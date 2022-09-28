package wallet

import (
	"gocoin/internal/core"
	"testing"
)

func TestWallet_ProcessTransaction(t *testing.T) {
	var w = NewWallet()

	var wAddr1 = w.Addresses[0]
	var wKey1 = w.Keys[wAddr1]
	var wAddr2 = w.NewAddress()

	// wAddr1: 100
	coinbase := core.NewCoinbaseTx([]byte("coinbase"), wAddr1, 100)

	w.ProcessTransaction(coinbase)
	if b1 := w.Balances[wAddr1]; b1 != 100 {
		t.Fatalf("Balance of wAddr1 is %d; want %d", b1, 100)
	}

	txb := core.NewTransactionBuilder()
	txb.AddInputFrom(coinbase, wKey1.PublicKey)
	txb.AddOutput(50, wAddr2)
	txb.AddOutput(50, wAddr1) // changes
	// pay 50 from wAddr1 to wAddr2
	tx1, _ := txb.Sign(wKey1)
	tx1.SetHash()

	w.ProcessTransaction(tx1)

	if b1 := w.Balances[wAddr1]; b1 != 50 {
		t.Fatalf("Balance of wAddr1 is %d; want %d", b1, 50)
	}

	if b2 := w.Balances[wAddr1]; b2 != 50 {
		t.Fatalf("Balance of wAddr2 is %d; want %d", b2, 50)
	}
}

func TestWallet_Connect(t *testing.T) {
	w := NewWallet()
	bc := core.NewBlockchain(w.Addresses[0])
	w.Connect(bc)

	if b := w.Balance(w.Addresses[0]); b != core.REWARD {
		t.Fatalf("Initial balance is %d; want %d", b, core.REWARD)
	}
}

func TestWallet_Send(t *testing.T) {
	w := NewWallet()
	bc := core.NewBlockchain(w.Addresses[0])
	w.Connect(bc)

	w.NewAddress()
	w.NewAddress()

	if _, err := w.Send(w.Addresses[0], w.Addresses[1], 100); err != nil {
		t.Fatalf("failed to send transaction: %s", err)
	} else {
		b := bc.Mine(w.Addresses[2])
		bc.AddBlock(b)
		w.ProcessBlock(b)

		if b := w.Balance(w.Addresses[0]); b != core.REWARD-100 {
			t.Fatalf("Address0 balance is %d; want %d", b, core.REWARD-100)
		}

		if b := w.Balance(w.Addresses[1]); b != 100 {
			t.Fatalf("Address0 balance is %d; want %d", b, 100)
		}
	}
}
