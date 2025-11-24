package p2p

import (
	"testing"
	"time"
)

func TestServerBroadcastBetweenPeers(t *testing.T) {
	cfgA := Config{
		ListenAddr:       "127.0.0.1:0",
		HandshakeTimeout: 2 * time.Second,
	}
	serverA := NewServer(cfgA)
	if err := serverA.Start(); err != nil {
		t.Fatalf("start server A: %v", err)
	}
	defer serverA.Close()

	addrA := serverA.listener.Addr().String()

	received := make(chan []byte, 1)

	cfgB := Config{
		ListenAddr:       "127.0.0.1:0",
		Seeds:            []string{addrA},
		HandshakeTimeout: 2 * time.Second,
	}
	serverB := NewServer(cfgB)
	serverB.RegisterHandler(MessageTypeTx, func(peer PeerInfo, payload []byte) {
		received <- payload
	})
	if err := serverB.Start(); err != nil {
		t.Fatalf("start server B: %v", err)
	}
	defer serverB.Close()

	waitForPeers := func(s *Server) bool {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			s.mu.RLock()
			count := len(s.peers)
			s.mu.RUnlock()
			if count > 0 {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		return false
	}

	if !waitForPeers(serverA) || !waitForPeers(serverB) {
		t.Fatal("peers failed to connect")
	}

	payload := []byte("hello")
	serverA.Broadcast(NewEnvelope(MessageTypeTx, payload, ""))

	select {
	case got := <-received:
		if string(got) != "hello" {
			t.Fatalf("unexpected payload %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive broadcast payload")
	}
}
