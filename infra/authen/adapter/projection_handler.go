package adapter

import (
	"context"
	"fmt"

	"todoe/infra/authen/domain"
	"todoe/infra/event"
)

func NewProjectionHandler(repo *MongoRepository) func(context.Context, event.Event) error {
	return func(ctx context.Context, e event.Event) error {
		session, ok := e.Payload.(domain.Session)
		if !ok {
			return fmt.Errorf("unexpected payload type %T", e.Payload)
		}
		switch e.Type {
		case domain.EventLoggedIn:
			result := repo.UpsertSession(ctx, session)
			if result.IsError() {
				return result.Error()
			}
		case domain.EventLoggedOut:
			result := repo.DeactivateSession(ctx, session.Token)
			if result.IsError() {
				return result.Error()
			}
		}
		return nil
	}
}
