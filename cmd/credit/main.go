package main

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	userdomain "todoe/domain/user/domain"
	"todoe/infra/messaging"
)

// fakeCreditAPI returns a deterministic score (300–850) based on email.
// Scores >= 600 are approved.
func fakeCreditAPI(email string) (score int, approved bool) {
	h := fnv.New32a()
	h.Write([]byte(email))
	score = 400 + int(h.Sum32()%551)
	return score, score >= 600
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
