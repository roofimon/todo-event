package httpadapter

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/application"
	"todoe/domain/user/domain"
)

type useCaseStub struct {
	register        func(context.Context, domain.RegisterInput) mo.Result[domain.User]
	verifyEmail     func(context.Context, bson.ObjectID, string) mo.Result[domain.User]
	completeProfile func(context.Context, bson.ObjectID, string) mo.Result[domain.User]
}

func (s useCaseStub) Register(ctx context.Context, input domain.RegisterInput) mo.Result[domain.User] {
	return s.register(ctx, input)
}

func (s useCaseStub) VerifyEmail(ctx context.Context, id bson.ObjectID, token string) mo.Result[domain.User] {
	return s.verifyEmail(ctx, id, token)
}

func TestVerifyEmailPipeline(t *testing.T) {
	userID := bson.NewObjectID()
	verifiedUser := domain.User{ID: userID, Status: domain.StatusEmailVerified}
	notFoundErr := errors.New("user not found")

	tests := []struct {
		name       string
		id         string
		body       string
		verify     func(context.Context, bson.ObjectID, string) mo.Result[domain.User]
		wantStatus int
		wantCalled bool
	}{
		{name: "invalid id", id: "invalid", body: `{"token":"token"}`, wantStatus: fiber.StatusBadRequest},
		{name: "malformed body", id: userID.Hex(), body: `{`, wantStatus: fiber.StatusBadRequest},
		{
			name: "invalid token",
			id:   userID.Hex(),
			body: `{"token":"wrong"}`,
			verify: func(context.Context, bson.ObjectID, string) mo.Result[domain.User] {
				return mo.Err[domain.User](application.ErrInvalidToken)
			},
			wantStatus: fiber.StatusBadRequest,
			wantCalled: true,
		},
		{
			name: "missing user",
			id:   userID.Hex(),
			body: `{"token":"token"}`,
			verify: func(context.Context, bson.ObjectID, string) mo.Result[domain.User] {
				return mo.Err[domain.User](notFoundErr)
			},
			wantStatus: fiber.StatusNotFound,
			wantCalled: true,
		},
		{
			name: "success",
			id:   userID.Hex(),
			body: `{"token":"token"}`,
			verify: func(_ context.Context, id bson.ObjectID, token string) mo.Result[domain.User] {
				if id != userID || token != "token" {
					t.Errorf("VerifyEmail(%v, %q), want (%v, %q)", id, token, userID, "token")
				}
				return mo.Ok(verifiedUser)
			},
			wantStatus: fiber.StatusOK,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			stub := useCaseStub{verifyEmail: func(ctx context.Context, id bson.ObjectID, token string) mo.Result[domain.User] {
				called = true
				return tt.verify(ctx, id, token)
			}}
			app := fiber.New()
			app.Post("/users/:id/verify-email", NewHandler(stub).VerifyEmail)
			request := httptest.NewRequest("POST", "/users/"+tt.id+"/verify-email", strings.NewReader(tt.body))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Errorf("VerifyEmail called = %t, want %t", called, tt.wantCalled)
			}
		})
	}
}

func (useCaseStub) RecordCreditScore(context.Context, bson.ObjectID, int, bool) mo.Result[domain.User] {
	panic("unexpected RecordCreditScore call")
}

func (s useCaseStub) CompleteProfile(ctx context.Context, id bson.ObjectID, bio string) mo.Result[domain.User] {
	return s.completeProfile(ctx, id, bio)
}

func TestCompleteProfilePipeline(t *testing.T) {
	userID := bson.NewObjectID()
	completedUser := domain.User{ID: userID, Bio: "Software engineer", Status: domain.StatusOnboardingComplete}
	repositoryErr := errors.New("repository unavailable")

	tests := []struct {
		name       string
		id         string
		body       string
		complete   func(context.Context, bson.ObjectID, string) mo.Result[domain.User]
		wantStatus int
		wantCalled bool
	}{
		{name: "invalid id", id: "invalid", body: `{"bio":"Software engineer"}`, wantStatus: fiber.StatusBadRequest},
		{name: "malformed body", id: userID.Hex(), body: `{`, wantStatus: fiber.StatusBadRequest},
		{
			name: "credit denied",
			id:   userID.Hex(),
			body: `{"bio":"Software engineer"}`,
			complete: func(context.Context, bson.ObjectID, string) mo.Result[domain.User] {
				return mo.Err[domain.User](application.ErrCreditDenied)
			},
			wantStatus: fiber.StatusForbidden,
			wantCalled: true,
		},
		{
			name: "credit not checked",
			id:   userID.Hex(),
			body: `{"bio":"Software engineer"}`,
			complete: func(context.Context, bson.ObjectID, string) mo.Result[domain.User] {
				return mo.Err[domain.User](application.ErrCreditNotChecked)
			},
			wantStatus: fiber.StatusConflict,
			wantCalled: true,
		},
		{
			name: "service failure",
			id:   userID.Hex(),
			body: `{"bio":"Software engineer"}`,
			complete: func(context.Context, bson.ObjectID, string) mo.Result[domain.User] {
				return mo.Err[domain.User](repositoryErr)
			},
			wantStatus: fiber.StatusInternalServerError,
			wantCalled: true,
		},
		{
			name: "success",
			id:   userID.Hex(),
			body: `{"bio":"Software engineer"}`,
			complete: func(_ context.Context, id bson.ObjectID, bio string) mo.Result[domain.User] {
				if id != userID || bio != "Software engineer" {
					t.Errorf("CompleteProfile(%v, %q), want (%v, %q)", id, bio, userID, "Software engineer")
				}
				return mo.Ok(completedUser)
			},
			wantStatus: fiber.StatusOK,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			stub := useCaseStub{completeProfile: func(ctx context.Context, id bson.ObjectID, bio string) mo.Result[domain.User] {
				called = true
				return tt.complete(ctx, id, bio)
			}}
			app := fiber.New()
			app.Post("/users/:id/complete-profile", NewHandler(stub).CompleteProfile)
			request := httptest.NewRequest("POST", "/users/"+tt.id+"/complete-profile", strings.NewReader(tt.body))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Errorf("CompleteProfile called = %t, want %t", called, tt.wantCalled)
			}
		})
	}
}

func (useCaseStub) GetUser(context.Context, bson.ObjectID) mo.Result[domain.User] {
	panic("unexpected GetUser call")
}

func TestRegisterPipeline(t *testing.T) {
	createdUser := domain.User{
		ID:        bson.NewObjectID(),
		Name:      "Alice",
		Email:     "alice@example.com",
		CreatedAt: time.Now(),
	}
	repositoryErr := errors.New("repository unavailable")

	tests := []struct {
		name       string
		body       string
		register   func(context.Context, domain.RegisterInput) mo.Result[domain.User]
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "malformed body",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid input",
			body:       `{"name":" ","email":"alice@example.com"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service failure",
			body: `{"name":"Alice","email":"alice@example.com"}`,
			register: func(context.Context, domain.RegisterInput) mo.Result[domain.User] {
				return mo.Err[domain.User](repositoryErr)
			},
			wantStatus: fiber.StatusInternalServerError,
			wantCalled: true,
		},
		{
			name: "success with normalized input",
			body: `{"name":" Alice ","email":" alice@example.com "}`,
			register: func(_ context.Context, input domain.RegisterInput) mo.Result[domain.User] {
				if input.Name() != "Alice" || input.Email() != "alice@example.com" {
					t.Errorf("input = (%q, %q), want normalized values", input.Name(), input.Email())
				}
				return mo.Ok(createdUser)
			},
			wantStatus: fiber.StatusCreated,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			stub := useCaseStub{register: func(ctx context.Context, input domain.RegisterInput) mo.Result[domain.User] {
				called = true
				return tt.register(ctx, input)
			}}
			app := fiber.New()
			app.Post("/users/register", NewHandler(stub).Register)
			request := httptest.NewRequest("POST", "/users/register", strings.NewReader(tt.body))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Errorf("Register called = %t, want %t", called, tt.wantCalled)
			}
		})
	}
}
