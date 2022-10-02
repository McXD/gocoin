package persistence

import (
	"gocoin/internal/core"
	"os"
	"testing"
)

func TestOpenAndWriteBlockFile(t *testing.T) {
	dir, _ := os.Getwd()
	t.Logf("CWD: %s", dir)

	bf, err := Open(0)
	if err != nil {
		t.Fatalf("canont open Blockfile: %s", err)
	}

	b1 := core.NewBlockBuilder().
		BaseOn(core.Hash256{}, -1).
		AddTransaction(core.NewCoinBaseTransaction([]byte("coinbase"), core.RandomHash160(), 100, 1)).
		SetBits(22).
		Build()

	if err := bf.WriteBlock(b1); err != nil {
		t.Fatalf("failed to write block: %s", err)
	}

	_ = bf.Close()

	bf, err = Open(0)
	if err != nil {
		t.Fatalf("canont re-open Blockfile: %s", err)
	}

	t.Logf("Latest Block: %-v", bf.blocks[len(bf.blocks)-1].Transactions[0])
}
