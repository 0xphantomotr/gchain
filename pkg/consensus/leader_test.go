package consensus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/0xphantomotr/gchain/pkg/chain"
	"github.com/0xphantomotr/gchain/pkg/mempool"
	"github.com/0xphantomotr/gchain/pkg/state"
	"github.com/0xphantomotr/gchain/pkg/types"
)

type mockValidatorSet struct {
	proposer types.Address
	size     int
}

func (m mockValidatorSet) Proposer(height, round uint64) types.Address { return m.proposer }
func (m mockValidatorSet) Size() int                                   { return m.size }
func (m mockValidatorSet) Has(types.Address) bool                      { return true }

type mockBroadcaster struct {
	mu        sync.Mutex
	messages  []Message
}

func (m *mockBroadcaster) Broadcast(msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockBroadcaster) Last() (Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return Message{}, false
	}
	return m.messages[len(m.messages)-1], true
}

func newStateManager(t *testing.T) *state.Manager {
	t.Helper()
	store := state.NewMemoryStore()
	return state.NewManager(store)
}

func newChainManager(t *testing.T) *chain.Manager {
	t.Helper()
	store := chain.NewMemoryStore()
	mgr, err := chain.NewManager(store)
	if err != nil {
		t.Fatalf("new chain manager: %v", err)
	}
	return mgr
}

func TestLeaderEngineProposesAndCommits(t *testing.T) {
	chainMgr := newChainManager(t)
	stateMgr := newStateManager(t)
	pool := mempool.New(10, nil)

	tx := types.Transaction{
		From:      types.Address{1},
		To:        types.Address{2},
		Amount:    0,
		Timestamp: time.Unix(0, 1),
	}
	if err := pool.Add(tx); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	nodeID := types.Address{1}
	validators := mockValidatorSet{proposer: nodeID, size: 1}
	broadcaster := &mockBroadcaster{}

	engine := NewLeaderEngine(chainMgr, pool, stateMgr, validators, broadcaster, nodeID, 5*time.Millisecond, 5)

	if err := engine.proposeBlock(context.Background(), engine.height, engine.round, types.Hash{}); err != nil {
		t.Fatalf("propose block: %v", err)
	}

	height, _ := chainMgr.Tip()
	if height != 1 {
		t.Fatalf("expected height 1, got %d", height)
	}
	if pool.Size() != 0 {
		t.Fatalf("expected empty mempool, got %d", pool.Size())
	}
}

func TestFollowerVotesAndCommitsOnQuorum(t *testing.T) {
	chainMgr := newChainManager(t)
	stateMgr := newStateManager(t)
	pool := mempool.New(10, nil)

	nodeID := types.Address{2}
	proposer := types.Address{1}
	validators := mockValidatorSet{proposer: proposer, size: 2}
	broadcaster := &mockBroadcaster{}

	engine := NewLeaderEngine(chainMgr, pool, stateMgr, validators, broadcaster, nodeID, 5*time.Millisecond, 5)

	block := &types.Block{
		Header: types.BlockHeader{
			Height:       engine.height,
			PreviousHash: types.Hash{},
			Proposer:     proposer,
			Timestamp:    time.Now(),
		},
	}
	block.Header.TxRoot = block.CalculateTxRoot()

	proposal := Message{
		From:   proposer,
		Height: engine.height,
		Round:  0,
		Type:   MessageTypeProposal,
		Block:  block,
	}

	engine.HandleMessage(proposal)

	last, ok := broadcaster.Last()
	if !ok || last.Type != MessageTypeVote {
		t.Fatalf("expected vote broadcast, got %#v", last)
	}

	engine.HandleMessage(Message{
		From:   proposer,
		Height: engine.height,
		Round:  0,
		Type:   MessageTypeVote,
		Block:  block,
	})

	height, _ := chainMgr.Tip()
	if height != 1 {
		t.Fatalf("expected chain height 1 after quorum, got %d", height)
	}
}
