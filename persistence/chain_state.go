package persistence

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"gocoin/core"
	"gocoin/marshal"
	"os"
	"time"
)

var ErrNotFound = errors.New("record not found")

// UXTORef is a reference to a UXTO. This is singled out because we need to use this as a key in the map.
type UXTORef struct {
	TxId core.Hash256
	N    uint32
}

func NewUXTORef(u *core.UXTO) *UXTORef {
	return &UXTORef{
		TxId: u.TxId,
		N:    u.N,
	}
}

func (ur *UXTORef) Serialize() []byte {
	var buf []byte

	buf = append(buf, ur.TxId[:]...)
	buf = append(buf, marshal.Uint32ToBytes(ur.N)...)

	return buf
}

func (ur *UXTORef) SetBytes(buf []byte) {
	ur.TxId = core.Hash256FromSlice(buf[:32])
	ur.N = marshal.Uint32FromBytes(buf[32:36])
}

type ChainStateRepo struct {
	db *bolt.DB
}

func NewChainStateRepo(rootDir string) (*ChainStateRepo, error) {
	repo := &ChainStateRepo{db: nil}

	if err := os.Mkdir(rootDir+"/db", os.ModePerm); !os.IsExist(err) {
		return nil, fmt.Errorf("cannot create db directory: %v", err)
	}

	db, err := bolt.Open(rootDir+"/db/chain_state.dat", 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	repo.db = db

	// create two buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("C")); err != nil {
			return fmt.Errorf("cannot create 'C': %w", err)
		} // txId:N -> UXTO
		if _, err := tx.CreateBucketIfNotExists([]byte("B")); err != nil {
			return fmt.Errorf("cannot create 'B': %w", err)
		} // "B" -> Hash256 (Terminating Block)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("cannot create buckets: %w", err)
	}

	return repo, nil
}

func (repo *ChainStateRepo) PutUXTO(u *core.UXTO) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("C"))
		err := b.Put(NewUXTORef(u).Serialize(), marshal.SerializeUXTO(u))
		return err
	})

	return err
}

func (repo *ChainStateRepo) GetUXTO(txId core.Hash256, n uint32) *core.UXTO {
	uxto := &core.UXTO{
		TxId:  core.Hash256{},
		N:     0,
		TxOut: nil,
	}

	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("C"))
		ret := b.Get((&UXTORef{
			TxId: txId,
			N:    n,
		}).Serialize())
		if ret == nil {
			return ErrNotFound
		}
		uxto = marshal.DeserializeUXTO(ret)
		return nil
	})

	if err != nil {
		return nil
	}

	return uxto
}

func (repo *ChainStateRepo) RemoveUXTO(txId core.Hash256, n uint32) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("C"))
		return b.Delete((&UXTORef{
			TxId: txId,
			N:    n,
		}).Serialize())
	})

	return err
}

func (repo *ChainStateRepo) GetCurrentBlockHash() (core.Hash256, error) {
	h := core.Hash256{}

	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("B"))
		ret := b.Get([]byte("B"))
		if ret == nil {
			return ErrNotFound
		}
		h = core.Hash256FromSlice(ret)
		return nil
	})

	return h, err
}

func (repo *ChainStateRepo) SetCurrentBlockHash(hash core.Hash256) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("B"))
		err := b.Put([]byte("B"), hash[:])
		return err
	})

	return err
}
