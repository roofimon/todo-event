package domain

import (
	"errors"
	"testing"
)

func TestNewRegisterInput(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		email     string
		wantName  string
		wantEmail string
		wantErr   error
	}{
		{name: "valid", inputName: "Alice", email: "alice@example.com", wantName: "Alice", wantEmail: "alice@example.com"},
		{name: "normalizes whitespace", inputName: "  Alice  ", email: "  alice@example.com  ", wantName: "Alice", wantEmail: "alice@example.com"},
		{name: "empty name", inputName: "  ", email: "alice@example.com", wantErr: ErrInvalidName},
		{name: "empty email", inputName: "Alice", email: "\t", wantErr: ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewRegisterInput(tt.inputName, tt.email)
			if !errors.Is(result.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %v", result.Error(), tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			input := result.MustGet()
			if input.Name() != tt.wantName || input.Email() != tt.wantEmail {
				t.Errorf("input = (%q, %q), want (%q, %q)", input.Name(), input.Email(), tt.wantName, tt.wantEmail)
			}
		})
	}
}
