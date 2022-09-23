package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

type Block struct {
	Timestamp     int64
	Index         int
	Hash          [32]byte
	PrevBlockHash [32]byte
	Nonce         int
	data          []byte

	// TODO: transactions
	// TODO: merkle root
	// TODO: difficulty
}

/*
 * Concatenate all fields as `[]byte` and use SHA256 hash. Set the field with result.
 */
func (b *Block) hash() {
	header := bytes.Join([][]byte{
		[]byte(strconv.FormatInt(b.Timestamp, 10)),
		[]byte(strconv.Itoa(b.Index)),
		b.Hash[:],
		b.PrevBlockHash[:],
		[]byte(strconv.Itoa(b.Nonce)),
		b.data,
	}, []byte{})

	b.Hash = sha256.Sum256(header)
}

// NewBlock returns a new `Block` without PoW hash (zero hash)./*
func NewBlock(index int, prevBlockHash [32]byte, data []byte) *Block {
	block := Block{
		Timestamp:     time.Now().Unix(),
		Index:         index,
		Hash:          [32]byte{},
		PrevBlockHash: prevBlockHash,
		Nonce:         0,
		data:          data,
	}

	block.hash()

	return &block
}

func (b *Block) String() string {
	return fmt.Sprintf("Block %d, hash=%x", b.Index, b.Hash)
}
