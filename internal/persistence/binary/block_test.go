package binary

import (
	"github.com/davecgh/go-spew/spew"
	"gocoin/internal/core"
	"reflect"
	"testing"
	"time"
)

func TestDeserializeBlockHeader(t *testing.T) {
	bh := &core.BlockHeader{
		Time:           time.Now().Unix(),
		Bits:           20,
		Nonce:          123144,
		HashPrevBlock:  core.RandomHash256(),
		HashMerkleRoot: core.RandomHash256(),
	}

	buf := SerializeBlockHeader(bh)
	bhDes := DeserializeBlockHeader(buf)

	t.Log(spew.Sdump(bh))
	t.Log(spew.Sdump(bhDes))

	if !reflect.DeepEqual(bh, bhDes) {
		t.Errorf("Objects not equal")
	}
}

func TestDeserializeBlock(t *testing.T) {
	PopulateTestData()

	tx1 := core.NewCoinBaseTransaction([]byte("COINBASE"), core.RandomHash160(), 1000, 100)
	tx2 := core.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[1]), PK[0]).
		AddInputFrom(USET.First(TXID[0]), PK[0]).
		AddOutput(100, ADDR[2]).
		AddChange(50).
		Sign(SK[0])
	tx3 := core.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[2]), PK[1]).
		AddInputFrom(USET.First(TXID[3]), PK[1]).
		AddOutput(100, ADDR[5]).
		AddChange(50).
		Sign(SK[0])
	b := core.NewBlockBuilder().
		BaseOn(core.EmptyHash256(), 1000).
		SetBits(20).
		AddTransaction(tx1).
		AddTransaction(tx2).
		AddTransaction(tx3).
		Build()

	buf := SerializeBlock(b)
	bDes := DeserializeBlock(buf)

	// not read from binary
	bDes.Height = b.Height
	bDes.Hash = b.Hash

	t.Log(spew.Sdump(b))
	t.Log(spew.Sdump(bDes))

	if !reflect.DeepEqual(b, bDes) {
		t.Errorf("Objects not equal")
	}
}
