package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

func newTestKemenagCalendar(t *testing.T) *KemenagCalendar {
	t.Helper()

	calendar, err := NewKemenagCalendar()
	if err != nil {
		t.Fatalf("NewKemenagCalendar() error = %v", err)
	}

	return calendar
}

func hijriDate(year, month, day int) domain.HijriDate {
	return domain.HijriDate{
		Year:  year,
		Month: month,
		Day:   day,
	}
}

func TestKemenagCalendarHijriDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		date string
		want domain.HijriDate
	}{
		{
			date: "2026-01-01",
			want: hijriDate(1447, 7, 12),
		},
		{
			date: "2026-02-18",
			want: hijriDate(1447, 8, 30),
		},
		{
			date: "2026-02-19",
			want: hijriDate(1447, 9, 1),
		},
		{
			date: "2026-03-21",
			want: hijriDate(1447, 10, 1),
		},
		{
			date: "2026-05-18",
			want: hijriDate(1447, 12, 1),
		},
		{
			date: "2026-06-25",
			want: hijriDate(1448, 1, 10),
		},
		{
			date: "2026-08-09",
			want: hijriDate(1448, 2, 25),
		},
		{
			date: "2026-08-14",
			want: hijriDate(1448, 3, 1),
		},
		{
			date: "2026-12-31",
			want: hijriDate(1448, 7, 22),
		},
		// Kalender Hijriah Indonesia Tahun 2027, read from the printed grid.
		{
			date: "2027-01-01",
			want: hijriDate(1448, 7, 23),
		},
		{
			date: "2027-01-08",
			want: hijriDate(1448, 7, 30),
		},
		{
			date: "2027-01-09",
			want: hijriDate(1448, 8, 1),
		},
		{
			date: "2027-02-08",
			want: hijriDate(1448, 9, 1),
		},
		{
			date: "2027-03-09",
			want: hijriDate(1448, 9, 30),
		},
		{
			date: "2027-03-10",
			want: hijriDate(1448, 10, 1),
		},
		{
			date: "2027-05-17",
			want: hijriDate(1448, 12, 10),
		},
		{
			date: "2027-06-06",
			want: hijriDate(1449, 1, 1),
		},
		{
			date: "2027-08-15",
			want: hijriDate(1449, 3, 12),
		},
		{
			date: "2027-12-26",
			want: hijriDate(1449, 7, 27),
		},
		{
			date: "2027-12-31",
			want: hijriDate(1449, 8, 3),
		},
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
	outside := []time.Time{
		date(2025, time.December, 27),
		date(2028, time.January, 1),
	}

	for _, value := range outside {
		_, err := calendar.HijriDate(context.Background(), value)
		if !errors.Is(err, domain.ErrDateOutsideCalendar) {
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

		if want := hijriDate(start.year, start.month, 1); got != want {
			t.Errorf("HijriDate(%s) = %+v, want %+v", start.date.Format(time.DateOnly), got, want)
		}
	}

	for _, endpoint := range []time.Time{calendar.coverageStart, calendar.coverageEnd} {
		if _, err := calendar.HijriDate(context.Background(), endpoint); err != nil {
			t.Errorf("coverage endpoint %s rejected: %v", endpoint.Format(time.DateOnly), err)
		}
	}

	coverage := calendar.coverageStart.Format(time.DateOnly) + ".." + calendar.coverageEnd.Format(time.DateOnly)
	if coverage != "2025-12-28..2027-12-31" {
		t.Errorf("coverage = %s, want 2025-12-28..2027-12-31", coverage)
	}
}

func TestKemenagCalendarSourceNamesPublications(t *testing.T) {
	t.Parallel()

	calendar := newTestKemenagCalendar(t)
	tests := []struct {
		start time.Time
		end   time.Time
		want  []string
	}{
		{
			start: date(2026, time.August, 1),
			end:   date(2026, time.August, 31),
			want:  []string{"kemenag-ri-2026"},
		},
		{
			start: date(2027, time.January, 1),
			end:   date(2027, time.December, 31),
			want:  []string{"kemenag-ri-2027"},
		},
		{
			start: date(2025, time.December, 29),
			end:   date(2026, time.January, 4),
			want:  []string{"kemenag-ri-2026"},
		},
		{
			start: date(2026, time.December, 28),
			end:   date(2027, time.January, 3),
			want:  []string{"kemenag-ri-2026", "kemenag-ri-2027"},
		},
	}

	for _, tt := range tests {
		source := calendar.Source(tt.start, tt.end)

		got := make([]string, 0, len(source.Publications))
		for _, publication := range source.Publications {
			got = append(got, publication.ID)
		}
		span := tt.start.Format(time.DateOnly) + ".." + tt.end.Format(time.DateOnly)
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("Source(%s) publications = %v, want %v", span, got, tt.want)
		}
		if source.URL != source.Publications[0].URL || source.CoverageEnd != "2027-12-31" {
			t.Errorf("Source(%s) url/coverage = %s/%s", span, source.URL, source.CoverageEnd)
		}
	}
}

func testPublication(
	year int,
	coverageStart string,
	coverageEnd string,
	starts ...kemenagMonthStart,
) kemenagPublication {
	return kemenagPublication{
		Year:          year,
		ID:            fmt.Sprintf("test-%d", year),
		Name:          "test",
		URL:           "https://example.test",
		SHA256:        strings.Repeat("0", 64),
		CoverageStart: coverageStart,
		CoverageEnd:   coverageEnd,
		MonthStarts:   starts,
	}
}

func testMonthStart(date string, month int) kemenagMonthStart {
	return kemenagMonthStart{
		Date:  date,
		Year:  1447,
		Month: month,
	}
}

func publicationFile(t *testing.T, publication kemenagPublication) *fstest.MapFile {
	t.Helper()

	content, err := json.Marshal(publication)
	if err != nil {
		t.Fatal(err)
	}

	return &fstest.MapFile{Data: content}
}

func TestLoadKemenagCalendarRejectsInconsistentData(t *testing.T) {
	t.Parallel()

	valid := testPublication(
		2026,
		"2026-01-01",
		"2026-01-31",
		testMonthStart("2025-12-21", 7),
		testMonthStart("2026-01-20", 8),
	)
	tests := map[string]kemenagPublication{
		"month length": testPublication(
			2027,
			"2026-02-01",
			"2026-02-10",
			testMonthStart("2026-01-20", 8),
			testMonthStart("2026-02-20", 9),
		),
		"conflicting boundary": testPublication(
			2027,
			"2026-02-01",
			"2026-02-10",
			testMonthStart("2026-01-20", 9),
		),
		"coverage gap": testPublication(
			2027,
			"2026-02-02",
			"2026-02-10",
			testMonthStart("2026-01-20", 8),
		),
		"skipped month": testPublication(
			2027,
			"2026-02-01",
			"2026-02-10",
			testMonthStart("2026-01-20", 8),
			testMonthStart("2026-02-19", 10),
		),
		"last month too long": testPublication(
			2027,
			"2026-02-01",
			"2026-02-25",
			testMonthStart("2026-01-20", 8),
		),
	}

	validFiles := fstest.MapFS{
		"kemenag/2026.json": publicationFile(t, valid),
	}
	if _, err := loadKemenagCalendar(validFiles); err != nil {
		t.Fatalf("valid data rejected: %v", err)
	}

	for name, second := range tests {
		files := fstest.MapFS{
			"kemenag/2026.json": publicationFile(t, valid),
			"kemenag/2027.json": publicationFile(t, second),
		}
		if _, err := loadKemenagCalendar(files); err == nil {
			t.Errorf("%s: loadKemenagCalendar() error = nil", name)
		}
	}

	unknownField := publicationFile(t, valid)
	unknownField.Data = bytes.Replace(unknownField.Data, []byte(`{"year":2026`), []byte(`{"year":2026,"extra":true`), 1)
	if _, err := loadKemenagCalendar(fstest.MapFS{"kemenag/2026.json": unknownField}); err == nil {
		t.Error("unknown field: loadKemenagCalendar() error = nil")
	}
}
