package core

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestTransaction_IsCoinbaseTx(t *testing.T) {
	acct, _ := rsa.GenerateKey(rand.Reader, 512)
	coinbaseTx := NewCoinbaseTx([]byte("coinbase"), HashPubKey(&acct.PublicKey))

	txIn := TxIn{coinbaseTx.Hash, 0,
		ScriptSig{
			PubKey:    acct.PublicKey,
			Signature: nil,
		},
		nil,
	}

	txOut := TxOut{
		Value:        10,
		ScriptPubKey: ScriptPubKey{PubKeyHash: HashPubKey(&acct.PublicKey)},
	}

	nonCoinbaseTx := Transaction{
		Ins:  []*TxIn{&txIn},
		Outs: []*TxOut{&txOut},
	}

	nonCoinbaseTx.Sign(acct)

	if !coinbaseTx.IsCoinbaseTx() {
		t.Fatalf("conbaseTx.IsCoinbaseTx() = false; want true")
	}

	if nonCoinbaseTx.IsCoinbaseTx() {
		t.Fatalf("nonCoinbaseTx.IsCoinbaseTx() = true; want false")
	}
}

func TestTransaction_SignAndVerify(t *testing.T) {
	txIn := TxIn{
		Hash256{}, 0, ScriptSig{acct1.PublicKey, nil}, nil,
	}
	txOut := TxOut{
		10, ScriptPubKey{HashPubKey(&acct1.PublicKey)},
	}
	tx := Transaction{Hash256{}, []*TxIn{&txIn}, []*TxOut{&txOut}}

	if err := tx.Sign(acct1); err != nil {
		t.Fatalf("tx.Sign(acct1) errored: %s", err)
	}

	tx.SetHash()

	if err := tx.VerifySignature(); err != nil {
		t.Fatalf("tx.VerifySignature() = false; want true: %s", err)
	}
}
