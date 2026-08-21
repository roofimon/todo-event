package messaging

import "encoding/json"

const (
	TaskSubject         = "task.events"
	UserSubject         = "user.events"
	CreditResultSubject = "credit.results"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
