package persistence

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gocoin/core"
	"gocoin/marshal"
	"os"
)

const (
	S_UXTO                 = 60 // size of an UXTO
	MAGIC_DIV_BLOCK uint64 = 0x11_22_33_44_55_66_77_88
)

var DIV_BLOCK []byte

func init() {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], MAGIC_DIV_BLOCK)
	DIV_BLOCK = buf[:]
}

type BlockFile struct {
	Id           uint32
	blkFileSize  int
	undoFileSize int
	bf           *os.File
	rf           *os.File
	Revs         [][]*core.UXTO
	Blocks       []*core.Block // in-memory cache
}

// NewBlockFile creates or opens a block file and the corresponding rev file
func NewBlockFile(rootDir string, id uint32) (*BlockFile, error) {
	blkFilePath := fmt.Sprintf("%s/data/blk_%06d.dat", rootDir, id)
	revFilePath := fmt.Sprintf("%s/data/rev_%06d.dat", rootDir, id)
	bf, err := os.OpenFile(blkFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	rf, err := os.OpenFile(revFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	blockFile := &BlockFile{
		Id:           id,
		blkFileSize:  0,
		undoFileSize: 0,
		bf:           bf,
		rf:           rf,
		Blocks:       make([]*core.Block, 0),
		Revs:         make([][]*core.UXTO, 0),
	}

	var buf [100_000]byte
	var pos int64

	// Read blk files
	for {
		n, _ := bf.ReadAt(buf[:], pos)
		if n == 0 { // no more reads
			break
		}
		blockFile.blkFileSize += n

		slices := bytes.Split(buf[:], DIV_BLOCK)

		for i, slice := range slices {
			if i != len(slices)-1 { // last slice may be incomplete
				blockFile.Blocks = append(blockFile.Blocks, marshal.UBlock(slice))
				pos += int64(len(slice))
			}

			pos += 8 // separator
		}
	}

	// Read rev file
	buf = [100_000]byte{}
	pos = 0

	for {
		n, _ := rf.ReadAt(buf[:], pos)
		if n == 0 {
			break
		}
		blockFile.undoFileSize += n

		blockUXTOs := bytes.Split(buf[:], DIV_BLOCK) // UXTO for each block

		for i, uxtos := range blockUXTOs {
			if i != len(blockUXTOs)-1 { // discard the last one
				count := len(uxtos) / S_UXTO
				blockFile.Revs = append(blockFile.Revs, []*core.UXTO{}) // initialize the slice
				for j := 0; j < count; j++ {
					blockFile.Revs[i] = append(blockFile.Revs[i], marshal.DeserializeUXTO(uxtos[S_UXTO*j:S_UXTO+S_UXTO*j]))
					pos += S_UXTO
				}
				pos += 8 // separator
			}
		}
	}

	return blockFile, nil
}

func (blockFile *BlockFile) WriteBlock(b *core.Block, uxtos []*core.UXTO) error {
	blockData := marshal.Block(b)
	blockData = append(blockData, DIV_BLOCK...)
	if _, err := blockFile.bf.Write(blockData); err != nil {
		return fmt.Errorf("failed to write to block file %d: %w", blockFile.Id, err)
	}
	blockFile.blkFileSize += len(blockData)

	var uxtoData []byte
	for _, u := range uxtos {
		uxtoData = append(uxtoData, marshal.SerializeUXTO(u)...)
	}
	uxtoData = append(uxtoData, DIV_BLOCK...)
	if _, err := blockFile.rf.Write(uxtoData); err != nil {
		return fmt.Errorf("failed to write to rev file: %w", err)
	}
	blockFile.undoFileSize += len(uxtoData)

	blockFile.Blocks = append(blockFile.Blocks, b)
	blockFile.Revs = append(blockFile.Revs, uxtos)
	return nil
}

func (blockFile *BlockFile) Close() error {
	return blockFile.bf.Close()
}

func (blockFile *BlockFile) GetBlockFileSize() int {
	return blockFile.blkFileSize
}

func (blockFile *BlockFile) GetUndoFileSize() int {
	return blockFile.undoFileSize
}

func (blockFile *BlockFile) GetBlockCount() int {
	return len(blockFile.Blocks)
}

func GetBlock(rootDir string, blkRec BlockIndexRecord) (*core.Block, error) {
	bf, err := NewBlockFile(rootDir, blkRec.BlockFileID)
	defer bf.Close()

	if err != nil {
		return nil, fmt.Errorf("cannot open block file: %w", err)
	}

	return bf.Blocks[blkRec.Offset], nil
}

func GetTransaction(rootDir string, txRec *TransactionRecord) (*core.Transaction, error) {
	bf, err := NewBlockFile(rootDir, txRec.BlockFileID)
	defer bf.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot open block file: %w", err)
	}

	tx := bf.Blocks[txRec.BlockOffset].Transactions[txRec.TxOffset]

	return tx, nil
}
