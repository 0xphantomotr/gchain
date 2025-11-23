package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0xphantomotr/gchain/pkg/chain"
	"github.com/0xphantomotr/gchain/pkg/mempool"
	"github.com/0xphantomotr/gchain/pkg/rpc"
	"github.com/0xphantomotr/gchain/pkg/state"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chainStore := chain.NewMemoryStore()
	chainMgr, err := chain.NewManager(chainStore)
	if err != nil {
		log.Fatalf("init chain manager: %v", err)
	}

	stateMgr := state.NewManager(state.NewMemoryStore())
	pool := mempool.New(1024, nil)

	rpcServer := rpc.NewServer(chainMgr, stateMgr, pool, ":8000")

	go func() {
		if err := rpcServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("rpc server error: %v", err)
		}
	}()
	log.Println("gchain node started; RPC listening on :8000")

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rpcServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("rpc shutdown error: %v", err)
	}

	log.Println("goodbye")
}
