package persistence

import (
	"crypto/rand"
	"crypto/rsa"
	core2 "gocoin/core"
)

var SK []*rsa.PrivateKey
var PK []*rsa.PublicKey
var ADDR []core2.Hash160

var TXID []core2.Hash256
var USET *core2.InMemUXTOSet

func PopulateTestData() {
	SK = []*rsa.PrivateKey{}
	PK = []*rsa.PublicKey{}
	ADDR = []core2.Hash160{}
	TXID = []core2.Hash256{}
	USET = core2.NewUXTOSet()

	// 10 accounts
	for i := 0; i < 10; i++ {
		sk, _ := rsa.GenerateKey(rand.Reader, 512)
		pk := &sk.PublicKey
		addr := core2.HashPubKey(pk)

		SK = append(SK, sk)
		PK = append(PK, pk)
		ADDR = append(ADDR, addr)
	}

	// 10 uxto, 2 per each account, 100 unit per tx
	for i := 0; i < 10; i++ {
		for j := 0; j < 2; j++ {
			uxto := NewUXTO(ADDR[i], 100)
			USET.Add(uxto)
			TXID = append(TXID, uxto.TxId)
		}
	}
}

func NewUXTO(to core2.Hash160, v uint32) *core2.UXTO {
	return &core2.UXTO{
		TxId: core2.RandomHash256(),
		N:    0,
		TxOut: &core2.TxOut{
			Value:        v,
			ScriptPubKey: core2.ScriptPubKey{PubKeyHash: to},
		},
	}
}
