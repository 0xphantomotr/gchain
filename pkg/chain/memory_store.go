package chain

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/0xphantomotr/gchain/pkg/types"
)

type MemoryStore struct {
	mu                sync.RWMutex
	blocksByHeight    map[uint64]*types.Block
	blocksByHash      map[types.Hash]*types.Block
	cannoicalByHeight uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		blocksByHeight: make(map[uint64]*types.Block),
		blocksByHash:   make(map[types.Hash]*types.Block),
	}
}

func cloneBlock(b *types.Block) (*types.Block, error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("marshal block: %w", err)
	}
	var cloned types.Block
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return nil, fmt.Errorf("unmarshal block: %w", err)
	}
	return &cloned, nil
}

func (s *MemoryStore) SaveBlock(block *types.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cloned, err := cloneBlock(block)
	if err != nil {
		return err
	}
	hash := block.Header.Hash()

	s.blocksByHeight[block.Header.Height] = cloned
	s.blocksByHash[hash] = cloned
	return nil
}

func (s *MemoryStore) GetBlockByHeight(height uint64) (*types.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	block, ok := s.blocksByHeight[height]
	if !ok {
		return nil, ErrBlockNotFound
	}
	return cloneBlock(block)
}

func (s *MemoryStore) GetBlockByHash(hash types.Hash) (*types.Block, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	block, ok := s.blocksByHash[hash]
	if !ok {
		return nil, ErrBlockNotFound
	}
	return cloneBlock(block)
}

func (s *MemoryStore) SetCannoicalHeight(height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cannoicalByHeight = height
	return nil
}

func (s *MemoryStore) GetCannoicalHeight() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cannoicalByHeight, nil
}
