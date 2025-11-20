package state

import (
	"errors"
	"testing"
	"time"

	"github.com/0xphantomotr/gchain/pkg/types"
)

func TestApplyTransactionUpdatesBalances(t *testing.T) {
	store := NewMemoryStore()
	mgr := NewManager(store)

	sender := types.Address{1}
	receiver := types.Address{2}
	mgr.cache[sender] = &Account{Address: sender, Balance: 100}

	tx := types.Transaction{
		From:      sender,
		To:        receiver,
		Amount:    40,
		Nonce:     0,
		Timestamp: time.Unix(0, 0),
	}

	if err := mgr.ApplyTransaction(tx); err != nil {
		t.Fatalf("apply transaction: %v", err)
	}

	gotSender, _ := mgr.GetAccount(sender)
	if gotSender.Balance != 60 {
		t.Fatalf("expected sender balance 60, got %d", gotSender.Balance)
	}
	gotReceiver, _ := mgr.GetAccount(receiver)
	if gotReceiver.Balance != 40 {
		t.Fatalf("expected receiver balance 40, got %d", gotReceiver.Balance)
	}
}

func TestApplyTransactionNonceMismatch(t *testing.T) {
	store := NewMemoryStore()
	mgr := NewManager(store)

	sender := types.Address{1}
	mgr.cache[sender] = &Account{Address: sender, Balance: 100, Nonce: 1}

	tx := types.Transaction{From: sender, To: types.Address{2}, Amount: 10, Nonce: 0, Timestamp: time.Unix(0, 0)}
	if err := mgr.ApplyTransaction(tx); !errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("expected nonce mismatch error, got %v", err)
	}
}

func TestApplyBlockRollbackOnFailure(t *testing.T) {
	store := NewMemoryStore()
	mgr := NewManager(store)

	sender := types.Address{1}
	mgr.cache[sender] = &Account{Address: sender, Balance: 30}

	block := types.Block{
		Header: types.BlockHeader{Height: 1},
		Transactions: []types.Transaction{
			{From: sender, To: types.Address{2}, Amount: 40, Nonce: 0, Timestamp: time.Unix(0, 0)},
		},
	}

	if err := mgr.ApplyBlock(block); err == nil {
		t.Fatalf("expected insufficient funds error")
	}
	got, _ := mgr.GetAccount(sender)
	if got.Balance != 30 {
		t.Fatalf("rollback failed, balance is %d", got.Balance)
	}
}
