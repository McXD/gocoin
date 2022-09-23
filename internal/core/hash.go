package core

import "fmt"

type SHA256Hash [32]byte

func (hash SHA256Hash) String() string {
	return fmt.Sprintf("%X", [32]byte(hash))
}
