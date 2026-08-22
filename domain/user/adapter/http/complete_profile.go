package httpadapter

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/application"
	"todoe/domain/user/domain"
	"todoe/infra/result"
)

type completeProfileRequest struct {
	userID bson.ObjectID
	bio    string
}

type completeProfileBody struct {
	Bio string `json:"bio"`
}

func (h *Handler) CompleteProfile(c *fiber.Ctx) error {
	completed := result.FlatMap(parseCompleteProfileRequest(c), h.completeProfile(c.Context()))
	return resolveCompleteProfileResult(c, completed)
}

func parseCompleteProfileRequest(c *fiber.Ctx) mo.Result[completeProfileRequest] {
	id := parseUserID(c)
	if id.IsError() {
		return mo.Err[completeProfileRequest](id.Error())
	}
	var body completeProfileBody
	if err := c.BodyParser(&body); err != nil {
		return mo.Err[completeProfileRequest](badRequestError{cause: err})
	}
	return mo.Ok(completeProfileRequest{userID: id.MustGet(), bio: body.Bio})
}

func (h *Handler) completeProfile(ctx context.Context) func(completeProfileRequest) mo.Result[domain.User] {
	return func(request completeProfileRequest) mo.Result[domain.User] {
		return h.useCase.CompleteProfile(ctx, request.userID, request.bio)
	}
}

func resolveCompleteProfileResult(c *fiber.Ctx, completed mo.Result[domain.User]) error {
	if completed.IsError() {
		var requestError badRequestError
		switch {
		case errors.As(completed.Error(), &requestError):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": completed.Error().Error()})
		case errors.Is(completed.Error(), application.ErrCreditDenied):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": completed.Error().Error()})
		case errors.Is(completed.Error(), application.ErrCreditNotChecked):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": completed.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(completed.MustGet())
}
