package repository

import (
	"context"
	"fmt"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"

	hijri "github.com/hablullah/go-hijri"
)

const (
	ummAlQuraID            = "umm-al-qura"
	ummAlQuraName          = "Umm al-Qura Calendar"
	ummAlQuraURL           = "https://webspace.science.uu.nl/~gent0113/islam/ummalqura.htm"
	ummAlQuraCoverageStart = "1937-03-14"
	ummAlQuraCoverageEnd   = "2077-11-16"
)

type UmmAlQuraCalendar struct{}

func NewUmmAlQuraCalendar() *UmmAlQuraCalendar {
	return &UmmAlQuraCalendar{}
}

func (r *UmmAlQuraCalendar) HijriDate(ctx context.Context, value time.Time) (domain.HijriDate, error) {
	if err := ctx.Err(); err != nil {
		return domain.HijriDate{}, err
	}

	date := time.Date(value.Year(), value.Month(), value.Day(), 12, 0, 0, 0, time.UTC)
	converted, err := hijri.CreateUmmAlQuraDate(date)
	if err != nil {
		return domain.HijriDate{}, fmt.Errorf(
			"%w: Umm al-Qura covers %s through %s: %w",
			domain.ErrDateOutsideCalendar,
			ummAlQuraCoverageStart,
			ummAlQuraCoverageEnd,
			err,
		)
	}

	return domain.HijriDate{
		Year:  int(converted.Year),
		Month: int(converted.Month),
		Day:   int(converted.Day),
	}, nil
}

func (r *UmmAlQuraCalendar) Source(time.Time, time.Time) domain.CalendarSource {
	return domain.CalendarSource{
		ID:            ummAlQuraID,
		Name:          ummAlQuraName,
		Publisher:     "KACST calendar; conversion data compiled by R.H. van Gent",
		URL:           ummAlQuraURL,
		CoverageStart: ummAlQuraCoverageStart,
		CoverageEnd:   ummAlQuraCoverageEnd,
		Status:        domain.CalendarStatusCalculated,
		Publications: []domain.CalendarPublication{
			{
				ID:            ummAlQuraID,
				Name:          ummAlQuraName,
				URL:           ummAlQuraURL,
				CoverageStart: ummAlQuraCoverageStart,
				CoverageEnd:   ummAlQuraCoverageEnd,
			},
		},
	}
}
