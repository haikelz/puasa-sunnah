package usecase

import (
	"context"
	"errors"
	"testing"

	"puasa-sunnah/internal/domains/fasting"
	fastingrepository "puasa-sunnah/internal/domains/fasting/repository"
)

func TestScheduleWeek(t *testing.T) {
	t.Parallel()

	got, err := NewFastingUsecase(fastingrepository.NewKemenagCalendar()).Schedule(context.Background(), fasting.ScheduleRequest{
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
	assertScheduleDates(t, got.Dates, []string{"2026-08-03", "2026-08-06"})
}

func TestScheduleMonthIncludesWhiteDaysAndOverlaps(t *testing.T) {
	t.Parallel()

	got, err := NewFastingUsecase(fastingrepository.NewKemenagCalendar()).Schedule(context.Background(), fasting.ScheduleRequest{
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
	if got.CalendarSource.ID != "kemenag-ri-2026" {
		t.Errorf("calendar source = %q, want kemenag-ri-2026", got.CalendarSource.ID)
	}
}

func TestScheduleYearSuppressesRestrictedDates(t *testing.T) {
	t.Parallel()

	got, err := NewFastingUsecase(fastingrepository.NewKemenagCalendar()).Schedule(context.Background(), fasting.ScheduleRequest{
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

func TestScheduleRejectsUnsupportedYear(t *testing.T) {
	t.Parallel()

	_, err := NewFastingUsecase(fastingrepository.NewKemenagCalendar()).Schedule(context.Background(), fasting.ScheduleRequest{
		Period: fasting.SchedulePeriodYear,
		Date:   "2027-08-09",
	})
	if !errors.Is(err, ErrCalendarUnavailable) {
		t.Fatalf("Schedule() error = %v, want ErrCalendarUnavailable", err)
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
