package http

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	domain "puasa-sunnah/internal/domains/fasting"
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
	schedule := domain.Schedule{
		Calendar: domain.CalendarIndonesia,
		Dates:    []domain.ScheduleDate{{Date: "2026-12-31", Recommendations: []domain.Occurrence{{Name: "A,B;C\\D\n" + strings.Repeat("é", 80)}}}},
	}
	encoded := scheduleICS(schedule, 0)
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

func TestScheduleICSReminder(t *testing.T) {
	t.Parallel()

	schedule := domain.Schedule{
		Calendar: domain.CalendarIndonesia,
		Dates: []domain.ScheduleDate{
			{
				Date: "2027-10-15",
				Recommendations: []domain.Occurrence{
					{Name: "Ayyam al-Bid"},
				},
			},
		},
	}
	tests := []struct {
		reminder string
		want     string
	}{
		{
			reminder: "20:00",
			want:     "TRIGGER:-PT4H",
		},
		{
			reminder: "21:30",
			want:     "TRIGGER:-PT2H30M",
		},
		{
			reminder: "00:00",
			want:     "TRIGGER:-PT24H",
		},
	}

	for _, tt := range tests {
		reminder, err := parseReminder(tt.reminder, formatICS, domain.LanguageEnglish)
		if err != nil {
			t.Fatalf("parseReminder(%q) error = %v", tt.reminder, err)
		}

		encoded := scheduleICS(schedule, reminder)

		alarm := "BEGIN:VALARM\r\nACTION:DISPLAY\r\nDESCRIPTION:Ayyam al-Bid\r\n" + tt.want + "\r\nEND:VALARM"
		if !strings.Contains(encoded, alarm) {
			t.Errorf("reminder %s: %s", tt.reminder, encoded)
		}
	}

	encoded := scheduleICS(schedule, 0)
	hasNoAlarm := !strings.Contains(encoded, "VALARM") &&
		strings.Contains(encoded, "REFRESH-INTERVAL;VALUE=DURATION:P1D")
	if !hasNoAlarm {
		t.Errorf("no-reminder calendar: %s", encoded)
	}

	invalid := []struct {
		value  string
		format string
	}{
		{
			value:  "8pm",
			format: formatICS,
		},
		{
			value:  "25:00",
			format: formatICS,
		},
		{
			value:  "20:00",
			format: formatJSON,
		},
	}
	for _, tt := range invalid {
		if _, err := parseReminder(tt.value, tt.format, domain.LanguageEnglish); err == nil {
			t.Errorf("parseReminder(%q, %q) error = nil", tt.value, tt.format)
		}
	}
}
