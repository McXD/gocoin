package core

import (
	"encoding/binary"
	"fmt"
	"github.com/cbergoon/merkletree"
	log "github.com/sirupsen/logrus"
	"math"
	"math/big"
	"time"
)

type BlockHeader struct {
	Time int64
	// Bits is a compact representation of the target value of PoW (i.e., number of leading zeros)
	Bits           uint8
	Nonce          uint32
	HashPrevBlock  Hash256
	HashMerkleRoot Hash256
}

func (header *BlockHeader) TargetValue() *big.Int {
	var targetInt = big.NewInt(1)
	targetInt.Lsh(targetInt, uint(256-int(header.Bits))).Add(targetInt, big.NewInt(-1))

	return targetInt
}

func (header *BlockHeader) Hash() Hash256 {
	data := make([]byte, 8+1+4+32+32)

	binary.BigEndian.PutUint64(data, uint64(header.Time))

	data[8] = header.Bits

	binary.BigEndian.PutUint32(data, header.Nonce)

	for i := 0; i < 32; i++ {
		data[i+8+1+4] = header.HashPrevBlock[i]
		data[i+8+1+4+32] = header.HashPrevBlock[i]
	}

	return HashTo256(data)
}

type Block struct {
	Hash   Hash256
	Height uint32
	BlockHeader
	Transactions []*Transaction
}

func (block *Block) CalculateMerkleRoot() (Hash256, error) {
	var leaves []merkletree.Content
	for _, tx := range block.Transactions {
		leaves = append(leaves, tx)
	}

	if tree, err := merkletree.NewTree(leaves); err != nil {
		return Hash256{}, err
	} else {
		return Hash256FromSlice(tree.MerkleRoot()), nil
	}
}

func (block *Block) ContainsTransaction(txId Hash256) bool {
	for _, tx := range block.Transactions {
		if tx.Hash() == txId {
			return true
		}
	}

	return false
}

func (block *Block) CalculateFee(uSet UXTOSet) (fee uint32, overflow bool) {
	var inValue, outValue uint32

	for _, tx := range block.Transactions {
		if !tx.IsCoinbaseTx() {
			tmp, _ := tx.CalculateOutValue() // ignore individual overflow, as this will be caught by individual tx verification
			outValue += tmp

			for _, txIn := range tx.Ins {
				inValue += uSet.GetUXTO(txIn.PrevTxId, txIn.N).Value
			}
		}
	}

	fee = inValue - outValue
	overflow = inValue < outValue

	return fee, overflow
}

func (block *Block) Verify(uSet UXTOSet, currentBits uint8, timeWindow int64, blockReward uint32) error {
	// verify header
	if math.Abs(float64(time.Now().Unix()-block.Time)) > float64(timeWindow) {
		return fmt.Errorf("invalid timestamp")
	}

	if block.Bits != currentBits {
		return fmt.Errorf("invalid Bits")
	}

	if mr, err := block.CalculateMerkleRoot(); err != nil {
		log.Warnf("Error calculating MerkleRoot for block %X: %s", block.Hash[:], err)
	} else {
		if block.HashMerkleRoot != mr {
			return fmt.Errorf("invalid MerkleRoot")
		}
	}

	// verify PoW
	if block.Hash != block.BlockHeader.Hash() {
		return fmt.Errorf("header does not match PoW")
	}

	if block.Hash.Int().Cmp(block.TargetValue()) == 1 {
		return fmt.Errorf("PoW does not meet difficulty")
	}

	// verify transactions
	if len(block.Transactions) == 0 {
		return fmt.Errorf("block contains zero transaction")
	}

	if !block.Transactions[0].IsCoinbaseTx() {
		return fmt.Errorf("first transaction is not coinbase")
	}

	for _, tx := range block.Transactions {
		if err := tx.Verify(uSet); err != nil {
			return fmt.Errorf("failed to verify transaction %s: %w", tx.Hash(), err)
		}
	}

	// verify balance
	if fee, overflow := block.CalculateFee(uSet); overflow { // this should not happen as we have verified each transaction
		return fmt.Errorf("transaction fee is negative")
	} else {
		coinbase, _ := block.Transactions[0].CalculateOutValue()
		if coinbase > blockReward+fee {
			fmt.Printf("reward %d; fee %d; coinbase %d\n", blockReward, fee, coinbase)

			return fmt.Errorf("invalid output value in coinbase")
		}
	}

	return nil
}

type BlockBuilder struct {
	*Block
}

func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{
		&Block{
			Hash:         Hash256{},
			Height:       0,
			BlockHeader:  BlockHeader{},
			Transactions: []*Transaction{},
		},
	}
}

func (bb *BlockBuilder) BaseOn(prevBlockHash Hash256, prevBlockHeight uint32) *BlockBuilder {
	bb.HashPrevBlock = prevBlockHash
	bb.Height = prevBlockHeight + 1
	return bb
}

func (bb *BlockBuilder) SetBits(bits uint8) *BlockBuilder {
	bb.Bits = bits
	return bb
}

func (bb *BlockBuilder) AddTransaction(tx *Transaction) *BlockBuilder {
	bb.Transactions = append(bb.Transactions, tx)
	return bb
}

// Build returns a full block
// A block is built through the following process:
//	1. set the timestamp to current time
//	2. calculate and set the HashMerkleTree in the block header
//	3. set the nonce until the header hashes to lower than value implied by Bits (PoW)
//  4. set the block hash
func (bb *BlockBuilder) Build() *Block {
	bb.Time = time.Now().Unix()

	if merkleRoot, err := bb.CalculateMerkleRoot(); err != nil {
		log.Warn(err)
	} else {
		bb.HashMerkleRoot = merkleRoot
	}

	target := bb.TargetValue()
	for {
		if bb.BlockHeader.Hash().Int().Cmp(target) == -1 {
			break
		}

		bb.Nonce++
	}

	bb.Hash = bb.BlockHeader.Hash()
	return bb.Block
}
