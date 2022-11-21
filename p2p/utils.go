package p2p

import (
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func FullMultiAddr(addr ma.Multiaddr, id peer.ID) string {
	p2pAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", id))
	return addr.Encapsulate(p2pAddr).String()
}
