package adapter

import (
	"context"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/internal/audit/domain"
	"todoe/internal/event"
)

type auditRepository interface {
	Save(context.Context, domain.AuditEntry) mo.Result[struct{}]
}

func NewAuditHandler(repo auditRepository) func(context.Context, event.Event) error {
	return func(ctx context.Context, e event.Event) error {
		entry := domain.AuditEntry{
			ID:        bson.NewObjectID(),
			EventType: e.Type,
			Payload:   e.Payload,
			CreatedAt: time.Now(),
		}
		result := repo.Save(ctx, entry)
		if result.IsError() {
			return result.Error()
		}
		return nil
	}
}
