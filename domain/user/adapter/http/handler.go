package httpadapter

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/application"
	"todoe/domain/user/domain"
	"todoe/domain/user/port"
)

type Handler struct {
	useCase port.UseCase
}

func NewHandler(useCase port.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	input := domain.NewRegisterInput(body.Name, body.Email)
	if input.IsError() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": input.Error().Error()})
	}
	result := h.useCase.Register(c.Context(), input.MustGet())
	if result.IsError() {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.Status(fiber.StatusCreated).JSON(result.MustGet())
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	result := h.useCase.VerifyEmail(c.Context(), id, body.Token)
	if result.IsError() {
		if errors.Is(result.Error(), application.ErrInvalidToken) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(result.MustGet())
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	result := h.useCase.GetUser(c.Context(), id)
	if result.IsError() {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(result.MustGet())
}

func (h *Handler) CompleteProfile(c *fiber.Ctx) error {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Bio string `json:"bio"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	result := h.useCase.CompleteProfile(c.Context(), id, body.Bio)
	if result.IsError() {
		switch {
		case errors.Is(result.Error(), application.ErrCreditDenied):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": result.Error().Error()})
		case errors.Is(result.Error(), application.ErrCreditNotChecked):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": result.Error().Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(result.MustGet())
}
