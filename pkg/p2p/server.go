package p2p

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type Peer struct {
	info     PeerInfo
	conn     net.Conn
	outgoing chan Envelope
	quit     chan struct{}
}

type Server struct {
	cfg      Config
	peers    map[string]*Peer
	handlers map[MessageType]HandlerFunc
	mu       sync.RWMutex
	listener net.Listener
	dialer   *Dialer
}

func NewServer(cfg Config) *Server {
	return &Server{
		cfg:      cfg,
		peers:    make(map[string]*Peer),
		handlers: make(map[MessageType]HandlerFunc),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptLoop()
	go s.connectSeeds()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return
		}
		go s.handleConnection(conn, true)
	}
}

func (s *Server) readLoop(p *Peer) {
	dec := json.NewDecoder(p.conn)
	for {
		var env Envelope
		if err := dec.Decode(&env); err != nil {
			s.removePeer(p.info.ID)
			return
		}
		s.dispatch(p.info, env)
	}
}

func (s *Server) writeLoop(p *Peer) {
	enc := json.NewEncoder(p.conn)
	for {
		select {
		case env := <-p.outgoing:
			if err := enc.Encode(env); err != nil {
				s.removePeer(p.info.ID)
				return
			}
		case <-p.quit:
			return
		}
	}
}

func (s *Server) dispatch(peer PeerInfo, env Envelope) {
	s.mu.RLock()
	handler := s.handlers[env.Type]
	s.mu.RUnlock()
	if handler != nil {
		handler(peer, env.Payload)
	}
}

func (s *Server) Broadcast(env Envelope) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, peer := range s.peers {
		if env.PeerID != "" && env.PeerID == id {
			continue
		}
		select {
		case peer.outgoing <- env.Clone():
		default:
			go s.removePeer(id) // drop slow peers
		}
	}
}

func (s *Server) BroadcastExcept(peerID string, env Envelope) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, peer := range s.peers {
		if id == peerID {
			continue
		}
		select {
		case peer.outgoing <- env.Clone():
		default:
			go s.removePeer(id)
		}
	}
}

func (s *Server) removePeer(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer, ok := s.peers[id]; ok {
		close(peer.quit)
		peer.conn.Close()
		delete(s.peers, id)
	}
}

func (s *Server) RegisterHandler(msgType MessageType, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[msgType] = handler
}

func (s *Server) Close() error {
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Lock()
	for _, peer := range s.peers {
		close(peer.quit)
		peer.conn.Close()
	}
	s.peers = map[string]*Peer{}
	s.mu.Unlock()
	return nil
}

func (s *Server) connectSeeds() {
	if len(s.cfg.Seeds) == 0 {
		return
	}
	for _, addr := range s.cfg.Seeds {
		go func(target string) {
			for {
				conn, err := net.DialTimeout("tcp", target, s.cfg.HandshakeTimeout)
				if err != nil {
					time.Sleep(time.Second)
					continue
				}
				s.handleConnection(conn, false)
				return
			}
		}(addr)
	}
}

func (s *Server) handleConnection(conn net.Conn, inbound bool) {
	peerID := conn.RemoteAddr().String()
	peer := &Peer{
		info: PeerInfo{
			ID:   peerID,
			Addr: conn.RemoteAddr().String(),
		},
		conn:     conn,
		outgoing: make(chan Envelope, 32),
		quit:     make(chan struct{}),
	}

	s.mu.Lock()
	if s.cfg.MaxPeers > 0 && len(s.peers) >= s.cfg.MaxPeers {
		s.mu.Unlock()
		conn.Close()
		return
	}
	s.peers[peerID] = peer
	s.mu.Unlock()

	go s.readLoop(peer)
	go s.writeLoop(peer)
}
