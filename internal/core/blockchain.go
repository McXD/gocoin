package core

import (
	"errors"
	"fmt"
	"strings"
)

type UXTORecord struct {
	isCoinbase    bool
	indices       []uint32
	amounts       map[uint]uint32
	scriptPubKeys map[uint]*ScriptPubKey
}

type Blockchain struct {
	Genesis *Block
	Head    *Block
	blocks  map[Hash256]*Block      // TODO: persistence
	uxtos   map[Hash256]*UXTORecord // TODO: persistence
}

func NewBlockchain(pubKeyHash Hash160) *Blockchain {
	blocks := make(map[Hash256]*Block)
	uxtos := make(map[Hash256]*UXTORecord, 0)

	coinbaseTx := NewCoinbaseTx([]byte("GoCoin spawned!"), pubKeyHash)
	genesis := NewBlock(0, Hash256{}, []*Transaction{coinbaseTx})

	blocks[genesis.Hash] = genesis

	return &Blockchain{
		Genesis: genesis,
		Head:    genesis,
		blocks:  blocks,
		uxtos:   uxtos,
	}
}

// AddBlock scans the chain and insert the given block after its stated previous block./*
// TODO: address branching
func (bc *Blockchain) AddBlock(b *Block) error {
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

func (bc *Blockchain) IsSpent(txHash Hash256, n uint32) bool {
	for _, ind := range bc.uxtos[txHash].indices {
		if ind == n {
			return true
		}
	}

	return false
}

func (bc *Blockchain) updateUXTOSet(b *Block) {
	for _, tx := range b.Transactions {
		for _, txIn := range tx.In {
			// delete the indices
			// TODO
			_ = txIn
		}
	}
}

func (bc *Blockchain) String() string {
	sb := strings.Builder{}
	current := bc.Head

	for current.PrevBlockHash != [32]byte{} {
		sb.WriteString(fmt.Sprintf("%s\n", current))
		current = bc.blocks[current.PrevBlockHash]
	}

	return sb.String()
}
