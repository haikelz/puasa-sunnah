package fasting

import (
	"context"
	"errors"
	"time"
)

var ErrDateOutsideCalendar = errors.New("date is outside calendar coverage")

type CalendarRepository interface {
	HijriDate(context.Context, time.Time) (HijriDate, error)
	// Source describes the publications covering start through end. It leaves
	// Caveat empty; the use case sets a localized one.
	Source(start, end time.Time) CalendarSource
}
