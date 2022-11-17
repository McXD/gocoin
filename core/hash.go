package core

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/ripemd160"
	"math/big"
)

type Hash256 [32]byte

func (hash Hash256) String() string {
	return fmt.Sprintf("%X", hash[:])
}

func EmptyHash256() Hash256 {
	return [32]byte{}
}

func Hash256FromSlice(byteSlice []byte) Hash256 {
	var arr [32]byte
	copy(arr[:], byteSlice)

	return arr
}

// Int returns the integer value of this hash bits.
func (hash Hash256) Int() *big.Int {
	return big.NewInt(0).SetBytes(hash[:])
}

func HashTo256(data []byte) Hash256 {
	return sha256.Sum256(data[:])
}

func DoubleHashTo256(data []byte) Hash256 {
	ret := HashTo256(data)
	return HashTo256(ret[:])
}

type Hash160 [20]byte

func HashTo160(data []byte) Hash160 {
	var sum160 Hash160

	md := ripemd160.New()
	sum256 := sha256.Sum256(data[:])
	md.Write(sum256[:])
	copy(sum160[:], md.Sum(nil))

	return sum160
}

func Hash160FromSlice(slice []byte) Hash160 {
	var arr [20]byte
	copy(arr[:], slice)

	return arr
}

func (hash *Hash160) String() string {
	return fmt.Sprintf("%X", hash[:])
}
