package marshal

import (
	"bytes"
	"gocoin/core"
)

const (
	MAGIC_TXIN uint32 = 0xff_ff_ff_ff
)

var SEP []byte

func init() {
	SEP = Uint32ToBytes(MAGIC_TXIN)
}

func Transaction(tx *core.Transaction) []byte {
	var buf []byte

	inputSize := IntToBytes(len(tx.Ins))
	outPutSize := IntToBytes(len(tx.Outs))

	buf = append(buf, inputSize...) // Input GetBlockFileSize, 8

	for _, txIn := range tx.Ins {
		buf = append(buf, SerializeTxIn(txIn)...) // TxIn, variable
		buf = append(buf, SEP...)                 // Separator, 4
	}

	buf = append(buf, outPutSize...) // Output GetBlockFileSize, 8

	for _, txOut := range tx.Outs {
		buf = append(buf, SerializeTxOut(txOut)...) // TxOut, 24
	}

	return buf
}

func UTransaction(buf []byte) *core.Transaction {
	tx := &core.Transaction{
		Ins:  []*core.TxIn{},
		Outs: []*core.TxOut{},
	}

	p := 0
	inputSize := IntFromBytes(buf[:8])

	p += 8
	txIns := bytes.Split(buf[8:], SEP)
	for i := 0; i < inputSize; i++ {
		tx.Ins = append(tx.Ins, DeserializeTxIn(txIns[i]))
		p += len(txIns[i])
		p += 4 // separator
	}

	outputSize := IntFromBytes(buf[p : p+8])

	p += 8
	for i := 0; i < outputSize; i++ {
		tx.Outs = append(tx.Outs, DeserializeTxOut(buf[p:p+24]))
		p += 24
	}

	return tx
}
