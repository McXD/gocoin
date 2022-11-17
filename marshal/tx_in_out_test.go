package marshal

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"github.com/davecgh/go-spew/spew"
	core2 "gocoin/core"
	"reflect"
	"testing"
)

func TestDeserializeScriptSig(t *testing.T) {
	sk, _ := rsa.GenerateKey(rand.Reader, 512)
	digest := core2.RandomHash256()
	sig, _ := sk.Sign(rand.Reader, digest[:], crypto.SHA256)

	ss := &core2.ScriptSig{
		PK:        &sk.PublicKey,
		Signature: sig,
	}

	var buf []byte
	buf = SerializeScriptSig(ss)
	ssDes := DeserializeScriptSig(buf)

	t.Logf("%-v", ss)
	t.Logf("%-v", ssDes)

	if !reflect.DeepEqual(ss, ssDes) {
		t.Fatalf("Objects are not equal")
	}
}

func TestDeserializeTxIn(t *testing.T) {
	sk, _ := rsa.GenerateKey(rand.Reader, 512)
	digest := core2.RandomHash256()
	sig, _ := sk.Sign(rand.Reader, digest[:], crypto.SHA256)

	ss := &core2.ScriptSig{
		PK:        &sk.PublicKey,
		Signature: sig,
	}

	txIn := &core2.TxIn{
		PrevTxId:  core2.RandomHash256(),
		N:         0,
		ScriptSig: *ss,
		Coinbase:  nil,
	}

	var buf []byte
	buf = SerializeTxIn(txIn)
	txInDes := DeserializeTxIn(buf)

	t.Logf("%-v", txIn)
	t.Logf("%-v", txInDes)

	if !reflect.DeepEqual(txIn, txInDes) {
		t.Errorf("Objects are not equal")
	}

	// --- coinbase input ---

	txIn = &core2.TxIn{
		PrevTxId: [32]byte{},
		N:        0,
		Coinbase: []byte("coinbase!!!!!"),
	}

	buf = SerializeTxIn(txIn)
	txInDes = DeserializeTxIn(buf)

	t.Logf("%-v", txIn)
	t.Logf("%-v", txInDes)

	if !reflect.DeepEqual(txIn, txInDes) {
		t.Errorf("Objects are not equal")
	}
}

func TestDeserializeTxOut(t *testing.T) {
	txOut := &core2.TxOut{
		Value: 100000,
		ScriptPubKey: core2.ScriptPubKey{
			PubKeyHash: core2.RandomHash160(),
		},
	}

	buf := SerializeTxOut(txOut)
	txOutDes := DeserializeTxOut(buf)

	t.Logf("%-v", txOut)
	t.Logf("%-v", txOutDes)

	if !reflect.DeepEqual(txOut, txOutDes) {
		t.Fatalf("Object not equal")
	}
}

func TestDeserializeUXTO(t *testing.T) {
	u := &core2.UXTO{
		TxId: core2.RandomHash256(),
		N:    11,
		TxOut: &core2.TxOut{
			Value: 100000,
			ScriptPubKey: core2.ScriptPubKey{
				PubKeyHash: core2.RandomHash160(),
			},
		},
	}

	buf := SerializeUXTO(u)
	uDes := DeserializeUXTO(buf)

	t.Logf(spew.Sdump(u))
	t.Logf(spew.Sdump(uDes))

	if !reflect.DeepEqual(u, uDes) {
		t.Fatalf("Objects not equal")
	}
}
