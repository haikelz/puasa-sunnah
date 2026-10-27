package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

var dayLanguages = []domain.Language{
	domain.LanguageEnglish,
	domain.LanguageIndonesian,
}

func dayPatterns(t *testing.T, request domain.DtoDayRequest) map[string]domain.PatternStatus {
	t.Helper()

	got, err := newCalendarUsecase(t).Day(context.Background(), request)
	if err != nil {
		t.Fatalf("Day(%s) error = %v", request.Date, err)
	}

	patterns := make(map[string]domain.PatternStatus, len(got.Patterns))
	for _, pattern := range got.Patterns {
		patterns[pattern.Key] = pattern
	}

	return patterns
}

func TestDayClassifiesWithSelectedCalendar(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Day(context.Background(), domain.DtoDayRequest{
		Date: "2027-03-10",
	})
	if err != nil {
		t.Fatalf("Day() error = %v", err)
	}

	wantHijri := domain.HijriDate{
		Year:  1448,
		Month: 10,
		Day:   1,
	}
	if got.Verdict != domain.VerdictProhibited || got.Hijri != wantHijri {
		t.Errorf("Idulfitri verdict/hijri = %s/%+v", got.Verdict, got.Hijri)
	}
	hasSource := got.Calendar == domain.CalendarIndonesia &&
		got.CalendarSource.Publications[0].ID == "kemenag-ri-2027" &&
		strings.Contains(got.CalendarSource.Caveat, "Sidang Isbat")
	if !hasSource {
		t.Errorf("calendar/source = %s/%+v", got.Calendar, got.CalendarSource)
	}
	if got.Patterns == nil || len(got.Patterns) != 0 {
		t.Errorf("patterns without history = %#v, want empty array", got.Patterns)
	}

	today, err := newCalendarUsecase(t).Day(context.Background(), domain.DtoDayRequest{
		Date:     domain.DateToday,
		Calendar: domain.CalendarUmmAlQura,
	})
	if err != nil {
		t.Fatalf("Day(today) error = %v", err)
	}

	if want := time.Now().In(jakarta).Format(time.DateOnly); today.Date != want {
		t.Errorf("today = %s, want %s", today.Date, want)
	}
}

func TestDayRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tooMany := make([]string, maxHistoryDates+1)
	for i := range tooMany {
		tooMany[i] = "2027-08-01"
	}
	tests := map[string]domain.DtoDayRequest{
		"malformed date": {
			Date: "2027-13-01",
		},
		"empty history date": {
			Date: "2027-08-06",
			History: &domain.DtoFastingHistory{
				FastingDates: []string{""},
			},
		},
		"malformed history date": {
			Date: "2027-08-06",
			History: &domain.DtoFastingHistory{
				FastingDates: []string{"06/08/2027"},
			},
		},
		"long history": {
			Date: "2027-08-06",
			History: &domain.DtoFastingHistory{
				FastingDates: tooMany,
			},
		},
		"unknown calendar": {
			Date:     "2027-08-06",
			Calendar: "other",
		},
	}

	for name, request := range tests {
		_, err := newCalendarUsecase(t).Day(context.Background(), request)
		if !errors.Is(err, domain.ErrInvalidRequest) {
			t.Errorf("%s: error = %v, want ErrInvalidRequest", name, err)
		}
	}

	_, err := newCalendarUsecase(t).Day(context.Background(), domain.DtoDayRequest{
		Date:     "2028-01-03",
		Language: domain.LanguageIndonesian,
	})
	var requestError *domain.RequestError
	isUnavailable := errors.Is(err, domain.ErrCalendarUnavailable) &&
		errors.As(err, &requestError) &&
		requestError.Code == domain.CodeOutsideCoverage &&
		requestError.Calendar == domain.CalendarIndonesia
	if !isUnavailable {
		t.Errorf("uncovered date error = %#v, want an outside_coverage error for calendar id", err)
	}
}

func TestDayFridayAlone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		date       string
		history    domain.DtoFastingHistory
		wantStatus string
	}{
		// 2027-08-06 is 3 Rabiulawal 1448: a Friday with no fixed-date recommendation.
		{
			name:       "alone",
			date:       "2027-08-06",
			wantStatus: domain.PatternStatusCaution,
		},
		{
			name: "with Thursday",
			date: "2027-08-06",
			history: domain.DtoFastingHistory{
				FastingDates: []string{"2027-08-05"},
			},
			wantStatus: domain.PatternStatusPaired,
		},
		{
			name: "with planned Saturday",
			date: "2027-08-06",
			history: domain.DtoFastingHistory{
				FastingDates: []string{"2027-08-07"},
			},
			wantStatus: domain.PatternStatusPaired,
		},
		// 2027-10-15 is 14 Jumadilawal: Ayyamul Bidh falls on the Friday.
		{
			name:       "white day",
			date:       "2027-10-15",
			wantStatus: domain.PatternStatusCoincides,
		},
		{
			name: "Dawud fast day",
			date: "2027-08-06",
			history: domain.DtoFastingHistory{
				FastingDates:  []string{"2027-08-04"},
				ObservesDawud: true,
			},
			wantStatus: domain.PatternStatusCoincides,
		},
	}

	for _, tt := range tests {
		for _, language := range dayLanguages {
			history := tt.history
			pattern, ok := dayPatterns(t, domain.DtoDayRequest{
				Date:     tt.date,
				Language: language,
				History:  &history,
			})[domain.PatternFridayAlone]

			if !ok || pattern.Status != tt.wantStatus {
				t.Errorf("%s/%s friday_alone = %+v, want status %s", tt.name, language, pattern, tt.wantStatus)
			}
			if language == domain.LanguageIndonesian && !strings.Contains(pattern.Message, "Jumat") {
				t.Errorf("%s Indonesian message = %q", tt.name, pattern.Message)
			}
			if tt.wantStatus == domain.PatternStatusCoincides && !strings.Contains(pattern.Caveat, "1144b") {
				t.Errorf("%s/%s coincides caveat = %q", tt.name, language, pattern.Caveat)
			}
		}
	}

	patterns := dayPatterns(t, domain.DtoDayRequest{
		Date:    "2027-08-05",
		History: &domain.DtoFastingHistory{},
	})
	if len(patterns) != 0 {
		t.Errorf("Thursday patterns = %+v, want none", patterns)
	}
}

func TestDayDawudAlternation(t *testing.T) {
	t.Parallel()

	for _, language := range dayLanguages {
		fastDay := dayPatterns(t, domain.DtoDayRequest{
			Date:     "2027-08-12",
			Language: language,
			History: &domain.DtoFastingHistory{
				FastingDates:  []string{"2027-08-10"},
				ObservesDawud: true,
			},
		})[domain.PatternDawud]
		restDay := dayPatterns(t, domain.DtoDayRequest{
			Date:     "2027-08-12",
			Language: language,
			History: &domain.DtoFastingHistory{
				FastingDates:  []string{"2027-08-11"},
				ObservesDawud: true,
			},
		})[domain.PatternDawud]

		isAlternating := fastDay.Status == domain.PatternStatusFastDay &&
			restDay.Status == domain.PatternStatusRestDay &&
			fastDay.Caveat != ""
		if !isAlternating {
			t.Errorf("%s dawud = %+v / %+v, want fast_day / rest_day with a caveat", language, fastDay, restDay)
		}
	}

	patterns := dayPatterns(t, domain.DtoDayRequest{
		Date: "2027-08-12",
		History: &domain.DtoFastingHistory{
			FastingDates: []string{"2027-08-10"},
		},
	})
	if _, ok := patterns[domain.PatternDawud]; ok {
		t.Error("dawud reported without observes_dawud")
	}
}

func TestDaySixShawwalProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fastingDates []string
		wantStatus   string
		wantRecorded int
	}{
		// 2027-03-15 is 6 Syawal 1448. Idulfitri (2027-03-10) and Zulkaidah (2027-04-10) never count.
		{
			name: "in progress",
			fastingDates: []string{
				"2027-03-10",
				"2027-03-11",
				"2027-03-12",
				"2027-03-13",
				"2027-03-14",
				"2027-04-10",
			},
			wantStatus:   domain.PatternStatusInProgress,
			wantRecorded: 4,
		},
		{
			name: "complete",
			fastingDates: []string{
				"2027-03-11",
				"2027-03-12",
				"2027-03-13",
				"2027-03-14",
				"2027-03-16",
				"2027-03-17",
			},
			wantStatus:   domain.PatternStatusComplete,
			wantRecorded: 6,
		},
		{
			name:         "requested day excluded",
			fastingDates: []string{"2027-03-15"},
			wantStatus:   domain.PatternStatusInProgress,
			wantRecorded: 0,
		},
	}

	for _, tt := range tests {
		for _, language := range dayLanguages {
			pattern := dayPatterns(t, domain.DtoDayRequest{
				Date:     "2027-03-15",
				Language: language,
				History: &domain.DtoFastingHistory{
					FastingDates: tt.fastingDates,
				},
			})[domain.PatternSixShawwal]

			// Kemenag's Shawwal 1448 has 29 days, so 6 Syawal leaves 24 including itself.
			isExpected := pattern.Status == tt.wantStatus &&
				pattern.Progress != nil &&
				pattern.Progress.Recorded == tt.wantRecorded &&
				pattern.Progress.Target == shawwalTarget &&
				pattern.Progress.EligibleDaysLeft == 24
			if !isExpected {
				t.Errorf("%s/%s six_shawwal_candidate = %+v, want %s with %d recorded and 24 days left", tt.name, language, pattern, tt.wantStatus, tt.wantRecorded)
			}
		}
	}

	patterns := dayPatterns(t, domain.DtoDayRequest{
		Date: "2027-04-10",
		History: &domain.DtoFastingHistory{
			FastingDates: []string{"2027-03-11"},
		},
	})
	if _, ok := patterns[domain.PatternSixShawwal]; ok {
		t.Error("six Shawwal reported outside Syawal")
	}
}

func TestDayRestrictionsSuppressPatterns(t *testing.T) {
	t.Parallel()

	history := domain.DtoFastingHistory{
		FastingDates:  []string{"2027-02-10"},
		ObservesDawud: true,
	}
	tests := map[string]domain.DtoDayRequest{
		// A Friday in Ramadan: obligatory, so no voluntary pattern applies.
		"Ramadan Friday": {
			Date:    "2027-02-12",
			History: &history,
		},
		"Idulfitri": {
			Date:    "2027-03-10",
			History: &history,
		},
		"Tasyrik": {
			Date:    "2027-05-19",
			History: &history,
		},
		"pilgrim Arafah": {
			Date:              "2027-05-16",
			IsPilgrimAtArafah: true,
			History:           &history,
		},
	}

	for name, request := range tests {
		for _, language := range dayLanguages {
			request.Language = language
			if patterns := dayPatterns(t, request); len(patterns) != 0 {
				t.Errorf("%s/%s patterns = %+v, want none", name, language, patterns)
			}
		}
	}
}
