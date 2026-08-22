package httpadapter

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/domain"
	"todoe/domain/user/port"
	"todoe/infra/result"
)

type Handler struct {
	useCase port.UseCase
}

func NewHandler(useCase port.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	parsed := parseRegisterBody(c)
	input := result.FlatMap(parsed, toRegisterInput)
	registered := result.FlatMap(input, h.register(c.Context()))
	return resolveResult(c, registered)
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	parsed := parseVerifyEmailRequest(c)
	verified := result.FlatMap(parsed, h.verifyEmail(c.Context()))
	return resolveVerifyEmailResult(c, verified)
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
	parsed := parseUserID(c)
	user := result.FlatMap(parsed, h.getUser(c.Context()))
	return resolveGetUserResult(c, user)
}

func (h *Handler) CompleteProfile(c *fiber.Ctx) error {
	parsed := parseCompleteProfileRequest(c)
	completed := result.FlatMap(parsed, h.completeProfile(c.Context()))
	return resolveCompleteProfileResult(c, completed)
}

func (h *Handler) register(ctx context.Context) func(domain.RegisterInput) mo.Result[domain.User] {
	return func(input domain.RegisterInput) mo.Result[domain.User] {
		return h.useCase.Register(ctx, input)
	}
}

func (h *Handler) verifyEmail(ctx context.Context) func(verifyEmailRequest) mo.Result[domain.User] {
	return func(request verifyEmailRequest) mo.Result[domain.User] {
		return h.useCase.VerifyEmail(ctx, request.userID, request.token)
	}
}

func (h *Handler) getUser(ctx context.Context) func(bson.ObjectID) mo.Result[domain.User] {
	return func(id bson.ObjectID) mo.Result[domain.User] {
		return h.useCase.GetUser(ctx, id)
	}
}

func (h *Handler) completeProfile(ctx context.Context) func(completeProfileRequest) mo.Result[domain.User] {
	return func(request completeProfileRequest) mo.Result[domain.User] {
		return h.useCase.CompleteProfile(ctx, request.userID, request.bio)
	}
}
