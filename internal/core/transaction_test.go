package core

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

var SK []*rsa.PrivateKey
var PK []*rsa.PublicKey
var ADDR []Hash160

var TXID []Hash256
var USET *InMemUXTOSet

func PopulateTestData() {
	SK = []*rsa.PrivateKey{}
	PK = []*rsa.PublicKey{}
	ADDR = []Hash160{}
	TXID = []Hash256{}
	USET = NewUXTOSet()

	// 10 accounts
	for i := 0; i < 10; i++ {
		sk, _ := rsa.GenerateKey(rand.Reader, 512)
		pk := &sk.PublicKey
		addr := HashPubKey(pk)

		SK = append(SK, sk)
		PK = append(PK, pk)
		ADDR = append(ADDR, addr)
	}

	// 10 uxto, 2 per each account, 100 unit per tx
	for i := 0; i < 10; i++ {
		for j := 0; j < 2; j++ {
			uxto := NewUXTO(ADDR[i], 100)
			USET.Add(uxto)
			TXID = append(TXID, uxto.TxId)
		}
	}
}

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
	PopulateTestData()

	tx := Transaction{
		Ins: []*TxIn{
			{
				PrevTxId: TXID[0],
				N:        0,
				ScriptSig: ScriptSig{
					PK:        PK[0],
					Signature: nil,
				},
				Coinbase: nil,
			},
		},
		Outs: []*TxOut{
			{
				Value:        USET.First(TXID[0]).Value,
				ScriptPubKey: ScriptPubKey{PubKeyHash: ADDR[0]},
			},
		},
	}

	if err := tx.SignTxIn(USET.First(TXID[0]), SK[0]); err != nil {
		t.Fatalf("failed to sign txIn: %s", err)
	}

	if err := tx.VerifyTxIn(USET.First(TXID[0]), PK[0]); err != nil {
		t.Fatalf("failed to verify txIn: %s", err)
	}

	if err := tx.VerifyTxIn(USET.First(TXID[1]), PK[0]); err == nil {
		t.Fatalf("expected verification error; got nil")
	}

	if err := tx.Verify(USET); err != nil {
		t.Fatalf("failed to verify transaction: %s", err)
	}
}

func TestTransactionBuilder(t *testing.T) {
	PopulateTestData()

	txb := NewTransactionBuilder()

	tx := txb.
		AddInputFrom(USET.First(TXID[0]), PK[0]).
		AddOutput(50, ADDR[2]).
		AddChange(1).
		Sign(SK[0])

	if err := tx.Verify(USET); err != nil {
		t.Fatalf("failed to verify built transaction: %s", err)
	}
}

func TestNewCoinBaseTransaction(t *testing.T) {
	PopulateTestData()

	cb := NewCoinBaseTransaction([]byte("Coinbase!"), ADDR[0], 100, 1)

	if !cb.IsCoinbaseTx() {
		t.Fatalf("cb is not of type coinbase")
	}

	if err := cb.Verify(USET); err != nil {
		t.Fatalf("cb is not verified: %s", err)
	}
}
