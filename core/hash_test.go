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

func TestParseHash256(t *testing.T) {
	txId := "7B7F2FCDEAE0D0740CA0AEC1061B17AE23923B8BA2BFE44BC6D8463520E83E22"
	parsed, _ := ParseHash256(txId)
	fmt.Printf("%s", parsed.String())
}
