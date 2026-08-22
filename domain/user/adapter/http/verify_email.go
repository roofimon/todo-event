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

type verifyEmailRequest struct {
	userID bson.ObjectID
	token  string
}

type verifyEmailBody struct {
	Token string `json:"token"`
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	verified := result.FlatMap(parseVerifyEmailRequest(c), h.verifyEmail(c.Context()))
	return resolveVerifyEmailResult(c, verified)
}

func parseVerifyEmailRequest(c *fiber.Ctx) mo.Result[verifyEmailRequest] {
	id := parseUserID(c)
	if id.IsError() {
		return mo.Err[verifyEmailRequest](id.Error())
	}
	var body verifyEmailBody
	if err := c.BodyParser(&body); err != nil {
		return mo.Err[verifyEmailRequest](badRequestError{cause: err})
	}
	return mo.Ok(verifyEmailRequest{userID: id.MustGet(), token: body.Token})
}

func (h *Handler) verifyEmail(ctx context.Context) func(verifyEmailRequest) mo.Result[domain.User] {
	return func(request verifyEmailRequest) mo.Result[domain.User] {
		return h.useCase.VerifyEmail(ctx, request.userID, request.token)
	}
}

func resolveVerifyEmailResult(c *fiber.Ctx, verified mo.Result[domain.User]) error {
	if verified.IsError() {
		var requestError badRequestError
		if errors.As(verified.Error(), &requestError) || errors.Is(verified.Error(), application.ErrInvalidToken) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": verified.Error().Error()})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(verified.MustGet())
}
