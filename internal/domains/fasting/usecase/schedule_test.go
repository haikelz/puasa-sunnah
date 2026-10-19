package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"puasa-sunnah/internal/domains/fasting"
	fastingrepository "puasa-sunnah/internal/domains/fasting/repository"
)

func TestScheduleWeek(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
		Period: fasting.SchedulePeriodWeek,
		Date:   "2026-08-09",
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if got.StartDate != "2026-08-03" || got.EndDate != "2026-08-09" {
		t.Errorf("range = %s..%s, want 2026-08-03..2026-08-09", got.StartDate, got.EndDate)
	}
	if got.WeekStartsOn != "Monday" {
		t.Errorf("WeekStartsOn = %q, want Monday", got.WeekStartsOn)
	}
	if got.Calendar != fasting.CalendarIndonesia {
		t.Errorf("Calendar = %q, want id", got.Calendar)
	}
	assertScheduleDates(t, got.Dates, []string{"2026-08-03", "2026-08-06"})
}

func TestScheduleMonthIncludesWhiteDaysAndOverlaps(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
		Period: fasting.SchedulePeriodMonth,
		Date:   "2026-08-09",
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	assertScheduleDates(t, got.Dates, []string{
		"2026-08-03", "2026-08-06", "2026-08-10", "2026-08-13", "2026-08-17", "2026-08-20",
		"2026-08-24", "2026-08-26", "2026-08-27", "2026-08-28", "2026-08-31",
	})
	if len(got.Dates[8].Recommendations) != 2 {
		t.Errorf("August 27 recommendation count = %d, want Monday/Thursday plus White Days", len(got.Dates[8].Recommendations))
	}
	if got.CalendarSource.ID != "kemenag-ri" || len(got.CalendarSource.Publications) != 1 || got.CalendarSource.Publications[0].ID != "kemenag-ri-2026" {
		t.Errorf("calendar source = %q %+v, want kemenag-ri with the 2026 publication", got.CalendarSource.ID, got.CalendarSource.Publications)
	}
}

func TestScheduleYearSuppressesRestrictedDates(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
		Period: fasting.SchedulePeriodYear,
		Date:   "2026-08-09",
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if !hasScheduleDate(got.Dates, "2026-05-26") {
		t.Error("year schedule does not contain Arafah")
	}
	for _, prohibited := range []string{"2026-03-21", "2026-05-27", "2026-05-28", "2026-05-29", "2026-05-30"} {
		if hasScheduleDate(got.Dates, prohibited) {
			t.Errorf("year schedule contains prohibited date %s", prohibited)
		}
	}
}

func TestScheduleIndonesia2027(t *testing.T) {
	t.Parallel()

	for _, language := range []fasting.Language{fasting.LanguageEnglish, fasting.LanguageIndonesian} {
		got, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
			Period:   fasting.SchedulePeriodYear,
			Date:     "2027-08-09",
			Language: language,
		})
		if err != nil {
			t.Fatalf("%s Schedule() error = %v", language, err)
		}
		if got.CalendarSource.Publications[0].ID != "kemenag-ri-2027" {
			t.Errorf("%s publications = %+v, want kemenag-ri-2027", language, got.CalendarSource.Publications)
		}
		// Arafah 9 Zulhijah 1448, Ashura 10 Muharam 1449, Ayyamul Bidh 13 Syaban 1448.
		for _, recommended := range []string{"2027-05-16", "2027-06-15", "2027-01-21"} {
			if !hasScheduleDate(got.Dates, recommended) {
				t.Errorf("%s 2027 schedule is missing %s", language, recommended)
			}
		}
		// Ramadan Monday, Idulfitri, Iduladha, and Tasyrik.
		for _, prohibited := range []string{"2027-02-08", "2027-03-10", "2027-05-17", "2027-05-18", "2027-05-19", "2027-05-20"} {
			if hasScheduleDate(got.Dates, prohibited) {
				t.Errorf("%s 2027 schedule contains %s", language, prohibited)
			}
		}
	}

	week, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
		Period: fasting.SchedulePeriodWeek,
		Date:   "2026-12-31",
	})
	if err != nil {
		t.Fatalf("cross-year week Schedule() error = %v", err)
	}
	if len(week.CalendarSource.Publications) != 2 {
		t.Errorf("cross-year week publications = %+v, want 2026 and 2027", week.CalendarSource.Publications)
	}
}

func TestScheduleRejectsUnsupportedYear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		language fasting.Language
		calendar fasting.Calendar
		date     string
		wantHint string
	}{
		{fasting.LanguageEnglish, fasting.CalendarIndonesia, "2028-08-09", "for other dates, calendar=umm-al-qura"},
		{fasting.LanguageIndonesian, fasting.CalendarIndonesia, "2028-08-09", "untuk tanggal lain, calendar=umm-al-qura"},
		{fasting.LanguageEnglish, fasting.CalendarUmmAlQura, "2100-08-09", ""},
	}
	for _, tt := range tests {
		_, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
			Period:   fasting.SchedulePeriodYear,
			Date:     tt.date,
			Language: tt.language,
			Calendar: tt.calendar,
		})
		if !errors.Is(err, fasting.ErrCalendarUnavailable) {
			t.Fatalf("%s/%s Schedule() error = %v, want ErrCalendarUnavailable", tt.language, tt.calendar, err)
		}
		hasHint := strings.Contains(err.Error(), "umm-al-qura")
		if tt.wantHint == "" && hasHint || tt.wantHint != "" && !strings.Contains(err.Error(), tt.wantHint) {
			t.Errorf("%s/%s error = %q, want hint %q", tt.language, tt.calendar, err, tt.wantHint)
		}
	}
}

func TestScheduleUmmAlQuraSupports2027(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
		Period:   fasting.SchedulePeriodYear,
		Date:     "2027-08-09",
		Calendar: fasting.CalendarUmmAlQura,
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if got.Calendar != fasting.CalendarUmmAlQura || got.CalendarSource.ID != "umm-al-qura" {
		t.Errorf("calendar/source = %q/%q, want umm-al-qura/umm-al-qura", got.Calendar, got.CalendarSource.ID)
	}
	if got.Total == 0 {
		t.Error("Umm al-Qura year schedule is empty")
	}
}

func TestScheduleRejectsUnsupportedCalendar(t *testing.T) {
	t.Parallel()

	_, err := newCalendarUsecase(t).Schedule(context.Background(), fasting.ScheduleRequest{
		Period:   fasting.SchedulePeriodMonth,
		Date:     "2026-08-09",
		Calendar: "other",
	})
	if !errors.Is(err, fasting.ErrInvalidRequest) {
		t.Fatalf("Schedule() error = %v, want ErrInvalidRequest", err)
	}
}

func newCalendarUsecase(t *testing.T) *FastingUsecase {
	t.Helper()
	kemenagCalendar, err := fastingrepository.NewKemenagCalendar()
	if err != nil {
		t.Fatalf("NewKemenagCalendar() error = %v", err)
	}
	service, err := NewFastingUsecase(kemenagCalendar, fastingrepository.NewUmmAlQuraCalendar())
	if err != nil {
		t.Fatalf("NewFastingUsecase() error = %v", err)
	}
	return service
}

func TestScheduleTypeFilter(t *testing.T) {
	t.Parallel()

	for _, language := range []fasting.Language{fasting.LanguageEnglish, fasting.LanguageIndonesian} {
		service := newCalendarUsecase(t)
		request := fasting.ScheduleRequest{Period: fasting.SchedulePeriodMonth, Date: "2026-08-09", Types: []string{"white_days", "white_days"}, Language: language}
		got, err := service.Schedule(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		assertScheduleDates(t, got.Dates, []string{"2026-08-26", "2026-08-27", "2026-08-28"})
		for _, date := range got.Dates {
			assertKeys(t, date.Recommendations, []string{"white_days"})
		}

		request.Types = []string{"white_days", "monday_thursday"}
		got, err = service.Schedule(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if got.Total != 11 {
			t.Fatalf("OR total = %d, want 11", got.Total)
		}

		request.Types = []string{"ashura"}
		got, err = service.Schedule(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if got.Total != 0 || got.Dates == nil {
			t.Fatalf("empty filter result = %+v", got)
		}

		request.Types = []string{"white_days", "invalid"}
		if _, err := service.Schedule(context.Background(), request); !errors.Is(err, fasting.ErrInvalidRequest) {
			t.Fatalf("invalid types error = %v", err)
		}

		request.Date = "2026-05-09"
		request.Types = []string{"white_days"}
		got, err = service.Schedule(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if hasScheduleDate(got.Dates, "2026-05-30") {
			t.Fatal("filter bypassed Tashriq restriction")
		}
	}
}

func assertScheduleDates(t *testing.T, dates []fasting.ScheduleDate, want []string) {
	t.Helper()
	if len(dates) != len(want) {
		t.Fatalf("schedule date count = %d, want %d: %#v", len(dates), len(want), dates)
	}
	for i := range want {
		if dates[i].Date != want[i] {
			t.Errorf("dates[%d] = %q, want %q", i, dates[i].Date, want[i])
		}
	}
}

func hasScheduleDate(dates []fasting.ScheduleDate, wanted string) bool {
	for _, date := range dates {
		if date.Date == wanted {
			return true
		}
	}
	return false
}
