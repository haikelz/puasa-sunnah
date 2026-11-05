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
	if isDawudActive {
		note, err := u.dawudGapNote(ctx, calendar, history, window.start, request.Language)
		if err != nil {
			return domain.PatternPlan{}, err
		}
		if note != "" {
			isDawudActive = false
			result.Notes = append(result.Notes, note)
		}
	}
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

// dawudGapNote checks the days between the caller's last recorded fast and
// start. A Ramadan or prohibited day there interrupted the alternation, and
// scholars differ on how to resume, so the plan must not resume on its own. It
// returns "" when nothing interrupts or the caller sent no fast before start.
func (u *FastingUsecase) dawudGapNote(
	ctx context.Context,
	calendar domain.CalendarRepository,
	history *fastingHistory,
	start time.Time,
	language domain.Language,
) (string, error) {
	lastFast := ""
	for date := range history.dates {
		if date < start.Format(time.DateOnly) && date > lastFast {
			lastFast = date
		}
	}
	if lastFast == "" {
		return "", nil
	}

	from, err := parseDate(lastFast, "fasting_dates")
	if err != nil {
		return "", err
	}
	// Restricted days lie at most about 255 days apart (14 Dhu al-Hijjah to
	// 1 Ramadan), so this loop stays short even for a year-old fast.
	for current := from.AddDate(0, 0, 1); current.Before(start); current = current.AddDate(0, 0, 1) {
		day := current.Format(time.DateOnly)
		hijri, err := calendar.HijriDate(ctx, current)
		if errors.Is(err, domain.ErrDateOutsideCalendar) {
			return text(
				language,
				"The Dawud projection does not start: the calendar does not cover "+day+", between your last recorded fast and start, so it cannot rule out Ramadan or a prohibited day. Record a recent fast, or ask a qualified scholar.",
				"Proyeksi puasa Dawud tidak dimulai: kalender tidak mencakup "+day+", di antara puasa terakhir yang Anda catat dan start, sehingga tidak dapat memastikan tidak ada Ramadan atau hari terlarang. Catat puasa terbaru, atau tanyakan kepada ulama yang kompeten.",
			), nil
		}
		if err != nil {
			return "", err
		}

		classification, err := u.Classify(ctx, domain.ClassificationRequest{
			Date:     day,
			Hijri:    hijri,
			Language: language,
		})
		if err != nil {
			return "", err
		}
		if len(classification.Restrictions) > 0 {
			return text(
				language,
				"The Dawud projection does not start: Ramadan or a prohibited day on "+day+" came after your last recorded fast. Scholars differ on how to resume; record your first fast after it, or ask a qualified scholar.",
				"Proyeksi puasa Dawud tidak dimulai: Ramadan atau hari terlarang pada "+day+" terjadi setelah puasa terakhir yang Anda catat. Ulama berbeda pendapat tentang cara melanjutkannya; catat puasa pertama Anda setelahnya, atau tanyakan kepada ulama yang kompeten.",
			), nil
		}
	}

	return "", nil
}

func dawudStopNote(language domain.Language, date string) string {
	return text(
		language,
		"The Dawud projection stops on "+date+", where Ramadan or a prohibited day interrupts the alternation. Scholars differ on how to resume; ask a qualified scholar.",
		"Proyeksi puasa Dawud berhenti pada "+date+", saat Ramadan atau hari terlarang menyela pergantian. Ulama berbeda pendapat tentang cara melanjutkannya; tanyakan kepada ulama yang kompeten.",
	)
}
