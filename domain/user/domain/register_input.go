package domain

import (
	"errors"
	"strings"

	"github.com/samber/mo"
)

var (
	ErrInvalidName  = errors.New("name must not be empty")
	ErrInvalidEmail = errors.New("email must not be empty")
)

// RegisterInput is a normalized, validated registration command.
// Consumers create it with NewRegisterInput before invoking the use case.
type RegisterInput struct {
	name  string
	email string
}

func NewRegisterInput(name, email string) mo.Result[RegisterInput] {
	input := RegisterInput{
		name:  strings.TrimSpace(name),
		email: strings.TrimSpace(email),
	}

	if err := input.Validate(); err != nil {
		return mo.Err[RegisterInput](err)
	}
	return mo.Ok(input)
}

func (i RegisterInput) Validate() error {
	switch {
	case i.name == "":
		return ErrInvalidName
	case i.email == "":
		return ErrInvalidEmail
	default:
		return nil
	}
}

func (i RegisterInput) Name() string {
	return i.name
}

func (i RegisterInput) Email() string {
	return i.email
}
