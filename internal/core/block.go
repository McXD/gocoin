package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strconv"
	"time"
)

type Block struct {
	Timestamp     int64
	Index         int
	Hash          Hash256
	PrevBlockHash Hash256
	Nonce         int
	Transactions  []*Transaction
	Bits          int
	MerkleRoot    Hash256
}

// TODO: merkle tree
func (b *Block) hashTxs() Hash256 {
	var txHashes []Hash256
	var txHash Hash256
	var txHashesBytes [][]byte

	// type conversion
	for _, Hash := range txHashes {
		txHashesBytes = append(txHashesBytes, Hash[:])
	}

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Hash)
	}
	txHash = sha256.Sum256(bytes.Join(txHashesBytes, []byte{}))

	return txHash
}

/*
 * Concatenate all fields as `[]byte` and use SHA256 hash. Set the field with result.
 */
func (b *Block) hash() {
	// TODO: do not repeat hashes
	merkleRoot := b.hashTxs()
	b.MerkleRoot = merkleRoot

	header := bytes.Join([][]byte{
		[]byte(strconv.FormatInt(b.Timestamp, 10)), // timestamp
		[]byte(strconv.Itoa(b.Index)),              // index
		b.Hash[:],                                  // SetHash
		b.PrevBlockHash[:],                         // prev_hash
		[]byte(strconv.Itoa(b.Nonce)),              // nonce
		merkleRoot[:],                              // merkle root
	}, []byte{})

	b.Hash = sha256.Sum256(header)
}

// hashPoW finds a Nonce so that the block's hash will be below the target.
// The TimeStamp will also be updated to reflect the time when the block is mined.
func (b *Block) hashPoW() {
	var targetInt = big.NewInt(1)
	targetInt.Lsh(targetInt, uint(256-b.Bits))
	targetInt.Add(targetInt, big.NewInt(-1))

	var start = time.Now().Unix() // starting time

	b.hash() // SetHash once to fill the initial value
	for hashInt := big.NewInt(0).SetBytes(b.Hash[:]); hashInt.Cmp(targetInt) != -1; b.Nonce += 1 {
		//log.Printf("Calculating POW for Block %d: nonce=%d, SetHash=%x, target=%x, comp=%d\n", b.Index, b.Nonce, hashInt, targetInt, hashInt.Cmp(targetInt))
		b.hash()
		hashInt.SetBytes(b.Hash[:])
	}

	b.Timestamp = time.Now().Unix()

	log.WithFields(log.Fields{
		"index":     b.Index,
		"timestamp": b.Timestamp,
		"nonce":     b.Nonce,
		"hash":      b.Hash,
		"spent":     b.Timestamp - start,
	}).Info("POW calculated for Block")
}

// NewBlock returns a new _valid_ block./*
func NewBlock(index int, prevBlockHash [32]byte, transactions []*Transaction) *Block {
	block := Block{
		Timestamp:     time.Now().Unix(),
		Index:         index,
		Hash:          [32]byte{},
		PrevBlockHash: prevBlockHash,
		Nonce:         0,
		Transactions:  transactions,
		Bits:          20, // TODO: move to config
	}

	block.hashPoW()

	return &block
}

func (b *Block) VerifyTransaction(ctx *Blockchain, tx *Transaction) error {
	for _, txIn := range tx.In {
		// verify that the referenced transaction output is not spent
		if ctx.IsSpent(txIn.Hash, txIn.N) {
			return fmt.Errorf("transaction is spent: id=%x, n=%d", txIn.Hash, txIn.N)
		}

		// verify that public key is correct
		// the key is read from UXTO's ScriptPubKey
		if utxo, _ := ctx.uxtos[txIn.Hash]; utxo.scriptPubKeys[uint(txIn.N)].PubKeyHash != HashPubKey(&txIn.PubKey) {
			return fmt.Errorf("unmached pubKeyHash: given=%x, want=%x", HashPubKey(&txIn.PubKey), utxo.scriptPubKeys[uint(txIn.N)].PubKeyHash)
		}
	}

	// verify that signature is correct
	if !tx.VerifiedSignature() {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func (b *Block) AddTransaction(ctx *Blockchain, tx *Transaction) error {
	if err := b.VerifyTransaction(ctx, tx); err != nil {
		return fmt.Errorf("cannot add transaction %x to block %d: %w", tx.Hash, b.Index, err)
	}

	b.Transactions = append(b.Transactions, tx)

	return nil
}

func (b *Block) verified() error {
	return nil
}

func (b *Block) String() string {
	return fmt.Sprintf("Block %d, SetHash=%s", b.Index, b.Hash)
}
