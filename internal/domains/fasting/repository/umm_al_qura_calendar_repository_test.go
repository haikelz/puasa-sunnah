package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

func TestUmmAlQuraCalendarHijriDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		gregorian string
		want      fasting.HijriDate
	}{
		{"2026-02-18", fasting.HijriDate{Year: 1447, Month: 9, Day: 1}},
		{"2026-03-20", fasting.HijriDate{Year: 1447, Month: 10, Day: 1}},
		{"2026-06-16", fasting.HijriDate{Year: 1448, Month: 1, Day: 1}},
		{"2027-02-08", fasting.HijriDate{Year: 1448, Month: 9, Day: 1}},
		{"2027-06-06", fasting.HijriDate{Year: 1449, Month: 1, Day: 1}},
	}

	calendar := NewUmmAlQuraCalendar()
	for _, tt := range tests {
		t.Run(tt.gregorian, func(t *testing.T) {
			date, err := time.Parse(time.DateOnly, tt.gregorian)
			if err != nil {
				t.Fatal(err)
			}
			got, err := calendar.HijriDate(context.Background(), date)
			if err != nil {
				t.Fatalf("HijriDate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("HijriDate() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestUmmAlQuraCalendarRejectsUnsupportedDate(t *testing.T) {
	t.Parallel()

	calendar := NewUmmAlQuraCalendar()
	for _, value := range []time.Time{
		time.Date(1937, time.March, 13, 0, 0, 0, 0, time.UTC),
		time.Date(2077, time.November, 17, 0, 0, 0, 0, time.UTC),
	} {
		_, err := calendar.HijriDate(context.Background(), value)
		if !errors.Is(err, fasting.ErrDateOutsideCalendar) {
			t.Errorf("HijriDate(%s) error = %v, want ErrDateOutsideCalendar", value.Format(time.DateOnly), err)
		}
	}
}
