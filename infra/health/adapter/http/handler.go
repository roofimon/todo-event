package httpadapter

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"todoe/infra/health/port"
)

type Handler struct {
	useCase port.UseCase
}

func NewHandler(useCase port.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) CheckHealth(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	result := h.useCase.CheckHealth(ctx)
	if result.IsError() {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error"})
	}

	health := result.MustGet()
	if health.Status != "ok" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(health)
	}
	return c.JSON(health)
}
