package fasting

import "context"

type Usecase interface {
	Classify(context.Context, ClassificationRequest) (Classification, error)
	Schedule(context.Context, ScheduleRequest) (Schedule, error)
	Day(context.Context, DtoDayRequest) (Day, error)
	Rules(context.Context, Language) []Rule
}
