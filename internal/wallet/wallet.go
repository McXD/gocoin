package wallet

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/internal/core"
)

type Wallet struct {
	Addresses []core.Hash160
	Keys      map[core.Hash160]*rsa.PrivateKey
	Balances  map[core.Hash160]uint32
	uxtos     map[core.Hash256]*core.UXTO // addr -> UXTO

	Receives map[core.Hash256]*core.Transaction
	Spends   map[core.Hash256]*core.Transaction

	blockchain *core.BlockchainInMem
}

// NewWallet returns a wallet with one address
func NewWallet() *Wallet {
	w := Wallet{
		Addresses: make([]core.Hash160, 0),
		Keys:      make(map[core.Hash160]*rsa.PrivateKey),
		Balances:  make(map[core.Hash160]uint32),
		uxtos:     make(map[core.Hash256]*core.UXTO),

		Receives: make(map[core.Hash256]*core.Transaction),
		Spends:   make(map[core.Hash256]*core.Transaction),
	}

	w.NewAddress()

	return &w
}

func (w *Wallet) NewAddress() core.Hash160 {
	sk, _ := rsa.GenerateKey(rand.Reader, 512)
	addr := core.HashPubKey(&sk.PublicKey)

	w.Addresses = append(w.Addresses, addr)
	w.Keys[addr] = sk

	return addr
}

func (w *Wallet) Send(from, to core.Hash160, value uint32) (core.Hash256, error) {
	var inVal uint32
	txb := core.NewTransactionBuilder()

	// inputs
	for _, u := range w.uxtos {
		if u.PubKeyHash == from {
			txb.AddInput(u.TxHash, u.Index, w.Keys[from].PublicKey)

			inVal += u.Value
			if inVal > value {
				break
			}
		}
	}

	if inVal < value {
		return core.Hash256{}, fmt.Errorf("insufficient fund, balance=%d, want=%d", inVal, value)
	}

	// outputs
	txb.AddOutput(value, to)
	if inVal > value {
		txb.AddOutput(inVal-value, from) // change
	}

	if tx, err := txb.Sign(w.Keys[from]); err != nil {
		return core.Hash256{}, fmt.Errorf("cannot sign transaction: %w", err)
	} else {
		tx.SetHash()

		if err := w.blockchain.AddTransaction(tx); err != nil {
			return core.Hash256{}, fmt.Errorf("failed to send transaction: %w", err)
		} else {
			return tx.Hash, nil
		}
	}
}

func (w *Wallet) Balance(addr core.Hash160) uint32 {
	return w.Balances[addr]
}

func (w *Wallet) ListUnspent(addr core.Hash160) []*core.UXTO {
	return nil
}

func (w *Wallet) GetTransaction(id core.Hash256) *core.Transaction {
	if tx := w.Receives[id]; tx == nil {
		return w.Spends[id]
	} else {
		return tx
	}
}

func (w *Wallet) ProcessTransaction(tx *core.Transaction) {
	// if an uxto occurs in input set, delete it
	for _, txIn := range tx.Ins {
		if uxto, ok := w.uxtos[txIn.Hash]; ok && uxto.Index == txIn.N {
			delete(w.uxtos, txIn.Hash)

			// update balance
			w.Balances[uxto.PubKeyHash] -= uxto.Value

			// record tx
			w.Spends[tx.Hash] = tx

			log.WithFields(log.Fields{
				"addr":  fmt.Sprintf("%X", uxto.PubKeyHash[:]),
				"txId":  fmt.Sprintf("%X", tx.Hash[:]),
				"index": uxto.Index,
				"value": uxto.Value,
				"to":    fmt.Sprintf("%X", tx.To()),
			}).Info("Spent UXTO")
		}
	}

	// if output contains one of our addresses, add it
	for i, txOut := range tx.Outs {
		for _, addr := range w.Addresses {
			if txOut.PubKeyHash == addr {
				w.uxtos[tx.Hash] = &core.UXTO{
					TxHash:       tx.Hash,
					Index:        uint32(i),
					Value:        txOut.Value,
					ScriptPubKey: core.ScriptPubKey{PubKeyHash: addr},
				}

				// update balance
				w.Balances[addr] += txOut.Value

				// record tx
				w.Receives[tx.Hash] = tx

				log.WithFields(log.Fields{
					"addr":  fmt.Sprintf("%X", addr[:]),
					"txId":  fmt.Sprintf("%X", tx.Hash[:]),
					"index": i,
					"value": txOut.Value,
					"from":  fmt.Sprintf("%X", tx.From()),
				}).Info("Received UXTO")
			}
		}
	}

}

func (w *Wallet) ProcessBlock(block *core.Block) {
	for _, tx := range block.Transactions {
		w.ProcessTransaction(tx)
	}
}

func (w *Wallet) Connect(bc *core.BlockchainInMem) {
	w.blockchain = bc

	// scan blocks
	currentBlock := bc.Head
	for currentBlock != nil {
		w.ProcessBlock(currentBlock)

		log.WithFields(log.Fields{
			"hash": fmt.Sprintf("%X", currentBlock.Hash[:]),
		}).Info("Scanned block")

		currentBlock = bc.GetBlock(currentBlock.PrevBlockHash)
	}
}
