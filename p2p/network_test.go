package p2p

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

func TestNewNetwork(t *testing.T) {
	n1, err := NewNetwork("localhost", 8844, 0)
	if err != nil {
		t.Fatal(err)
	}

	n2, err := NewNetwork("localhost", 8845, 0)
	if err != nil {
		t.Fatal(err)
	}

	n1Addr := n1.GetAddress()
	n2Addr := n2.GetAddress()

	err = n1.AddPeer(n2Addr)
	if err != nil {
		t.Fatal(err)
		return
	}
	err = n2.AddPeer(n1Addr)
	if err != nil {
		t.Fatal(err)
		return
	}

	spew.Dump(n1.ListKnownAddrs())

	if len(n1.ListPeerIDs()) != 1 {
		t.Error("n1 should have 1 peer")
	}

	if len(n2.ListPeerIDs()) != 1 {
		t.Error("n1 should have 1 peer")
	}
}

func TestNetwork_GetAddr(t *testing.T) {
	n1, _ := NewNetwork("localhost", 8844, 0)
	n2, _ := NewNetwork("localhost", 8845, 0)
	n3, _ := NewNetwork("localhost", 8846, 0)

	// n2 <-> n1 <-> n3
	_ = n1.AddPeer(n2.GetAddress())
	_ = n1.AddPeer(n3.GetAddress())
	_ = n2.AddPeer(n1.GetAddress())
	_ = n3.AddPeer(n1.GetAddress())

	n1.StartListening(nil)
	ch := make(chan bool)

	go func() {
		addr, err := n2.GetAddr(n2.ListPeerIDs()[0])
		if err != nil {
			t.Error(err)
		}

		spew.Dump(addr)
		ch <- true
	}()

	<-ch
}
