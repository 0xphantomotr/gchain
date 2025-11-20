package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/0xphantomotr/gchain/pkg/types"
)

var (
	ErrNotFound          = errors.New("state: not found")
	ErrNonceMismatch     = errors.New("state: nonce mismatch")
	ErrInsufficientFunds = errors.New("state: insufficient funds")
)

type Account struct {
	Address types.Address `json:"address"`
	Balance uint64        `json:"balance"`
	Nonce   uint64        `json:"nonce"`
}

type Manager struct {
	mu    sync.RWMutex
	store Store
	cache map[types.Address]*Account
}

func NewManager(store Store) *Manager {
	return &Manager{
		store: store,
		cache: make(map[types.Address]*Account),
	}
}

type Store interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
	Delete(key []byte) error
}

func accountKey(addr types.Address) []byte {
	key := make([]byte, 0, len("acct:")+len(addr))
	key = append(key, []byte("acct:")...)
	key = append(key, addr[:]...)
	return key
}

func (m *Manager) GetAccount(addr types.Address) (*Account, error) {
	m.mu.RLock()
	if acc, ok := m.cache[addr]; ok {
		clone := *acc
		m.mu.RUnlock()
		return &clone, nil
	}
	m.mu.RUnlock()

	data, err := m.store.Get(accountKey(addr))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &Account{Address: addr}, nil
		}
		return nil, fmt.Errorf("get account %s: %w", addr.String(), err)
	}

	var acct Account
	if err := json.Unmarshal(data, &acct); err != nil {
		return nil, fmt.Errorf("decode account %s: %w", addr.String(), err)
	}

	m.mu.Lock()
	m.cache[addr] = &acct
	m.mu.Unlock()

	clone := acct
	return &clone, nil
}

func (m *Manager) ApplyTransaction(tx types.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.applyTransactionLocked(tx)
}

func (m *Manager) ApplyBlock(block types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot := m.cloneCache()
	for _, tx := range block.Transactions {
		if err := m.applyTransactionLocked(tx); err != nil {
			m.cache = snapshot
			return fmt.Errorf("apply tx %s: %w", tx.Hash.String(), err)
		}
	}
	if err := m.commitLocked(); err != nil {
		return fmt.Errorf("commit block height=%d: %w", block.Header.Height, err)
	}
	return nil
}

func (m *Manager) applyTransactionLocked(tx types.Transaction) error {
	sender := m.getOrCreate(tx.From)
	receiver := m.getOrCreate(tx.To)

	if sender.Nonce != tx.Nonce {
		return ErrNonceMismatch
	}
	if sender.Balance < tx.Amount {
		return ErrInsufficientFunds
	}

	sender.Balance -= tx.Amount
	sender.Nonce++
	receiver.Balance += tx.Amount
	return nil
}

func (m *Manager) getOrCreate(addr types.Address) *Account {
	if acc, ok := m.cache[addr]; ok {
		return acc
	}
	acc := &Account{Address: addr}
	m.cache[addr] = acc
	return acc
}

func (m *Manager) cloneCache() map[types.Address]*Account {
	dup := make(map[types.Address]*Account, len(m.cache))
	for addr, acct := range m.cache {
		clone := *acct
		dup[addr] = &clone
	}
	return dup
}

func (m *Manager) commitLocked() error {
	for addr, acct := range m.cache {
		payload, err := json.Marshal(acct)
		if err != nil {
			return fmt.Errorf("marshal account %s: %w", addr.String(), err)
		}
		if err := m.store.Set(accountKey(addr), payload); err != nil {
			return fmt.Errorf("persist account %s: %w", addr.String(), err)
		}
	}
	return nil
}
