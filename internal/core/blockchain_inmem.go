package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type UXTORecord struct {
	isCoinbase    bool
	indices       []uint32
	amounts       map[uint]uint32
	scriptPubKeys map[uint]*ScriptPubKey
}

type BlockchainInMem struct {
	Genesis *Block
	Head    *Block
	blocks  map[Hash256]*Block      // TODO: persistence
	uxtos   map[Hash256]*UXTORecord // TODO: persistence
}

// NewBlockchain creates a new blockchain with the fist transaction being a coinbase transaction paid to the `pubKeyHash`.
// This transaction occupied the entire genesis block.
func NewBlockchain(pubKeyHash Hash160) *BlockchainInMem {
	blocks := make(map[Hash256]*Block)
	uxtos := make(map[Hash256]*UXTORecord, 0)
	coinbaseTx := NewCoinbaseTx([]byte("GoCoin spawned!"), pubKeyHash)

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
	}
}

// AddBlock scans the chain and insert the given block after its stated previous block./*
// TODO: address branching
func (bc *BlockchainInMem) AddBlock(b *Block) error {
	if err := b.verified(); err != nil {
		return fmt.Errorf("failed to add block: %w", err)
	}

	// check if the block is on the longest chain, if so, change the head
	if bc.blocks[b.PrevBlockHash] != nil {
		bc.blocks[b.Hash] = b
		bc.Head = b

		return nil
	}

	return errors.New("cannot find parent block")
}

func (bc *BlockchainInMem) IsSpent(txHash Hash256, n uint32) bool {
	for _, ind := range bc.uxtos[txHash].indices {
		if ind == n {
			return true
		}
	}

	return false
}

func (bc *BlockchainInMem) updateUXTOSet(b *Block) {
	for _, tx := range b.Transactions {
		for _, txIn := range tx.Ins {
			// delete the indices
			// TODO
			_ = txIn
		}
	}
}

// AddTransaction verifies a transaction and if valid, add it to the mempool;
// If invalid, an error is returned
func (bc *BlockchainInMem) AddTransaction(tx *Transaction) error {
	return nil
}

// Mine collects a batch of transactions from the mempool and tries to generate a block.
// The mining process is executed in a separate go routine.
func (bc *BlockchainInMem) Mine() *Block {
	return nil
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
