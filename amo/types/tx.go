package types

import (
	"encoding/json"
	"strings"
)

const (
	TxTransfer = "transfer"
	TxPurchase = "purchase"
)

type Message struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type Transfer struct {
	From   Address `json:"from"`
	To     Address `json:"to"`
	Amount uint64  `json:"amount"`
}

type Purchase struct {
	From     Address `json:"from"`
	FileHash Hash    `json:"file_hash"`
}

func ParseTx(tx []byte) (Message, interface{}) {
	var message Message

	err := json.Unmarshal(tx, &message)
	if err != nil {
		panic(err)
	}

	message.Type = strings.ToLower(message.Type)

	var payload interface{}
	switch message.Type {
	case TxTransfer:
		payload = new(Transfer)
	case TxPurchase:
		payload = new(Purchase)
	}

	err = json.Unmarshal(message.Payload, &payload)
	if err != nil {
		panic(err)
	}

	return message, payload
}
