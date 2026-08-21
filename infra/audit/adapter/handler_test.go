package adapter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/infra/audit/domain"
	"todoe/infra/event"
)

type auditRepositoryStub struct {
	save func(context.Context, domain.AuditEntry) mo.Result[struct{}]
}

func (s auditRepositoryStub) Save(ctx context.Context, entry domain.AuditEntry) mo.Result[struct{}] {
	return s.save(ctx, entry)
}

func TestAuditHandlerSavesEvent(t *testing.T) {
	t.Parallel()

	type contextKey string
	const key contextKey = "request-id"
	ctx := context.WithValue(context.Background(), key, "request-123")
	payload := map[string]any{"task_id": "task-1", "completed": true}
	e := event.Event{Type: "task.completed", Payload: payload}
	before := time.Now()

	var saved domain.AuditEntry
	repo := auditRepositoryStub{save: func(gotCtx context.Context, entry domain.AuditEntry) mo.Result[struct{}] {
		if gotCtx != ctx {
			t.Error("Save received a different context")
		}
		saved = entry
		return mo.Ok(struct{}{})
	}}

	if err := NewAuditHandler(repo)(ctx, e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	after := time.Now()

	if saved.ID == bson.NilObjectID {
		t.Error("saved entry has a nil ID")
	}
	if saved.EventType != e.Type {
		t.Errorf("EventType = %q, want %q", saved.EventType, e.Type)
	}
	if !reflect.DeepEqual(saved.Payload, payload) {
		t.Errorf("Payload = %#v, want %#v", saved.Payload, payload)
	}
	if saved.CreatedAt.Before(before) || saved.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want a time between %v and %v", saved.CreatedAt, before, after)
	}
}

func TestAuditHandlerReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("save audit entry")
	repo := auditRepositoryStub{save: func(context.Context, domain.AuditEntry) mo.Result[struct{}] {
		return mo.Err[struct{}](wantErr)
	}}

	err := NewAuditHandler(repo)(context.Background(), event.Event{Type: "task.created"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("handler error = %v, want %v", err, wantErr)
	}
}
