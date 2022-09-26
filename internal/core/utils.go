package core

import "encoding/binary"

func UintToBytes(i uint32) []byte {
	valueBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valueBytes, i)

	return valueBytes
}
