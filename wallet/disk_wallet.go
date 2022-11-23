package wallet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"github.com/boltdb/bolt"
	log "github.com/sirupsen/logrus"
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

	log.Infof("added new address: %s", addr.String())

	return addr, nil
}

func (w *DiskWallet) ListAddresses() []core.Hash160 {
	var addresses []core.Hash160

	_ = w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("addresses"))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			addresses = append(addresses, core.Hash160FromSlice(k))
		}
		return nil
	})

	return addresses
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
				if inVal >= value+fee {
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

// GetBalances sums up all the UXTOS for all addresses.
func (w *DiskWallet) GetBalances() map[core.Hash160]uint32 {
	balances := make(map[core.Hash160]uint32)
	for _, addr := range w.ListAddresses() {
		balances[addr] = 0
	}

	_ = w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("uxtos"))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			uxto := marshal.DeserializeUXTO(v)
			balances[uxto.PubKeyHash] += uxto.Value
		}

		return nil
	})

	return balances
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
		transactions := btx.Bucket([]byte("transactions"))

		// if an uxto occurs in input set, delete it
		log.Debugf("Processing inputs of transaction %s", txId)
		for _, in := range tx.Ins {
			uRef := persistence.UXTORef{
				TxId: in.PrevTxId,
				N:    in.N,
			}

			log.Debugf("Processsing input %s:%d", uRef.TxId, uRef.N)

			uxtoBytes := uxtos.Get(uRef.Serialize())
			if uxtoBytes == nil {
				continue
			}
			relevant = true

			if err := uxtos.Delete(uRef.Serialize()); err != nil {
				return fmt.Errorf("failed to delete uxto: %w", err)
			}

			log.Infof("Deleted uxto: txId=%s, vout=%d", uRef.TxId, uRef.N)
		}

		log.Debugf("Processing outputs of transaction %s", txId)
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

				log.Infof("Added uxto: txId=%s, vout=%d, value=%d", uRef.TxId, uRef.N, out.Value)
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

func (w *DiskWallet) ProcessBlock(block *core.Block) {
	for _, tx := range block.Transactions {
		if err := w.ProcessTransaction(tx); err != nil {
			log.Errorf("Failed to process transaction %s: %v", tx.Hash(), err)
		}
	}
}

func (w *DiskWallet) RollBack(block *core.Block, spent []*core.UXTO) {
	err := w.db.Update(func(tx *bolt.Tx) error {
		uxtos := tx.Bucket([]byte("uxtos"))
		addresses := tx.Bucket([]byte("addresses"))

		for _, uxto := range spent {
			if addresses.Get(uxto.TxOut.PubKeyHash[:]) == nil {
				// not ours
				continue
			}

			uRef := persistence.UXTORef{
				TxId: uxto.TxId,
				N:    uxto.N,
			}

			if err := uxtos.Put(uRef.Serialize(), marshal.SerializeUXTO(uxto)); err != nil {
				return fmt.Errorf("failed to put uxto: %w", err)
			}

			log.Infof("Added uxto (Rollback): txId=%s, vout=%d, value=%d", uRef.TxId, uRef.N, uxto.Value)
		}

		generated := core.GenerateUXTOsFromBlock(block)
		for _, uxto := range generated {
			uRef := persistence.UXTORef{
				TxId: uxto.TxId,
				N:    uxto.N,
			}

			if uxtos.Get(uRef.Serialize()) == nil {
				// not ours
				continue
			}

			if err := uxtos.Delete(uRef.Serialize()); err != nil {
				return fmt.Errorf("failed to delete uxto: %w", err)
			}

			log.Infof("Deleted uxto (Rollback): txId=%s, vout=%d, value=%d", uRef.TxId, uRef.N, uxto.Value)
		}

		return nil
	})

	if err != nil {
		log.Errorf("Failed to rollback: %v", err)
	}
}
