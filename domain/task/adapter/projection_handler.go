package adapter

import (
	"context"
	"fmt"

	"todoe/domain/task/domain"
	"todoe/infra/event"
)

func NewProjectionHandler(repo *MongoRepository) func(context.Context, event.Event) error {
	return func(ctx context.Context, e event.Event) error {
		task, ok := e.Payload.(domain.Task)
		if !ok {
			return fmt.Errorf("unexpected payload type %T", e.Payload)
		}
		result := repo.Upsert(ctx, task)
		if result.IsError() {
			return result.Error()
		}
		return nil
	}
}
