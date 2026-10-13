package http

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

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

func TestCalendarTextEncoding(t *testing.T) {
	t.Parallel()
	schedule := fasting.Schedule{
		Calendar: fasting.CalendarIndonesia,
		Dates:    []fasting.ScheduleDate{{Date: "2026-12-31", Recommendations: []fasting.Occurrence{{Name: "A,B;C\\D\n" + strings.Repeat("é", 80)}}}},
	}
	encoded := scheduleICS(schedule)
	for _, line := range strings.Split(encoded, "\r\n") {
		if len(line) > 75 || !utf8.ValidString(line) {
			t.Fatalf("invalid folded line %q", line)
		}
	}
	unfolded := strings.ReplaceAll(encoded, "\r\n ", "")
	if !strings.Contains(unfolded, "SUMMARY:A\\,B\\;C\\\\D\\n") || !strings.Contains(unfolded, "DTEND;VALUE=DATE:20270101") {
		t.Fatalf("encoding = %s", unfolded)
	}
}
