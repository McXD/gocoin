package core

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type UXTORecord struct {
	TxHash        Hash256
	IsCoinbase    bool
	Indices       []uint32 // keep track of who is in
	Amounts       map[uint32]uint32
	ScriptPubKeys map[uint32]*ScriptPubKey
}

func (u *UXTORecord) AmountOf(index uint32) uint32 {
	return u.Amounts[index]
}

func (u *UXTORecord) ScriptPubKeyOf(index uint32) *ScriptPubKey {
	return u.ScriptPubKeys[index]
}

func (u *UXTORecord) GetTxOut(index uint32) *TxOut {
	return &TxOut{
		Value:        u.AmountOf(index),
		ScriptPubKey: *u.ScriptPubKeyOf(index),
	}
}

func NewUXTORecord(tx *Transaction) *UXTORecord {
	indices := make([]uint32, len(tx.Outs))
	for i, _ := range indices {
		indices[i] = uint32(i)
	}

	amounts := make(map[uint32]uint32)
	scripts := make(map[uint32]*ScriptPubKey)

	for i, out := range tx.Outs {
		amounts[uint32(i)] = out.Value
		scripts[uint32(i)] = &out.ScriptPubKey
	}

	return &UXTORecord{
		TxHash:        tx.Hash,
		IsCoinbase:    tx.IsCoinbaseTx(),
		Indices:       indices,
		Amounts:       amounts,
		ScriptPubKeys: scripts,
	}
}

func (u *UXTORecord) Consume(txIn *TxIn) {
	if u.TxHash != txIn.Hash {
		return
	}

	// delete the index
	for i, ind := range u.Indices {
		if ind == txIn.N {
			u.Indices = append(u.Indices[:i], u.Indices[i+1:]...)
		}
	}

	// zero-out the amount
	u.Amounts[txIn.N] = 0 // no partial-spend for a UXTO; all or nothing
}

func (u *UXTORecord) IsEmpty() bool {
	return len(u.Indices) == 0
}

type BlockchainInMem struct {
	Genesis *Block
	Head    *Block
	blocks  map[Hash256]*Block      // blockHash -> Block
	uxtos   map[Hash256]*UXTORecord // txHash -> UXTO record
	mempool []*Transaction
}

// NewBlockchain creates a new blockchain with the fist transaction being a coinbase transaction paid to the `pubKeyHash`.
// This transaction occupied the entire genesis block.
func NewBlockchain(pubKeyHash Hash160) *BlockchainInMem {
	blocks := make(map[Hash256]*Block)
	uxtos := make(map[Hash256]*UXTORecord)
	coinbaseTx := NewCoinbaseTx([]byte("GoCoin spawned!"), pubKeyHash)
	mempool := []*Transaction{}

	bb := NewBlockBuilder().
		Now().
		SetDifficulty(20).
		SetPrevBlockHash(Hash256{}).
		SetTimeStamp(time.Now().Unix()).
		SetIndex(0)
	bb.AddTransaction(nil, coinbaseTx)
	bb.SetMerkleTreeRoot()

	genesis := bb.Mine()

	blocks[genesis.Hash] = genesis
	//uxtos[coinbaseTx.Hash]

	return &BlockchainInMem{
		Genesis: genesis,
		Head:    genesis,
		blocks:  blocks,
		uxtos:   uxtos,
		mempool: mempool,
	}
}

func (bc *BlockchainInMem) IsNotSpent(txIn *TxIn) bool {
	uxto := bc.uxtos[txIn.Hash]
	if uxto == nil {
		return false
	}

	for _, ind := range uxto.Indices {
		if ind == txIn.N {
			return true
		}
	}

	return false
}

func (bc *BlockchainInMem) VerifyBlock(b *Block) error {
	// metadata

	return nil
}

// AddBlock scans the chain and insert the given block after its stated previous block./*
// TODO: address branching
func (bc *BlockchainInMem) AddBlock(b *Block) error {
	// WARNING: assume chain has no fork, no orphan

	// add to block storage
	bc.blocks[b.Hash] = b

	// update head
	b.PrevBlockHash = bc.Head.Hash
	bc.Head.NextBlockHash = b.Hash
	bc.Head = b

	// update uxto
	for _, tx := range b.Transactions {
		// mark all inputs as spent
		if !tx.IsCoinbaseTx() { // skip for coinbase transaction
			for _, txIn := range tx.Ins {
				bc.uxtos[txIn.Hash].Consume(txIn)
			}
		}

		// add new unspent output
		bc.uxtos[tx.Hash] = NewUXTORecord(tx)
	}

	log.WithFields(log.Fields{
		"preBlockHash":     b.PrevBlockHash,
		"currentBlockHash": b.Hash,
		"height":           b.Index,
	}).Info("Appended new block")

	return nil
}

func (bc *BlockchainInMem) VerifyTransaction(tx *Transaction) error {
	// signature
	if err := tx.VerifySignature(); err != nil {
		return fmt.Errorf("cannot verify signature: %w", err)
	}

	for _, txIn := range tx.Ins {
		uxto := bc.uxtos[txIn.Hash]

		// input is not spent
		if uxto == nil || !Contains(uxto.Indices, txIn.N) {
			return fmt.Errorf("uxto is spent: id=%x, index=%d", txIn.Hash[:], txIn.N)
		}

		// inputs are spendable by the pubKey
		uxto.GetTxOut(txIn.N).CanBeSpentBy(txIn.PubKey)
	}

	// TODO: balance

	return nil
}

func (bc *BlockchainInMem) AddTransaction(tx *Transaction) error {
	bc.mempool = append(bc.mempool, tx)
	return nil
}

func (bc *BlockchainInMem) GenerateBlockTo(pubKeyHash Hash160, txs []*Transaction) *Block {
	// TODO: collect transaction fee
	bb := NewBlockBuilder()
	bb.
		SetDifficulty(bc.Head.Bits). // TODO: dynamic
		SetPrevBlockHash(bc.Head.Hash).
		SetIndex(bc.Head.Index + 1).
		Now()

	coinbase := NewCoinbaseTx([]byte("coinbase"), pubKeyHash)
	bb.AddTransaction(nil, coinbase)
	for _, tx := range txs {
		bb.AddTransaction(nil, tx)
	}
	bb.SetMerkleTreeRoot()

	return bb.Mine()
}

func (bc *BlockchainInMem) String() string {
	sb := strings.Builder{}
	current := bc.Head

	for current.PrevBlockHash != [32]byte{} {
		sb.WriteString(fmt.Sprintf("%s\n", current))
		current = bc.blocks[current.PrevBlockHash]
	}

	return sb.String()
}
