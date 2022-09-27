package core

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"
)

var acct1, _ = rsa.GenerateKey(rand.Reader, 512)

var bc *BlockchainInMem

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	rng := rand.Reader
	acct1, _ = rsa.GenerateKey(rng, 512)

	bc = NewBlockchain(HashPubKey(&acct1.PublicKey))
}

func TestNewBlockchain(t *testing.T) {

}

func TestBlockchain_AddTransaction(t *testing.T) {
	bc := NewBlockchain(HashPubKey(&acct1.PublicKey))
	_ = bc
}

func TestBlockchain_Mine(t *testing.T) {

}

func TestBlockchain_AddBlock(t *testing.T) {

}
