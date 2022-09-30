package core

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/ripemd160"
)

type Hash256 [32]byte

func (hash Hash256) String() string {
	return fmt.Sprintf("%X", hash[:])
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

func (hash *Hash160) String() string {
	return fmt.Sprintf("%X", hash[:])
}
