package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

// planByDate indexes a plan's patterns by date and key.
func planByDate(plan domain.PatternPlan) map[string]map[string]domain.PatternStatus {
	byDate := make(map[string]map[string]domain.PatternStatus, len(plan.Dates))
	for _, date := range plan.Dates {
		byDate[date.Date] = make(map[string]domain.PatternStatus, len(date.Patterns))
		for _, pattern := range date.Patterns {
			byDate[date.Date][pattern.Key] = pattern
		}
	}

	return byDate
}

func TestPatternsProjectsDawudUntilRamadan(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)
	lastFast := time.Date(2027, time.January, 24, 0, 0, 0, 0, time.UTC)

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		plan, err := service.Patterns(context.Background(), domain.DtoPatternRequest{
			Start:    "2027-01-25",
			End:      "2027-02-12",
			Language: language,
			History: domain.DtoFastingHistory{
				FastingDates:  []string{"2027-01-24"},
				ObservesDawud: true,
			},
		})
		if err != nil {
			t.Fatalf("Patterns(%s) error = %v", language, err)
		}

		// Ramadan 1448 starts on 2027-02-08 in the Kemenag calendar.
		byDate := planByDate(plan)
		if plan.Total != 14 || len(byDate) != 14 {
			t.Fatalf("Patterns(%s) total = %d, want 14 dates before Ramadan", language, plan.Total)
		}
		for offset := 1; offset <= 14; offset++ {
			date := lastFast.AddDate(0, 0, offset).Format(time.DateOnly)
			want := domain.PatternStatusRestDay
			if offset%2 == 0 {
				want = domain.PatternStatusFastDay
			}

			if got := byDate[date][domain.PatternDawud].Status; got != want {
				t.Errorf("Patterns(%s) %s Dawud = %q, want %q", language, date, got, want)
			}
		}

		fridays := map[string]string{
			"2027-01-29": domain.PatternStatusCaution,
			"2027-02-05": domain.PatternStatusCoincides,
		}
		for date, want := range fridays {
			if got := byDate[date][domain.PatternFridayAlone].Status; got != want {
				t.Errorf("Patterns(%s) %s Friday = %q, want %q", language, date, got, want)
			}
		}

		hasStopNote := len(plan.Notes) == 1 &&
			strings.Contains(plan.Notes[0], "2027-02-08")
		if !hasStopNote {
			t.Errorf("Patterns(%s) notes = %q, want one naming 2027-02-08", language, plan.Notes)
		}
	}
}

func TestPatternsCountsShawwalDaysLeft(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)
	tests := []struct {
		name     string
		calendar domain.Calendar
		start    string
		want     map[string]int
	}{
		// Kemenag's Shawwal 1448 runs 29 days, 2027-03-10 through 2027-04-07.
		{
			name:     "29-day month",
			calendar: domain.CalendarIndonesia,
			start:    "2027-04-05",
			want: map[string]int{
				"2027-04-05": 3,
				"2027-04-06": 2,
				"2027-04-07": 1,
			},
		},
		// Umm al-Qura's Shawwal 1448 runs 30 days, ending on 2027-04-07.
		{
			name:     "30-day month",
			calendar: domain.CalendarUmmAlQura,
			start:    "2027-04-05",
			want: map[string]int{
				"2027-04-05": 3,
				"2027-04-06": 2,
				"2027-04-07": 1,
			},
		},
		{
			name:     "start of Kemenag Shawwal",
			calendar: domain.CalendarIndonesia,
			start:    "2027-03-11",
			want: map[string]int{
				"2027-03-11": 28,
				"2027-03-12": 27,
				"2027-03-13": 26,
				"2027-03-14": 25,
			},
		},
	}

	for _, tt := range tests {
		plan, err := service.Patterns(context.Background(), domain.DtoPatternRequest{
			Start:    tt.start,
			Days:     new(4),
			Calendar: tt.calendar,
			Language: domain.LanguageIndonesian,
			History: domain.DtoFastingHistory{
				FastingDates: []string{"2027-03-11", "2027-03-12"},
			},
		})
		if err != nil {
			t.Fatalf("%s: Patterns() error = %v", tt.name, err)
		}

		byDate := planByDate(plan)
		for date, want := range tt.want {
			shawwal := byDate[date][domain.PatternSixShawwal]
			if shawwal.Progress == nil || shawwal.Progress.EligibleDaysLeft != want {
				t.Errorf("%s: %s Shawwal progress = %+v, want %d days left", tt.name, date, shawwal.Progress, want)
			}
		}
		if len(plan.Notes) != 0 {
			t.Errorf("%s: notes = %q without Dawud, want none", tt.name, plan.Notes)
		}
	}

	plan, err := service.Patterns(context.Background(), domain.DtoPatternRequest{
		Start:    "2027-03-11",
		End:      "2027-03-13",
		Language: domain.LanguageEnglish,
		History: domain.DtoFastingHistory{
			FastingDates: []string{"2027-03-11", "2027-03-12"},
		},
	})
	if err != nil {
		t.Fatalf("Patterns() error = %v", err)
	}

	// Each date counts the other recorded Shawwal days, never itself.
	for date, want := range map[string]int{"2027-03-11": 1, "2027-03-12": 1, "2027-03-13": 2} {
		progress := planByDate(plan)[date][domain.PatternSixShawwal].Progress
		if progress == nil || progress.Recorded != want {
			t.Errorf("%s recorded = %+v, want %d", date, progress, want)
		}
	}
}

func TestPatternsSkipsPilgrimArafah(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)

	// 9 Dhu al-Hijjah 1448 is 2027-05-16 in the Kemenag calendar.
	plan, err := service.Patterns(context.Background(), domain.DtoPatternRequest{
		Start:             "2027-05-15",
		End:               "2027-05-16",
		IsPilgrimAtArafah: true,
		Language:          domain.LanguageEnglish,
		History: domain.DtoFastingHistory{
			FastingDates:  make([]string, 0),
			ObservesDawud: true,
		},
	})
	if err != nil {
		t.Fatalf("Patterns() error = %v", err)
	}

	byDate := planByDate(plan)
	if _, ok := byDate["2027-05-15"]; !ok || plan.Total != 1 {
		t.Errorf("pilgrim plan dates = %+v, want only 2027-05-15", plan.Dates)
	}
}

func TestPatternsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)
	tooLong := make([]string, maxHistoryDates+1)
	for i := range tooLong {
		tooLong[i] = "2027-01-01"
	}
	tests := map[string]struct {
		request  domain.DtoPatternRequest
		wantCode string
	}{
		"missing end": {
			request: domain.DtoPatternRequest{
				Start: "2027-01-01",
			},
			wantCode: domain.CodeRangeNeedsEnd,
		},
		"too many days": {
			request: domain.DtoPatternRequest{
				Start: "2027-01-01",
				Days:  new(367),
			},
			wantCode: domain.CodeRangeDays,
		},
		"history too long": {
			request: domain.DtoPatternRequest{
				Start: "2027-01-01",
				Days:  new(7),
				History: domain.DtoFastingHistory{
					FastingDates: tooLong,
				},
			},
			wantCode: domain.CodeHistoryTooLong,
		},
		"bad history date": {
			request: domain.DtoPatternRequest{
				Start: "2027-01-01",
				Days:  new(7),
				History: domain.DtoFastingHistory{
					FastingDates: []string{"2027-1-1"},
				},
			},
			wantCode: domain.CodeDateFormat,
		},
	}

	for name, tt := range tests {
		_, err := service.Patterns(context.Background(), tt.request)

		var requestErr *domain.RequestError
		if !errors.As(err, &requestErr) || requestErr.Code != tt.wantCode {
			t.Errorf("%s: Patterns() error = %v, want code %s", name, err, tt.wantCode)
		}
	}

	_, err := service.Patterns(context.Background(), domain.DtoPatternRequest{
		Start: "2028-01-01",
		Days:  new(7),
	})
	if !errors.Is(err, domain.ErrCalendarUnavailable) {
		t.Errorf("outside coverage: Patterns() error = %v, want ErrCalendarUnavailable", err)
	}
}
