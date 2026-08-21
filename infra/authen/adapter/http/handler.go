package httpadapter

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"todoe/infra/authen/application"
	"todoe/infra/authen/port"
)

type Handler struct {
	useCase port.UseCase
}

func NewHandler(useCase port.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) RegisterCredential(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	result := h.useCase.RegisterCredential(c.Context(), body.Email, body.Password)
	if result.IsError() {
		if errors.Is(result.Error(), application.ErrInvalidCredentials) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	cred := result.MustGet()
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": cred.ID, "email": cred.Email, "created_at": cred.CreatedAt})
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	result := h.useCase.Login(c.Context(), body.Email, body.Password)
	if result.IsError() {
		if errors.Is(result.Error(), application.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(fiber.StatusOK).JSON(result.MustGet())
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing authorization token"})
	}
	result := h.useCase.Logout(c.Context(), token)
	if result.IsError() {
		if errors.Is(result.Error(), application.ErrSessionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ValidateToken(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing authorization token"})
	}
	result := h.useCase.ValidateToken(c.Context(), token)
	if result.IsError() {
		switch {
		case errors.Is(result.Error(), application.ErrSessionExpired):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": result.Error().Error()})
		case errors.Is(result.Error(), application.ErrSessionNotFound):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(result.MustGet())
}
