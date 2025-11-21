package mempool

import (
	"container/heap"
	"sync"

	"github.com/0xphantomotr/gchain/pkg/types"
)

type TxSource interface {
	Validate(tx types.Transaction) error
}

type Pool interface {
	Add(tx types.Transaction) error
	Pending(limit int) []types.Transaction
	Remove(txHash types.Hash)
	Size() int
}

type entry struct {
	tx       types.Transaction
	priority int64
	index    int
}

type Mempool struct {
	mu     sync.RWMutex
	txs    map[types.Hash]*entry
	pq     priorityQueue
	maxTxs int
	source TxSource
}

func New(maxTxs int, source TxSource) *Mempool {
	pq := make(priorityQueue, 0)
	heap.Init(&pq)
	return &Mempool{
		maxTxs: maxTxs,
		source: source,
		txs:    make(map[types.Hash]*entry),
		pq:     pq,
	}
}

func (m *Mempool) Add(tx types.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := tx.CalculateHash()
	tx.Hash = hash
	if _, exists := m.txs[hash]; exists {
		return nil
	}
	if m.source != nil {
		if err := m.source.Validate(tx); err != nil {
			return err
		}
	}
	if m.maxTxs > 0 && len(m.txs) >= m.maxTxs {
		evicted := heap.Pop(&m.pq).(*entry)
		delete(m.txs, evicted.tx.Hash)
	}
	e := &entry{tx: tx, priority: -tx.Timestamp.UnixNano()}
	heap.Push(&m.pq, e)
	m.txs[hash] = e
	return nil
}

func (m *Mempool) Pending(limit int) []types.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || len(m.pq) == 0 {
		return nil
	}

	snapshot := make(priorityQueue, len(m.pq))
	copy(snapshot, m.pq)
	heap.Init(&snapshot)

	results := make([]types.Transaction, 0, min(limit, snapshot.Len()))
	for i := 0; i < limit && snapshot.Len() > 0; i++ {
		entry := heap.Pop(&snapshot).(*entry)
		results = append(results, entry.tx)
	}
	return results
}

func (m *Mempool) Remove(hash types.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.txs[hash]
	if !ok {
		return
	}
	heap.Remove(&m.pq, entry.index)
	delete(m.txs, hash)
}

func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.txs)
}

func (m *Mempool) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs = make(map[types.Hash]*entry)
	m.pq = make(priorityQueue, 0)
	heap.Init(&m.pq)
}

type priorityQueue []*entry

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].priority < pq[j].priority }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i]; pq[i].index = i; pq[j].index = j }
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*entry)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}
