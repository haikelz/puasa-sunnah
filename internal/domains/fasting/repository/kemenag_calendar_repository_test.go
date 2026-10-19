package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

func newTestKemenagCalendar(t *testing.T) *KemenagCalendar {
	t.Helper()
	calendar, err := NewKemenagCalendar()
	if err != nil {
		t.Fatalf("NewKemenagCalendar() error = %v", err)
	}
	return calendar
}

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
		// Kalender Hijriah Indonesia Tahun 2027, read from the printed grid.
		{date: "2027-01-01", want: fasting.HijriDate{Year: 1448, Month: 7, Day: 23}},
		{date: "2027-01-08", want: fasting.HijriDate{Year: 1448, Month: 7, Day: 30}},
		{date: "2027-01-09", want: fasting.HijriDate{Year: 1448, Month: 8, Day: 1}},
		{date: "2027-02-08", want: fasting.HijriDate{Year: 1448, Month: 9, Day: 1}},
		{date: "2027-03-09", want: fasting.HijriDate{Year: 1448, Month: 9, Day: 30}},
		{date: "2027-03-10", want: fasting.HijriDate{Year: 1448, Month: 10, Day: 1}},
		{date: "2027-05-17", want: fasting.HijriDate{Year: 1448, Month: 12, Day: 10}},
		{date: "2027-06-06", want: fasting.HijriDate{Year: 1449, Month: 1, Day: 1}},
		{date: "2027-08-15", want: fasting.HijriDate{Year: 1449, Month: 3, Day: 12}},
		{date: "2027-12-26", want: fasting.HijriDate{Year: 1449, Month: 7, Day: 27}},
		{date: "2027-12-31", want: fasting.HijriDate{Year: 1449, Month: 8, Day: 3}},
	}

	calendar := newTestKemenagCalendar(t)
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

	calendar := newTestKemenagCalendar(t)
	for _, value := range []time.Time{date(2025, time.December, 27), date(2028, time.January, 1)} {
		_, err := calendar.HijriDate(context.Background(), value)
		if !errors.Is(err, fasting.ErrDateOutsideCalendar) {
			t.Errorf("HijriDate(%s) error = %v, want ErrDateOutsideCalendar", value.Format(time.DateOnly), err)
		}
	}
}

func TestKemenagCalendarHonorsCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestKemenagCalendar(t).HijriDate(ctx, date(2026, time.January, 1))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("HijriDate() error = %v, want context.Canceled", err)
	}
}

func TestKemenagCalendarMonthBoundaries(t *testing.T) {
	t.Parallel()

	calendar := newTestKemenagCalendar(t)
	for _, start := range calendar.starts {
		if start.date.Before(calendar.coverageStart) {
			continue
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

	for _, endpoint := range []time.Time{calendar.coverageStart, calendar.coverageEnd} {
		if _, err := calendar.HijriDate(context.Background(), endpoint); err != nil {
			t.Errorf("coverage endpoint %s rejected: %v", endpoint.Format(time.DateOnly), err)
		}
	}
	if got := calendar.coverageStart.Format(time.DateOnly) + ".." + calendar.coverageEnd.Format(time.DateOnly); got != "2025-12-28..2027-12-31" {
		t.Errorf("coverage = %s, want 2025-12-28..2027-12-31", got)
	}
}

func TestKemenagCalendarSourceNamesPublications(t *testing.T) {
	t.Parallel()

	calendar := newTestKemenagCalendar(t)
	tests := []struct {
		start, end time.Time
		want       []string
	}{
		{date(2026, time.August, 1), date(2026, time.August, 31), []string{"kemenag-ri-2026"}},
		{date(2027, time.January, 1), date(2027, time.December, 31), []string{"kemenag-ri-2027"}},
		{date(2025, time.December, 29), date(2026, time.January, 4), []string{"kemenag-ri-2026"}},
		{date(2026, time.December, 28), date(2027, time.January, 3), []string{"kemenag-ri-2026", "kemenag-ri-2027"}},
	}
	for _, tt := range tests {
		source := calendar.Source(tt.start, tt.end)
		got := make([]string, 0, len(source.Publications))
		for _, publication := range source.Publications {
			got = append(got, publication.ID)
		}
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("Source(%s..%s) publications = %v, want %v", tt.start.Format(time.DateOnly), tt.end.Format(time.DateOnly), got, tt.want)
		}
		if source.URL != source.Publications[0].URL || source.CoverageEnd != "2027-12-31" {
			t.Errorf("Source(%s..%s) url/coverage = %s/%s", tt.start.Format(time.DateOnly), tt.end.Format(time.DateOnly), source.URL, source.CoverageEnd)
		}
	}
}

func TestLoadKemenagCalendarRejectsInconsistentData(t *testing.T) {
	t.Parallel()

	const header = `"id":"x-%s","name":"x","url":"https://example.test","sha256":"` +
		"0000000000000000000000000000000000000000000000000000000000000000" + `"`
	publication := func(year, coverage, starts string) string {
		return `{"year":` + year + `,` + strings.Replace(header, "%s", year, 1) + `,` + coverage + `,"month_starts":[` + starts + `]}`
	}
	valid := publication("2026", `"coverage_start":"2026-01-01","coverage_end":"2026-01-31"`,
		`{"date":"2025-12-21","hijri_year":1447,"hijri_month":7},{"date":"2026-01-20","hijri_year":1447,"hijri_month":8}`)

	tests := map[string]string{
		"month length": publication("2027", `"coverage_start":"2026-02-01","coverage_end":"2026-02-10"`,
			`{"date":"2026-01-20","hijri_year":1447,"hijri_month":8},{"date":"2026-02-20","hijri_year":1447,"hijri_month":9}`),
		"conflicting boundary": publication("2027", `"coverage_start":"2026-02-01","coverage_end":"2026-02-10"`,
			`{"date":"2026-01-20","hijri_year":1447,"hijri_month":9}`),
		"coverage gap": publication("2027", `"coverage_start":"2026-02-02","coverage_end":"2026-02-10"`,
			`{"date":"2026-01-20","hijri_year":1447,"hijri_month":8}`),
		"skipped month": publication("2027", `"coverage_start":"2026-02-01","coverage_end":"2026-02-10"`,
			`{"date":"2026-01-20","hijri_year":1447,"hijri_month":8},{"date":"2026-02-19","hijri_year":1447,"hijri_month":10}`),
		"last month too long": publication("2027", `"coverage_start":"2026-02-01","coverage_end":"2026-02-25"`,
			`{"date":"2026-01-20","hijri_year":1447,"hijri_month":8}`),
		"unknown field": strings.Replace(valid, `"year":2026`, `"year":2027,"extra":true`, 1),
	}

	if _, err := loadKemenagCalendar(fstest.MapFS{"kemenag/2026.json": {Data: []byte(valid)}}); err != nil {
		t.Fatalf("valid data rejected: %v", err)
	}
	for name, second := range tests {
		files := fstest.MapFS{
			"kemenag/2026.json": {Data: []byte(valid)},
			"kemenag/2027.json": {Data: []byte(second)},
		}
		if _, err := loadKemenagCalendar(files); err == nil {
			t.Errorf("%s: loadKemenagCalendar() error = nil", name)
		}
	}
}
