package core

import (
	"crypto/rsa"
	"testing"
)

func NewTransaction(uxto *UXTO, sk *rsa.PrivateKey, to Hash160, value, fee uint32) *Transaction {
	return NewTransactionBuilder().
		AddInputFrom(uxto, &sk.PublicKey).
		AddOutput(value, to).
		AddChange(fee).
		Sign(sk)
}

func TestBlock_BuildAndVerify(t *testing.T) {
	PopulateTestData()

	coinbaseNoFee := NewCoinBaseTransaction([]byte("coinbaseNoFee"), ADDR[5], 100, 0)
	tx1 := NewTransaction(USET.First(TXID[0]), SK[0], ADDR[3], 60, 0)
	tx2 := NewTransaction(USET.First(TXID[1]), SK[0], ADDR[3], 60, 0)
	txInvalid := NewTransaction(USET.First(TXID[3]), SK[0], ADDR[4], 60, 0)

	// valid transaction
	b := NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(20).
		AddTransaction(coinbaseNoFee).
		AddTransaction(tx1).
		AddTransaction(tx2).
		Build()

	if err := b.Verify(USET, 20, 1000, 100); err != nil {
		t.Fatalf("failed to verify block: %s", err)
	}

	// block with invalid transactions
	b = NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(20).
		AddTransaction(coinbaseNoFee).
		AddTransaction(tx1).
		AddTransaction(tx2).
		AddTransaction(txInvalid).
		Build()

	if err := b.Verify(USET, 20, 1000, 100); err == nil {
		t.Fatalf("verification passed; expected transaction validation error")
	} else {
		t.Log(err)
	}

	// block with incorrect header
	b = NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(10).
		AddTransaction(coinbaseNoFee).
		Build()

	if err := b.Verify(USET, 20, 1000, 100); err == nil {
		t.Fatalf("verification passed; expected Bits error")
	} else {
		t.Log(err)
	}

	// block contains a transaction not included in merkle root
	b = NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(20).
		AddTransaction(coinbaseNoFee).
		Build()

	b.Transactions = append(b.Transactions, tx1)

	if err := b.Verify(USET, 20, 1000, 100); err == nil {
		t.Fatalf("verification passed; expected invalid merkle root")
	} else {
		t.Log(err)
	}

	// block contains zero transactions
	b = NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(20).
		Build()

	if err := b.Verify(USET, 20, 1000, 100); err == nil {
		t.Fatalf("verification passed; expected no transaction found")
	} else {
		t.Log(err)
	}

	// block contains no coinbase
	b = NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(20).
		AddTransaction(tx1).
		Build()

	if err := b.Verify(USET, 20, 1000, 100); err == nil {
		t.Fatalf("verification passed; expected no coinbase transaction")
	} else {
		t.Log(err)
	}

	// balance not matched
	txPayFee := NewTransaction(USET.First(TXID[0]), SK[0], ADDR[3], 50, 60)
	coinbaseWithFee := NewCoinBaseTransaction([]byte("coinbase"), ADDR[0], 100, 20)
	b = NewBlockBuilder().
		BaseOn(Hash256{}, 0).
		SetBits(20).
		AddTransaction(coinbaseWithFee).
		AddTransaction(txPayFee).
		Build()

	if err := b.Verify(USET, 20, 1000, 100); err == nil {
		t.Fatalf("verification passed; expected invalid coinbase")
	} else {
		t.Log(err)
	}
}
