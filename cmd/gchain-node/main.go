package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/0xphantomotr/gchain/pkg/chain"
	"github.com/0xphantomotr/gchain/pkg/consensus"
	"github.com/0xphantomotr/gchain/pkg/mempool"
	"github.com/0xphantomotr/gchain/pkg/p2p"
	"github.com/0xphantomotr/gchain/pkg/rpc"
	"github.com/0xphantomotr/gchain/pkg/state"
	"github.com/0xphantomotr/gchain/pkg/types"
)

func main() {
	rpcAddr := flag.String("rpc-listen", ":8000", "RPC listen address")
	p2pAddr := flag.String("p2p-listen", ":9000", "P2P listen address")
	seedsFlag := flag.String("p2p-seeds", "", "comma-separated list of peer addresses")
	nodeIDFlag := flag.String("node-id", "0101010101010101010101010101010101010101010101010101010101010101", "validator address (64 hex chars)")
	genesisFlag := flag.String("genesis", "", "comma-separated list of addr:balance pairs (hex:amount)")
	flag.Parse()

	nodeID, err := parseHexAddress(*nodeIDFlag)
	if err != nil {
		log.Fatalf("invalid node-id: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chainStore := chain.NewMemoryStore()
	chainMgr, err := chain.NewManager(chainStore)
	if err != nil {
		log.Fatalf("init chain manager: %v", err)
	}

	stateMgr := state.NewManager(state.NewMemoryStore())
	if err := applyGenesis(stateMgr, *genesisFlag); err != nil {
		log.Fatalf("apply genesis: %v", err)
	}
	if *genesisFlag == "" {
		if err := stateMgr.SeedAccount(nodeID, 1_000_000_000, 0); err != nil {
			log.Fatalf("seed default balance: %v", err)
		}
	}
	pool := mempool.New(1024, nil)

	var seeds []string
	if *seedsFlag != "" {
		for _, entry := range strings.Split(*seedsFlag, ",") {
			addr := strings.TrimSpace(entry)
			if addr != "" {
				seeds = append(seeds, addr)
			}
		}
	}

	p2pServer := p2p.NewServer(p2p.Config{
		ListenAddr:       *p2pAddr,
		Seeds:            seeds,
		HandshakeTimeout: 5 * time.Second,
		MaxPeers:         50,
	})

	p2pServer.RegisterHandler(p2p.MessageTypeTx, func(peer p2p.PeerInfo, payload []byte) {
		var tx types.Transaction
		if err := json.Unmarshal(payload, &tx); err != nil {
			log.Printf("p2p: invalid tx payload from %s: %v", peer.ID, err)
			return
		}
		if err := pool.Add(tx); err != nil {
			return
		}
		p2pServer.BroadcastExcept(peer.ID, p2p.NewEnvelope(p2p.MessageTypeTx, payload, ""))
	})

	if err := p2pServer.Start(); err != nil {
		log.Fatalf("start p2p server: %v", err)
	}
	defer p2pServer.Close()

	validatorSet := singleValidatorSet{id: nodeID}
	consensusBroadcaster := &p2pConsensusBroadcaster{transport: p2pServer}
	engine := consensus.NewLeaderEngine(chainMgr, pool, stateMgr, validatorSet, consensusBroadcaster, nodeID, 2*time.Second, 64)

	p2pServer.RegisterHandler(p2p.MessageTypeConsensus, func(peer p2p.PeerInfo, payload []byte) {
		var msg consensus.Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Printf("p2p: invalid consensus payload from %s: %v", peer.ID, err)
			return
		}
		engine.HandleMessage(msg)
	})

	go func() {
		if err := engine.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("consensus stopped: %v", err)
		}
	}()

	rpcServer := rpc.NewServer(chainMgr, stateMgr, pool, p2pServer, *rpcAddr)

	go func() {
		if err := rpcServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("rpc server error: %v", err)
		}
	}()
	log.Printf("gchain node started; RPC on %s, P2P on %s\n", *rpcAddr, *p2pAddr)

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rpcServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("rpc shutdown error: %v", err)
	}

	log.Println("goodbye")
}

type singleValidatorSet struct {
	id types.Address
}

func (s singleValidatorSet) Proposer(height, round uint64) types.Address { return s.id }
func (s singleValidatorSet) Size() int                                   { return 1 }
func (s singleValidatorSet) Has(addr types.Address) bool                 { return addr == s.id }

type p2pConsensusBroadcaster struct {
	transport p2p.Transport
}

func (b *p2pConsensusBroadcaster) Broadcast(msg consensus.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	b.transport.Broadcast(p2p.NewEnvelope(p2p.MessageTypeConsensus, payload, ""))
	return nil
}

func parseHexAddress(input string) (types.Address, error) {
	var addr types.Address
	data, err := hex.DecodeString(strings.TrimPrefix(input, "0x"))
	if err != nil {
		return addr, err
	}
	if len(data) != len(addr) {
		return addr, fmt.Errorf("expected %d bytes, got %d", len(addr), len(data))
	}
	copy(addr[:], data)
	return addr, nil
}

func applyGenesis(stateMgr *state.Manager, cfg string) error {
	if cfg == "" {
		return nil
	}
	pairs := strings.Split(cfg, ",")
	for _, pair := range pairs {
		p := strings.TrimSpace(pair)
		if p == "" {
			continue
		}
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid genesis entry %q", p)
		}
		addr, err := parseHexAddress(parts[0])
		if err != nil {
			return fmt.Errorf("parse addr %q: %w", parts[0], err)
		}
		amount, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parse balance %q: %w", parts[1], err)
		}
		if err := stateMgr.SeedAccount(addr, amount, 0); err != nil {
			return err
		}
	}
	return nil
}
