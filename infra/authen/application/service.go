package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"todoe/infra/authen/domain"
	"todoe/infra/authen/port"
	"todoe/infra/event"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session has expired")
)

const sessionTTL = 24 * time.Hour

type Service struct {
	repo      port.Repository
	publisher port.Publisher
}

var _ port.UseCase = (*Service)(nil)

func NewService(repo port.Repository, publisher port.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) RegisterCredential(ctx context.Context, email, password string) mo.Result[domain.Credential] {
	if email == "" || password == "" {
		return mo.Err[domain.Credential](ErrInvalidCredentials)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return mo.Err[domain.Credential](err)
	}
	cred := domain.Credential{
		ID:           bson.NewObjectID(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if r := s.repo.CreateCredential(ctx, cred); r.IsError() {
		return mo.Err[domain.Credential](r.Error())
	}
	return mo.Ok(cred)
}

func (s *Service) ActivateUser(ctx context.Context, userID bson.ObjectID, email, name string) mo.Result[domain.Credential] {
	tempPassword := bson.NewObjectID().Hex()
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return mo.Err[domain.Credential](err)
	}
	cred := domain.Credential{
		ID:           bson.NewObjectID(),
		UserID:       userID,
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if r := s.repo.CreateCredential(ctx, cred); r.IsError() {
		return mo.Err[domain.Credential](r.Error())
	}
	slog.Info("auth: user activated — credential created", "email", email, "name", name, "temp_password", tempPassword)
	return mo.Ok(cred)
}

func (s *Service) Login(ctx context.Context, email, password string) mo.Result[domain.Session] {
	cred := s.repo.FindCredentialByEmail(ctx, email)
	if cred.IsError() {
		return mo.Err[domain.Session](ErrInvalidCredentials)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cred.MustGet().PasswordHash), []byte(password)); err != nil {
		return mo.Err[domain.Session](ErrInvalidCredentials)
	}
	now := time.Now()
	session := domain.Session{
		ID:        bson.NewObjectID(),
		UserID:    cred.MustGet().UserID,
		Token:     bson.NewObjectID().Hex(),
		Active:    true,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}
	payload := domain.LoggedInPayload{
		SessionID: session.ID,
		UserID:    session.UserID,
		Token:     session.Token,
	}
	if r := s.repo.Append(ctx, session.ID, domain.EventLoggedIn, payload); r.IsError() {
		return mo.Err[domain.Session](r.Error())
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventLoggedIn, Payload: session})
	return mo.Ok(session)
}

func (s *Service) Logout(ctx context.Context, token string) mo.Result[struct{}] {
	sessionResult := s.repo.FindActiveSessionByToken(ctx, token)
	if sessionResult.IsError() {
		return mo.Err[struct{}](ErrSessionNotFound)
	}
	session := sessionResult.MustGet()
	payload := domain.LoggedOutPayload{
		SessionID: session.ID,
		Token:     token,
	}
	if r := s.repo.Append(ctx, session.ID, domain.EventLoggedOut, payload); r.IsError() {
		return mo.Err[struct{}](r.Error())
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventLoggedOut, Payload: session})
	return mo.Ok(struct{}{})
}

func (s *Service) ValidateToken(ctx context.Context, token string) mo.Result[domain.Session] {
	sessionResult := s.repo.FindActiveSessionByToken(ctx, token)
	if sessionResult.IsError() {
		return mo.Err[domain.Session](ErrSessionNotFound)
	}
	session := sessionResult.MustGet()
	if session.IsExpired() {
		return mo.Err[domain.Session](ErrSessionExpired)
	}
	return mo.Ok(session)
}
