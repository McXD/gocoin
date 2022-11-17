package core

import (
	"fmt"
	"testing"
)

func TestHashTo160(t *testing.T) {
	raw := []byte("Howdy")
	want := "7ba243548c7b57a54de0b0c5349c65eb6b4841d0"
	sum := fmt.Sprintf("%x", HashTo160(raw))
	if sum != want {
		t.Fatalf(`HashTo160("Howdy") = %q, want %#q`, sum, want)
	}
}
