package metrics

import "expvar"

var (
	txSubmitted     = expvar.NewInt("tx_submitted_total")
	blocksCommitted = expvar.NewInt("blocks_committed_total")
	currentHeight   = expvar.NewInt("current_block_height")
	peerCount       = expvar.NewInt("peer_count")
)

// IncTxSubmitted increments the transaction submission counter.
func IncTxSubmitted() {
	txSubmitted.Add(1)
}

// ObserveBlockCommit records a committed block height.
func ObserveBlockCommit(height uint64) {
	blocksCommitted.Add(1)
	currentHeight.Set(int64(height))
}

// SetPeerCount sets the current connected peer count.
func SetPeerCount(count int) {
	peerCount.Set(int64(count))
}
