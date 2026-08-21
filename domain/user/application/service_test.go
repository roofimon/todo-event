package application

import (
	"context"
	"errors"
	"testing"

	"todoe/domain/user/domain"
)

func TestRegisterRejectsUninitializedInput(t *testing.T) {
	service := NewService(nil, nil)

	result := service.Register(context.Background(), domain.RegisterInput{})

	if !errors.Is(result.Error(), domain.ErrInvalidName) {
		t.Fatalf("Register error = %v, want %v", result.Error(), domain.ErrInvalidName)
	}
}
