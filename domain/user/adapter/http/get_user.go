package httpadapter

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/domain"
	"todoe/infra/result"
)

func (h *Handler) GetUser(c *fiber.Ctx) error {
	user := result.FlatMap(parseUserID(c), h.getUser(c.Context()))
	return resolveGetUserResult(c, user)
}

func (h *Handler) getUser(ctx context.Context) func(bson.ObjectID) mo.Result[domain.User] {
	return func(id bson.ObjectID) mo.Result[domain.User] {
		return h.useCase.GetUser(ctx, id)
	}
}

func resolveGetUserResult(c *fiber.Ctx, user mo.Result[domain.User]) error {
	if user.IsError() {
		var requestError badRequestError
		if errors.As(user.Error(), &requestError) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": user.Error().Error()})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(user.MustGet())
}
