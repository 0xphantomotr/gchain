package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Hash [32]byte

type Address [32]byte

type Transaction struct {
	Hash      Hash      `json:"hash"`
	From      Address   `json:"from"`
	To        Address   `json:"to"`
	Amount    uint64    `json:"amount"`
	Nonce     uint64    `json:"nonce"`
	Signature []byte    `json:"signature,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type BlockHeader struct {
	Height       uint64    `json:"height"`
	PreviousHash Hash      `json:"previous_hash"`
	StateRoot    Hash      `json:"state_root"`
	TxRoot       Hash      `json:"tx_root"`
	Proposer     Address   `json:"proposer"`
	Timestamp    time.Time `json:"timestamp"`
}

type Block struct {
	Header       BlockHeader   `json:"header"`
	Transactions []Transaction `json:"transactions"`
}

type Vote struct {
	Voter     Address `json:"voter"`
	Height    uint64  `json:"height"`
	BlockHash Hash    `json:"block_hash"`
	Signature []byte  `json:"signature,omitempty"`
}

type PeerInfo struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	PublicKey []byte `json:"public_key,omitempty"`
}

func (tx *Transaction) CalculateHash() Hash {
	payload, _ := json.Marshal(struct {
		From   Address `json:"from"`
		To     Address `json:"to"`
		Amount uint64  `json:"amount"`
		Nonce  uint64  `json:"nonce"`
		Time   int64   `json:"timestamp"`
	}{
		From: tx.From, To: tx.To, Amount: tx.Amount, Nonce: tx.Nonce, Time: tx.Timestamp.UnixNano(),
	})

	return sha256.Sum256(payload)
}

func (b *Block) CalculateTxRoot() Hash {
	h := sha256.New()
	for _, tx := range b.Transactions {
		sum := tx.CalculateHash()
		h.Write(sum[:])
	}
	var out Hash
	copy(out[:], h.Sum(nil))
	return out
}

func (h *BlockHeader) Hash() Hash {
	payload, _ := json.Marshal(h)
	return sha256.Sum256(payload)
}

func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}
