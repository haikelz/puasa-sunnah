package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

func TestKemenagCalendarHijriDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		date string
		want fasting.HijriDate
	}{
		{date: "2026-01-01", want: fasting.HijriDate{Year: 1447, Month: 7, Day: 12}},
		{date: "2026-02-18", want: fasting.HijriDate{Year: 1447, Month: 8, Day: 30}},
		{date: "2026-02-19", want: fasting.HijriDate{Year: 1447, Month: 9, Day: 1}},
		{date: "2026-03-21", want: fasting.HijriDate{Year: 1447, Month: 10, Day: 1}},
		{date: "2026-05-18", want: fasting.HijriDate{Year: 1447, Month: 12, Day: 1}},
		{date: "2026-06-25", want: fasting.HijriDate{Year: 1448, Month: 1, Day: 10}},
		{date: "2026-08-09", want: fasting.HijriDate{Year: 1448, Month: 2, Day: 25}},
		{date: "2026-08-14", want: fasting.HijriDate{Year: 1448, Month: 3, Day: 1}},
		{date: "2026-12-31", want: fasting.HijriDate{Year: 1448, Month: 7, Day: 22}},
	}

	calendar := NewKemenagCalendar()
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			t.Parallel()
			date, err := time.Parse(time.DateOnly, tt.date)
			if err != nil {
				t.Fatal(err)
			}
			got, err := calendar.HijriDate(context.Background(), date)
			if err != nil {
				t.Fatalf("HijriDate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("HijriDate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestKemenagCalendarRejectsUnsupportedDate(t *testing.T) {
	t.Parallel()

	calendar := NewKemenagCalendar()
	for _, value := range []time.Time{coverageStart.AddDate(0, 0, -1), coverageEnd.AddDate(0, 0, 1)} {
		_, err := calendar.HijriDate(context.Background(), value)
		if !errors.Is(err, fasting.ErrDateOutsideCalendar) {
			t.Errorf("HijriDate(%s) error = %v, want ErrDateOutsideCalendar", value.Format(time.DateOnly), err)
		}
	}
}

func TestKemenagCalendarMonthBoundaries(t *testing.T) {
	t.Parallel()

	calendar := NewKemenagCalendar()
	for i := 1; i < len(calendar.starts); i++ {
		start := calendar.starts[i]
		previous := calendar.starts[i-1]
		monthLength := int(start.date.Sub(previous.date).Hours() / 24)
		if monthLength != 29 && monthLength != 30 {
			t.Errorf("month before %s has %d days, want 29 or 30", start.date.Format(time.DateOnly), monthLength)
		}

		got, err := calendar.HijriDate(context.Background(), start.date)
		if err != nil {
			t.Fatalf("HijriDate(%s) error = %v", start.date.Format(time.DateOnly), err)
		}
		want := fasting.HijriDate{Year: start.year, Month: start.month, Day: 1}
		if got != want {
			t.Errorf("HijriDate(%s) = %+v, want %+v", start.date.Format(time.DateOnly), got, want)
		}
	}

	for _, endpoint := range []time.Time{coverageStart, coverageEnd} {
		if _, err := calendar.HijriDate(context.Background(), endpoint); err != nil {
			t.Errorf("coverage endpoint %s rejected: %v", endpoint.Format(time.DateOnly), err)
		}
	}

	source := calendar.Source()
	if source.CoverageStart != coverageStart.Format(time.DateOnly) || source.CoverageEnd != coverageEnd.Format(time.DateOnly) {
		t.Errorf("source coverage = %s..%s, want %s..%s", source.CoverageStart, source.CoverageEnd, coverageStart.Format(time.DateOnly), coverageEnd.Format(time.DateOnly))
	}
}
