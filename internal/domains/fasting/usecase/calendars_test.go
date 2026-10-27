package usecase

import (
	"context"
	"testing"

	domain "puasa-sunnah/internal/domains/fasting"
)

func TestCalendarsListsFullCoverage(t *testing.T) {
	t.Parallel()

	service := newCalendarUsecase(t)

	for _, language := range []domain.Language{domain.LanguageEnglish, domain.LanguageIndonesian} {
		coverages, err := service.Calendars(context.Background(), language)
		if err != nil {
			t.Fatalf("Calendars(%s) error = %v", language, err)
		}
		if len(coverages) != 2 {
			t.Fatalf("Calendars(%s) = %d calendars, want 2", language, len(coverages))
		}

		indonesia, ummAlQura := coverages[0], coverages[1]
		isIndonesiaDefault := indonesia.Calendar == domain.CalendarIndonesia &&
			indonesia.IsDefault &&
			ummAlQura.Calendar == domain.CalendarUmmAlQura &&
			!ummAlQura.IsDefault
		if !isIndonesiaDefault {
			t.Errorf("Calendars(%s) order/default = %s %t, %s %t", language, indonesia.Calendar, indonesia.IsDefault, ummAlQura.Calendar, ummAlQura.IsDefault)
		}

		source := indonesia.CalendarSource
		hasEveryYear := source.CoverageStart == "2025-12-28" &&
			source.CoverageEnd == "2027-12-31" &&
			len(source.Publications) == 2 &&
			len(source.IsbatDecisions) == 3
		if !hasEveryYear {
			t.Errorf("Calendars(%s) id source = %+v", language, source)
		}
		if source.Caveat != calendarCaveat(language, domain.CalendarIndonesia) {
			t.Errorf("Calendars(%s) id caveat = %q", language, source.Caveat)
		}
		if ummAlQura.CalendarSource.Caveat != calendarCaveat(language, domain.CalendarUmmAlQura) {
			t.Errorf("Calendars(%s) umm-al-qura caveat = %q", language, ummAlQura.CalendarSource.Caveat)
		}
	}
}
