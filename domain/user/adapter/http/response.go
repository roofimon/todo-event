package httpadapter

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"

	"todoe/domain/user/application"
	"todoe/domain/user/domain"
)

func resolveResult(c *fiber.Ctx, result mo.Result[domain.User]) error {
	if result.IsError() {
		var requestError badRequestError
		if errors.As(result.Error(), &requestError) ||
			errors.Is(result.Error(), domain.ErrInvalidName) ||
			errors.Is(result.Error(), domain.ErrInvalidEmail) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(fiber.StatusCreated).JSON(result.MustGet())
}

func resolveVerifyEmailResult(c *fiber.Ctx, result mo.Result[domain.User]) error {
	if result.IsError() {
		var requestError badRequestError
		if errors.As(result.Error(), &requestError) || errors.Is(result.Error(), application.ErrInvalidToken) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(result.MustGet())
}

func resolveGetUserResult(c *fiber.Ctx, result mo.Result[domain.User]) error {
	if result.IsError() {
		var requestError badRequestError
		if errors.As(result.Error(), &requestError) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(result.MustGet())
}

func resolveCompleteProfileResult(c *fiber.Ctx, result mo.Result[domain.User]) error {
	if result.IsError() {
		var requestError badRequestError
		switch {
		case errors.As(result.Error(), &requestError):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": result.Error().Error()})
		case errors.Is(result.Error(), application.ErrCreditDenied):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": result.Error().Error()})
		case errors.Is(result.Error(), application.ErrCreditNotChecked):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(result.MustGet())
}
