package wallet

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"gocoin/internal/core"
)

type TransactionBuilder struct {
	consumed map[core.Hash256]*core.TxOut
	*core.Transaction
}

func (txb *TransactionBuilder) AddInput(in *core.TxIn) *TransactionBuilder {
	txb.In = append(txb.In, *in)
	return txb
}

func (txb *TransactionBuilder) AddOutput(out *core.TxOut) *TransactionBuilder {
	txb.Out = append(txb.Out, *out)
	return txb
}

func (txb *TransactionBuilder) Sign(privKey *rsa.PrivateKey) *core.Transaction {
	var raw []byte
	var txHash core.Hash256

	raw = txb.ToBytes(false)
	txHash = core.DoubleHashTo256(raw)

	// sign txHash
	rng := rand.Reader
	sig, err := rsa.SignPKCS1v15(rng, privKey, crypto.SHA256, txHash[:])
	if err != nil {
		for _, TxIn := range txb.In {
			TxIn.Signature = sig
		}
	}

	return txb.Transaction
}

func (txb *TransactionBuilder) Verify() *core.Transaction {
	// TODO
	return nil
}
