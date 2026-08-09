package fasting

import (
	"context"
	"errors"
	"time"
)

var ErrDateOutsideCalendar = errors.New("date is outside calendar coverage")

type CalendarRepository interface {
	HijriDate(context.Context, time.Time) (HijriDate, error)
	Source() CalendarSource
}
