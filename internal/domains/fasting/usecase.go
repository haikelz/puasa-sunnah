package fasting

import "context"

type Usecase interface {
	Classify(context.Context, ClassificationRequest) (Classification, error)
	Schedule(context.Context, ScheduleRequest) (Schedule, error)
	Rules(context.Context, Language) []Rule
}
