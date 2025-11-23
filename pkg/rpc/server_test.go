package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/0xphantomotr/gchain/pkg/chain"
	"github.com/0xphantomotr/gchain/pkg/mempool"
	"github.com/0xphantomotr/gchain/pkg/state"
	"github.com/0xphantomotr/gchain/pkg/types"
)

func newTestServer(t *testing.T) (*Server, *chain.Manager, *state.Manager, *mempool.Mempool) {
	t.Helper()
	chainStore := chain.NewMemoryStore()
	chainMgr, err := chain.NewManager(chainStore)
	if err != nil {
		t.Fatalf("new chain manager: %v", err)
	}

	stateMgr := state.NewManager(state.NewMemoryStore())
	pool := mempool.New(100, nil)

	server := NewServer(chainMgr, stateMgr, pool, ":0")
	return server, chainMgr, stateMgr, pool
}

func TestSubmitTx(t *testing.T) {
	server, _, _, pool := newTestServer(t)
	ts := httptest.NewServer(server.httpServer.Handler)
	defer ts.Close()

	req := SubmitTxRequest{
		From:   "0101010101010101010101010101010101010101010101010101010101010101",
		To:     "0202020202020202020202020202020202020202020202020202020202020202",
		Amount: 10,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/tx", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("submit tx request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	if pool.Size() != 1 {
		t.Fatalf("expected mempool size 1, got %d", pool.Size())
	}
}

func TestGetBlock(t *testing.T) {
	server, chainMgr, _, _ := newTestServer(t)
	ts := httptest.NewServer(server.httpServer.Handler)
	defer ts.Close()

	block := &types.Block{
		Header: types.BlockHeader{
			Height:    1,
			Timestamp: time.Now(),
		},
	}
	block.Header.TxRoot = block.CalculateTxRoot()
	if err := chainMgr.AddBlock(block); err != nil {
		t.Fatalf("add block: %v", err)
	}

	resp, err := http.Get(ts.URL + "/block/" + strconv.Itoa(1))
	if err != nil {
		t.Fatalf("get block request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var out BlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Header.Height != 1 {
		t.Fatalf("expected height 1, got %d", out.Header.Height)
	}
}

func TestGetBalance(t *testing.T) {
	chainStore := chain.NewMemoryStore()
	chainMgr, err := chain.NewManager(chainStore)
	if err != nil {
		t.Fatalf("new chain manager: %v", err)
	}

	stateStore := state.NewMemoryStore()
	addr := types.Address{1}
	seedAccount(t, stateStore, state.Account{Address: addr, Balance: 42})
	stateMgr := state.NewManager(stateStore)

	server := NewServer(chainMgr, stateMgr, mempool.New(10, nil), ":0")
	ts := httptest.NewServer(server.httpServer.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/balance/" + addr.String())
	if err != nil {
		t.Fatalf("get balance request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var out BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Balance != 42 {
		t.Fatalf("expected balance 42, got %d", out.Balance)
	}
}

func seedAccount(t *testing.T, store *state.MemoryStore, account state.Account) {
	t.Helper()
	payload, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("marshal account: %v", err)
	}
	key := append([]byte("acct:"), account.Address[:]...)
	if err := store.Set(key, payload); err != nil {
		t.Fatalf("seed store: %v", err)
	}
}
