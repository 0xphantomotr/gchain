package p2p

import "encoding/json"

type MessageType uint8

const (
	MessageTypeTx MessageType = iota
	MessageTypeBlock
	MessageTypeConsensus
	MessageTypePing
	MessageTypePong
)

type Envelope struct {
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
	PeerID  string      `json:"peer_id,omitempty"`
}

func NewEnvelope(t MessageType, payload []byte, peerID string) Envelope {
	return Envelope{
		Type:    t,
		Payload: payload,
		PeerID:  peerID,
	}
}

func (e Envelope) Clone() Envelope {
	dup := make([]byte, len(e.Payload))
	copy(dup, e.Payload)
	e.Payload = dup
	return e
}

func MustMarshalPayload(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
