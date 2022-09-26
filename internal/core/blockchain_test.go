package core

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"
)

var acct1, acct2, acct3 *rsa.PrivateKey
var bc *Blockchain

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	rng := rand.Reader
	acct1, _ = rsa.GenerateKey(rng, 512)
	acct2, _ = rsa.GenerateKey(rng, 512)
	acct3, _ = rsa.GenerateKey(rng, 512)

	bc = NewBlockchain(HashPubKey(&acct1.PublicKey))
}

func TestBlockchain_AddBlock(t *testing.T) {

}
