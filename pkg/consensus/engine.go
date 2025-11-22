package consensus

import (
	"context"

	"github.com/0xphantomotr/gchain/pkg/types"
)

type MessageType uint8

const (
	MessageTypeProposal MessageType = iota
	MessageTypeVote
)

type Message struct {
	From   types.Address
	Height uint64
	Round  uint64
	Type   MessageType
	Block  *types.Block
}

// type Executor struct {
// 	chainMgr    *chain.Manager
// 	stateMgr    *state.Manager
// 	txPool      *mempool.Mempool
// 	validators  ValidatorSet
// 	broadcaster Broadcaster
// }

type ValidatorSet interface {
	Proposer(height, round uint64) types.Address
	Size() int
	Has(addr types.Address) bool
}

type Broadcaster interface {
	Broadcast(msg Message) error
}

type Engine interface {
	Start(ctx context.Context) error
	HandleMessage(msg Message)
}
