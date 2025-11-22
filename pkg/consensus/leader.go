package consensus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/0xphantomotr/gchain/pkg/chain"
	"github.com/0xphantomotr/gchain/pkg/mempool"
	"github.com/0xphantomotr/gchain/pkg/state"
	"github.com/0xphantomotr/gchain/pkg/types"
)

type LeaderEngine struct {
	mu          sync.Mutex
	chain       *chain.Manager
	mempool     *mempool.Mempool
	state       *state.Manager
	validators  ValidatorSet
	broadcaster Broadcaster

	nodeID         types.Address
	height         uint64
	round          uint64
	votes          map[types.Hash]int
	roundDuration  time.Duration
	maxTxsPerBlock int
}

func NewLeaderEngine(chainMgr *chain.Manager, mem *mempool.Mempool, stateMgr *state.Manager, validators ValidatorSet, broadcaster Broadcaster, nodeID types.Address, roundDuration time.Duration, maxTxsPerBlock int) *LeaderEngine {
	height, _ := chainMgr.Tip()
	return &LeaderEngine{
		chain:          chainMgr,
		mempool:        mem,
		state:          stateMgr,
		validators:     validators,
		broadcaster:    broadcaster,
		nodeID:         nodeID,
		height:         height + 1,
		votes:          make(map[types.Hash]int),
		roundDuration:  roundDuration,
		maxTxsPerBlock: maxTxsPerBlock,
	}
}

func (e *LeaderEngine) Start(ctx context.Context) error {
	ticker := time.NewTicker(e.roundDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := e.runRound(ctx); err != nil {
				log.Printf("consensus round error: %v", err)
			}
		}
	}
}

func (e *LeaderEngine) runRound(ctx context.Context) error {
	e.mu.Lock()
	height := e.height
	round := e.round
	proposer := e.validators.Proposer(height, round)
	e.mu.Unlock()

	if proposer != e.nodeID {
		return nil
	}

	_, tipHash := e.chain.Tip()
	return e.proposeBlock(ctx, height, round, tipHash)
}

func (e *LeaderEngine) proposeBlock(ctx context.Context, height uint64, round uint64, previousHash types.Hash) error {
	txs := e.mempool.Pending(e.maxTxsPerBlock)
	block := &types.Block{
		Header: types.BlockHeader{
			Height:       height,
			PreviousHash: previousHash,
			Proposer:     e.nodeID,
			Timestamp:    time.Now(),
			StateRoot:    types.Hash{},
		},
		Transactions: txs,
	}
	block.Header.TxRoot = block.CalculateTxRoot()

	msg := Message{
		From:   e.nodeID,
		Height: height,
		Round:  round,
		Type:   MessageTypeProposal,
		Block:  block,
	}
	if err := e.broadcaster.Broadcast(msg); err != nil {
		return fmt.Errorf("broadcast proposal: %w", err)
	}

	e.HandleMessage(Message{
		From:   e.nodeID,
		Height: height,
		Round:  round,
		Type:   MessageTypeVote,
		Block:  block,
	})

	return nil
}

func (e *LeaderEngine) HandleMessage(msg Message) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if msg.Block == nil || msg.Height != e.height {
		return
	}

	switch msg.Type {
	case MessageTypeProposal:
		expected := e.validators.Proposer(msg.Height, msg.Round)
		if expected != msg.From {
			return
		}
		if err := e.validateBlock(msg.Block); err != nil {
			return
		}
		e.broadcastVoteLocked(msg.Block, msg.Height, msg.Round)
		e.applyVoteLocked(msg.Block)
	case MessageTypeVote:
		e.applyVoteLocked(msg.Block)
	}
}

func (e *LeaderEngine) broadcastVoteLocked(block *types.Block, height, round uint64) {
	msg := Message{
		From:   e.nodeID,
		Height: height,
		Round:  round,
		Type:   MessageTypeVote,
		Block:  block,
	}
	if err := e.broadcaster.Broadcast(msg); err != nil {
		log.Printf("broadcast vote error: %v", err)
	}
}

func (e *LeaderEngine) applyVoteLocked(block *types.Block) {
	hash := block.Header.Hash()
	e.votes[hash]++
	if e.votes[hash] >= e.quorumThreshold() {
		e.commitBlockLocked(block)
	}
}

func (e *LeaderEngine) commitBlockLocked(block *types.Block) {
	if err := e.state.ApplyBlock(*block); err != nil {
		log.Printf("commit apply block error: %v", err)
		return
	}
	if err := e.chain.AddBlock(block); err != nil {
		log.Printf("commit add block error: %v", err)
		return
	}

	for _, tx := range block.Transactions {
		e.mempool.Remove(tx.Hash)
	}

	e.height = block.Header.Height + 1
	e.round = 0
	e.votes = make(map[types.Hash]int)
}

func (e *LeaderEngine) validateBlock(block *types.Block) error {
	tipHeight, tipHash := e.chain.Tip()
	if block.Header.Height != tipHeight+1 {
		return fmt.Errorf("unexpected height: got %d, want %d", block.Header.Height, tipHeight+1)
	}
	if block.Header.PreviousHash != tipHash {
		return fmt.Errorf("previous hash mismatch")
	}
	return nil
}

func (e *LeaderEngine) quorumThreshold() int {
	return e.validators.Size()/2 + 1
}
