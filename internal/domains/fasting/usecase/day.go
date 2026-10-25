package usecase

import (
	"context"
	"errors"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

// Day classifies one Gregorian date with a Hijri date derived from the
// caller-selected calendar. Unlike a schedule, it keeps restrictions.
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

	return domain.Day{
		Classification: classification,
		Calendar:       calendarID,
		CalendarSource: source,
	}, nil
}
