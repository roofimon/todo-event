package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"todoe/infra/authen/domain"
	"todoe/infra/event"
)

type repositoryStub struct {
	append                   func(context.Context, bson.ObjectID, string, any) mo.Result[struct{}]
	createCredential         func(context.Context, domain.Credential) mo.Result[struct{}]
	findCredentialByEmail    func(context.Context, string) mo.Result[domain.Credential]
	findActiveSessionByToken func(context.Context, string) mo.Result[domain.Session]
}

func (r repositoryStub) Append(ctx context.Context, id bson.ObjectID, eventType string, payload any) mo.Result[struct{}] {
	return r.append(ctx, id, eventType, payload)
}

func (r repositoryStub) CreateCredential(ctx context.Context, cred domain.Credential) mo.Result[struct{}] {
	return r.createCredential(ctx, cred)
}

func (r repositoryStub) FindCredentialByEmail(ctx context.Context, email string) mo.Result[domain.Credential] {
	return r.findCredentialByEmail(ctx, email)
}

func (r repositoryStub) FindActiveSessionByToken(ctx context.Context, token string) mo.Result[domain.Session] {
	return r.findActiveSessionByToken(ctx, token)
}

func (repositoryStub) UpsertSession(context.Context, domain.Session) mo.Result[struct{}] {
	panic("unexpected UpsertSession call")
}

func (repositoryStub) DeactivateSession(context.Context, string) mo.Result[struct{}] {
	panic("unexpected DeactivateSession call")
}

type publisherSpy struct {
	events []event.Event
}

func (p *publisherSpy) Publish(_ context.Context, e event.Event) {
	p.events = append(p.events, e)
}

func TestRegisterCredential(t *testing.T) {
	t.Run("rejects missing credentials", func(t *testing.T) {
		svc := NewService(repositoryStub{}, &publisherSpy{})
		for _, input := range []struct{ email, password string }{{"", "password"}, {"user@example.com", ""}} {
			result := svc.RegisterCredential(context.Background(), input.email, input.password)
			if !errors.Is(result.Error(), ErrInvalidCredentials) {
				t.Errorf("RegisterCredential(%q, %q) error = %v, want %v", input.email, input.password, result.Error(), ErrInvalidCredentials)
			}
		}
	})

	t.Run("creates a hashed credential", func(t *testing.T) {
		ctx := context.Background()
		var saved domain.Credential
		repo := repositoryStub{createCredential: func(gotCtx context.Context, cred domain.Credential) mo.Result[struct{}] {
			if gotCtx != ctx {
				t.Error("CreateCredential received a different context")
			}
			saved = cred
			return mo.Ok(struct{}{})
		}}
		before := time.Now()
		result := NewService(repo, &publisherSpy{}).RegisterCredential(ctx, "user@example.com", "secret")

		if result.IsError() {
			t.Fatalf("RegisterCredential returned error: %v", result.Error())
		}
		cred := result.MustGet()
		if cred != saved {
			t.Errorf("returned credential = %#v, want saved credential %#v", cred, saved)
		}
		if cred.ID == bson.NilObjectID || cred.Email != "user@example.com" {
			t.Errorf("credential identity = (%v, %q), want non-nil ID and requested email", cred.ID, cred.Email)
		}
		if cred.CreatedAt.Before(before) || cred.CreatedAt.After(time.Now()) {
			t.Errorf("CreatedAt = %v, want current time", cred.CreatedAt)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte("secret")); err != nil {
			t.Errorf("PasswordHash does not match password: %v", err)
		}
		if cred.PasswordHash == "secret" {
			t.Error("password was stored in plaintext")
		}
	})

	t.Run("returns repository error", func(t *testing.T) {
		wantErr := errors.New("create credential")
		repo := repositoryStub{createCredential: func(context.Context, domain.Credential) mo.Result[struct{}] {
			return mo.Err[struct{}](wantErr)
		}}
		result := NewService(repo, &publisherSpy{}).RegisterCredential(context.Background(), "user@example.com", "secret")
		if !errors.Is(result.Error(), wantErr) {
			t.Fatalf("error = %v, want %v", result.Error(), wantErr)
		}
	})
}

func TestLogin(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	userID := bson.NewObjectID()
	credential := domain.Credential{UserID: userID, Email: "user@example.com", PasswordHash: string(passwordHash)}

	t.Run("rejects unknown email", func(t *testing.T) {
		repo := repositoryStub{findCredentialByEmail: func(context.Context, string) mo.Result[domain.Credential] {
			return mo.Err[domain.Credential](errors.New("not found"))
		}}
		result := NewService(repo, &publisherSpy{}).Login(context.Background(), credential.Email, "secret")
		if !errors.Is(result.Error(), ErrInvalidCredentials) {
			t.Fatalf("error = %v, want %v", result.Error(), ErrInvalidCredentials)
		}
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		repo := repositoryStub{findCredentialByEmail: func(context.Context, string) mo.Result[domain.Credential] {
			return mo.Ok(credential)
		}}
		result := NewService(repo, &publisherSpy{}).Login(context.Background(), credential.Email, "wrong")
		if !errors.Is(result.Error(), ErrInvalidCredentials) {
			t.Fatalf("error = %v, want %v", result.Error(), ErrInvalidCredentials)
		}
	})

	t.Run("appends event and publishes session", func(t *testing.T) {
		var appendedID bson.ObjectID
		var appendedType string
		var appendedPayload domain.LoggedInPayload
		repo := repositoryStub{
			findCredentialByEmail: func(context.Context, string) mo.Result[domain.Credential] { return mo.Ok(credential) },
			append: func(_ context.Context, id bson.ObjectID, eventType string, payload any) mo.Result[struct{}] {
				appendedID, appendedType = id, eventType
				appendedPayload = payload.(domain.LoggedInPayload)
				return mo.Ok(struct{}{})
			},
		}
		publisher := &publisherSpy{}
		before := time.Now()
		result := NewService(repo, publisher).Login(context.Background(), credential.Email, "secret")

		if result.IsError() {
			t.Fatalf("Login returned error: %v", result.Error())
		}
		session := result.MustGet()
		if session.ID == bson.NilObjectID || session.UserID != userID || session.Token == "" || !session.Active {
			t.Errorf("invalid session: %#v", session)
		}
		if session.CreatedAt.Before(before) || session.ExpiresAt.Sub(session.CreatedAt) != sessionTTL {
			t.Errorf("session times = (%v, %v), want current creation and %v TTL", session.CreatedAt, session.ExpiresAt, sessionTTL)
		}
		if appendedID != session.ID || appendedType != domain.EventLoggedIn {
			t.Errorf("appended event = (%v, %q), want (%v, %q)", appendedID, appendedType, session.ID, domain.EventLoggedIn)
		}
		if appendedPayload.SessionID != session.ID || appendedPayload.UserID != session.UserID || appendedPayload.Token != session.Token {
			t.Errorf("appended payload = %#v, want session identity", appendedPayload)
		}
		if len(publisher.events) != 1 || publisher.events[0].Type != domain.EventLoggedIn || publisher.events[0].Payload != session {
			t.Errorf("published events = %#v, want logged-in session", publisher.events)
		}
	})

	t.Run("does not publish when append fails", func(t *testing.T) {
		wantErr := errors.New("append event")
		repo := repositoryStub{
			findCredentialByEmail: func(context.Context, string) mo.Result[domain.Credential] { return mo.Ok(credential) },
			append: func(context.Context, bson.ObjectID, string, any) mo.Result[struct{}] {
				return mo.Err[struct{}](wantErr)
			},
		}
		publisher := &publisherSpy{}
		result := NewService(repo, publisher).Login(context.Background(), credential.Email, "secret")
		if !errors.Is(result.Error(), wantErr) || len(publisher.events) != 0 {
			t.Fatalf("error = %v, events = %d; want %v and no events", result.Error(), len(publisher.events), wantErr)
		}
	})
}

func TestLogout(t *testing.T) {
	session := domain.Session{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Token: "token", Active: true}

	t.Run("returns not found", func(t *testing.T) {
		repo := repositoryStub{findActiveSessionByToken: func(context.Context, string) mo.Result[domain.Session] {
			return mo.Err[domain.Session](errors.New("missing"))
		}}
		result := NewService(repo, &publisherSpy{}).Logout(context.Background(), session.Token)
		if !errors.Is(result.Error(), ErrSessionNotFound) {
			t.Fatalf("error = %v, want %v", result.Error(), ErrSessionNotFound)
		}
	})

	t.Run("appends event and publishes session", func(t *testing.T) {
		var payload domain.LoggedOutPayload
		repo := repositoryStub{
			findActiveSessionByToken: func(context.Context, string) mo.Result[domain.Session] { return mo.Ok(session) },
			append: func(_ context.Context, id bson.ObjectID, eventType string, value any) mo.Result[struct{}] {
				if id != session.ID || eventType != domain.EventLoggedOut {
					t.Errorf("Append(%v, %q), want (%v, %q)", id, eventType, session.ID, domain.EventLoggedOut)
				}
				payload = value.(domain.LoggedOutPayload)
				return mo.Ok(struct{}{})
			},
		}
		publisher := &publisherSpy{}
		result := NewService(repo, publisher).Logout(context.Background(), session.Token)
		if result.IsError() {
			t.Fatalf("Logout returned error: %v", result.Error())
		}
		if payload.SessionID != session.ID || payload.Token != session.Token {
			t.Errorf("payload = %#v, want session ID and token", payload)
		}
		if len(publisher.events) != 1 || publisher.events[0].Type != domain.EventLoggedOut || publisher.events[0].Payload != session {
			t.Errorf("published events = %#v, want logged-out session", publisher.events)
		}
	})
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name    string
		result  mo.Result[domain.Session]
		wantErr error
	}{
		{name: "missing", result: mo.Err[domain.Session](errors.New("missing")), wantErr: ErrSessionNotFound},
		{name: "expired", result: mo.Ok(domain.Session{ExpiresAt: time.Now().Add(-time.Minute)}), wantErr: ErrSessionExpired},
		{name: "valid", result: mo.Ok(domain.Session{ID: bson.NewObjectID(), ExpiresAt: time.Now().Add(time.Minute)})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repositoryStub{findActiveSessionByToken: func(context.Context, string) mo.Result[domain.Session] { return tt.result }}
			result := NewService(repo, &publisherSpy{}).ValidateToken(context.Background(), "token")
			if !errors.Is(result.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %v", result.Error(), tt.wantErr)
			}
			if tt.wantErr == nil && result.MustGet() != tt.result.MustGet() {
				t.Errorf("session = %#v, want %#v", result.MustGet(), tt.result.MustGet())
			}
		})
	}
}
