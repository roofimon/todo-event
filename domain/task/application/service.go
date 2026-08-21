package application

import (
	"context"
	"errors"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/task/domain"
	"todoe/domain/task/port"
	"todoe/infra/event"
)

var (
	ErrInvalidTitle  = errors.New("title must not be empty")
	ErrInvalidStatus = errors.New("invalid status")
)

type Service struct {
	repo      port.Repository
	publisher port.Publisher
}

var _ port.UseCase = (*Service)(nil)

func NewService(repo port.Repository, publisher port.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func validateTitle(title string) error {
	if title == "" {
		return ErrInvalidTitle
	}
	return nil
}

func validateStatus(s domain.Status) error {
	switch s {
	case domain.StatusPending, domain.StatusInProgress, domain.StatusDone:
		return nil
	}
	return ErrInvalidStatus
}

func (s *Service) CreateTask(ctx context.Context, title string) mo.Result[domain.Task] {
	if err := validateTitle(title); err != nil {
		return mo.Err[domain.Task](err)
	}
	id := bson.NewObjectID()
	if result := s.repo.Append(ctx, id, domain.EventCreated, domain.TaskCreatedPayload{Title: title}); result.IsError() {
		return mo.Err[domain.Task](result.Error())
	}
	task := domain.Task{ID: id, Title: title, Status: domain.StatusPending, CreatedAt: time.Now()}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventCreated, Payload: task})
	return mo.Ok(task)
}

func (s *Service) ListTasks(ctx context.Context) mo.Result[[]domain.Task] {
	return s.repo.FindAll(ctx)
}

func (s *Service) GetTask(ctx context.Context, id bson.ObjectID) mo.Result[domain.Task] {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ChangeStatus(ctx context.Context, id bson.ObjectID, status domain.Status) mo.Result[domain.Task] {
	if err := validateStatus(status); err != nil {
		return mo.Err[domain.Task](err)
	}
	current := s.repo.FindByID(ctx, id)
	if current.IsError() {
		return mo.Err[domain.Task](current.Error())
	}
	next := current.MustGet().ChangeStatus(status)
	if result := s.repo.Append(ctx, id, domain.EventStatusChanged, domain.StatusChangedPayload{Status: status}); result.IsError() {
		return mo.Err[domain.Task](result.Error())
	}
	s.publisher.Publish(ctx, event.Event{Type: domain.EventStatusChanged, Payload: next})
	return mo.Ok(next)
}
