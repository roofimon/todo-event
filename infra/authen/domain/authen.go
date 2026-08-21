package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	EventLoggedIn  = "auth.logged_in"
	EventLoggedOut = "auth.logged_out"
)

type Credential struct {
	ID           bson.ObjectID `bson:"_id"           json:"id"`
	UserID       bson.ObjectID `bson:"user_id"       json:"user_id"`
	Email        string        `bson:"email"         json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}

type Session struct {
	ID        bson.ObjectID `bson:"_id"        json:"id"`
	UserID    bson.ObjectID `bson:"user_id"    json:"user_id"`
	Token     string        `bson:"token"      json:"token"`
	Active    bool          `bson:"active"     json:"active"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	ExpiresAt time.Time     `bson:"expires_at" json:"expires_at"`
}

func (s Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

type LoggedInPayload struct {
	SessionID bson.ObjectID `bson:"session_id" json:"session_id"`
	UserID    bson.ObjectID `bson:"user_id"    json:"user_id"`
	Token     string        `bson:"token"      json:"token"`
}

type LoggedOutPayload struct {
	SessionID bson.ObjectID `bson:"session_id" json:"session_id"`
	Token     string        `bson:"token"      json:"token"`
}
