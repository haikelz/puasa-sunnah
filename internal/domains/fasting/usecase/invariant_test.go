package usecase

import (
	"context"
	"testing"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
	fastingRepository "puasa-sunnah/internal/domains/fasting/repository"
)

// TestInvariantsAcrossCoverage walks every date each calendar covers and checks
// the rules no data or rule change may break: Hijri dates advance one day at a
// time through 29- or 30-day months, restrictions always suppress
// recommendations, a pilgrim at Arafah never gets the Arafah fast, and
// restricted days never carry personal patterns.
func TestInvariantsAcrossCoverage(t *testing.T) {
	t.Parallel()

	kemenagCalendar, err := fastingRepository.NewKemenagCalendar()
	if err != nil {
		t.Fatalf("NewKemenagCalendar() error = %v", err)
	}
	calendars := map[domain.Calendar]domain.CalendarRepository{
		domain.CalendarIndonesia: kemenagCalendar,
		domain.CalendarUmmAlQura: fastingRepository.NewUmmAlQuraCalendar(),
	}

	for calendarID, calendar := range calendars {
		t.Run(string(calendarID), func(t *testing.T) {
			t.Parallel()

			assertCoverageInvariants(t, calendarID, calendar)
		})
	}
}

func assertCoverageInvariants(
	t *testing.T,
	calendarID domain.Calendar,
	calendar domain.CalendarRepository,
) {
	t.Helper()

	ctx := context.Background()
	service := newCalendarUsecase(t)
	source := calendar.Source(time.Now(), time.Now())
	start, err := parseDate(source.CoverageStart, "coverage_start")
	if err != nil {
		t.Fatalf("coverage start %q: %v", source.CoverageStart, err)
	}
	end, err := parseDate(source.CoverageEnd, "coverage_end")
	if err != nil {
		t.Fatalf("coverage end %q: %v", source.CoverageEnd, err)
	}

	var previous domain.HijriDate
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		day := date.Format(time.DateOnly)
		hijri, err := calendar.HijriDate(ctx, date)
		if err != nil {
			t.Fatalf("%s HijriDate() error = %v", day, err)
		}

		isBrokenSequence := date.After(start) &&
			!followsHijri(previous, hijri) &&
			!isKnownSourceAnomaly(calendarID, day)
		if isBrokenSequence {
			t.Fatalf("%s is %+v, which does not follow %+v", day, hijri, previous)
		}
		previous = hijri

		for _, isPilgrim := range []bool{false, true} {
			classification, err := service.Classify(ctx, domain.ClassificationRequest{
				Date:              day,
				Hijri:             hijri,
				IsPilgrimAtArafah: isPilgrim,
				Language:          domain.LanguageEnglish,
			})
			if err != nil {
				t.Fatalf("%s Classify() error = %v", day, err)
			}

			assertClassificationInvariants(t, day, classification, isPilgrim)
		}

		if !isRestricted(hijri) {
			continue
		}

		res, err := service.Day(ctx, domain.DtoDayRequest{
			Date:     day,
			Calendar: calendarID,
			History: &domain.DtoFastingHistory{
				FastingDates:  []string{date.AddDate(0, 0, -2).Format(time.DateOnly)},
				ObservesDawud: true,
			},
		})
		if err != nil {
			t.Fatalf("%s Day() error = %v", day, err)
		}
		if len(res.Patterns) != 0 {
			t.Fatalf("%s is restricted but returned patterns %+v", day, res.Patterns)
		}
	}
}

func assertClassificationInvariants(
	t *testing.T,
	day string,
	classification domain.Classification,
	isPilgrim bool,
) {
	t.Helper()

	hijri := classification.Hijri
	if isRestricted(hijri) {
		isSuppressed := len(classification.Restrictions) > 0 &&
			len(classification.Recommendations) == 0 &&
			classification.Verdict != domain.VerdictRecommended
		if !isSuppressed {
			t.Fatalf("%s (%+v) is restricted but classified as %+v", day, hijri, classification)
		}

		return
	}

	if len(classification.Restrictions) != 0 {
		t.Fatalf("%s (%+v) has unexpected restrictions %+v", day, hijri, classification.Restrictions)
	}
	if !isPilgrim || hijri.Month != 12 || hijri.Day != 9 {
		return
	}
	for _, recommendation := range classification.Recommendations {
		if recommendation.Key == "arafah" {
			t.Fatalf("%s recommends the Arafah fast to a pilgrim at Arafah", day)
		}
	}
	if classification.Verdict == domain.VerdictRecommended {
		t.Fatalf("%s gives a pilgrim at Arafah the verdict %s", day, classification.Verdict)
	}
}

// isKnownSourceAnomaly allows the one month the source data itself gets wrong.
// Van Gent's Umm al-Qura table, which go-hijri copies, starts Sha'ban 1364 on
// 1945-07-11 and Ramadan on 1945-08-08, 28 days later (islamcalendar_dat.js
// entries 31648 and 31676). Allow that transition instead of hiding it.
func isKnownSourceAnomaly(calendarID domain.Calendar, day string) bool {
	return calendarID == domain.CalendarUmmAlQura && day == "1945-08-08"
}

// isRestricted mirrors the precedence rules: Ramadan, Idulfitri, Iduladha, and
// the days of Tasyrik.
func isRestricted(hijri domain.HijriDate) bool {
	return hijri.Month == 9 ||
		hijri.Month == 10 && hijri.Day == 1 ||
		hijri.Month == 12 && hijri.Day >= 10 && hijri.Day <= 13
}

// followsHijri reports whether next is the day after previous: the same month
// one day later, or day 1 of the next month after a 29- or 30-day month.
func followsHijri(previous, next domain.HijriDate) bool {
	if next.Day < 1 || next.Day > 30 {
		return false
	}
	if next.Year == previous.Year && next.Month == previous.Month {
		return next.Day == previous.Day+1
	}

	wantYear, wantMonth := previous.Year, previous.Month+1
	if wantMonth > 12 {
		wantYear, wantMonth = wantYear+1, 1
	}

	return next.Year == wantYear &&
		next.Month == wantMonth &&
		next.Day == 1 &&
		(previous.Day == 29 || previous.Day == 30)
}
