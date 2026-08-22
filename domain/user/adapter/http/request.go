package httpadapter

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"todoe/domain/user/domain"
)

type badRequestError struct {
	cause error
}

func (e badRequestError) Error() string {
	return e.cause.Error()
}

type registerBody struct {
	Name  string `json:"name"`
	Email string `json:"email"`
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

type verifyEmailRequest struct {
	userID bson.ObjectID
	token  string
}

type verifyEmailBody struct {
	Token string `json:"token"`
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

type completeProfileRequest struct {
	userID bson.ObjectID
	bio    string
}

type completeProfileBody struct {
	Bio string `json:"bio"`
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

func parseUserID(c *fiber.Ctx) mo.Result[bson.ObjectID] {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return mo.Err[bson.ObjectID](badRequestError{cause: errors.New("invalid id")})
	}
	return mo.Ok(id)
}
