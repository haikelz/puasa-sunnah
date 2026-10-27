package usecase

import (
	"context"
	"errors"
	"maps"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

// Patterns evaluates the personal patterns for every date in a range. Six
// Shawwal and Friday read only the dates the caller sent. Dawud is projected:
// each projected fast day makes the next day a rest day. The projection stops
// at the first Ramadan or prohibited day, because scholars differ on how to
// resume, and a note names that date. Restricted days and a pilgrim's Arafah
// day carry no patterns.
func (u *FastingUsecase) Patterns(
	ctx context.Context,
	request domain.DtoPatternRequest,
) (domain.PatternPlan, error) {
	calendarID, calendar, err := u.scheduleCalendar(request.Calendar)
	if err != nil {
		return domain.PatternPlan{}, err
	}

	window, err := scheduleBounds(domain.ScheduleRequest{
		Period: domain.SchedulePeriodRange,
		Start:  request.Start,
		End:    request.End,
		Days:   request.Days,
	})
	if err != nil {
		return domain.PatternPlan{}, err
	}

	history, err := parseHistory(&request.History)
	if err != nil {
		return domain.PatternPlan{}, err
	}

	source := calendar.Source(window.start, window.end)
	source.Caveat = calendarCaveat(request.Language, calendarID)
	result := domain.PatternPlan{
		Calendar:       calendarID,
		CalendarSource: source,
		StartDate:      window.start.Format(time.DateOnly),
		EndDate:        window.end.Format(time.DateOnly),
		Timezone:       jakarta.String(),
		Dates:          make([]domain.PatternDate, 0),
		Notes:          make([]string, 0),
		Disclaimer:     disclaimer(request.Language),
	}

	// projected holds the caller's dates plus the Dawud fast days this plan adds.
	projected := &fastingHistory{
		dates: maps.Clone(history.dates),
	}
	isDawudActive := history.observesDawud
	for current := window.start; !current.After(window.end); current = current.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return domain.PatternPlan{}, err
		}
		hijri, err := calendar.HijriDate(ctx, current)
		if errors.Is(err, domain.ErrDateOutsideCalendar) {
			return domain.PatternPlan{}, calendarUnavailable(calendarID, source)
		}
		if err != nil {
			return domain.PatternPlan{}, err
		}

		classification, err := u.Classify(ctx, domain.ClassificationRequest{
			Date:              current.Format(time.DateOnly),
			Hijri:             hijri,
			IsPilgrimAtArafah: request.IsPilgrimAtArafah,
			Language:          request.Language,
		})
		if err != nil {
			return domain.PatternPlan{}, err
		}

		if len(classification.Restrictions) > 0 {
			if isDawudActive {
				isDawudActive = false
				result.Notes = append(result.Notes, dawudStopNote(request.Language, classification.Date))
			}
			continue
		}
		isPilgrimOnArafah := request.IsPilgrimAtArafah &&
			hijri.Month == 12 &&
			hijri.Day == 9
		if isPilgrimOnArafah {
			continue
		}

		var dawud *dawudDay
		if isDawudActive {
			dawud = &dawudDay{
				isFastDay:   !projected.has(current.AddDate(0, 0, -1)),
				isProjected: true,
			}
			if dawud.isFastDay {
				projected.dates[classification.Date] = true
			}
		}

		patterns, err := personalPatterns(
			ctx,
			calendar,
			current,
			classification,
			history,
			dawud,
			request.Language,
		)
		if err != nil {
			return domain.PatternPlan{}, err
		}
		if len(patterns) == 0 {
			continue
		}

		result.Dates = append(result.Dates, domain.PatternDate{
			Date:     classification.Date,
			Weekday:  classification.Weekday,
			Hijri:    classification.Hijri,
			Patterns: patterns,
		})
	}

	result.Total = len(result.Dates)
	return result, nil
}

func dawudStopNote(language domain.Language, date string) string {
	return text(
		language,
		"The Dawud projection stops on "+date+", where Ramadan or a prohibited day interrupts the alternation. Scholars differ on how to resume; ask a qualified scholar.",
		"Proyeksi puasa Dawud berhenti pada "+date+", saat Ramadan atau hari terlarang menyela pergantian. Ulama berbeda pendapat tentang cara melanjutkannya; tanyakan kepada ulama yang kompeten.",
	)
}
