package p2p

import (
	"github.com/davecgh/go-spew/spew"
	"reflect"
	"testing"
)

func TestHeader(t *testing.T) {
	h := Header{
		Magic:    HEADER_MAGIC,
		Command:  CMD_GETADDR,
		SPayload: 0,
	}

	data := h.ToBytes()
	h2 := Header{}
	h2.SetBytes(data)

	if !reflect.DeepEqual(h, h2) {
		t.Error("Header does not match")
	}
}

func TestAddr(t *testing.T) {
	s1 := "/ip4/127.0.0.1/tcp/8844/p2p/QmdD74V7HZ5vmtqck38bHEnToQNmr729fYrUWWo9Jajkke"
	s2 := "/ip4/127.0.0.1/tcp/8845/p2p/QmQrzYCyC1ZYwLvCqsPgm1NiZwJcnD1KjVoA957jNn4e5d"

	sent := []string{s1, s2}
	data := SendAddr(sent)
	recv := ReceiveAddr(data)

	if !reflect.DeepEqual(recv, sent) {
		spew.Dump(recv)
		spew.Dump(sent)
		t.Error("Addr does not match")
	}
}
