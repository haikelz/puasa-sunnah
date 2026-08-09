package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

var jakarta = time.FixedZone("Asia/Jakarta", 7*60*60)

func (u *FastingUsecase) Schedule(ctx context.Context, request fasting.ScheduleRequest) (fasting.Schedule, error) {
	if u.calendar == nil {
		return fasting.Schedule{}, fmt.Errorf("%w: %s", ErrCalendarUnavailable, text(request.Language, "no calendar source configured", "sumber kalender belum dikonfigurasi"))
	}

	anchor, err := scheduleAnchor(request.Date, request.Language)
	if err != nil {
		return fasting.Schedule{}, err
	}
	start, end, err := scheduleRange(anchor, request.Period, request.Language)
	if err != nil {
		return fasting.Schedule{}, err
	}

	calendarSource := u.calendar.Source()
	calendarSource.Caveat = calendarCaveat(request.Language)
	result := fasting.Schedule{
		Period:         request.Period,
		AnchorDate:     anchor.Format(time.DateOnly),
		StartDate:      start.Format(time.DateOnly),
		EndDate:        end.Format(time.DateOnly),
		Timezone:       jakarta.String(),
		Audience:       "general_non_pilgrim",
		CalendarSource: calendarSource,
		Dates:          make([]fasting.ScheduleDate, 0),
		Disclaimer:     disclaimer(request.Language),
	}
	if request.IsPilgrimAtArafah {
		result.Audience = "pilgrim_at_arafah"
	}
	if request.Period == fasting.SchedulePeriodWeek {
		result.WeekStartsOn = text(request.Language, "Monday", "Senin")
	}

	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return fasting.Schedule{}, err
		}
		hijri, err := u.calendar.HijriDate(ctx, current)
		if err != nil {
			if errors.Is(err, fasting.ErrDateOutsideCalendar) {
				source := u.calendar.Source()
				return fasting.Schedule{}, fmt.Errorf("%w: %s", ErrCalendarUnavailable, text(request.Language,
					fmt.Sprintf("calendar covers %s through %s", source.CoverageStart, source.CoverageEnd),
					fmt.Sprintf("kalender hanya mencakup %s sampai %s", source.CoverageStart, source.CoverageEnd),
				))
			}
			return fasting.Schedule{}, err
		}
		classification, err := u.Classify(ctx, fasting.ClassificationRequest{
			Date:              current.Format(time.DateOnly),
			Hijri:             hijri,
			IsPilgrimAtArafah: request.IsPilgrimAtArafah,
			Language:          request.Language,
		})
		if err != nil {
			return fasting.Schedule{}, err
		}
		if classification.Verdict != fasting.VerdictRecommended {
			continue
		}
		result.Dates = append(result.Dates, fasting.ScheduleDate{
			Date:            classification.Date,
			Weekday:         classification.Weekday,
			Hijri:           classification.Hijri,
			Recommendations: classification.Recommendations,
			Notes:           classification.Notes,
		})
	}

	result.Total = len(result.Dates)
	return result, nil
}

func scheduleAnchor(value string, language fasting.Language) (time.Time, error) {
	if value == "" {
		now := time.Now().In(jakarta)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakarta), nil
	}
	parsed, err := time.ParseInLocation(time.DateOnly, value, jakarta)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidRequest, text(language, "date must use YYYY-MM-DD", "tanggal harus menggunakan format YYYY-MM-DD"))
	}
	return parsed, nil
}

func scheduleRange(anchor time.Time, period fasting.SchedulePeriod, language fasting.Language) (time.Time, time.Time, error) {
	switch period {
	case fasting.SchedulePeriodWeek:
		daysSinceMonday := (int(anchor.Weekday()) + 6) % 7
		start := anchor.AddDate(0, 0, -daysSinceMonday)
		return start, start.AddDate(0, 0, 6), nil
	case fasting.SchedulePeriodMonth:
		start := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, jakarta)
		return start, start.AddDate(0, 1, -1), nil
	case fasting.SchedulePeriodYear:
		start := time.Date(anchor.Year(), time.January, 1, 0, 0, 0, 0, jakarta)
		return start, start.AddDate(1, 0, -1), nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("%w: %s", ErrInvalidRequest, text(language, "period must be week, month, or year", "period harus bernilai week, month, atau year"))
	}
}
