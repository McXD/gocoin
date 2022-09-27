package core

import (
	"fmt"
	"testing"
)

func TestBlockBuilder(t *testing.T) {
	bb := NewBlockBuilder()
	bb.
		SetDifficulty(20).
		SetIndex(12).
		SetPrevBlockHash(RandomHash256()).
		Now()

	txb := NewTransactionBuilder()
	tx, _ := txb.
		AddInput(RandomHash256(), 0, acct1.PublicKey).
		AddOutput(1000, HashPubKey(&acct1.PublicKey)).
		Sign(acct1)

	bb.AddTransaction(nil, tx)
	bb.SetMerkleTreeRoot()
	b := bb.Mine()

	fmt.Printf("Block: %+v\n", *b)
}
