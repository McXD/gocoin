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
	Hash          SHA256Hash
	PrevBlockHash SHA256Hash
	Nonce         int
	Transactions  []*Transaction
	Bits          int
	MerkleRoot    SHA256Hash
}

// TODO: merkle tree
func (b *Block) hashTxs() SHA256Hash {
	var txHashes []SHA256Hash
	var txHash SHA256Hash
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
		b.Hash[:],                                  // hash
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

	b.hash() // hash once to fill the initial value
	for hashInt := big.NewInt(0).SetBytes(b.Hash[:]); hashInt.Cmp(targetInt) != -1; b.Nonce += 1 {
		//log.Printf("Calculating POW for Block %d: nonce=%d, hash=%x, target=%x, comp=%d\n", b.Index, b.Nonce, hashInt, targetInt, hashInt.Cmp(targetInt))
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

// Verify in a block's context (PoW and transactions)
func (b *Block) verified() error {
	// TODO
	return nil
}

func (b *Block) String() string {
	return fmt.Sprintf("Block %d, hash=%s", b.Index, b.Hash)
}
