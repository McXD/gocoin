package persistence

import (
	"fmt"
	"gocoin/internal/core"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestOpen(t *testing.T) {
	tmpPath := fmt.Sprintf("/tmp/blk_%x.index", core.RandomHash256().String())

	repo, err := NewBlockIndexRepo(tmpPath)
	if err != nil {
		t.Fatalf("cannot open repo: %s", err)
	}
	defer os.Remove(tmpPath)

	txId := core.RandomHash256()
	txRec := &TransactionRecord{
		BlockFileID: 123,
		BlockOffset: 12,
		TxOffset:    2,
	}

	blkId := core.RandomHash256()
	blkRec := &BlockIndexRecord{
		BlockHeader: core.BlockHeader{
			Time:           time.Now().Unix(),
			Bits:           20,
			Nonce:          1231414,
			HashPrevBlock:  core.RandomHash256(),
			HashMerkleRoot: core.RandomHash256(),
		},
		Height:      123,
		TxCount:     12,
		BlockFileID: 12,
		Offset:      2,
	}

	fileId := uint32(12)
	fileRec := &FileInfoRecord{
		BlockCount:    150,
		BlockFileSize: 123124124,
		UndoFileSize:  1231312,
	}

	// Transaction Record

	if err := repo.PutTransactionRecord(txId, txRec); err != nil {
		t.Errorf("cannot put record: %s", err)
	}

	gotTxRec, err := repo.GetTransactionRecord(txId)
	if err != nil {
		t.Errorf("cannot get record: %s", err)
	}

	if !reflect.DeepEqual(txRec, gotTxRec) {
		t.Fatalf("records not equal")
	}

	// Block Record

	if err := repo.PutBlockIndexRecord(blkId, blkRec); err != nil {
		t.Errorf("cannot put record: %s", err)
	}

	gotBlkRec, err := repo.GetBlockIndexRecord(blkId)
	if err != nil {
		t.Errorf("cannot get record: %s", err)
	}

	if !reflect.DeepEqual(blkRec, gotBlkRec) {
		t.Fatalf("records not equal")
	}

	// File Record

	if err := repo.PutFileInfoRecord(fileId, fileRec); err != nil {
		t.Errorf("cannot put record: %s", err)
	}

	gotFileRec, err := repo.GetFileInfoRecord(fileId)
	if err != nil {
		t.Errorf("cannot get record: %s", err)
	}

	if !reflect.DeepEqual(fileRec, gotFileRec) {
		t.Fatalf("records not equal")
	}

	// File Id

	if err := repo.PutCurrentFileId(fileId); err != nil {
		t.Errorf("cannot put file id: %s", err)
	}

	if gotFileId, _ := repo.GetCurrentFileId(); gotFileId != fileId {
		t.Errorf("got %d; want %d", gotFileId, fileId)
	}

	_ = repo.IncrementFileId()
	if gotFileId, _ := repo.GetCurrentFileId(); gotFileId != fileId+1 {
		t.Errorf("failed to increment file id: got %d", gotFileId)
	}
}
