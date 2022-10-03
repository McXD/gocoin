package persistence

import (
	"fmt"
	"gocoin/internal/core"
	"os"
	"reflect"
	"testing"
)

func TestAllChainState(t *testing.T) {
	PopulateTestData()

	tmpPath := fmt.Sprintf("/tmp/chainstate_%x.index", core.RandomHash256().String())

	repo, err := NewChainStateRepo(tmpPath)
	if err != nil {
		t.Fatalf("cannot open repo: %s", err)
	}
	defer os.Remove(tmpPath)

	u1 := USET.First(TXID[0])
	blkId := core.RandomHash256()

	// --- Put ---
	if err := repo.PutUXTO(u1); err != nil {
		t.Errorf("failed to put: %s", err)
	}

	// --- Get ---
	if u1G, err := repo.GetUXTO(u1.TxId, u1.N); err != nil {
		t.Errorf("failed to get: %s", err)
	} else if !reflect.DeepEqual(u1G, u1) {
		t.Errorf("objects not equal")
	}

	// --- Delete ---
	if err := repo.RemoveUXTO(u1.TxId, u1.N); err != nil {
		t.Errorf("failed to delete: %s", err)
	}

	// --- Set Block ---
	if err := repo.SetCurrentBlockHash(blkId); err != nil {
		t.Errorf("failed to set: %s", err)
	}

	// --- Get Block ---
	if blkIdG, err := repo.GetCurrentBlockHash(); err != nil {
		t.Errorf("failed to get: %s", err)
	} else if !reflect.DeepEqual(blkId, blkIdG) {
		t.Errorf("ids not equal")
	}
}
