package main

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"todoe/domain/user/domain"
	userdomain "todoe/domain/user/domain"
	"todoe/internal/event"
	"todoe/internal/messaging"

	"github.com/nats-io/nats.go"
)

type multiPublisher struct{ publishers []event.Publisher }

// fakeCreditAPI returns a deterministic score (300–850) based on email.
// Scores >= 600 are approved.
func fakeCreditAPI(email string) (score int, approved bool) {
	h := fnv.New32a()
	h.Write([]byte(email))
	score = 400 + int(h.Sum32()%551)
	return score, score >= 600
}

type natsPublisher struct {
	conn    *nats.Conn
	subject string
}

func (p *natsPublisher) Publish(_ context.Context, e event.Event) {
	payload, _ := json.Marshal(e.Payload)
	data, _ := json.Marshal(messaging.Message{Type: e.Type, Payload: payload})
	if err := p.conn.Publish(p.subject, data); err != nil {
		slog.Error("nats: publish error", "subject", p.subject, "err", err)
	}
}

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("nats:", err)
	}
	defer nc.Drain()

	nc.Subscribe(messaging.UserSubject, func(m *nats.Msg) {
		var msg messaging.Message
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			slog.Error("credit: unmarshal", "err", err)
			return
		}
		if msg.Type != userdomain.EventEmailVerified {
			return
		}

		var user userdomain.User
		if err := json.Unmarshal(msg.Payload, &user); err != nil {
			slog.Error("credit: unmarshal user", "err", err)
			return
		}

		score, approved := fakeCreditAPI(user.Email)
		slog.Info("credit: scored", "user_id", user.ID.Hex(), "email", user.Email, "score", score, "approved", approved)

		payload, _ := json.Marshal(userdomain.CreditScoredPayload{
			UserID:   user.ID.Hex(),
			Score:    score,
			Approved: approved,
		})
		if score < 600 {
			user.Status = domain.StatusRegistered
			user.CreditScore = 0
			slog.Error("score disqualified: user score disqualified")
			mUser, _ := json.Marshal(user)
			data, _ := json.Marshal(messaging.Message{
				Type:    domain.EventScoreDisqualified,
				Payload: mUser,
			})
			if err := nc.Publish(messaging.UserSubject, data); err != nil {
				slog.Error("credit: publish result", "err", err)
			}
			return
		}
		data, _ := json.Marshal(messaging.Message{
			Type:    userdomain.EventCreditScored,
			Payload: payload,
		})
		if err := nc.Publish(messaging.CreditResultSubject, data); err != nil {
			slog.Error("credit: publish result", "err", err)
		}
	})

	slog.Info("credit service listening", "subject", messaging.UserSubject)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("credit service stopping")
}
