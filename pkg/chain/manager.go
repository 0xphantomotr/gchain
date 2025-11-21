package chain

import (
	"errors"
	"fmt"
	"sync"

	"github.com/0xphantomotr/gchain/pkg/types"
)

var (
	ErrBlockNotFound    = errors.New("chain: block not found")
	ErrUnexpectedHeight = errors.New("chain: unexpected block height")
	ErrBadPrevHash      = errors.New("chain: previous hash mismatch")
)

type Store interface {
	SaveBlock(block *types.Block) error
	GetBlockByHeight(height uint64) (*types.Block, error)
	GetBlockByHash(hash types.Hash) (*types.Block, error)
	SetCannoicalHeight(height uint64) error
	GetCannoicalHeight() (uint64, error)
}

type Manager struct {
	mu      sync.RWMutex
	store   Store
	tip     uint64
	tipHash types.Hash
}

func NewManager(store Store) (*Manager, error) {
	height, err := store.GetCannoicalHeight()
	if err != nil && !errors.Is(err, ErrBlockNotFound) {
		return nil, fmt.Errorf("load cannoical height: %w", err)
	}
	var tipHash types.Hash
	if height > 0 {
		block, err := store.GetBlockByHeight(height)
		if err != nil {
			return nil, fmt.Errorf("load tip block: %w", err)
		}
		tipHash = block.Header.Hash()
	}

	return &Manager{
		store:   store,
		tip:     height,
		tipHash: tipHash,
	}, nil
}

func (m *Manager) AddBlock(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	expectedHeight := m.tip + 1
	if block.Header.Height != expectedHeight {
		return fmt.Errorf("%w: got %d want %d", ErrUnexpectedHeight, block.Header.Height, expectedHeight)
	}
	if expectedHeight > 1 && block.Header.PreviousHash != m.tipHash {
		return ErrBadPrevHash
	}
	if block.Header.TxRoot == (types.Hash{}) {
		block.Header.TxRoot = block.CalculateTxRoot()
	}

	if err := m.store.SaveBlock(block); err != nil {
		return fmt.Errorf("save block: %w", err)
	}
	if err := m.store.SetCannoicalHeight(block.Header.Height); err != nil {
		return fmt.Errorf("persist cannoical height: %w", err)
	}

	m.tip = block.Header.Height
	m.tipHash = block.Header.Hash()
	return nil
}

func (m *Manager) GetBlockByHeight(height uint64) (*types.Block, error) {
	return m.store.GetBlockByHeight(height)
}

func (m *Manager) GetBlockByHash(hash types.Hash) (*types.Block, error) {
	return m.store.GetBlockByHash(hash)
}

func (m *Manager) Tip() (uint64, types.Hash) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tip, m.tipHash
}
