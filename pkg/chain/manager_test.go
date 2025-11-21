package chain

import (
	"testing"
	"time"

	"github.com/0xphantomotr/gchain/pkg/types"
)

func makeBlock(height uint64, prev types.Hash) *types.Block {
	block := &types.Block{
		Header: types.BlockHeader{
			Height:       height,
			PreviousHash: prev,
			StateRoot:    types.Hash{byte(height)},
			Proposer:     types.Address{byte(height)},
			Timestamp:    time.Unix(int64(height), 0),
		},
		Transactions: []types.Transaction{
			{From: types.Address{1}, To: types.Address{2}, Amount: height, Timestamp: time.Unix(0, int64(height))},
		},
	}
	block.Header.TxRoot = block.CalculateTxRoot()
	return block
}

func TestAddBlockUpdatesTip(t *testing.T) {
	store := NewMemoryStore()
	mgr, err := NewManager(store)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	block := makeBlock(1, types.Hash{})
	if err := mgr.AddBlock(block); err != nil {
		t.Fatalf("add block: %v", err)
	}

	height, hash := mgr.Tip()
	if height != 1 {
		t.Fatalf("expected tip height 1, got %d", height)
	}
	if hash != block.Header.Hash() {
		t.Fatalf("expected hash mismatch")
	}

	stored, err := mgr.GetBlockByHeight(1)
	if err != nil {
		t.Fatalf("get block: %v", err)
	}
	if stored.Header.StateRoot != block.Header.StateRoot {
		t.Fatalf("stored block differs")
	}
}

func TestAddBlockRejectsWrongHeight(t *testing.T) {
	store := NewMemoryStore()
	mgr, _ := NewManager(store)

	block := makeBlock(2, types.Hash{})
	if err := mgr.AddBlock(block); err == nil {
		t.Fatal("expected height error, got nil")
	}
}

func TestAddBlockRejectsBadPrevHash(t *testing.T) {
	store := NewMemoryStore()
	mgr, _ := NewManager(store)

	if err := mgr.AddBlock(makeBlock(1, types.Hash{})); err != nil {
		t.Fatalf("add first block: %v", err)
	}
	badBlock := makeBlock(2, types.Hash{1})
	if err := mgr.AddBlock(badBlock); err == nil {
		t.Fatal("expected prev hash mismatch")
	}
}

func TestCanonicalHeightPersists(t *testing.T) {
	store := NewMemoryStore()
	mgr, err := NewManager(store)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	block1 := makeBlock(1, types.Hash{})
	if err := mgr.AddBlock(block1); err != nil {
		t.Fatalf("add block1: %v", err)
	}

	block2 := makeBlock(2, block1.Header.Hash())
	if err := mgr.AddBlock(block2); err != nil {
		t.Fatalf("add block2: %v", err)
	}

	height, _ := mgr.Tip()
	if height != 2 {
		t.Fatalf("expected tip height 2, got %d", height)
	}

	newMgr, err := NewManager(store)
	if err != nil {
		t.Fatalf("new manager reload: %v", err)
	}
	reloadedHeight, _ := newMgr.Tip()
	if reloadedHeight != 2 {
		t.Fatalf("expected persisted height 2, got %d", reloadedHeight)
	}
}
