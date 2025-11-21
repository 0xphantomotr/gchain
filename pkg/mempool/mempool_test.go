package mempool

import (
	"testing"
	"time"

	"github.com/0xphantomotr/gchain/pkg/types"
)

func makeTx(ts int64) types.Transaction {
	tx := types.Transaction{
		From:      types.Address{1},
		To:        types.Address{2},
		Amount:    1,
		Timestamp: time.Unix(0, ts),
	}
	tx.Hash = tx.CalculateHash()
	return tx
}

func TestPendingRespectsOrder(t *testing.T) {
	pool := New(10, nil)
	tx1 := makeTx(1)
	tx2 := makeTx(2)
	if err := pool.Add(tx2); err != nil {
		t.Fatal(err)
	}
	if err := pool.Add(tx1); err != nil {
		t.Fatal(err)
	}

	pending := pool.Pending(2)
	if len(pending) != 2 || pending[0].Hash != tx2.Hash {
		t.Fatalf("expected tx2 first, got %#v", pending)
	}
}

func TestAddDeduplicates(t *testing.T) {
	pool := New(10, nil)
	tx := makeTx(1)
	if err := pool.Add(tx); err != nil {
		t.Fatal(err)
	}
	if err := pool.Add(tx); err != nil {
		t.Fatal(err)
	}
	if pool.Size() != 1 {
		t.Fatalf("expected size 1, got %d", pool.Size())
	}
}

func TestEvictWhenFull(t *testing.T) {
	pool := New(1, nil)
	tx1 := makeTx(1)
	tx2 := makeTx(2)
	_ = pool.Add(tx1)
	_ = pool.Add(tx2)

	pending := pool.Pending(2)
	if len(pending) != 1 || pending[0].Hash != tx2.Hash {
		t.Fatalf("expected only newest tx2")
	}
}

func TestRemoveDeletesEntry(t *testing.T) {
	pool := New(10, nil)
	tx := makeTx(1)
	_ = pool.Add(tx)
	pool.Remove(tx.Hash)
	if pool.Size() != 0 {
		t.Fatalf("expected empty pool")
	}
}
