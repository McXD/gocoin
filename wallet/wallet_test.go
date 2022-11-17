package wallet

import (
	"gocoin/core"
	"testing"
)

func TestWallet_ProcessTransaction(t *testing.T) {
	var w = NewInMemWallet()

	var wAddr1 = w.Addresses[0]
	var wKey1 = w.Keys[wAddr1]
	var wAddr2 = w.NewAddress()

	// wAddr1: 100
	coinbase := core.NewCoinBaseTransaction([]byte("coinbase"), wAddr1, 100, 0)

	w.ProcessTransaction(coinbase)
	if b1 := w.Balances[wAddr1]; b1 != 100 {
		t.Fatalf("GetBalance of wAddr1 is %d; want %d", b1, 100)
	}

	txb := core.NewTransactionBuilder()
	txb.AddInputFrom(&core.UXTO{
		TxId:  coinbase.Hash(),
		N:     0,
		TxOut: coinbase.Outs[0],
	}, &wKey1.PublicKey)
	txb.AddOutput(50, wAddr2)

	// pay 50 from wAddr1 to wAddr2
	tx1 := txb.Sign(wKey1)
	w.ProcessTransaction(tx1)

	if b1 := w.Balances[wAddr1]; b1 != 50 {
		t.Fatalf("GetBalance of wAddr1 is %d; want %d", b1, 50)
	}

	if b2 := w.Balances[wAddr1]; b2 != 50 {
		t.Fatalf("GetBalance of wAddr2 is %d; want %d", b2, 50)
	}
}

func TestWallet_CreateTransaction(t *testing.T) {
	w := NewInMemWallet()
	w.NewAddress()
	w.NewAddress()

	// wAddr0: 100
	coinbase := core.NewCoinBaseTransaction([]byte("coinbase"), w.Addresses[0], 100, 0)
	w.ProcessTransaction(coinbase)

	if _, err := w.CreateTransaction(w.Addresses[0], w.Addresses[1], 100, 0); err != nil {
		t.Fatalf("failed to create transaction: %s", err)
	}
}
