package httpadapter

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"

	"todoe/domain/user/domain"
	"todoe/infra/result"
)

type registerBody struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *Handler) Register(c *fiber.Ctx) error {
	registered := result.FlatMap3(
		parseRegisterBody(c),
		toRegisterInput,
		h.register(c.Context()),
	)
	return resolveRegisterResult(c, registered)
}

func parseRegisterBody(c *fiber.Ctx) mo.Result[registerBody] {
	var body registerBody
	if err := c.BodyParser(&body); err != nil {
		return mo.Err[registerBody](badRequestError{cause: err})
	}
	return mo.Ok(body)
}

func toRegisterInput(body registerBody) mo.Result[domain.RegisterInput] {
	return domain.NewRegisterInput(body.Name, body.Email)
}

func (h *Handler) register(ctx context.Context) func(domain.RegisterInput) mo.Result[domain.User] {
	return func(input domain.RegisterInput) mo.Result[domain.User] {
		return h.useCase.Register(ctx, input)
	}
}

func resolveRegisterResult(c *fiber.Ctx, registered mo.Result[domain.User]) error {
	if registered.IsError() {
		var requestError badRequestError
		if errors.As(registered.Error(), &requestError) ||
			errors.Is(registered.Error(), domain.ErrInvalidName) ||
			errors.Is(registered.Error(), domain.ErrInvalidEmail) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": registered.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(fiber.StatusCreated).JSON(registered.MustGet())
}
