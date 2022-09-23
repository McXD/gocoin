package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"
)

type Block struct {
	Timestamp     int64
	Index         int
	Hash          [32]byte
	PrevBlockHash [32]byte
	Nonce         int
	Data          []byte
	Bits          int

	// TODO: merkle root
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
		b.Data,
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

	log.Printf("POW calculated for Block %d: nonce=%10d, hash=%x, spent=%4ds\n", b.Index, b.Nonce, b.Hash, b.Timestamp-start)
}

// NewBlock returns a new _valid_ block./*
func NewBlock(index int, prevBlockHash [32]byte, data []byte) *Block {
	block := Block{
		Timestamp:     time.Now().Unix(),
		Index:         index,
		Hash:          [32]byte{},
		PrevBlockHash: prevBlockHash,
		Nonce:         0,
		Data:          data,
		Bits:          20, // TODO: move to config
	}

	block.hashPoW()

	return &block
}

func (b *Block) String() string {
	return fmt.Sprintf("Block %d, hash=%x", b.Index, b.Hash)
}
