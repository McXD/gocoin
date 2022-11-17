package persistence

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gocoin/internal/core"
	myBinary "gocoin/internal/persistence/binary"
	"os"
)

const (
	S_UXTO                 = 60 // size of an UXTO
	MAGIC_DIV_BLOCK uint64 = 0x11_22_33_44_55_66_77_88
	BLK_BASE               = "data"
	REV_BASE               = "data"
)

var DIV_BLOCK []byte

func init() {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], MAGIC_DIV_BLOCK)
	DIV_BLOCK = buf[:]
}

type BlockFile struct {
	blkFileSize int
	bf          *os.File
	rf          *os.File
	revs        [][]core.UXTO
	blocks      []core.Block // in-memory cache
}

func blkFileName(id uint32) string {
	return fmt.Sprintf("%s/blk_%06d.dat", BLK_BASE, id)
}

func revFileName(id uint32) string {
	return fmt.Sprintf("%s/rev_%06d.dat", REV_BASE, id)
}

func OpenBlockFile(rootDir string, id uint32) (*BlockFile, error) {
	// open or create files
	absBlkFileName := fmt.Sprintf("%s/%s", rootDir, blkFileName(id))
	absRevFileName := fmt.Sprintf("%s/%s", rootDir, revFileName(id))
	bf, err := os.OpenFile(absBlkFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	rf, err := os.OpenFile(absRevFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	blockFile := &BlockFile{
		blkFileSize: 0,
		bf:          bf,
		rf:          rf,
		blocks:      make([]core.Block, 0),
		revs:        make([][]core.UXTO, 0),
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
				blockFile.blocks = append(blockFile.blocks, *myBinary.DeserializeBlock(slice))
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

		blockUXTOs := bytes.Split(buf[:], DIV_BLOCK) // UXTO for each block

		for i, uxtos := range blockUXTOs {
			if i != len(blockUXTOs)-1 { // discard the last one
				count := len(uxtos) / S_UXTO
				blockFile.revs = append(blockFile.revs, []core.UXTO{}) // initialize the slice
				for j := 0; j < count; j++ {
					blockFile.revs[i] = append(blockFile.revs[i], *myBinary.DeserializeUXTO(uxtos[S_UXTO*j : S_UXTO+S_UXTO*j]))
					pos += S_UXTO
				}
				pos += 8 // separator
			}
		}
	}

	return blockFile, nil
}

func (blockFile *BlockFile) WriteBlock(b *core.Block, uxtos []*core.UXTO) error {
	blockData := myBinary.SerializeBlock(b)
	blockData = append(blockData, DIV_BLOCK...)

	if _, err := blockFile.bf.Write(blockData); err != nil {
		return fmt.Errorf("failed to write to block file: %w", err)
	}

	var uxtoData []byte
	for _, u := range uxtos {
		uxtoData = append(uxtoData, myBinary.SerializeUXTO(u)...)
	}
	uxtoData = append(uxtoData, DIV_BLOCK...)

	if _, err := blockFile.rf.Write(uxtoData); err != nil {
		return fmt.Errorf("failed to write to rev file: %w", err)
	}

	return nil
}

func (blockFile *BlockFile) Close() error {
	return blockFile.bf.Close()
}

func (blockFile *BlockFile) Size() int {
	return blockFile.blkFileSize
}

func (blockFile *BlockFile) GetBlockSize() int {
	return len(blockFile.blocks)
}
