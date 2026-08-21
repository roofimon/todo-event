package port

import (
	"context"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"todoe/domain/task/domain"
	"todoe/infra/event"
)

type UseCase interface {
	CreateTask(ctx context.Context, title string) mo.Result[domain.Task]
	ListTasks(ctx context.Context) mo.Result[[]domain.Task]
	GetTask(ctx context.Context, id bson.ObjectID) mo.Result[domain.Task]
	ChangeStatus(ctx context.Context, id bson.ObjectID, status domain.Status) mo.Result[domain.Task]
}

type Repository interface {
	Append(ctx context.Context, aggregateID bson.ObjectID, eventType string, payload any) mo.Result[struct{}]
	FindAll(ctx context.Context) mo.Result[[]domain.Task]
	FindByID(ctx context.Context, id bson.ObjectID) mo.Result[domain.Task]
}

type Publisher interface {
	Publish(ctx context.Context, e event.Event)
}
