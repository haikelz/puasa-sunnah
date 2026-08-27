package repository

import (
	"context"
	"fmt"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

const sourceURL = "https://aceh.kemenag.go.id/storage/berita_informasi/01K7GE96M0ZVRJ62Z520RMYG1B.pdf"

var (
	coverageStart = date(2025, time.December, 28)
	coverageEnd   = date(2027, time.January, 2)
)

type monthStart struct {
	date  time.Time
	year  int
	month int
}

type KemenagCalendar struct {
	starts []monthStart
}

func NewKemenagCalendar() *KemenagCalendar {
	return &KemenagCalendar{starts: []monthStart{
		{date: date(2025, time.December, 21), year: 1447, month: 7},
		{date: date(2026, time.January, 20), year: 1447, month: 8},
		{date: date(2026, time.February, 19), year: 1447, month: 9},
		{date: date(2026, time.March, 21), year: 1447, month: 10},
		{date: date(2026, time.April, 19), year: 1447, month: 11},
		{date: date(2026, time.May, 18), year: 1447, month: 12},
		{date: date(2026, time.June, 16), year: 1448, month: 1},
		{date: date(2026, time.July, 16), year: 1448, month: 2},
		{date: date(2026, time.August, 14), year: 1448, month: 3},
		{date: date(2026, time.September, 13), year: 1448, month: 4},
		{date: date(2026, time.October, 12), year: 1448, month: 5},
		{date: date(2026, time.November, 11), year: 1448, month: 6},
		{date: date(2026, time.December, 10), year: 1448, month: 7},
	}}
}

func (c *KemenagCalendar) HijriDate(ctx context.Context, value time.Time) (fasting.HijriDate, error) {
	if err := ctx.Err(); err != nil {
		return fasting.HijriDate{}, err
	}

	value = date(value.Year(), value.Month(), value.Day())
	if value.Before(coverageStart) || value.After(coverageEnd) {
		return fasting.HijriDate{}, fmt.Errorf("%w: Kemenag 2026 covers %s through %s", fasting.ErrDateOutsideCalendar, coverageStart.Format(time.DateOnly), coverageEnd.Format(time.DateOnly))
	}

	for i := len(c.starts) - 1; i >= 0; i-- {
		start := c.starts[i]
		if !value.Before(start.date) {
			return fasting.HijriDate{
				Year:  start.year,
				Month: start.month,
				Day:   int(value.Sub(start.date).Hours()/24) + 1,
			}, nil
		}
	}

	return fasting.HijriDate{}, fmt.Errorf("%w: no Kemenag month boundary for %s", fasting.ErrDateOutsideCalendar, value.Format(time.DateOnly))
}

func (c *KemenagCalendar) Source() fasting.CalendarSource {
	return fasting.CalendarSource{
		ID:            "kemenag-ri-2026",
		Name:          "Kalender Hijriah Indonesia Tahun 2026",
		Publisher:     "Ditjen Bimbingan Masyarakat Islam, Kementerian Agama RI",
		URL:           sourceURL,
		CoverageStart: coverageStart.Format(time.DateOnly),
		CoverageEnd:   coverageEnd.Format(time.DateOnly),
		Status:        "published_planning_calendar",
		Caveat:        "For worship, Ramadan, Shawwal, and Dhu al-Hijjah dates can be superseded by an official Kemenag Sidang Isbat decision.",
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
