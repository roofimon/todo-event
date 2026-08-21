package port

import (
	"context"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/domain"
	"todoe/infra/event"
)

type UseCase interface {
	Register(ctx context.Context, input domain.RegisterInput) mo.Result[domain.User]
	VerifyEmail(ctx context.Context, id bson.ObjectID, token string) mo.Result[domain.User]
	RecordCreditScore(ctx context.Context, id bson.ObjectID, score int, approved bool) mo.Result[domain.User]
	CompleteProfile(ctx context.Context, id bson.ObjectID, bio string) mo.Result[domain.User]
	GetUser(ctx context.Context, id bson.ObjectID) mo.Result[domain.User]
}

type Repository interface {
	Append(ctx context.Context, aggregateID bson.ObjectID, eventType string, payload any) mo.Result[struct{}]
	FindByID(ctx context.Context, id bson.ObjectID) mo.Result[domain.User]
}

type Publisher interface {
	Publish(ctx context.Context, e event.Event)
}
