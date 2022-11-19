package persistence

import (
	"fmt"
	"github.com/boltdb/bolt"
	"gocoin/core"
	"gocoin/marshal"
	"os"
	"time"
)

type BlockIndexRecord struct {
	core.BlockHeader
	Height      uint32
	TxCount     uint32
	BlockFileID uint32
	Offset      uint32
}

func (b *BlockIndexRecord) Marshall() []byte {
	var buf []byte

	buf = append(buf, marshal.BlockHeader(&b.BlockHeader)...)
	buf = append(buf, marshal.Uint32ToBytes(b.Height)...)
	buf = append(buf, marshal.Uint32ToBytes(b.TxCount)...)
	buf = append(buf, marshal.Uint32ToBytes(b.BlockFileID)...)
	buf = append(buf, marshal.Uint32ToBytes(b.Offset)...)

	return buf
}

func UBlockIndexRecord(buf []byte) *BlockIndexRecord {
	record := &BlockIndexRecord{
		BlockHeader: core.BlockHeader{},
		Height:      0,
		TxCount:     0,
		BlockFileID: 0,
		Offset:      0,
	}

	p := 0
	record.BlockHeader = *marshal.UBlockHeader(buf[:marshal.S_BLOCKHEADER])

	p += marshal.S_BLOCKHEADER
	record.Height = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.TxCount = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.BlockFileID = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.Offset = marshal.Uint32FromBytes(buf[p : p+4])

	return record
}

type FileInfoRecord struct {
	BlockCount    uint32
	BlockFileSize uint32
	UndoFileSize  uint32
}

func (r *FileInfoRecord) Marshall() []byte {
	var buf []byte

	buf = append(buf, marshal.Uint32ToBytes(r.BlockCount)...)
	buf = append(buf, marshal.Uint32ToBytes(r.BlockFileSize)...)
	buf = append(buf, marshal.Uint32ToBytes(r.UndoFileSize)...)

	return buf
}

func UFileInfoRecord(buf []byte) *FileInfoRecord {
	record := &FileInfoRecord{
		BlockCount:    0,
		BlockFileSize: 0,
		UndoFileSize:  0,
	}

	p := 0
	record.BlockCount = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.BlockFileSize = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.UndoFileSize = marshal.Uint32FromBytes(buf[p : p+4])

	return record
}

type TransactionRecord struct {
	BlockFileID uint32
	BlockOffset uint32
	TxOffset    uint32
}

func (r *TransactionRecord) Marshall() []byte {
	var buf []byte

	buf = append(buf, marshal.Uint32ToBytes(r.BlockFileID)...)
	buf = append(buf, marshal.Uint32ToBytes(r.BlockOffset)...)
	buf = append(buf, marshal.Uint32ToBytes(r.TxOffset)...)

	return buf
}

func UTransactionRecord(buf []byte) *TransactionRecord {
	record := &TransactionRecord{
		BlockFileID: 0,
		BlockOffset: 0,
		TxOffset:    0,
	}

	p := 0
	record.BlockFileID = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.BlockOffset = marshal.Uint32FromBytes(buf[p : p+4])

	p += 4
	record.TxOffset = marshal.Uint32FromBytes(buf[p : p+4])

	return record
}

type BlockIndexRepo struct {
	db *bolt.DB
}

func NewBlockIndexRepo(rootDir string) (*BlockIndexRepo, error) {
	repo := &BlockIndexRepo{db: nil}

	if err := os.Mkdir(rootDir+"/db", os.ModePerm); !os.IsExist(err) {
		return nil, fmt.Errorf("cannot create db directory: %v", err)
	}

	db, err := bolt.Open(rootDir+"/db/block_index.dat", 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	repo.db = db

	// create four buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("b")); err != nil {
			return fmt.Errorf("cannot create 'b': %w", err)
		} // Block Index
		if _, err := tx.CreateBucketIfNotExists([]byte("f")); err != nil {
			return fmt.Errorf("cannot create 'f': %w", err)
		} // File Index
		if _, err := tx.CreateBucketIfNotExists([]byte("t")); err != nil {
			return fmt.Errorf("cannot create 't': %w", err)
		} // Transaction Index
		if _, err := tx.CreateBucketIfNotExists([]byte("l")); err != nil {
			return fmt.Errorf("cannot create 'l': %w", err)
		} // Counter
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create buckets: %w", err)
	}

	return repo, nil
}

func (repo *BlockIndexRepo) PutTransactionRecord(txId core.Hash256, r *TransactionRecord) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("t"))
		err := b.Put(txId[:], r.Marshall())
		return err
	})

	return err
}

func (repo *BlockIndexRepo) GetTransactionRecord(txId core.Hash256) (*TransactionRecord, error) {
	var tr *TransactionRecord

	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("t"))
		ret := b.Get(txId[:])
		if ret == nil {
			return fmt.Errorf("record not found")
		}

		tr = UTransactionRecord(ret)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tr, nil
}

func (repo *BlockIndexRepo) PutBlockIndexRecord(blkId core.Hash256, r *BlockIndexRecord) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("b"))
		err := b.Put(blkId[:], r.Marshall())
		return err
	})

	return err
}

func (repo *BlockIndexRepo) GetBlockIndexRecord(blkId core.Hash256) (*BlockIndexRecord, error) {
	var tr *BlockIndexRecord

	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("b"))
		ret := b.Get(blkId[:])
		if ret == nil {
			return ErrNotFound
		}
		tr = UBlockIndexRecord(ret)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tr, nil
}

func (repo *BlockIndexRepo) PutFileInfoRecord(fileId uint32, r *FileInfoRecord) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("f"))
		err := b.Put(marshal.Uint32ToBytes(fileId), r.Marshall())
		return err
	})

	return err
}

func (repo *BlockIndexRepo) GetFileInfoRecord(fileId uint32) (*FileInfoRecord, error) {
	var tr *FileInfoRecord

	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("f"))
		ret := b.Get(marshal.Uint32ToBytes(fileId))
		if ret == nil {
			return fmt.Errorf("record not found")
		}
		tr = UFileInfoRecord(ret)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tr, nil
}

func (repo *BlockIndexRepo) PutCurrentFileId(id uint32) error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("l"))
		if err := b.Put([]byte("l"), marshal.Uint32ToBytes(id)); err != nil {
			return fmt.Errorf("failed to put Id: %w", err)
		}
		return nil
	})

	return err
}

func (repo *BlockIndexRepo) IncrementFileId() error {
	err := repo.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("l"))
		id := marshal.Uint32FromBytes(b.Get([]byte("l")))
		if err := b.Put([]byte("l"), marshal.Uint32ToBytes(id+1)); err != nil {
			return fmt.Errorf("failed to put Id: %w", err)
		}
		return nil
	})

	return err
}

func (repo *BlockIndexRepo) GetCurrentFileId() (uint32, error) {
	var id uint32

	err := repo.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("l"))
		ret := b.Get([]byte("l"))
		if ret == nil {
			return ErrNotFound
		}
		id = marshal.Uint32FromBytes(ret)
		return nil
	})

	if err != nil {
		return 0, err
	}

	return id, nil
}
