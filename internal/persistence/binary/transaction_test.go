package binary

import (
	"github.com/davecgh/go-spew/spew"
	"gocoin/internal/core"
	"reflect"
	"testing"
)

func TestDeserializeTransaction(t *testing.T) {
	PopulateTestData()

	tx := core.NewCoinBaseTransaction([]byte("coin!"), core.RandomHash160(), 1000, 10)

	buf := SerializeTransaction(tx)
	txDes := DeserializeTransaction(buf)

	t.Logf("%s", spew.Sdump(tx))
	t.Logf("%s", spew.Sdump(txDes))

	if !reflect.DeepEqual(tx, txDes) {
		t.Errorf("Object not equal")
	}

	// --- General Transaction ---

	tx = core.NewTransactionBuilder().
		AddInputFrom(USET.First(TXID[1]), PK[0]).
		AddInputFrom(USET.First(TXID[0]), PK[0]).
		AddOutput(50, ADDR[2]).
		AddChange(1).
		Sign(SK[0])

	buf = SerializeTransaction(tx)
	txDes = DeserializeTransaction(buf)

	t.Logf("%s", spew.Sdump(tx))
	t.Logf("%s", spew.Sdump(txDes))

	if !reflect.DeepEqual(tx, txDes) {
		t.Errorf("Object not equal")
	}
}
