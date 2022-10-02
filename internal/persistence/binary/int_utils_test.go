package binary

import "testing"

func TestIntToAndFromBytes(t *testing.T) {
	var v int = 0x123456

	bv := IntToBytes(v)
	if i := IntFromBytes(bv); i != v {
		t.Errorf("IntFromBytes(bi) = %d; expected %d", i, v)
	}
}
