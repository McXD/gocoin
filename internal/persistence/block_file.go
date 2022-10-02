package persistence

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"gocoin/internal/core"
	"os"
)

const (
	MAGIC    = 0x11223344
	BLK_BASE = "data/blk"
	REV_BASE = "data/rev"
	N_BLK    = 100
)

var DIV []byte

func init() {
	buf := [8]byte{}
	binary.PutVarint(buf[:], MAGIC)
	DIV = buf[:]
}

type BlockFile struct {
	f      *os.File
	revs   [][]core.UXTO
	blocks []core.Block // in-memory cache
	enc    *gob.Encoder
}

func blkFileName(id int) string {
	return fmt.Sprintf("%s/blk_%d.dat", BLK_BASE, id)
}

func revFileName(id int) string {
	return fmt.Sprintf("%s/rev_%d.dat", REV_BASE, id)
}

func Open(id int) (*BlockFile, error) {
	homeDir, err := os.UserHomeDir()
	absFileName := fmt.Sprintf("%s/.gocoin/%s", homeDir, blkFileName(id))

	f, err := os.OpenFile(absFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	bf := &BlockFile{
		f:      f,
		blocks: []core.Block{},
		enc:    gob.NewEncoder(f),
	}

	var buf [100_000]byte
	var pos int64
	var currentBlock core.Block

	for {
		n, _ := f.ReadAt(buf[:], pos)
		if n == 0 { // no more reads
			break
		}

		slices := bytes.Split(buf[:], DIV)

		for i, slice := range slices {
			if i != len(slices)-1 { // last slice may be incomplete
				if err := gob.NewDecoder(bytes.NewReader(slice)).Decode(&currentBlock); err != nil {
					return nil, fmt.Errorf("error decoding block: %w", err)
				}

				bf.blocks = append(bf.blocks, currentBlock)
			}
		}

		pos = pos + int64(n) - int64(len(slices[len(slices)-1]))
	}

	return bf, nil
}

func (bf *BlockFile) WriteBlock(b *core.Block) error {
	if len(bf.blocks) == N_BLK {
		return fmt.Errorf("blocks reached max amount")
	}

	if err := bf.enc.Encode(*b); err != nil {
		return err
	}

	if _, err := bf.f.Write(DIV); err != nil {
		return err
	}

	return nil
}

func (bf *BlockFile) Close() error {
	return bf.f.Close()
}
