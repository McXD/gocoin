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
	NextBlockHash Hash256
	Nonce         int
	Transactions  []*Transaction
	Bits          int
	MerkleRoot    Hash256
}

func (b *Block) verified() error {
	// TODO
	return nil
}

func (b *Block) String() string {
	return fmt.Sprintf("Block %d, SetHash=%s", b.Index, b.Hash)
}

type BlockBuilder struct {
	*Block
}

func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{&Block{}}
}

func (bb *BlockBuilder) SetIndex(index int) *BlockBuilder {
	bb.Index = index
	return bb
}

func (bb *BlockBuilder) SetTimeStamp(timestamp int64) *BlockBuilder {
	bb.Timestamp = timestamp
	return bb
}

func (bb *BlockBuilder) SetPrevBlockHash(hash Hash256) *BlockBuilder {
	bb.PrevBlockHash = hash
	return bb
}

func (bb *BlockBuilder) Now() *BlockBuilder {
	return bb.SetTimeStamp(time.Now().Unix())
}

func (bb *BlockBuilder) SetDifficulty(bits int) *BlockBuilder {
	bb.Bits = bits
	return bb
}

func (bb *BlockBuilder) AddTransaction(ctx *BlockchainInMem, tx *Transaction) (*BlockBuilder, error) {
	if ctx != nil {
		if err := bb.VerifyTransaction(ctx, tx); err != nil {
			return bb, fmt.Errorf("transaction verification failed: %w", err)
		}
	}

	bb.Transactions = append(bb.Transactions, tx)

	return bb, nil
}

func (bb *BlockBuilder) SetMerkleTreeRoot() *BlockBuilder {
	// TODO: merkle tree implementation

	var txHashes []Hash256
	var txHash Hash256
	var txHashesBytes [][]byte

	// type conversion
	for _, Hash := range txHashes {
		txHashesBytes = append(txHashesBytes, Hash[:])
	}

	for _, tx := range bb.Transactions {
		txHashes = append(txHashes, tx.Hash)
	}

	txHash = sha256.Sum256(bytes.Join(txHashesBytes, []byte{}))
	bb.MerkleRoot = txHash

	return bb
}

// Mine finds a Nonce so that the block's hash will be below the target.
// The TimeStamp will also be updated to reflect the time when the block is mined.
// This method DOES NOT ensure that all fields are properly set.
func (bb *BlockBuilder) Mine() *Block {
	var targetInt = big.NewInt(1)
	targetInt.Lsh(targetInt, uint(256-bb.Bits))
	targetInt.Add(targetInt, big.NewInt(-1))

	startTime := time.Now().Unix()

	for {
		bb.hash()
		hashInt := big.NewInt(0).SetBytes(bb.Hash[:])
		if hashInt.Cmp(targetInt) == -1 {
			break
		}
		bb.Nonce++
	}

	endTime := time.Now().Unix()
	bb.Timestamp = endTime

	log.WithFields(log.Fields{
		"index":     bb.Index,
		"timestamp": bb.Timestamp,
		"nonce":     bb.Nonce,
		"hash":      bb.Hash,
		"spent":     endTime - startTime,
	}).Info("POW calculated for Block")

	return bb.Block
}

/*
 * Concatenate all fields as `[]byte` and use SHA256 hash. Set the field with result.
 */
func (bb *BlockBuilder) hash() {
	header := bytes.Join([][]byte{
		[]byte(strconv.FormatInt(bb.Timestamp, 10)), // timestamp
		[]byte(strconv.Itoa(bb.Index)),              // index
		bb.Hash[:],                                  // SetHash
		bb.PrevBlockHash[:],                         // prev_hash
		[]byte(strconv.Itoa(bb.Nonce)),              // nonce
		bb.MerkleRoot[:],                            // merkle root
	}, []byte{})

	bb.Hash = sha256.Sum256(header)
}

func (bb *BlockBuilder) VerifyTransaction(ctx *BlockchainInMem, tx *Transaction) error {
	for _, txIn := range tx.Ins {
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
	if err := tx.VerifiedSignature(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (bb *BlockBuilder) Verified() (*BlockBuilder, error) {
	// TODO
	return bb, nil
}

func (bb *BlockBuilder) Build() *Block {
	return bb.Block
}
