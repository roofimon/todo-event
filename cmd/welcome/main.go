package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	userdomain "todoe/domain/user/domain"
	"todoe/infra/messaging"
)

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
			slog.Error("welcome: unmarshal", "err", err)
			return
		}

		switch msg.Type {
		case userdomain.EventRegistered:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("step 1/4: verification email sent",
				"to", user.Email, "name", user.Name, "token", user.VerificationToken)

		case userdomain.EventEmailVerified:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("step 2/4: email confirmed — running credit check...",
				"user_id", user.ID.Hex())

		case userdomain.EventCreditScored:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			if user.CreditApproved {
				slog.Info("step 3/4: credit approved — complete your profile",
					"score", user.CreditScore)
			} else {
				slog.Info("step 3/4: credit denied — onboarding blocked",
					"score", user.CreditScore)
			}

		case userdomain.EventProfileCompleted:
			var user userdomain.User
			json.Unmarshal(msg.Payload, &user)
			slog.Info("step 4/4: onboarding complete — welcome!",
				"name", user.Name, "bio", user.Bio)
		}
	})

	slog.Info("welcome service listening", "subject", messaging.UserSubject)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("welcome service stopping")
}
