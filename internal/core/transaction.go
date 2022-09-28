package core

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
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

// CanBeSpentBy checks whether the given pubKey is entitled to this output
func (txOut *TxOut) CanBeSpentBy(pubKey rsa.PublicKey) bool {
	return HashPubKey(&pubKey) == txOut.PubKeyHash
}

type TxIn struct {
	Hash Hash256 // Txid
	N    uint32  // output index
	ScriptSig
	CoinBase []byte
}

type Transaction struct {
	Hash Hash256
	Ins  []*TxIn
	Outs []*TxOut
}

func (tx *Transaction) From() Hash160 {
	var from Hash160

	if tx.IsCoinbaseTx() {
		return from
	}

	from = HashPubKey(&tx.Ins[0].PubKey)

	return from
}

func (tx *Transaction) To() Hash160 {
	return tx.Outs[0].PubKeyHash
}

func (tx *Transaction) SetHash() {
	tx.Hash = HashTo256(tx.ToBytes(true))
}

func NewCoinbaseTx(coinbase []byte, pubKeyHash Hash160, reward uint32) *Transaction {
	txIn := TxIn{
		Hash:     Hash256{}, // zeros
		N:        reward,
		CoinBase: coinbase,
	}

	txOut := TxOut{
		Value:        reward, // TODO: dynamic
		ScriptPubKey: ScriptPubKey{pubKeyHash},
	}

	tx := Transaction{
		Ins:  []*TxIn{&txIn},
		Outs: []*TxOut{&txOut},
	}

	tx.SetHash()

	return &tx
}

// ToBytes returns the raw bytes of the transaction for signing or hashing
func (tx *Transaction) ToBytes(withSig bool) []byte {
	var raw []byte

	// TxIns
	for _, txIn := range tx.Ins {
		raw = append(raw, txIn.Hash[:]...)        // tx reference
		raw = append(raw, UintToBytes(txIn.N)...) // index
		if withSig {
			raw = append(raw, txIn.Signature...) // signature
		}
		//raw = append(raw, txIn.PubKey.N.Bytes()...)              // pubKey: N TODO: segmentation fault
		raw = append(raw, UintToBytes(uint32(txIn.PubKey.E))...) // pubKey: E
	}

	// TxOuts
	for _, txOut := range tx.Outs {
		raw = append(raw, UintToBytes(txOut.Value)...) // value
		raw = append(raw, txOut.PubKeyHash[:]...)      // scriptPubKey (only the public key in our case)
	}

	return raw
}

func (tx *Transaction) Sign(privKey *rsa.PrivateKey) error {
	var raw []byte
	var txHash Hash256

	raw = tx.ToBytes(false)
	txHash = DoubleHashTo256(raw)

	// sign txHash
	rng := rand.Reader
	sig, err := rsa.SignPKCS1v15(rng, privKey, crypto.SHA256, txHash[:])
	if err != nil {
		return err
	}

	for _, txIn := range tx.Ins {
		txIn.Signature = sig
	}

	return nil
}

func (tx *Transaction) VerifySignature() error {
	raw := tx.ToBytes(false)
	txHash := DoubleHashTo256(raw)

	for _, txIn := range tx.Ins {
		err := rsa.VerifyPKCS1v15(&txIn.PubKey, crypto.SHA256, txHash[:], txIn.Signature)
		if err != nil {
			return fmt.Errorf("invalid signature for input %x:%d: %w", txIn.Hash[:], txIn.N, err)
		}
	}

	return nil
}

func (tx *Transaction) IsCoinbaseTx() bool {
	return len(tx.Ins) == 1 && len(tx.Ins[0].CoinBase) != 0
}

type TransactionBuilder struct {
	*Transaction
}

func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{&Transaction{}}
}

func (txb *TransactionBuilder) AddInput(id Hash256, n uint32, pubKey rsa.PublicKey) *TransactionBuilder {
	txIn := TxIn{
		Hash: id,
		N:    n,
		ScriptSig: ScriptSig{
			PubKey: pubKey,
		},
		CoinBase: nil,
	}

	txb.Transaction.Ins = append(txb.Transaction.Ins, &txIn)
	return txb
}

// AddInputFrom adds all outputs in the transaction the given public key is entitled to
// Returns total value collected
func (txb *TransactionBuilder) AddInputFrom(tx *Transaction, pubKey rsa.PublicKey) uint32 {
	var inputValue uint32

	for i, txOut := range tx.Outs {
		if txOut.CanBeSpentBy(pubKey) {
			txb.AddInput(tx.Hash, uint32(i), pubKey)
			inputValue += txOut.Value
		}
	}

	return inputValue
}

func (txb *TransactionBuilder) AddOutput(v uint32, pubKeyHash Hash160) *TransactionBuilder {
	txOut := TxOut{
		Value: v,
		ScriptPubKey: ScriptPubKey{
			PubKeyHash: pubKeyHash,
		},
	}

	txb.Transaction.Outs = append(txb.Transaction.Outs, &txOut)
	return txb
}

func (txb *TransactionBuilder) GetOutputValue() uint32 {
	var sum uint32
	for _, txOut := range txb.Outs {
		sum += txOut.Value
	}

	return sum
}

func (txb *TransactionBuilder) Sign(privKey *rsa.PrivateKey) (*Transaction, error) {
	if err := txb.Transaction.Sign(privKey); err != nil {
		return txb.Transaction, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return txb.Transaction, nil
}
