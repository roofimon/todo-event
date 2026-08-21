package adapter

import (
	"context"
	"fmt"

	"todoe/domain/user/domain"
	"todoe/infra/event"
)

func NewProjectionHandler(repo *MongoRepository) func(context.Context, event.Event) error {
	return func(ctx context.Context, e event.Event) error {
		user, ok := e.Payload.(domain.User)
		if !ok {
			return fmt.Errorf("unexpected payload type %T", e.Payload)
		}
		result := repo.Upsert(ctx, user)
		if result.IsError() {
			return result.Error()
		}
		return nil
	}
}
