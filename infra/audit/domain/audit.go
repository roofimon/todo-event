package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type AuditEntry struct {
	ID        bson.ObjectID `bson:"_id"`
	EventType string        `bson:"event_type"`
	Payload   any           `bson:"payload"`
	CreatedAt time.Time     `bson:"created_at"`
}
