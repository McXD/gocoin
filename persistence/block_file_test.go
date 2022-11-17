package persistence

import (
	"github.com/davecgh/go-spew/spew"
	core2 "gocoin/core"
	"testing"
)

func TestOpenAndWriteBlockFile(t *testing.T) {
	PopulateTestData()

	bf, err := NewBlockFile(0)
	if err != nil {
		t.Fatalf("canont open Blockfile: %s", err)
	}

	tx0 := core2.NewCoinBaseTransaction([]byte("coinbase"), core2.RandomHash160(), 100, 1)
	tx1 := core2.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[1]), PK[0]).
		AddInputFrom(USET.First(TXID[0]), PK[0]).
		AddOutput(100, ADDR[2]).
		AddChange(50).
		Sign(SK[0])
	tx2 := core2.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[2]), PK[1]).
		AddInputFrom(USET.First(TXID[3]), PK[1]).
		AddOutput(100, ADDR[5]).
		AddChange(50).
		Sign(SK[0])
	b := core2.NewBlockBuilder().
		BaseOn(core2.EmptyHash256(), 1000).
		SetBits(20).
		AddTransaction(tx0).
		AddTransaction(tx1).
		AddTransaction(tx2).
		Build()
	spent := []*core2.UXTO{
		USET.First(TXID[0]),
		USET.First(TXID[1]),
		USET.First(TXID[2]),
		USET.First(TXID[3]),
	}

	if err := bf.WriteBlock(b, spent); err != nil {
		t.Fatalf("failed to write block: %s", err)
	}

	_ = bf.Close()

	bf, err = NewBlockFile(0)
	if err != nil {
		t.Fatalf("canont re-open Blockfile: %s", err)
	}

	t.Logf(spew.Sdump(bf))
}
