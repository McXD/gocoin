package marshal

import (
	"github.com/davecgh/go-spew/spew"
	core2 "gocoin/core"
	"reflect"
	"testing"
	"time"
)

func TestDeserializeBlockHeader(t *testing.T) {
	bh := &core2.BlockHeader{
		Time:           time.Now().Unix(),
		Bits:           20,
		Nonce:          123144,
		HashPrevBlock:  core2.RandomHash256(),
		HashMerkleRoot: core2.RandomHash256(),
	}

	buf := BlockHeader(bh)
	bhDes := DeserializeBlockHeader(buf)

	t.Log(spew.Sdump(bh))
	t.Log(spew.Sdump(bhDes))

	if !reflect.DeepEqual(bh, bhDes) {
		t.Errorf("Objects not equal")
	}
}

func TestDeserializeBlock(t *testing.T) {
	PopulateTestData()

	tx1 := core2.NewCoinBaseTransaction([]byte("COINBASE"), core2.RandomHash160(), 1000, 100)
	tx2 := core2.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[1]), PK[0]).
		AddInputFrom(USET.First(TXID[0]), PK[0]).
		AddOutput(100, ADDR[2]).
		AddChange(50).
		Sign(SK[0])
	tx3 := core2.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[2]), PK[1]).
		AddInputFrom(USET.First(TXID[3]), PK[1]).
		AddOutput(100, ADDR[5]).
		AddChange(50).
		Sign(SK[0])
	b := core2.NewBlockBuilder().
		BaseOn(core2.EmptyHash256(), 1000).
		SetBits(20).
		AddTransaction(tx1).
		AddTransaction(tx2).
		AddTransaction(tx3).
		Build()

	buf := Block(b)
	bDes := UBlock(buf)

	// not read from serialize
	bDes.Height = b.Height
	bDes.Hash = b.Hash

	t.Log(spew.Sdump(b))
	t.Log(spew.Sdump(bDes))

	if !reflect.DeepEqual(b, bDes) {
		t.Errorf("Objects not equal")
	}
}
