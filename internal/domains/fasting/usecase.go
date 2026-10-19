package fasting

import (
	"context"
	"errors"
)

var (
	ErrInvalidRequest      = errors.New("invalid request")
	ErrCalendarUnavailable = errors.New("calendar does not cover requested range")
)

type Usecase interface {
	Classify(context.Context, ClassificationRequest) (Classification, error)
	Schedule(context.Context, ScheduleRequest) (Schedule, error)
	Rules(context.Context, Language) []Rule
}
