package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/0xphantomotr/gchain/pkg/chain"
	"github.com/0xphantomotr/gchain/pkg/mempool"
	"github.com/0xphantomotr/gchain/pkg/state"
	"github.com/0xphantomotr/gchain/pkg/types"
)

type SubmitTxRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
}
type SubmitTxResponse struct {
	TxHash string `json:"tx_hash"`
}
type BlockResponse struct {
	Header       types.BlockHeader   `json:"header"`
	Transactions []types.Transaction `json:"transactions"`
}
type BalanceResponse struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
}

type Server struct {
	chain      *chain.Manager
	state      *state.Manager
	mempool    *mempool.Mempool
	httpServer *http.Server
}

func NewServer(chain *chain.Manager, state *state.Manager, pool *mempool.Mempool, listenAddr string) *Server {
	mux := http.NewServeMux()
	srv := &Server{chain: chain, state: state, mempool: pool}
	mux.HandleFunc("/healthz", srv.handleHealth)
	mux.HandleFunc("/tx", srv.handleSubmitTx)
	mux.HandleFunc("/block/", srv.handleGetBlock)
	mux.HandleFunc("/balance/", srv.handleGetBalance)
	mux.HandleFunc("/tip", srv.handleGetTip)
	srv.httpServer = &http.Server{Addr: listenAddr, Handler: mux}
	return srv
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSubmitTx(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    defer r.Body.Close()

    var req SubmitTxRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse(err))
        return
    }

    from, err := parseAddress(req.From)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse(err))
        return
    }
    to, err := parseAddress(req.To)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse(err))
        return
    }

	tx := types.Transaction{
		From:      from,
		To:        to,
		Amount:    req.Amount,
		Timestamp: time.Now(),
	}

    tx.Hash = tx.CalculateHash()

    if err := s.mempool.Add(tx); err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse(err))
        return
    }

    writeJSON(w, http.StatusOK, SubmitTxResponse{TxHash: tx.Hash.String()})
}

func (s *Server) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	heightStr := strings.TrimPrefix(r.URL.Path, "/block/")
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse(err))
		return
	}

	block, err := s.chain.GetBlockByHeight(height)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse(err))
		return
	}

	writeJSON(w, http.StatusOK, BlockResponse{
		Header:       block.Header,
		Transactions: block.Transactions,
	})
}

func (s *Server) handleGetBalance(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    addrStr := strings.TrimPrefix(r.URL.Path, "/balance/")
    addr, err := parseAddress(addrStr)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse(err))
        return
    }

    account, err := s.state.GetAccount(addr)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, errorResponse(err))
        return
    }

    writeJSON(w, http.StatusOK, BalanceResponse{
        Address: addr.String(),
        Balance: account.Balance,
    })
}

func (s *Server) handleGetTip(w http.ResponseWriter, r *http.Request) {
    height, hash := s.chain.Tip()
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "height": height,
        "hash":   hash.String(),
    })
}

// Helpers

type errorPayload struct {
    Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(payload)
}

func errorResponse(err error) errorPayload {
    return errorPayload{Error: err.Error()}
}

func parseAddress(hexStr string) (types.Address, error) {
    var addr types.Address
    b, err := hex.DecodeString(strings.TrimPrefix(hexStr, "0x"))
    if err != nil {
        return addr, err
    }
    if len(b) != len(addr) {
        return addr, fmt.Errorf("invalid address length")
    }
    copy(addr[:], b)
    return addr, nil
}
