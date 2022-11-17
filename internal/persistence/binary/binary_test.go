package binary

import (
	"crypto/rand"
	"crypto/rsa"
	"gocoin/internal/core"
)

var SK []*rsa.PrivateKey
var PK []*rsa.PublicKey
var ADDR []core.Hash160

var TXID []core.Hash256
var USET *core.InMemUXTOSet

func PopulateTestData() {
	SK = []*rsa.PrivateKey{}
	PK = []*rsa.PublicKey{}
	ADDR = []core.Hash160{}
	TXID = []core.Hash256{}
	USET = core.NewUXTOSet()

	// 10 accounts
	for i := 0; i < 10; i++ {
		sk, _ := rsa.GenerateKey(rand.Reader, 512)
		pk := &sk.PublicKey
		addr := core.HashPubKey(pk)

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

func NewUXTO(to core.Hash160, v uint32) *core.UXTO {
	return &core.UXTO{
		TxId: core.RandomHash256(),
		N:    0,
		TxOut: &core.TxOut{
			Value:        v,
			ScriptPubKey: core.ScriptPubKey{PubKeyHash: to},
		},
	}
}
