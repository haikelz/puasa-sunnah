package usecase

import (
	"context"
	"fmt"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

// Calendars lists every selectable calendar with its full coverage, so a client
// can learn the supported range before a request returns 422.
func (u *FastingUsecase) Calendars(
	_ context.Context,
	language domain.Language,
) ([]domain.CalendarCoverage, error) {
	calendarIDs := []domain.Calendar{
		domain.CalendarIndonesia,
		domain.CalendarUmmAlQura,
	}
	coverages := make([]domain.CalendarCoverage, 0, len(calendarIDs))
	for _, calendarID := range calendarIDs {
		_, calendar, err := u.scheduleCalendar(calendarID)
		if err != nil {
			return nil, err
		}

		// Source reports the coverage for any range, so ask once for the
		// bounds and again for everything inside them.
		bounds := calendar.Source(time.Time{}, time.Time{})
		start, err := time.Parse(time.DateOnly, bounds.CoverageStart)
		if err != nil {
			return nil, fmt.Errorf("parse %s coverage start: %w", calendarID, err)
		}
		end, err := time.Parse(time.DateOnly, bounds.CoverageEnd)
		if err != nil {
			return nil, fmt.Errorf("parse %s coverage end: %w", calendarID, err)
		}

		source := calendar.Source(start, end)
		source.Caveat = calendarCaveat(language, calendarID)
		coverages = append(coverages, domain.CalendarCoverage{
			Calendar:       calendarID,
			IsDefault:      calendarID == defaultCalendar,
			CalendarSource: source,
		})
	}

	return coverages, nil
}
