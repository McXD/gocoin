package core

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

var sk1, _ = rsa.GenerateKey(rand.Reader, 512)
var addr1 = HashPubKey(&sk1.PublicKey)
var uxto1 = NewUXTO(addr1, 100)
var sk2, _ = rsa.GenerateKey(rand.Reader, 512)
var addr2 = HashPubKey(&sk1.PublicKey)
var uxto2 = NewUXTO(addr1, 100)

func NewUXTO(to Hash160, v uint32) *UXTO {
	return &UXTO{
		TxId: RandomHash256(),
		N:    0,
		TxOut: &TxOut{
			Value:        v,
			ScriptPubKey: ScriptPubKey{to},
		},
	}
}

func TestTransaction_SignAndVerify(t *testing.T) {
	tx := Transaction{
		Ins: []*TxIn{
			{
				PrevTxId: uxto1.TxId,
				N:        uxto1.N,
				ScriptSig: ScriptSig{
					PK:        &sk1.PublicKey,
					Signature: nil,
				},
				Coinbase: nil,
			},
		},
		Outs: []*TxOut{
			{
				Value:        uxto1.Value,
				ScriptPubKey: ScriptPubKey{PubKeyHash: addr1},
			},
		},
	}

	if err := tx.SignTxIn(uxto1, sk1); err != nil {
		t.Fatalf("failed to sign txIn: %s", err)
	}

	if err := tx.VerifyTxIn(uxto1, &sk1.PublicKey); err != nil {
		t.Fatalf("failed to verify txIn: %s", err)
	}

	if err := tx.VerifyTxIn(uxto2, &sk1.PublicKey); err == nil {
		t.Fatalf("expected verification error; got nil")
	}

	if err := tx.Verify(map[Hash256]*UXTO{uxto1.TxId: uxto1}); err != nil {
		t.Fatalf("failed to verify transaction: %s", err)
	}
}

func TestTransactionBuilder(t *testing.T) {
	txb := NewTransactionBuilder()

	tx := txb.
		AddInputFrom(uxto1, &sk1.PublicKey).
		AddOutput(50, addr2).
		AddChange(1).
		Sign(sk1)

	if err := tx.Verify(map[Hash256]*UXTO{uxto1.TxId: uxto1}); err != nil {
		t.Fatalf("failed to verify built transaction: %s", err)
	}
}

func TestNewCoinBaseTransaction(t *testing.T) {
	cb := NewCoinBaseTransaction([]byte("Coinbase!"), addr1, 100, 1)

	if !cb.IsCoinbaseTx() {
		t.Fatalf("cb is not of type coinbase")
	}

	if err := cb.Verify(make(map[Hash256]*UXTO)); err != nil {
		t.Fatalf("cb is not verified: %s", err)
	}
}
