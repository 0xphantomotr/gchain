package types

import (
	"testing"
	"time"
)

func TestTransactionHashDeterministic(t *testing.T) {
	tx := Transaction{
		From:      Address{1},
		To:        Address{2},
		Amount:    10,
		Nonce:     1,
		Timestamp: time.Unix(0, 123),
	}
	hash1 := tx.CalculateHash()
	hash2 := tx.CalculateHash()
	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash, got %s and %s", hash1.String(), hash2.String())
	}
}

func TestBlockHeaderHashChangesWithStateRoot(t *testing.T) {
	header := BlockHeader{
		Height:    1,
		Timestamp: time.Now(),
	}
	hash1 := header.Hash()
	header.StateRoot = Hash{1}
	hash2 := header.Hash()
	if hash1 == hash2 {
		t.Fatal("expected header hash to change when state root changes")
	}
}

func TestCalculateTxRootAggregatesTransactions(t *testing.T) {
	block := Block{
		Transactions: []Transaction{
			{From: Address{1}, To: Address{2}, Amount: 5, Timestamp: time.Now()},
			{From: Address{3}, To: Address{4}, Amount: 7, Timestamp: time.Now()},
		},
	}
	root := block.CalculateTxRoot()
	if root == (Hash{}) {
		t.Fatal("tx root should not be zero hash when block has transactions")
	}
}
