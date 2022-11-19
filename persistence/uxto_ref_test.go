package persistence

import (
	"fmt"
	"gocoin/core"
	"testing"
)

func TestUXTORef_Serialize(t *testing.T) {
	ref := UXTORef{
		TxId: core.RandomHash256(),
		N:    1,
	}

	refBin := ref.Serialize()

	refRec := UXTORef{}
	refRec.SetBytes(refBin)

	fmt.Printf("ref: %v\n", refBin)
	fmt.Printf("%-v\n", ref)
	fmt.Printf("%-v\n", refRec)
}
