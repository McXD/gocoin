package marshal

import (
	"crypto/rsa"
	"gocoin/core"
	"math/big"
)

func SerializeScriptSig(ss *core.ScriptSig) []byte {
	var buf []byte

	pknB := ss.PK.N.Bytes()
	pknSizeB := IntToBytes(len(pknB))

	buf = append(buf, pknSizeB...)            // PK.N GetBlockFileSize, 8
	buf = append(buf, ss.PK.N.Bytes()...)     // PK.N, variable
	buf = append(buf, IntToBytes(ss.PK.E)...) // PK.E, 8
	buf = append(buf, ss.Signature...)        // Signature, variable

	return buf
}

func DeserializeScriptSig(buf []byte) *core.ScriptSig {
	ss := &core.ScriptSig{
		PK: &rsa.PublicKey{
			N: big.NewInt(0),
			E: 0,
		},
		Signature: []byte{},
	}

	p := 0
	pknSize := IntFromBytes(buf[:8])
	p += 8
	ss.PK.N.SetBytes(buf[p : p+pknSize])
	p += pknSize
	ss.PK.E = IntFromBytes(buf[p : p+8])
	p += 8
	sig := make([]byte, len(buf)-p)
	copy(sig, buf[p:])
	ss.Signature = sig

	return ss
}

func SerializeTxIn(txIn *core.TxIn) []byte {
	var data []byte
	var dataScriptSig []byte

	if txIn.PrevTxId == core.EmptyHash256() {
		dataScriptSig = txIn.Coinbase
	} else {
		dataScriptSig = SerializeScriptSig(&txIn.ScriptSig)
	}
	sizeScriptSig := IntToBytes(len(dataScriptSig))

	data = append(data, txIn.PrevTxId[:]...)      // TxId, 32
	data = append(data, Uint32ToBytes(txIn.N)...) // vOut, 4
	data = append(data, sizeScriptSig...)         // ScriptSig GetBlockFileSize, 8
	data = append(data, dataScriptSig...)         // ScriptSig, variable

	return data
}

func DeserializeTxIn(buf []byte) *core.TxIn {
	txIn := &core.TxIn{
		PrevTxId:  core.Hash256{},
		N:         0,
		ScriptSig: core.ScriptSig{},
		Coinbase:  nil,
	}

	p := 0

	txIn.PrevTxId = core.Hash256FromSlice(buf[p : p+32])
	p += 32

	txIn.N = Uint32FromBytes(buf[p : p+4])
	p += 4

	scripSigSize := IntFromBytes(buf[p : p+8])
	p += 8

	if txIn.PrevTxId != core.EmptyHash256() { // read to scripSig
		txIn.ScriptSig = *DeserializeScriptSig(buf[p : p+scripSigSize])
	} else { // read to Coinbase
		scriptSig := make([]byte, scripSigSize)
		copy(scriptSig, buf[p:p+scripSigSize])
		txIn.Coinbase = scriptSig
	}
	p += scripSigSize

	return txIn
}

func SerializeScriptPubKey(skp *core.ScriptPubKey) []byte {
	return skp.PubKeyHash[:] // PubKeyHash, 20
}

func DeserializeScriptPubKey(buf []byte) *core.ScriptPubKey {
	return &core.ScriptPubKey{PubKeyHash: core.Hash160FromSlice(buf)}
}

func SerializeTxOut(txOut *core.TxOut) []byte {
	var buf []byte

	buf = append(buf, SerializeScriptPubKey(&txOut.ScriptPubKey)...) // ScriptPubKey, 20
	buf = append(buf, Uint32ToBytes(txOut.Value)...)                 // Value, 4

	return buf
}

func DeserializeTxOut(buf []byte) *core.TxOut {
	txOut := core.TxOut{
		Value:        0,
		ScriptPubKey: core.ScriptPubKey{},
	}

	p := 0
	txOut.ScriptPubKey = *DeserializeScriptPubKey(buf[:20])

	p += 20
	txOut.Value = Uint32FromBytes(buf[p : p+4])

	return &txOut
}

func SerializeUXTO(u *core.UXTO) []byte {
	var buf []byte

	buf = append(buf, u.TxId[:]...)               // TxId, 32
	buf = append(buf, Uint32ToBytes(u.N)...)      // N, 4
	buf = append(buf, SerializeTxOut(u.TxOut)...) // TxOut (PubKey and Value), 24

	return buf
}

func DeserializeUXTO(buf []byte) *core.UXTO {
	u := &core.UXTO{
		TxId:  core.Hash256{},
		N:     0,
		TxOut: nil,
	}

	p := 0
	u.TxId = core.Hash256FromSlice(buf[:32])

	p += 32
	u.N = Uint32FromBytes(buf[p : p+4])

	p += 4
	u.TxOut = DeserializeTxOut(buf[p : p+24])

	return u
}
