package usecase

import (
	"context"
	"errors"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

var jakarta = time.FixedZone("Asia/Jakarta", 7*60*60)

func (u *FastingUsecase) Schedule(ctx context.Context, request domain.ScheduleRequest) (domain.Schedule, error) {
	for _, key := range request.Types {
		switch key {
		case "monday_thursday", "white_days", "ashura", "six_shawwal_candidate", "first_nine_dhul_hijjah", "arafah":
		default:
			return domain.Schedule{}, invalidRequest(domain.CodeUnsupportedType, "types")
		}
	}

	calendarID, calendar, err := u.scheduleCalendar(request.Calendar)
	if err != nil {
		return domain.Schedule{}, err
	}

	window, err := scheduleBounds(request)
	if err != nil {
		return domain.Schedule{}, err
	}

	calendarSource := calendar.Source(window.start, window.end)
	calendarSource.Caveat = calendarCaveat(request.Language, calendarID)
	result := domain.Schedule{
		Period:         request.Period,
		Calendar:       calendarID,
		AnchorDate:     window.anchor.Format(time.DateOnly),
		StartDate:      window.start.Format(time.DateOnly),
		EndDate:        window.end.Format(time.DateOnly),
		Timezone:       jakarta.String(),
		Audience:       "general_non_pilgrim",
		CalendarSource: calendarSource,
		Dates:          make([]domain.ScheduleDate, 0),
		Disclaimer:     disclaimer(request.Language),
	}
	if request.IsPilgrimAtArafah {
		result.Audience = "pilgrim_at_arafah"
	}
	if request.Period == domain.SchedulePeriodWeek {
		result.WeekStartsOn = text(request.Language, "Monday", "Senin")
	}

	for current := window.start; !current.After(window.end); current = current.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return domain.Schedule{}, err
		}
		hijri, err := calendar.HijriDate(ctx, current)
		if errors.Is(err, domain.ErrDateOutsideCalendar) {
			return domain.Schedule{}, calendarUnavailable(calendarID, calendarSource)
		}
		if err != nil {
			return domain.Schedule{}, err
		}

		classification, err := u.Classify(ctx, domain.ClassificationRequest{
			Date:              current.Format(time.DateOnly),
			Hijri:             hijri,
			IsPilgrimAtArafah: request.IsPilgrimAtArafah,
			Language:          request.Language,
		})
		if err != nil {
			return domain.Schedule{}, err
		}
		if classification.Verdict != domain.VerdictRecommended {
			continue
		}
		if len(request.Types) > 0 {
			filtered := make([]domain.Occurrence, 0)
			for _, recommendation := range classification.Recommendations {
				for _, key := range request.Types {
					if recommendation.Key == key {
						filtered = append(filtered, recommendation)
						break
					}
				}
			}
			classification.Recommendations = filtered
			if len(filtered) == 0 {
				continue
			}
		}
		result.Dates = append(result.Dates, domain.ScheduleDate{
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

func (u *FastingUsecase) scheduleCalendar(calendarID domain.Calendar) (domain.Calendar, domain.CalendarRepository, error) {
	if calendarID == "" {
		calendarID = domain.CalendarIndonesia
	}

	switch calendarID {
	case domain.CalendarIndonesia:
		return calendarID, u.indonesiaCalendar, nil
	case domain.CalendarUmmAlQura:
		return calendarID, u.ummAlQuraCalendar, nil
	default:
		return "", nil, invalidRequest(domain.CodeUnsupportedCalendar, "calendar")
	}
}

func invalidRequest(code, field string) error {
	return &domain.RequestError{
		Kind:  domain.ErrInvalidRequest,
		Code:  code,
		Field: field,
	}
}

// calendarUnavailable reports the coverage so the delivery layer can point to
// the other calendar without switching to it: the caller must choose.
func calendarUnavailable(calendarID domain.Calendar, source domain.CalendarSource) error {
	return &domain.RequestError{
		Kind:          domain.ErrCalendarUnavailable,
		Code:          domain.CodeOutsideCoverage,
		Calendar:      calendarID,
		CoverageStart: source.CoverageStart,
		CoverageEnd:   source.CoverageEnd,
	}
}

// scheduleWindow is the anchor and inclusive date range a schedule covers.
type scheduleWindow struct {
	anchor time.Time
	start  time.Time
	end    time.Time
}

// scheduleBounds resolves the window for a request: the calendar period that
// contains the anchor date.
func scheduleBounds(request domain.ScheduleRequest) (scheduleWindow, error) {
	anchor, err := scheduleAnchor(request.Date, "date")
	if err != nil {
		return scheduleWindow{}, err
	}

	start, end, err := scheduleRange(anchor, request.Period)
	if err != nil {
		return scheduleWindow{}, err
	}

	return scheduleWindow{
		anchor: anchor,
		start:  start,
		end:    end,
	}, nil
}

// scheduleAnchor parses a strict date in Asia/Jakarta; empty means today.
func scheduleAnchor(value, field string) (time.Time, error) {
	if value == "" {
		now := time.Now().In(jakarta)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakarta), nil
	}

	return parseDate(value, field)
}

func parseDate(value, field string) (time.Time, error) {
	parsed, err := time.ParseInLocation(time.DateOnly, value, jakarta)
	if err != nil {
		return time.Time{}, invalidRequest(domain.CodeDateFormat, field)
	}

	return parsed, nil
}

func scheduleRange(anchor time.Time, period domain.SchedulePeriod) (time.Time, time.Time, error) {
	switch period {
	case domain.SchedulePeriodWeek:
		daysSinceMonday := (int(anchor.Weekday()) + 6) % 7
		start := anchor.AddDate(0, 0, -daysSinceMonday)
		return start, start.AddDate(0, 0, 6), nil
	case domain.SchedulePeriodMonth:
		start := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, jakarta)
		return start, start.AddDate(0, 1, -1), nil
	case domain.SchedulePeriodYear:
		start := time.Date(anchor.Year(), time.January, 1, 0, 0, 0, 0, jakarta)
		return start, start.AddDate(1, 0, -1), nil
	default:
		return time.Time{}, time.Time{}, invalidRequest(domain.CodeUnsupportedPeriod, "period")
	}
}
