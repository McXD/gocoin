package core

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
)

var acct1, _ = rsa.GenerateKey(rand.Reader, 512)
var acct2, _ = rsa.GenerateKey(rand.Reader, 512)

func TestBlockchainInMem_GenerateBlockTo(t *testing.T) {
	bc := NewBlockchain(HashPubKey(&acct1.PublicKey))
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{}))
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{}))
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{}))
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{}))
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{}))
}

func TestBlockchainInMem_VerifyTransaction(t *testing.T) {
	bc := NewBlockchain(HashPubKey(&acct1.PublicKey))
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{}))

	txb := NewTransactionBuilder()
	txb.AddInputFrom(bc.Head.Transactions[0], acct1.PublicKey)
	txb.AddOutput(10, HashPubKey(&acct2.PublicKey))
	tx1, _ := txb.Sign(acct1)

	if err := bc.VerifyTransaction(tx1); err != nil {
		t.Fatalf("bc.VerifyTransaction(tx1) = false; want true: %s", err)
	}

	// Mine the block
	bc.AddBlock(bc.GenerateBlockTo(HashPubKey(&acct1.PublicKey), []*Transaction{tx1}))

	txb = NewTransactionBuilder()
	txb.AddInputFrom(bc.Head.Transactions[0], acct1.PublicKey)
	txb.AddOutput(10, HashPubKey(&acct2.PublicKey))
	txDoubleSpend, _ := txb.Sign(acct1)

	if err := bc.VerifyTransaction(txDoubleSpend); err == nil {
		t.Fatalf("bc.VerifyTransaction(tx1) = true; want false")
	} else {
		fmt.Printf("tx.VerifyTransaction(txDoubleSpend) erred: %s\n", err)
	}
}
