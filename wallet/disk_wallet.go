package wallet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"github.com/boltdb/bolt"
	"gocoin/core"
	"gocoin/marshal"
	"gocoin/persistence"
	"time"
)

type DiskWallet struct {
	db *bolt.DB
}

// NewDiskWallet creates or loads a new disk wallet
func NewDiskWallet(rootDir string) (*DiskWallet, error) {
	w := &DiskWallet{db: nil}

	db, err := bolt.Open(rootDir+"/db/wallet.dat", 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	w.db = db

	// create four buckets
	bucketKeys := [][]byte{
		[]byte("addresses"),    // address -> 0
		[]byte("keys"),         // address -> sk
		[]byte("balances"),     // address -> uint32
		[]byte("uxtos"),        // uRef -> UXTO
		[]byte("transactions"), // txid -> transactions
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucketKey := range bucketKeys {
			_, err := tx.CreateBucketIfNotExists(bucketKey)
			if err != nil {
				return fmt.Errorf("failed create bucket: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("cannot create buckets: %w", err)
	}

	return w, nil
}

func (w *DiskWallet) NewAddress() (core.Hash160, error) {
	sk, _ := rsa.GenerateKey(rand.Reader, 512) // TODO: param bit length
	addr := core.HashPubKey(&sk.PublicKey)

	err := w.db.Update(func(tx *bolt.Tx) error {
		addresses := tx.Bucket([]byte("addresses"))
		keys := tx.Bucket([]byte("keys"))

		if err := addresses.Put(addr[:], []byte{}); err != nil {
			return fmt.Errorf("failed to put address: %w", err)
		}

		if err := keys.Put(addr[:], x509.MarshalPKCS1PrivateKey(sk)); err != nil {
			return fmt.Errorf("failed to put key: %w", err)
		}

		return nil
	})

	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to update db: %w", err)
	}

	return addr, nil
}

func (w *DiskWallet) getKey(address core.Hash160) (*rsa.PrivateKey, error) {
	var sk *rsa.PrivateKey

	err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("keys"))

		skBytes := b.Get(address[:])
		if skBytes == nil {
			return fmt.Errorf("key not found")
		}

		var err error
		sk, err = x509.ParsePKCS1PrivateKey(skBytes)
		if err != nil {
			return fmt.Errorf("failed to parse key: %w", err)
		}

		return nil
	})

	return sk, err
}

func (w *DiskWallet) CreateTransaction(from, to core.Hash160, value, fee uint32) (*core.Transaction, error) {
	var inVal uint32
	sk, err := w.getKey(from)
	if err != nil {
		return nil, fmt.Errorf("failed to get key for address %s: %w", from, err)
	}

	txb := core.NewTransactionBuilder()
	err = w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("uxtos"))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			uRef := persistence.UXTORef{}
			uRef.SetBytes(k)
			uxto := marshal.DeserializeUXTO(v)

			if uxto.PubKeyHash == from {
				txb.AddInputFrom(uxto, &sk.PublicKey)

				inVal += uxto.Value
				if inVal > value {
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if inVal < value {
		return nil, fmt.Errorf("insufficient fund, balance=%d, want=%d", inVal, value)
	}
	txb.AddOutput(value, to)
	txb.AddChange(fee)

	return txb.Sign(sk), nil
}

func (w *DiskWallet) GetBalance(addr core.Hash160) (uint32, error) {
	var balance uint32

	err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("balances"))

		balanceBytes := b.Get(addr[:])
		if balanceBytes == nil {
			balance = 0
			return nil
		}

		balance = marshal.Uint32FromBytes(balanceBytes)

		return nil
	})

	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (w *DiskWallet) ListUnspent(addr core.Hash160) ([]*core.UXTO, error) {
	var uxtoList []*core.UXTO

	err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("uxtos"))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			uRef := persistence.UXTORef{}
			uRef.SetBytes(k)
			uxto := marshal.DeserializeUXTO(v)

			if uxto.PubKeyHash == addr {
				uxtoList = append(uxtoList, uxto)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return uxtoList, nil
}

func (w *DiskWallet) ListTransactions() ([]*core.Transaction, error) {
	var txList []*core.Transaction

	err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("transactions"))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			tx := marshal.UTransaction(v)
			txList = append(txList, tx)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return txList, nil
}

func (w *DiskWallet) ProcessTransaction(tx *core.Transaction) error {
	txId := tx.Hash()

	err := w.db.Update(func(btx *bolt.Tx) error {
		relevant := false // whether the database is updated
		uxtos := btx.Bucket([]byte("uxtos"))
		addresses := btx.Bucket([]byte("addresses"))
		balances := btx.Bucket([]byte("balances"))
		transactions := btx.Bucket([]byte("transactions"))

		// if an uxto occurs in input set, delete it
		for _, in := range tx.Ins {
			uRef := persistence.UXTORef{
				TxId: txId,
				N:    in.N,
			}

			uxtoBytes := uxtos.Get(uRef.Serialize())
			if uxtoBytes == nil {
				continue
			}
			relevant = true
			uxto := marshal.DeserializeUXTO(uxtoBytes)

			if err := uxtos.Delete(uRef.Serialize()); err != nil {
				return fmt.Errorf("failed to delete uxto: %w", err)
			}

			// update balance
			balance, _ := w.GetBalance(uxto.PubKeyHash)
			balances.Put(uxto.PubKeyHash[:], marshal.Uint32ToBytes(balance-uxto.Value))
		}

		// if output contains one of our addresses, add it
		for i, out := range tx.Outs {
			if addresses.Get(out.PubKeyHash[:]) != nil {
				relevant = true

				uRef := persistence.UXTORef{
					TxId: txId,
					N:    uint32(i),
				}

				newUXTO := &core.UXTO{
					TxId:  txId,
					N:     uint32(i),
					TxOut: out,
				}

				if err := uxtos.Put(uRef.Serialize(), marshal.SerializeUXTO(newUXTO)); err != nil {
					return fmt.Errorf("failed to put uxto: %w", err)
				}

				// update balance
				balance, _ := w.GetBalance(out.PubKeyHash)
				balances.Put(out.PubKeyHash[:], marshal.Uint32ToBytes(balance+out.Value))
			}
		}

		// record this transaction
		if relevant {
			if err := transactions.Put(txId[:], marshal.Transaction(tx)); err != nil {
				return fmt.Errorf("failed to put transaction: %w", err)
			}
		}

		return nil
	})

	return err
}

func (w *DiskWallet) ProcessBlock(block *core.Block) error {
	for _, tx := range block.Transactions {
		if err := w.ProcessTransaction(tx); err != nil {
			return fmt.Errorf("failed to process transaction %s: %w", tx.Hash(), err)
		}
	}

	return nil
}