package repository

import (
	"context"
	"fmt"
	"time"

	hijri "github.com/hablullah/go-hijri"

	"puasa-sunnah/internal/domains/fasting"
)

const (
	ummAlQuraCoverageStart = "1937-03-14"
	ummAlQuraCoverageEnd   = "2077-11-16"
)

type UmmAlQuraCalendar struct{}

func NewUmmAlQuraCalendar() *UmmAlQuraCalendar {
	return &UmmAlQuraCalendar{}
}

func (c *UmmAlQuraCalendar) HijriDate(ctx context.Context, value time.Time) (fasting.HijriDate, error) {
	if err := ctx.Err(); err != nil {
		return fasting.HijriDate{}, err
	}

	date := time.Date(value.Year(), value.Month(), value.Day(), 12, 0, 0, 0, time.UTC)
	converted, err := hijri.CreateUmmAlQuraDate(date)
	if err != nil {
		return fasting.HijriDate{}, fmt.Errorf("%w: Umm al-Qura covers %s through %s: %w", fasting.ErrDateOutsideCalendar, ummAlQuraCoverageStart, ummAlQuraCoverageEnd, err)
	}

	return fasting.HijriDate{
		Year:  int(converted.Year),
		Month: int(converted.Month),
		Day:   int(converted.Day),
	}, nil
}

func (c *UmmAlQuraCalendar) Source() fasting.CalendarSource {
	return fasting.CalendarSource{
		ID:            "umm-al-qura",
		Name:          "Umm al-Qura Calendar",
		Publisher:     "KACST calendar; conversion data compiled by R.H. van Gent",
		URL:           "https://webspace.science.uu.nl/~gent0113/islam/ummalqura.htm",
		CoverageStart: ummAlQuraCoverageStart,
		CoverageEnd:   ummAlQuraCoverageEnd,
		Status:        "calculated_planning_calendar",
		Caveat:        "Umm al-Qura is Saudi Arabia's civil calendar and can differ from calendars accepted by other authorities and localities.",
	}
}
