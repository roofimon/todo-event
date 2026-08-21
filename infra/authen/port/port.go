package port

import (
	"context"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/infra/authen/domain"
	"todoe/infra/event"
)

type UseCase interface {
	RegisterCredential(ctx context.Context, email, password string) mo.Result[domain.Credential]
	ActivateUser(ctx context.Context, userID bson.ObjectID, email, name string) mo.Result[domain.Credential]
	Login(ctx context.Context, email, password string) mo.Result[domain.Session]
	Logout(ctx context.Context, token string) mo.Result[struct{}]
	ValidateToken(ctx context.Context, token string) mo.Result[domain.Session]
}

type Repository interface {
	Append(ctx context.Context, aggregateID bson.ObjectID, eventType string, payload any) mo.Result[struct{}]
	CreateCredential(ctx context.Context, cred domain.Credential) mo.Result[struct{}]
	FindCredentialByEmail(ctx context.Context, email string) mo.Result[domain.Credential]
	UpsertSession(ctx context.Context, session domain.Session) mo.Result[struct{}]
	FindActiveSessionByToken(ctx context.Context, token string) mo.Result[domain.Session]
	DeactivateSession(ctx context.Context, token string) mo.Result[struct{}]
}

type Publisher interface {
	Publish(ctx context.Context, e event.Event)
}
