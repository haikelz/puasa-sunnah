package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

const (
	// maxHistoryDates bounds caller history; only dates near the requested day matter.
	maxHistoryDates = 366
	// shawwalTarget is the number of Shawwal days in the six-day pattern.
	shawwalTarget = 6
)

// Day classifies one Gregorian date with a Hijri date derived from the
// caller-selected calendar. Personal patterns run only when the caller sends
// history, and never on restricted days.
func (u *FastingUsecase) Day(ctx context.Context, request domain.DtoDayRequest) (domain.Day, error) {
	calendarID, calendar, err := u.scheduleCalendar(request.Calendar)
	if err != nil {
		return domain.Day{}, err
	}

	dateValue := request.Date
	if dateValue == domain.DateToday {
		dateValue = ""
	}
	date, err := scheduleAnchor(dateValue, "date")
	if err != nil {
		return domain.Day{}, err
	}

	history, err := parseHistory(request.History)
	if err != nil {
		return domain.Day{}, err
	}

	source := calendar.Source(date, date)
	source.Caveat = calendarCaveat(request.Language, calendarID)
	hijri, err := calendar.HijriDate(ctx, date)
	if errors.Is(err, domain.ErrDateOutsideCalendar) {
		return domain.Day{}, calendarUnavailable(calendarID, source)
	}
	if err != nil {
		return domain.Day{}, err
	}

	classification, err := u.Classify(ctx, domain.ClassificationRequest{
		Date:              date.Format(time.DateOnly),
		Hijri:             hijri,
		IsPilgrimAtArafah: request.IsPilgrimAtArafah,
		Language:          request.Language,
	})
	if err != nil {
		return domain.Day{}, err
	}

	result := domain.Day{
		Classification: classification,
		Calendar:       calendarID,
		CalendarSource: source,
		Patterns:       make([]domain.PatternStatus, 0),
	}
	isPilgrimOnArafah := request.IsPilgrimAtArafah &&
		hijri.Month == 12 &&
		hijri.Day == 9
	if history == nil || len(classification.Restrictions) > 0 || isPilgrimOnArafah {
		return result, nil
	}

	result.Patterns, err = personalPatterns(
		ctx,
		calendar,
		date,
		classification,
		history,
		request.Language,
	)
	if err != nil {
		return domain.Day{}, err
	}

	return result, nil
}

type fastingHistory struct {
	dates         map[string]bool
	observesDawud bool
}

func (h *fastingHistory) has(date time.Time) bool {
	return h.dates[date.Format(time.DateOnly)]
}

func parseHistory(history *domain.DtoFastingHistory) (*fastingHistory, error) {
	if history == nil {
		return nil, nil
	}
	if len(history.FastingDates) > maxHistoryDates {
		return nil, invalidRequest(domain.CodeHistoryTooLong, "fasting_dates")
	}

	parsed := &fastingHistory{
		dates:         make(map[string]bool, len(history.FastingDates)),
		observesDawud: history.ObservesDawud,
	}
	for _, value := range history.FastingDates {
		date, err := parseDate(value, "fasting_dates")
		if err != nil {
			return nil, err
		}

		parsed.dates[date.Format(time.DateOnly)] = true
	}

	return parsed, nil
}

// personalPatterns follows the /rules order: six Shawwal, Dawud, Friday alone.
func personalPatterns(
	ctx context.Context,
	calendar domain.CalendarRepository,
	date time.Time,
	classification domain.Classification,
	history *fastingHistory,
	language domain.Language,
) ([]domain.PatternStatus, error) {
	patterns := make([]domain.PatternStatus, 0, 3)

	if classification.Hijri.Month == 10 && classification.Hijri.Day >= 2 {
		shawwal, err := sixShawwalStatus(
			ctx,
			calendar,
			date,
			classification.Hijri,
			history,
			language,
		)
		if err != nil {
			return nil, err
		}

		patterns = append(patterns, shawwal)
	}

	isDawudFastDay := false
	if history.observesDawud {
		isDawudFastDay = !history.has(date.AddDate(0, 0, -1))
		patterns = append(patterns, dawudStatus(isDawudFastDay, language))
	}

	if date.Weekday() == time.Friday {
		patterns = append(patterns, fridayStatus(
			date,
			classification,
			history,
			isDawudFastDay,
			language,
		))
	}

	return patterns, nil
}

func sixShawwalStatus(
	ctx context.Context,
	calendar domain.CalendarRepository,
	date time.Time,
	hijri domain.HijriDate,
	history *fastingHistory,
	language domain.Language,
) (domain.PatternStatus, error) {
	// A Hijri month spans at most 30 days, so only nearby dates can share it.
	recorded := 0
	for offset := -30; offset <= 30; offset++ {
		candidate := date.AddDate(0, 0, offset)
		if offset == 0 || !history.has(candidate) {
			continue
		}

		candidateHijri, err := calendar.HijriDate(ctx, candidate)
		if errors.Is(err, domain.ErrDateOutsideCalendar) {
			// Coverage boundaries never split a Shawwal, so an uncovered date lies in another month.
			continue
		}
		if err != nil {
			return domain.PatternStatus{}, err
		}

		isSameShawwal := candidateHijri.Year == hijri.Year &&
			candidateHijri.Month == 10 &&
			candidateHijri.Day >= 2
		if isSameShawwal {
			recorded++
		}
	}

	status := domain.PatternStatus{
		Key:    domain.PatternSixShawwal,
		Name:   text(language, "Six days of Shawwal", "Enam hari puasa Syawal"),
		Status: domain.PatternStatusInProgress,
		Message: text(
			language,
			fmt.Sprintf("You recorded %d of 6 Shawwal days; fasting today makes %d.", recorded, recorded+1),
			fmt.Sprintf("Anda mencatat %d dari 6 hari puasa Syawal; berpuasa hari ini menjadikannya %d.", recorded, recorded+1),
		),
		Evidence: "Sahih Muslim 1164a",
		Caveat: text(
			language,
			"Schools differ on details such as consecutiveness and combining intentions.",
			"Terdapat perbedaan pendapat mengenai rincian seperti pelaksanaan berturut-turut dan penggabungan niat.",
		),
		Progress: &domain.PatternProgress{
			Recorded: recorded,
			Target:   shawwalTarget,
		},
	}
	if recorded >= shawwalTarget {
		status.Status = domain.PatternStatusComplete
		status.Message = text(
			language,
			fmt.Sprintf("You recorded %d Shawwal days; the six-day pattern is complete.", recorded),
			fmt.Sprintf("Anda mencatat %d hari puasa Syawal; pola enam hari sudah lengkap.", recorded),
		)
	}

	return status, nil
}

func dawudStatus(isFastDay bool, language domain.Language) domain.PatternStatus {
	status := domain.PatternStatus{
		Key:    domain.PatternDawud,
		Name:   text(language, "Fast of Prophet Dawud", "Puasa Nabi Dawud"),
		Status: domain.PatternStatusFastDay,
		Message: text(
			language,
			"You recorded no fast yesterday, so the alternation falls on today.",
			"Anda tidak mencatat puasa kemarin, sehingga giliran berpuasa jatuh hari ini.",
		),
		Evidence: "Sahih al-Bukhari 1976",
		Caveat: text(
			language,
			"When Ramadan or a prohibited day interrupts the alternation, scholars differ on how to resume; ask a qualified scholar.",
			"Ketika Ramadan atau hari terlarang menyela pergantian ini, ulama berbeda pendapat tentang cara melanjutkannya; tanyakan kepada ulama yang kompeten.",
		),
	}
	if !isFastDay {
		status.Status = domain.PatternStatusRestDay
		status.Message = text(
			language,
			"You recorded a fast yesterday; the alternation rests today.",
			"Anda mencatat puasa kemarin; pergantian ini libur hari ini.",
		)
	}

	return status
}

func fridayStatus(
	date time.Time,
	classification domain.Classification,
	history *fastingHistory,
	isDawudFastDay bool,
	language domain.Language,
) domain.PatternStatus {
	status := domain.PatternStatus{
		Key:    domain.PatternFridayAlone,
		Name:   text(language, "Friday alone", "Puasa Jumat secara tersendiri"),
		Status: domain.PatternStatusCaution,
		Message: text(
			language,
			"Fasting Friday alone is discouraged; also fast Thursday or Saturday.",
			"Berpuasa pada hari Jumat saja tidak dianjurkan; sertakan puasa pada hari Kamis atau Sabtu.",
		),
		Evidence: "Sahih al-Bukhari 1985; Sahih Muslim 1144b",
	}

	customary := make([]string, 0, len(classification.Recommendations)+1)
	for _, recommendation := range classification.Recommendations {
		customary = append(customary, recommendation.Name)
	}
	if isDawudFastDay {
		customary = append(customary, text(language, "your Dawud alternation", "pergantian puasa Dawud Anda"))
	}

	isPaired := history.has(date.AddDate(0, 0, -1)) ||
		history.has(date.AddDate(0, 0, 1))
	switch {
	case isPaired:
		status.Status = domain.PatternStatusPaired
		status.Message = text(
			language,
			"You recorded a fast on the adjacent Thursday or Saturday, so this Friday is not singled out.",
			"Anda mencatat puasa pada hari Kamis atau Sabtu yang berdekatan, sehingga Jumat ini tidak dikhususkan.",
		)
	case len(customary) > 0:
		names := strings.Join(customary, ", ")
		status.Status = domain.PatternStatusCoincides
		status.Message = text(
			language,
			"This Friday coincides with "+names+".",
			"Jumat ini bertepatan dengan "+names+".",
		)
		status.Caveat = text(
			language,
			"Sahih Muslim 1144b excepts a fast you customarily observe; scholars differ on how far that reaches, so ask a qualified scholar if unsure.",
			"Sahih Muslim 1144b mengecualikan puasa yang biasa Anda jalankan; ulama berbeda pendapat tentang cakupannya, sehingga tanyakan kepada ulama yang kompeten bila ragu.",
		)
	}

	return status
}
