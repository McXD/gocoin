package core

import (
	"crypto"
	"crypto/rsa"
)

type ScriptPubKey struct {
	PubKeyHash Hash160
}

type ScriptSig struct {
	PubKey    rsa.PublicKey
	Signature []byte
}

type TxOut struct {
	Value uint32 // number of satoshi (100,000,000)
	ScriptPubKey
}

type TxIn struct {
	Hash Hash256 // Txid
	N    uint32  // output index
	ScriptSig
	CoinBase []byte
}

type Transaction struct {
	Hash Hash256
	In   []TxIn
	Out  []TxOut
}

func (tx *Transaction) SetHash() {
	tx.Hash = HashTo256(tx.ToBytes(true))
}

func NewCoinbaseTx(coinbase []byte, pubKeyHash Hash160) *Transaction {
	txIn := TxIn{
		Hash:     Hash256{}, // zeros
		N:        10000,     // TODO: specification
		CoinBase: coinbase,
	}

	txOut := TxOut{
		Value:        100, // TODO: dynamic
		ScriptPubKey: ScriptPubKey{pubKeyHash},
	}

	tx := Transaction{
		In:  []TxIn{txIn},
		Out: []TxOut{txOut},
	}

	tx.SetHash()

	return &tx
}

// ToBytes returns the raw bytes of the transaction for signing or hashing
func (tx *Transaction) ToBytes(withSig bool) []byte {
	var raw []byte

	// TxIns
	for _, txIn := range tx.In {
		raw = append(raw, txIn.Hash[:]...)        // tx reference
		raw = append(raw, UintToBytes(txIn.N)...) // index
		if withSig {
			raw = append(raw, txIn.Signature...) // signature
		}
		//raw = append(raw, txIn.PubKey.N.Bytes()...)              // pubKey: N TODO: segmentation fault
		raw = append(raw, UintToBytes(uint32(txIn.PubKey.E))...) // pubKey: E
	}

	// TxOuts
	for _, txOut := range tx.Out {
		raw = append(raw, UintToBytes(txOut.Value)...) // value
		raw = append(raw, txOut.PubKeyHash[:]...)      // scriptPubKey (only the public key in our case)
	}

	return raw
}

func (txIn *TxIn) verifiedSignature(txHash []byte) bool {
	err := rsa.VerifyPKCS1v15(&txIn.PubKey, crypto.SHA256, txHash, txIn.Signature)
	if err != nil {
		return false
	}

	return true
}

func (tx *Transaction) VerifiedSignature() bool {
	raw := tx.ToBytes(false)
	txHash := DoubleHashTo256(raw)

	for _, txIn := range tx.In {
		if !txIn.verifiedSignature(txHash[:]) {
			return false
		}
	}

	return true
}

func HashPubKey(pubKey *rsa.PublicKey) Hash160 {
	var raw []byte
	raw = append(raw, pubKey.N.Bytes()...)
	raw = append(raw, UintToBytes(uint32(pubKey.E))...)

	return HashTo160(raw)
}
