package application

import (
	"context"
	"errors"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/domain"
	"todoe/domain/user/port"
	"todoe/infra/event"
)

var (
	ErrInvalidName      = errors.New("name must not be empty")
	ErrInvalidEmail     = errors.New("email must not be empty")
	ErrInvalidToken     = errors.New("invalid verification token")
	ErrCreditNotChecked = errors.New("credit score not yet checked")
	ErrCreditDenied     = errors.New("credit application was denied")
)

type Service struct {
	repo      port.Repository
	publisher port.Publisher
}

var _ port.UseCase = (*Service)(nil)

func NewService(repo port.Repository, publisher port.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Register(ctx context.Context, name, email string) mo.Result[domain.User] {
	if name == "" {
		return mo.Err[domain.User](ErrInvalidName)
	}
	if email == "" {
		return mo.Err[domain.User](ErrInvalidEmail)
	}
	id := bson.NewObjectID()
	token := bson.NewObjectID().Hex()
	payload := domain.RegisteredPayload{Name: name, Email: email, VerificationToken: token}
	if r := s.repo.Append(ctx, id, domain.EventRegistered, payload); r.IsError() {
		return mo.Err[domain.User](r.Error())
	}
	user := domain.User{
		ID:                id,
		Name:              name,
		Email:             email,
		Status:            domain.StatusRegistered,
		VerificationToken: token,
		CreatedAt:         time.Now(),
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventRegistered, Payload: user})
	return mo.Ok(user)
}

func (s *Service) VerifyEmail(ctx context.Context, id bson.ObjectID, token string) mo.Result[domain.User] {
	current := s.repo.FindByID(ctx, id)
	if current.IsError() {
		return mo.Err[domain.User](current.Error())
	}
	u := current.MustGet()
	if u.VerificationToken != token {
		return mo.Err[domain.User](ErrInvalidToken)
	}
	next := u.WithEmailVerified()
	if r := s.repo.Append(ctx, id, domain.EventEmailVerified, domain.EmailVerifiedPayload{UserID: id.Hex()}); r.IsError() {
		return mo.Err[domain.User](r.Error())
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventEmailVerified, Payload: next})
	return mo.Ok(next)
}

func (s *Service) RecordCreditScore(ctx context.Context, id bson.ObjectID, score int, approved bool) mo.Result[domain.User] {
	current := s.repo.FindByID(ctx, id)
	if current.IsError() {
		return mo.Err[domain.User](current.Error())
	}
	next := current.MustGet().WithCreditScore(score, approved)
	payload := domain.CreditScoredPayload{UserID: id.Hex(), Score: score, Approved: approved}
	if r := s.repo.Append(ctx, id, domain.EventCreditScored, payload); r.IsError() {
		return mo.Err[domain.User](r.Error())
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventCreditScored, Payload: next})
	return mo.Ok(next)
}

func (s *Service) GetUser(ctx context.Context, id bson.ObjectID) mo.Result[domain.User] {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) CompleteProfile(ctx context.Context, id bson.ObjectID, bio string) mo.Result[domain.User] {
	current := s.repo.FindByID(ctx, id)
	if current.IsError() {
		return mo.Err[domain.User](current.Error())
	}
	u := current.MustGet()
	switch u.Status {
	case domain.StatusCreditDenied:
		return mo.Err[domain.User](ErrCreditDenied)
	case domain.StatusRegistered, domain.StatusEmailVerified:
		return mo.Err[domain.User](ErrCreditNotChecked)
	}
	next := u.WithProfile(bio)
	payload := domain.ProfileCompletedPayload{UserID: id.Hex(), Bio: bio}
	if r := s.repo.Append(ctx, id, domain.EventProfileCompleted, payload); r.IsError() {
		return mo.Err[domain.User](r.Error())
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventProfileCompleted, Payload: next})
	s.publisher.Publish(ctx, event.Event{
		Type: domain.EventUserActivated,
		Payload: domain.UserActivatedPayload{
			UserID: next.ID.Hex(),
			Email:  next.Email,
			Name:   next.Name,
		},
	})
	return mo.Ok(next)
}
