# gchain

gchain is a minimal modular blockchain node written in Go. It is designed as a portfolio project to showcase protocol‑level engineering skills: P2P networking, consensus, state execution, RPC APIs, and tooling.

## Features

- **State + Chain**: account-based state machine with transaction validation, persistent block storage, and tip tracking.
- **Mempool**: priority queue with basic validation and gossip via the P2P layer.
- **Consensus**: single-validator leader-based engine (round-robin ready) that builds blocks from the mempool and commits them via the state manager.
- **Networking**: TCP-based P2P transport that gossips transactions and consensus messages; metrics for peer counts.
- **RPC / CLI**: HTTP API (`/tx`, `/block/{height}`, `/balance/{addr}`, `/tip`, health, metrics) and a `gchain-light` CLI for querying and submitting transactions without running a full node.
- **Observability**: `expvar` metrics (tx submitted, block commits, current height, peer count) served via `/metrics`.

## Quickstart

```bash
# run a node (RPC :8000, P2P :9000) with a funded validator
go run ./cmd/gchain-node \
  --rpc-listen :8000 \
  --p2p-listen :9000 \
  --p2p-seeds "" \
  --node-id 0101010101010101010101010101010101010101010101010101010101010101 \
  --genesis 0101010101010101010101010101010101010101010101010101010101010101:1000

# submit a transaction
curl -X POST http://localhost:8000/tx \
  -H "Content-Type: application/json" \
  -d '{"from":"0101...0101","to":"0202...0202","amount":5}'

# query state / tip
curl http://localhost:8000/balance/0202...0202
curl http://localhost:8000/block/1
curl http://localhost:8000/tip
curl http://localhost:8000/metrics

# use the light client
go run ./cmd/gchain-light --rpc http://localhost:8000 tip
go run ./cmd/gchain-light --rpc http://localhost:8000 balance 0202...0202
go run ./cmd/gchain-light --rpc http://localhost:8000 send --from 0101...0101 --to 0202...0202 --amount 5
```

(Replace `0101...0101` / `0202...0202` with full 64‑hex-character addresses.)

## Structure

```
cmd/
  gchain-node     # full node binary
  gchain-light    # RPC-based light client
pkg/
  chain           # block storage + tip tracking
  consensus       # leader-based consensus engine
  mempool         # transaction pool
  metrics         # expvar metric helpers
  p2p             # TCP transport
  rpc             # HTTP API server
  state           # state manager + persistence
  types           # shared structs (blocks, tx, votes)
docs/             # (TODO) architecture / consensus notes
```

## Development

```bash
go test ./pkg/...
```

The node can be configured via CLI flags (`--rpc-listen`, `--p2p-listen`, `--p2p-seeds`, `--node-id`, `--genesis`). Peers can be chained together by listing seed addresses.