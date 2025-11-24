package p2p

import (
	"sync"
	"time"
)

type Config struct {
	ListenAddr       string
	Seeds            []string
	MaxPeers         int
	HandshakeTimeout time.Duration
	ReadBufferSize   int
	WriteBufferSize  int
}

type PeerInfo struct {
	ID   string
	Addr string
}

type HandlerFunc func(peer PeerInfo, payload []byte)

type Transport interface {
	Start() error
	Close() error
	Broadcast(env Envelope)
	BroadcastExcept(peerID string, env Envelope)
	RegisterHandler(msgType MessageType, handler HandlerFunc)
}

type peerManager struct {
	mu    sync.RWMutex
	peers map[string]*Peer
	max   int
}
