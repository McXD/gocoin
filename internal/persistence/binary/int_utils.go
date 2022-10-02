package binary

import "encoding/binary"

func IntToBytes(v int) []byte {
	// assume int to be uint64
	return Uint64ToBytes(uint64(v))
}

func IntFromBytes(buf []byte) int {
	return int(Uint64FromBytes(buf))
}

func Uint32ToBytes(v uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, v)

	return buf
}

func Uint32FromBytes(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}

func Uint64ToBytes(v uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(v))

	return buf
}

func Uint64FromBytes(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}
