package http

import (
	"errors"
	"testing"

	"puasa-sunnah/internal/domains/fasting"
	fastingusecase "puasa-sunnah/internal/domains/fasting/usecase"
)

func TestNewHandlerRequiresUsecase(t *testing.T) {
	t.Parallel()

	var typedNil *fastingusecase.FastingUsecase
	tests := []struct {
		name    string
		usecase fasting.Usecase
	}{
		{name: "nil"},
		{name: "typed nil", usecase: typedNil},
	}
	for _, tt := range tests {
		handler, err := NewHandler(tt.usecase)
		if handler != nil {
			t.Errorf("%s: NewHandler() handler = %#v, want nil", tt.name, handler)
		}
		if !errors.Is(err, errUsecaseRequired) {
			t.Errorf("%s: NewHandler() error = %v, want %v", tt.name, err, errUsecaseRequired)
		}
	}
}
