package p2p

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"gocoin/core"
	"io"
	mrand "math/rand"
)

const PROTOCOL = "/gocoin/1.0.0"

type Network struct {
	host.Host
}

func NewNetwork(hostname string, port int) (*Network, error) {
	n := &Network{Host: nil}

	basicHost, err := makeBasicHost(hostname, port, true, 0)
	if err != nil {
		return nil, err
	}

	n.Host = basicHost

	return n, nil
}

func makeBasicHost(listenDnsName string, listenPort int, insecure bool, randseed int64) (host.Host, error) {
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/%d", listenDnsName, listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}

	return libp2p.New(opts...)
}

func (n *Network) GetAddress() string {
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", n.Host.ID()))
	addr := n.Host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func (n *Network) AddPeer(targetPeer string) error {
	maddr, err := ma.NewMultiaddr(targetPeer)
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	n.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	log.Infof("Added peer %s", targetPeer)
	return nil
}

func (n *Network) ListPeers() []peer.ID {
	ret := make([]peer.ID, 0)
	store := n.Host.Peerstore()
	for _, id := range store.Peers() {
		if id != n.Host.ID() {
			ret = append(ret, id)
		}
	}
	return ret
}

func (n *Network) ListKnownAddrs() []string {
	ret := make([]string, 0)
	store := n.Host.Peerstore()
	for _, id := range store.Peers() {
		p2pAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", id))
		addr := store.Addrs(id)[0]
		ret = append(ret, addr.Encapsulate(p2pAddr).String())
	}
	return ret
}

func (n *Network) handleStream(s network.Stream) {
	ctx := context.Background()
	log.Infof("Got a new stream!")
	ctx = context.WithValue(ctx, "addr", FullMultiAddr(s.Conn().RemoteMultiaddr(), s.Conn().RemotePeer()))

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	buf := make([]byte, S_HEADER)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading header: %s", err)
		return
	}
	h := ReceiveHeader(buf)

	switch h.Command {
	case CMD_GETADDR:
		n.handleGetAddr(ctx, rw)
		break
	}
}

func (n *Network) StartListening() {
	n.Host.SetStreamHandler(PROTOCOL, n.handleStream)
}

// GetAddr requests multi-addresses the given peer knows of
func (n *Network) GetAddr(id peer.ID) ([]string, error) {
	log.Infof("GetAddr request to %s", id)
	s, err := n.Host.NewStream(context.Background(), id, PROTOCOL)
	if err != nil {
		return nil, fmt.Errorf("error creating stream: %s", err)
	}

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	_, err = rw.Write(SendGetAddr())
	if err != nil {
		return nil, fmt.Errorf("error writing request: %s", err)
	}

	err = rw.Flush()
	if err != nil {
		return nil, fmt.Errorf("error flushing request: %s", err)
	}

	buf := make([]byte, S_HEADER)
	_, err = io.ReadFull(rw, buf)
	h := ReceiveHeader(buf)

	if h.Command != CMD_ADDR {
		return nil, fmt.Errorf("unexpected response: %s", h.Command)
	}

	buf = make([]byte, h.SPayload)
	_, err = io.ReadFull(rw, buf)
	addrs := ReceiveAddr(buf)

	return addrs, nil
}

func (n *Network) handleGetAddr(ctx context.Context, rw *bufio.ReadWriter) {
	// add to our own neighbor list
	addr := ctx.Value("addr").(string)
	err := n.AddPeer(addr)
	if err != nil {
		log.Errorf("Error adding peer: %s", err)
		return
	}

	_, err = rw.Write(SendAddr(n.ListKnownAddrs()))
	if err != nil {
		log.Errorf("Error writing response: %s", err)
		return
	}

	err = rw.Flush()
	if err != nil {
		log.Errorf("Error flushing response: %s", err)
		return
	}
}

// GetBlocks requests block ids from the given peer
// L ---- getblocks ----> R
// L <--- inv blocks ---- R
func (n *Network) GetBlocks(peer peer.ID, knownHashes []core.Hash256, endHash core.Hash256) []Inventory {
	log.Infof("GetBlocks request to %s", peer)
	s, err := n.Host.NewStream(context.Background(), peer, PROTOCOL)
	if err != nil {
		log.Errorf("Error creating stream: %s", err)
		return nil
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	_, err = rw.Write(SendGetBlocks(MsgGetBlocks{
		NBlocks:     uint32(len(knownHashes)),
		BlockHashes: knownHashes,
		EndHash:     endHash,
	}))
	if err != nil {
		log.Errorf("Error writing request: %s", err)
		return nil
	}

	return nil
}

func (n *Network) handleGetBlocks(ctx context.Context, rw *bufio.ReadWriter, h Header) {
	buf := make([]byte, h.SPayload)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading payload: %s", err)
		return
	}

	_ = ReceiveGetBlocks(buf)
}

// DownloadBlocks requests a list of blocks from the given peer
// L ----  getdata  ---> R (INV is of type block)
// L <----  block  ----- R
// L <----  block  ----- R
// ....
// L <----  block  ----- R
func (n *Network) DownloadBlocks(peer peer.ID, invs []Inventory) []*core.Block {
	return nil
}

// BroadcastBlock broadcasts a block to all peers the node currently knows of
// L ----- block ------> R
func (n *Network) BroadcastBlock(block *core.Block) {

}

// BroadcastTx broadcasts a transaction to all peers the node currently knows of
// L ------- tx -------> R
func (n *Network) BroadcastTx(tx *core.Transaction) {

}
