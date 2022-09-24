package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

type TxOut struct {
	Value        uint32 // number of satoshi (100,000,000)
	ScriptPubKey []byte
}

func (o TxOut) hash() SHA256Hash {
	valueBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valueBytes, o.Value)

	return sha256.Sum256(bytes.Join([][]byte{
		valueBytes, o.ScriptPubKey,
	}, []byte{}))
}

type TxIn struct {
	Hash      SHA256Hash // Txid
	N         uint32     // output index
	ScriptSig []byte
}

func (i TxIn) hash() SHA256Hash {
	nBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nBytes, i.N)

	return sha256.Sum256(bytes.Join([][]byte{
		i.Hash[:], nBytes, i.ScriptSig,
	}, []byte{}))
}

type Transaction struct {
	Hash SHA256Hash
	In   []TxIn
	Out  []TxOut
}

func (tx *Transaction) hash() {
	var allHashes [][]byte

	// inputs
	for _, in := range tx.In {
		t := in.hash()
		allHashes = append(allHashes, t[:])
	}

	// outputs
	for _, in := range tx.Out {
		t := in.hash()
		allHashes = append(allHashes, t[:])
	}

	tx.Hash = sha256.Sum256(bytes.Join(allHashes, []byte{}))
}

func NewCoinbaseTx() *Transaction {
	txIn := TxIn{
		Hash:      SHA256Hash{}, // zeros
		N:         10000,        // TODO: specification
		ScriptSig: []byte{0xb, 0xa, 0xb, 0xe},
	}

	txOut := TxOut{
		Value:        100,                            // TODO: dynamic
		ScriptPubKey: []byte{0x11, 0x22, 0x33, 0x44}, // TODO: scripting
	}

	tx := Transaction{
		In:  []TxIn{txIn},
		Out: []TxOut{txOut},
	}

	tx.hash()

	return &tx
}
