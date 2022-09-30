package core

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/cbergoon/merkletree"
)

type ScriptPubKey struct {
	PubKeyHash Hash160
}

func (spk *ScriptPubKey) IsGeneratedFrom(pk *rsa.PublicKey) bool {
	return spk.PubKeyHash == HashPubKey(pk)
}

type ScriptSig struct {
	PK        *rsa.PublicKey
	Signature []byte
}

type TxIn struct {
	PrevTxId Hash256
	N        uint32 // output index
	ScriptSig
	Coinbase []byte
}

func (txIn *TxIn) SpentBy() Hash160 {
	return HashPubKey(txIn.PK)
}

type TxOut struct {
	Value uint32 // number of satoshi (100,000,000)
	ScriptPubKey
}

func (txOut *TxOut) AddressedTo() Hash160 {
	return txOut.PubKeyHash
}

// CanBeSpentBy checks whether the given pubKey is entitled to this output
func (txOut *TxOut) CanBeSpentBy(pk *rsa.PublicKey) bool {
	return txOut.ScriptPubKey.IsGeneratedFrom(pk)
}

func (txOut *TxOut) Serialized() []byte {
	return []byte{}
}

type UXTO struct {
	TxId Hash256
	N    uint32
	*TxOut
}

type UXTOSet struct {
	uxtos map[Hash256][]*UXTO
}

func NewUXTOSet() *UXTOSet {
	return &UXTOSet{uxtos: make(map[Hash256][]*UXTO)}
}

func (uSet *UXTOSet) Add(uxto *UXTO) {
	if uxto == nil {
		return
	}

	if uSet.uxtos[uxto.TxId] == nil {
		uSet.uxtos[uxto.TxId] = []*UXTO{}
	}

	uSet.uxtos[uxto.TxId] = append(uSet.uxtos[uxto.TxId], uxto)
}

func (uSet *UXTOSet) First(txId Hash256) *UXTO {
	uxtos := uSet.uxtos[txId]
	if uxtos == nil || len(uxtos) == 0 {
		return nil
	}

	return uxtos[0]
}

func (uSet *UXTOSet) Get(txId Hash256, n uint32) *UXTO {
	uxtos := uSet.uxtos[txId]
	if uxtos == nil || len(uxtos) == 0 {
		return nil
	}

	for _, uxto := range uxtos {
		if uxto.N == n {
			return uxto
		}
	}

	return nil
}

type Transaction struct {
	Ins  []*TxIn
	Outs []*TxOut
}

func (tx *Transaction) Hash() Hash256 {
	return HashTo256(tx.Serialized())
}

func (tx *Transaction) Serialized() []byte {
	var data []byte

	for _, txIn := range tx.Ins {
		data = append(data, txIn.PrevTxId[:]...)
		data = append(data, UintToBytes(txIn.N)...)

		if txIn.PrevTxId == [32]byte{} { // coinbase input
			data = append(data, txIn.Coinbase[:]...)
		}
	}

	for _, txOut := range tx.Outs {
		data = append(data, txOut.PubKeyHash[:]...)
		data = append(data, UintToBytes(txOut.Value)[:]...)
	}

	return data
}

func (tx *Transaction) InputOf(prevTxId Hash256, n uint32) *TxIn {
	for _, txIn := range tx.Ins {
		if txIn.PrevTxId == prevTxId && txIn.N == n { // find the consuming input
			return txIn
		}
	}

	return nil
}

func (tx *Transaction) generateSigningDigest(uxto *UXTO) []byte {
	var data []byte

	subScript := uxto.PubKeyHash

	for _, txIn := range tx.Ins {
		data = append(data, txIn.PrevTxId[:]...)
		data = append(data, UintToBytes(txIn.N)...)

		if uxto.TxId == txIn.PrevTxId && uxto.N == txIn.N {
			data = append(data, subScript[:]...)
		}
	}

	for _, txOut := range tx.Outs {
		data = append(data, txOut.PubKeyHash[:]...)
		data = append(data, UintToBytes(txOut.Value)[:]...)
	}

	digest := DoubleHashTo256(data)

	return digest[:]
}

func (tx *Transaction) SignTxIn(uxto *UXTO, sk *rsa.PrivateKey) error {
	digest := tx.generateSigningDigest(uxto)

	if sig, err := sk.Sign(rand.Reader, digest[:], crypto.SHA256); err != nil {
		return err
	} else {
		if txIn := tx.InputOf(uxto.TxId, uxto.N); txIn != nil {
			txIn.Signature = sig
			return nil
		} else { // no input matched
			return fmt.Errorf("no txIn matches the given uxto")
		}
	}
}

func (tx *Transaction) VerifyTxIn(uxto *UXTO, pk *rsa.PublicKey) error { // this is equivalent to running Script in Bitcoin
	digest := tx.generateSigningDigest(uxto)

	if txIn := tx.InputOf(uxto.TxId, uxto.N); txIn != nil {
		if !uxto.CanBeSpentBy(pk) {
			return fmt.Errorf("uxto cannot be spent")
		}

		if err := rsa.VerifyPKCS1v15(pk, crypto.SHA256, digest, txIn.Signature); err != nil {
			return fmt.Errorf("cannot verify signature: %w", err)
		}

		return nil
	} else { // no input matched
		return fmt.Errorf("no txIn matches the given uxto")
	}
}

func (tx *Transaction) Verify(uSet *UXTOSet) error {
	var inValue uint32

	// size
	if len(tx.Ins) == 0 {
		return fmt.Errorf("transaction contains 0 input")
	}

	if len(tx.Outs) == 0 {
		return fmt.Errorf("transaction contains 0 output")
	}

	// sender
	if !tx.IsCoinbaseTx() {
		sender := tx.Ins[0].SpentBy()
		for _, txIn := range tx.Ins {
			if txIn.SpentBy() != sender {
				return fmt.Errorf("transaction has multiple senders")
			}
		}
	} else {
		if len(tx.Ins) != 1 {
			return fmt.Errorf("coinbase transaction has more than 1 input")
		}
	}

	// skip for coinbase tx
	if !tx.IsCoinbaseTx() { // in a coinbase tx, input value is essentially zero
		for _, txIn := range tx.Ins {
			if uxto := uSet.Get(txIn.PrevTxId, txIn.N); uxto == nil { // no double-spend
				return fmt.Errorf("transaction input not found in UXTO set")
			} else {
				// "running Script"
				if err := tx.VerifyTxIn(uxto, txIn.PK); err != nil {
					return fmt.Errorf("txIn verification failed: %w", err)
				}

				// accumulate inValue
				inValue += uxto.Value
			}
		}

		// balance
		if outValue, overflow := tx.CalculateOutValue(); inValue < outValue || overflow {
			return fmt.Errorf("out-value is bigger than in-value")
		}
	}

	return nil
}

// From returns the address of the transaction payer.
// There is no a "To()" counterpart as a transaction can be sent to multiple addresses (e.g., change).
func (tx *Transaction) From() Hash160 {
	if tx.IsCoinbaseTx() {
		return Hash160{}
	}

	return tx.Ins[0].SpentBy()
}

func (tx *Transaction) CalculateOutValue() (outValue uint32, overflow bool) {
	for _, txOut := range tx.Outs {
		// overflow check
		twoSum := outValue + txOut.Value
		overflow = twoSum < outValue || twoSum < txOut.Value
		outValue = twoSum
	}

	return outValue, overflow
}

func (tx *Transaction) IsCoinbaseTx() bool {
	return len(tx.Ins) == 1 && len(tx.Ins[0].Coinbase) != 0
}

// CalculateHash is implements the interface function required by merkletree.Content
func (tx Transaction) CalculateHash() ([]byte, error) {
	hash := tx.Hash()
	return hash[:], nil
}

func (tx Transaction) Equals(other merkletree.Content) (bool, error) {
	otherHash, err := other.CalculateHash()
	thisHash := tx.Hash()

	if err != nil {
		return false, err
	}
	for i, v := range otherHash {
		if v != thisHash[i] {
			return false, nil
		}
	}

	return true, nil
}

type TransactionBuilder struct {
	uxtos   map[Hash256][]*UXTO
	inValue uint32
	*Transaction
}

func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		uxtos:   make(map[Hash256][]*UXTO),
		inValue: 0,
		Transaction: &Transaction{
			Ins:  nil,
			Outs: nil,
		},
	}
}

// AddInputFrom adds uxto to the tx input set
func (txb *TransactionBuilder) AddInputFrom(uxto *UXTO, pk *rsa.PublicKey) *TransactionBuilder {
	txIn := &TxIn{
		PrevTxId: uxto.TxId,
		N:        uxto.N,
		ScriptSig: ScriptSig{
			PK: pk,
		},
		Coinbase: nil,
	}

	txb.Ins = append(txb.Ins, txIn)
	txb.inValue += uxto.Value
	txb.uxtos[uxto.TxId] = append(txb.uxtos[uxto.TxId], uxto)
	return txb
}

func (txb *TransactionBuilder) AddOutput(v uint32, pubKeyHash Hash160) *TransactionBuilder {
	txOut := &TxOut{
		Value: v,
		ScriptPubKey: ScriptPubKey{
			PubKeyHash: pubKeyHash,
		},
	}

	txb.Outs = append(txb.Outs, txOut)
	return txb
}

func (txb *TransactionBuilder) AddChange(txFee uint32) *TransactionBuilder {
	currentOut, _ := txb.CalculateOutValue()
	txb.AddOutput(txb.inValue-currentOut-txFee, txb.From())
	return txb
}

func (txb *TransactionBuilder) Build() *Transaction {
	return txb.Transaction
}

func (txb *TransactionBuilder) Sign(privKey *rsa.PrivateKey) *Transaction {
	for _, uxtos := range txb.uxtos {
		for _, uxto := range uxtos {
			txb.SignTxIn(uxto, privKey) // shouldn't err
		}
	}

	return txb.Transaction
}

func NewCoinBaseTransaction(coinbase []byte, payTo Hash160, blockReward uint32, txFee uint32) *Transaction {
	txIn := &TxIn{
		PrevTxId: Hash256{},
		N:        0xcafebabe,
		Coinbase: coinbase,
	}

	txOut := &TxOut{
		Value: blockReward + txFee,
		ScriptPubKey: ScriptPubKey{
			PubKeyHash: payTo,
		},
	}

	// no signature is required since there's no ScriptPubKey

	return &Transaction{
		Ins:  []*TxIn{txIn},
		Outs: []*TxOut{txOut},
	}
}
