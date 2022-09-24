package core

import (
	"errors"
	"fmt"
	"strings"
)

type Blockchain struct {
	Genesis *Block
	Head    *Block
	blocks  map[[32]byte]*Block // TODO: persistence
}

func NewBlockchain() *Blockchain {
	blocks := make(map[[32]byte]*Block)
	genesis := NewBlock(0, SHA256Hash{}, []*Transaction{NewCoinbaseTx()})
	blocks[genesis.Hash] = genesis

	return &Blockchain{
		Genesis: genesis,
		Head:    genesis,
		blocks:  blocks,
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

func (bc *Blockchain) String() string {
	sb := strings.Builder{}
	current := bc.Head

	for current.PrevBlockHash != [32]byte{} {
		sb.WriteString(fmt.Sprintf("%s\n", current))
		current = bc.blocks[current.PrevBlockHash]
	}

	return sb.String()
}
