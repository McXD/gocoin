package blockchain

import (
	"github.com/davecgh/go-spew/spew"
	"gocoin/core"
	"math/rand"
	"os"
	"runtime/debug"
	"testing"
	"time"
)

var bc1, bc2 *Blockchain
var addr core.Hash160

func shouldFail(err error) {
	if err != nil {
		debug.PrintStack()
		panic(err)
	}
}

func init() {
	var err error
	dirRoots := []string{"/tmp/test-gocoin1", "/tmp/test-gocoin2", "/tmp/test-gocoin3"}
	for _, dir := range dirRoots {
		err := os.RemoveAll(dir)
		shouldFail(err)
		err = os.Mkdir(dir, os.ModePerm)
		shouldFail(err)
		err = os.Mkdir(dir+"/data", os.ModePerm)
		shouldFail(err)
		err = os.Mkdir(dir+"/db", os.ModePerm)
		shouldFail(err)
	}

	bc1, err = NewBlockchain("/tmp/test-gocoin1", "localhost", 8844)
	shouldFail(err)
	bc2, err = NewBlockchain("/tmp/test-gocoin2", "localhost", 8845)
	shouldFail(err)

	addr = core.RandomHash160()

	// Add peers
	err = bc1.AddPeer(bc2.Network.GetAddress())
	shouldFail(err)
	err = bc2.AddPeer(bc1.Network.GetAddress())
	shouldFail(err)
	// Add ten blocks to bc1
	for i := 0; i < 10; i++ {
		b, err := bc1.Mine(getCoinbase(), addr, BLOCK_REWARD)
		shouldFail(err)
		err = bc1.AddBlockAsTip(b)
		shouldFail(err)
	}

	bc1.StartP2PListener()
	bc2.StartP2PListener()
}

func TestHandleGetBlocks(t *testing.T) {
	currentHash, err := bc2.GetCurrentBlockHash()
	shouldFail(err)

	go bc2.Network.GetBlocks(bc1.ID(), []core.Hash256{currentHash}, core.Hash256{})

	time.Sleep(100 * time.Second)
}

func TestHandleGetBlockData(t *testing.T) {
	currentHash, err := bc2.GetCurrentBlockHash()
	shouldFail(err)

	go func() {
		invs := bc2.Network.GetBlocks(bc1.ID(), []core.Hash256{currentHash}, core.Hash256{})
		spew.Dump(invs)
		blocks := bc2.Network.DownloadBlocks(bc1.ID(), invs)
		spew.Dump(blocks)
	}()

	time.Sleep(100 * time.Second)
}

func TestHandleBroadcast(t *testing.T) {
	currentHash, err := bc2.GetCurrentBlockHash()
	shouldFail(err)

	// sync
	invs := bc2.Network.GetBlocks(bc1.ID(), []core.Hash256{currentHash}, core.Hash256{})
	blocks := bc2.Network.DownloadBlocks(bc1.ID(), invs[:len(invs)])
	for _, b := range blocks[1:] { // we already have the first block
		err := bc2.AddBlockAsTip(b)
		shouldFail(err)
	}

	// mine 5 blocks at bc2 and send them to bc1
	for i := 0; i < 5; i++ {
		b, err := bc2.Mine(getCoinbase(), addr, BLOCK_REWARD)
		shouldFail(err)
		err = bc2.AddBlockAsTip(b)
		shouldFail(err)
		bc2.Network.BroadcastBlock(b)
	}

	time.Sleep(100 * time.Second)
}

func TestHandleReorg(t *testing.T) {
	currentHash, err := bc2.GetCurrentBlockHash()
	shouldFail(err)

	// sync up to the second-to-last block
	invs := bc2.Network.GetBlocks(bc1.ID(), []core.Hash256{currentHash}, core.Hash256{})
	blocks := bc2.Network.DownloadBlocks(bc1.ID(), invs[:len(invs)])
	for _, b := range blocks[1 : len(blocks)-1] {
		err := bc2.AddBlockAsTip(b)
		shouldFail(err)
	}

	// mine 5 blocks at bc2 and send them to bc1
	for i := 0; i < 5; i++ {
		b, err := bc2.Mine(getCoinbase(), addr, BLOCK_REWARD)
		shouldFail(err)

		err = bc2.AddBlockAsTip(b)
		shouldFail(err)
		bc2.Network.BroadcastBlock(b)
	}

	time.Sleep(100 * time.Second)
}

func getCoinbase() []byte {
	i := rand.Int()
	coinbase := append(core.UintToBytes(uint32(i)), []byte("coinbase")...)
	return coinbase
}
