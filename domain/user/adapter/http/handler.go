package httpadapter

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/port"
)

type Handler struct {
	useCase port.UseCase
}

func NewHandler(useCase port.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

type badRequestError struct {
	cause error
}

func (e badRequestError) Error() string {
	return e.cause.Error()
}

func parseUserID(c *fiber.Ctx) mo.Result[bson.ObjectID] {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return mo.Err[bson.ObjectID](badRequestError{cause: errors.New("invalid id")})
	}
	return mo.Ok(id)
}
