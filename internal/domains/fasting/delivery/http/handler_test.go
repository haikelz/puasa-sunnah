package http

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"puasa-sunnah/internal/domains/fasting"
)

func TestNewHandlerRequiresUsecase(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(nil)
	if handler != nil {
		t.Errorf("NewHandler() handler = %#v, want nil", handler)
	}
	if !errors.Is(err, errUsecaseRequired) {
		t.Errorf("NewHandler() error = %v, want %v", err, errUsecaseRequired)
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
