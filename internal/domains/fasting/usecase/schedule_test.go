package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
	fastingRepository "puasa-sunnah/internal/domains/fasting/repository"
)

func TestScheduleWeek(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period: domain.SchedulePeriodWeek,
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
	if got.Calendar != domain.CalendarIndonesia {
		t.Errorf("Calendar = %q, want id", got.Calendar)
	}
	assertScheduleDates(t, got.Dates, []string{"2026-08-03", "2026-08-06"})
}

func TestScheduleMonthIncludesWhiteDaysAndOverlaps(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period: domain.SchedulePeriodMonth,
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

	got, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period: domain.SchedulePeriodYear,
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

	// Arafah 9 Zulhijah 1448, Ashura 10 Muharam 1449, Ayyamul Bidh 13 Syaban 1448.
	recommended := []string{
		"2027-05-16",
		"2027-06-15",
		"2027-01-21",
	}
	// Ramadan Monday, Idulfitri, Iduladha, and Tasyrik.
	prohibited := []string{
		"2027-02-08",
		"2027-03-10",
		"2027-05-17",
		"2027-05-18",
		"2027-05-19",
		"2027-05-20",
	}

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		got, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
			Period:   domain.SchedulePeriodYear,
			Date:     "2027-08-09",
			Language: language,
		})
		if err != nil {
			t.Fatalf("%s Schedule() error = %v", language, err)
		}

		if got.CalendarSource.Publications[0].ID != "kemenag-ri-2027" {
			t.Errorf("%s publications = %+v, want kemenag-ri-2027", language, got.CalendarSource.Publications)
		}
		for _, date := range recommended {
			if !hasScheduleDate(got.Dates, date) {
				t.Errorf("%s 2027 schedule is missing %s", language, date)
			}
		}
		for _, date := range prohibited {
			if hasScheduleDate(got.Dates, date) {
				t.Errorf("%s 2027 schedule contains %s", language, date)
			}
		}
	}

	week, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period: domain.SchedulePeriodWeek,
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
		calendar        domain.Calendar
		date            string
		wantCoverageEnd string
	}{
		{
			calendar:        domain.CalendarIndonesia,
			date:            "2028-08-09",
			wantCoverageEnd: "2027-12-31",
		},
		{
			calendar:        domain.CalendarUmmAlQura,
			date:            "2100-08-09",
			wantCoverageEnd: "2077-11-16",
		},
	}

	for _, tt := range tests {
		_, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
			Period:   domain.SchedulePeriodYear,
			Date:     tt.date,
			Calendar: tt.calendar,
		})

		var requestError *domain.RequestError
		isCoverageError := errors.Is(err, domain.ErrCalendarUnavailable) &&
			errors.As(err, &requestError) &&
			requestError.Code == domain.CodeOutsideCoverage &&
			requestError.Calendar == tt.calendar &&
			requestError.CoverageEnd == tt.wantCoverageEnd
		if !isCoverageError {
			t.Errorf("%s Schedule() error = %#v, want coverage error ending %s", tt.calendar, err, tt.wantCoverageEnd)
		}
	}
}

func TestScheduleUmmAlQuraSupports2027(t *testing.T) {
	t.Parallel()

	got, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period:   domain.SchedulePeriodYear,
		Date:     "2027-08-09",
		Calendar: domain.CalendarUmmAlQura,
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if got.Calendar != domain.CalendarUmmAlQura || got.CalendarSource.ID != "umm-al-qura" {
		t.Errorf("calendar/source = %q/%q, want umm-al-qura/umm-al-qura", got.Calendar, got.CalendarSource.ID)
	}
	if got.Total == 0 {
		t.Error("Umm al-Qura year schedule is empty")
	}
}

func TestScheduleRejectsUnsupportedCalendar(t *testing.T) {
	t.Parallel()

	_, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period:   domain.SchedulePeriodMonth,
		Date:     "2026-08-09",
		Calendar: "other",
	})
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("Schedule() error = %v, want ErrInvalidRequest", err)
	}
}

func newCalendarUsecase(t *testing.T) *FastingUsecase {
	t.Helper()

	kemenagCalendar, err := fastingRepository.NewKemenagCalendar()
	if err != nil {
		t.Fatalf("NewKemenagCalendar() error = %v", err)
	}

	service, err := NewFastingUsecase(kemenagCalendar, fastingRepository.NewUmmAlQuraCalendar())
	if err != nil {
		t.Fatalf("NewFastingUsecase() error = %v", err)
	}

	return service
}

func TestScheduleTypeFilter(t *testing.T) {
	t.Parallel()

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		service := newCalendarUsecase(t)
		request := domain.ScheduleRequest{Period: domain.SchedulePeriodMonth, Date: "2026-08-09", Types: []string{"white_days", "white_days"}, Language: language}
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
		if _, err := service.Schedule(context.Background(), request); !errors.Is(err, domain.ErrInvalidRequest) {
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

func assertScheduleDates(t *testing.T, dates []domain.ScheduleDate, want []string) {
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

func hasScheduleDate(dates []domain.ScheduleDate, wanted string) bool {
	for _, date := range dates {
		if date.Date == wanted {
			return true
		}
	}
	return false
}

func TestScheduleRange(t *testing.T) {
	t.Parallel()

	days := func(n int) *int { return &n }
	tests := []struct {
		name      string
		request   domain.ScheduleRequest
		wantStart string
		wantEnd   string
	}{
		{
			name: "days",
			request: domain.ScheduleRequest{
				Start: "2027-03-01",
				Days:  days(10),
			},
			wantStart: "2027-03-01",
			wantEnd:   "2027-03-10",
		},
		{
			name: "end",
			request: domain.ScheduleRequest{
				Start: "2027-03-01",
				End:   "2027-03-31",
			},
			wantStart: "2027-03-01",
			wantEnd:   "2027-03-31",
		},
		{
			name: "one day",
			request: domain.ScheduleRequest{
				Start: "2027-10-15",
				End:   "2027-10-15",
			},
			wantStart: "2027-10-15",
			wantEnd:   "2027-10-15",
		},
		{
			name: "full span",
			request: domain.ScheduleRequest{
				Start: "2027-01-01",
				Days:  days(maxRangeDays),
			},
			wantStart: "2027-01-01",
			wantEnd:   "2028-01-01",
		},
	}

	for _, tt := range tests {
		tt.request.Period = domain.SchedulePeriodRange
		tt.request.Calendar = domain.CalendarUmmAlQura

		got, err := newCalendarUsecase(t).Schedule(context.Background(), tt.request)
		if err != nil {
			t.Fatalf("%s: Schedule() error = %v", tt.name, err)
		}

		isExpected := got.Period == domain.SchedulePeriodRange &&
			got.StartDate == tt.wantStart &&
			got.EndDate == tt.wantEnd &&
			got.AnchorDate == tt.wantStart &&
			got.WeekStartsOn == ""
		if !isExpected {
			t.Errorf("%s: period/range/anchor = %s %s..%s %s", tt.name, got.Period, got.StartDate, got.EndDate, got.AnchorDate)
		}
	}

	rolling, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period:   domain.SchedulePeriodRange,
		Days:     days(30),
		Calendar: domain.CalendarUmmAlQura,
	})
	if err != nil {
		t.Fatalf("rolling Schedule() error = %v", err)
	}

	if today := time.Now().In(jakarta).Format(time.DateOnly); rolling.StartDate != today {
		t.Errorf("rolling start = %s, want today %s", rolling.StartDate, today)
	}
}

func TestScheduleRangeRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	days := func(n int) *int { return &n }
	tests := map[string]domain.ScheduleRequest{
		"neither end nor days": {
			Period: domain.SchedulePeriodRange,
			Start:  "2027-03-01",
		},
		"end and days": {
			Period: domain.SchedulePeriodRange,
			Start:  "2027-03-01",
			End:    "2027-03-02",
			Days:   days(2),
		},
		"end before start": {
			Period: domain.SchedulePeriodRange,
			Start:  "2027-03-02",
			End:    "2027-03-01",
		},
		"too long by end": {
			Period: domain.SchedulePeriodRange,
			Start:  "2027-01-01",
			End:    "2028-01-02",
		},
		"zero days": {
			Period: domain.SchedulePeriodRange,
			Days:   days(0),
		},
		"too many days": {
			Period: domain.SchedulePeriodRange,
			Days:   days(maxRangeDays + 1),
		},
		"date with range": {
			Period: domain.SchedulePeriodRange,
			Date:   "2027-03-01",
			Days:   days(2),
		},
		"malformed start": {
			Period: domain.SchedulePeriodRange,
			Start:  "1 March 2027",
			Days:   days(2),
		},
		"start with month": {
			Period: domain.SchedulePeriodMonth,
			Start:  "2027-03-01",
		},
		"days with year": {
			Period: domain.SchedulePeriodYear,
			Days:   days(5),
		},
	}

	for name, request := range tests {
		for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
			request.Language = language
			_, err := newCalendarUsecase(t).Schedule(context.Background(), request)
			if !errors.Is(err, domain.ErrInvalidRequest) {
				t.Errorf("%s/%s: error = %v, want ErrInvalidRequest", name, language, err)
			}
		}
	}

	_, err := newCalendarUsecase(t).Schedule(context.Background(), domain.ScheduleRequest{
		Period: domain.SchedulePeriodRange,
		Start:  "2027-12-20",
		Days:   days(30),
	})
	if !errors.Is(err, domain.ErrCalendarUnavailable) {
		t.Errorf("range past Kemenag coverage error = %v, want ErrCalendarUnavailable", err)
	}
}
