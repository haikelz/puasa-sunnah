package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

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

	tests := map[string]domain.DtoDayRequest{
		"malformed date": {
			Date: "2027-13-01",
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
