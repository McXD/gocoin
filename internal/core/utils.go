package core

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
)

func UintToBytes(i uint32) []byte {
	valueBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valueBytes, i)

	return valueBytes
}

func HashPubKey(pubKey *rsa.PublicKey) Hash160 {
	var raw []byte
	raw = append(raw, pubKey.N.Bytes()...)
	raw = append(raw, UintToBytes(uint32(pubKey.E))...)

	return HashTo160(raw)
}

func RandomHash256() Hash256 {
	var buf [32]byte
	rand.Reader.Read(buf[:])

	return HashTo256(buf[:])
}
