package repository

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

// Each file transcribes one official Kemenag PDF. Add a year only from a
// published Kemenag calendar; never extrapolate month lengths.
//
//go:embed kemenag/*.json
var kemenagFiles embed.FS

type kemenagPublication struct {
	Year          int                 `json:"year"`
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	URL           string              `json:"url"`
	SHA256        string              `json:"sha256"`
	CoverageStart string              `json:"coverage_start"`
	CoverageEnd   string              `json:"coverage_end"`
	MonthStarts   []kemenagMonthStart `json:"month_starts"`

	coverageStart time.Time
	coverageEnd   time.Time
}

type kemenagMonthStart struct {
	Date  string `json:"date"`
	Year  int    `json:"hijri_year"`
	Month int    `json:"hijri_month"`
}

type monthStart struct {
	date  time.Time
	year  int
	month int
}

type KemenagCalendar struct {
	publications  []kemenagPublication
	starts        []monthStart
	coverageStart time.Time
	coverageEnd   time.Time
}

func NewKemenagCalendar() (*KemenagCalendar, error) {
	return loadKemenagCalendar(kemenagFiles)
}

func loadKemenagCalendar(files fs.FS) (*KemenagCalendar, error) {
	names, err := fs.Glob(files, "kemenag/*.json")
	if err != nil {
		return nil, fmt.Errorf("list Kemenag calendars: %w", err)
	}
	if len(names) == 0 {
		return nil, errors.New("no Kemenag calendar data")
	}

	publications := make([]kemenagPublication, 0, len(names))
	for _, name := range names {
		publication, err := readKemenagPublication(files, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		publications = append(publications, publication)
	}
	sort.Slice(publications, func(i, j int) bool {
		return publications[i].coverageStart.Before(publications[j].coverageStart)
	})

	calendar := &KemenagCalendar{
		publications:  publications,
		coverageStart: publications[0].coverageStart,
		coverageEnd:   publications[0].coverageEnd,
	}
	for _, publication := range publications[1:] {
		if publication.coverageStart.After(calendar.coverageEnd.AddDate(0, 0, 1)) {
			return nil, fmt.Errorf("coverage gap between %s and %s", calendar.coverageEnd.Format(time.DateOnly), publication.CoverageStart)
		}
		if publication.coverageEnd.After(calendar.coverageEnd) {
			calendar.coverageEnd = publication.coverageEnd
		}
	}

	calendar.starts, err = mergeMonthStarts(publications)
	if err != nil {
		return nil, err
	}
	if calendar.starts[0].date.After(calendar.coverageStart) {
		return nil, fmt.Errorf("no month start on or before %s", calendar.coverageStart.Format(time.DateOnly))
	}
	if lastLength := daysBetween(calendar.starts[len(calendar.starts)-1].date, calendar.coverageEnd) + 1; lastLength > 30 {
		return nil, fmt.Errorf("coverage ends %d days into the last month", lastLength)
	}

	return calendar, nil
}

func readKemenagPublication(files fs.FS, name string) (kemenagPublication, error) {
	content, err := fs.ReadFile(files, name)
	if err != nil {
		return kemenagPublication{}, err
	}

	var publication kemenagPublication
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publication); err != nil {
		return kemenagPublication{}, err
	}
	if publication.Year < 1 || publication.ID == "" || publication.Name == "" || publication.URL == "" || len(publication.SHA256) != 64 || len(publication.MonthStarts) == 0 {
		return kemenagPublication{}, errors.New("year, id, name, url, sha256, and month_starts are required")
	}

	if publication.coverageStart, err = time.Parse(time.DateOnly, publication.CoverageStart); err != nil {
		return kemenagPublication{}, fmt.Errorf("coverage_start: %w", err)
	}
	if publication.coverageEnd, err = time.Parse(time.DateOnly, publication.CoverageEnd); err != nil {
		return kemenagPublication{}, fmt.Errorf("coverage_end: %w", err)
	}
	if publication.coverageEnd.Before(publication.coverageStart) {
		return kemenagPublication{}, errors.New("coverage_end precedes coverage_start")
	}

	return publication, nil
}

// mergeMonthStarts joins every publication's boundaries. Publications overlap at
// year edges, so a shared boundary must agree exactly.
func mergeMonthStarts(publications []kemenagPublication) ([]monthStart, error) {
	byDate := make(map[time.Time]monthStart)
	for _, publication := range publications {
		for _, entry := range publication.MonthStarts {
			date, err := time.Parse(time.DateOnly, entry.Date)
			if err != nil {
				return nil, fmt.Errorf("%s month start: %w", publication.ID, err)
			}
			if entry.Month < 1 || entry.Month > 12 || entry.Year < 1 {
				return nil, fmt.Errorf("%s month start %s has invalid Hijri month %d/%d", publication.ID, entry.Date, entry.Month, entry.Year)
			}

			start := monthStart{date: date, year: entry.Year, month: entry.Month}
			if existing, ok := byDate[date]; ok && existing != start {
				return nil, fmt.Errorf("%s disagrees on the month starting %s", publication.ID, entry.Date)
			}
			byDate[date] = start
		}
	}

	starts := make([]monthStart, 0, len(byDate))
	for _, start := range byDate {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i].date.Before(starts[j].date) })

	for i := 1; i < len(starts); i++ {
		previous, current := starts[i-1], starts[i]
		wantYear, wantMonth := previous.year, previous.month+1
		if wantMonth > 12 {
			wantYear, wantMonth = wantYear+1, 1
		}
		if current.year != wantYear || current.month != wantMonth {
			return nil, fmt.Errorf("month starting %s is %d/%d, want %d/%d", current.date.Format(time.DateOnly), current.month, current.year, wantMonth, wantYear)
		}
		if length := daysBetween(previous.date, current.date); length != 29 && length != 30 {
			return nil, fmt.Errorf("month starting %s has %d days, want 29 or 30", previous.date.Format(time.DateOnly), length)
		}
	}

	return starts, nil
}

func (c *KemenagCalendar) HijriDate(ctx context.Context, value time.Time) (fasting.HijriDate, error) {
	if err := ctx.Err(); err != nil {
		return fasting.HijriDate{}, err
	}

	value = date(value.Year(), value.Month(), value.Day())
	if value.Before(c.coverageStart) || value.After(c.coverageEnd) {
		return fasting.HijriDate{}, fmt.Errorf("%w: Kemenag calendars cover %s through %s", fasting.ErrDateOutsideCalendar, c.coverageStart.Format(time.DateOnly), c.coverageEnd.Format(time.DateOnly))
	}

	for i := len(c.starts) - 1; i >= 0; i-- {
		start := c.starts[i]
		if !value.Before(start.date) {
			return fasting.HijriDate{
				Year:  start.year,
				Month: start.month,
				Day:   daysBetween(start.date, value) + 1,
			}, nil
		}
	}

	return fasting.HijriDate{}, fmt.Errorf("%w: no Kemenag month boundary for %s", fasting.ErrDateOutsideCalendar, value.Format(time.DateOnly))
}

func (c *KemenagCalendar) Source() fasting.CalendarSource {
	latest := c.publications[len(c.publications)-1]
	return fasting.CalendarSource{
		ID:            latest.ID,
		Name:          latest.Name,
		Publisher:     "Ditjen Bimbingan Masyarakat Islam, Kementerian Agama RI",
		URL:           latest.URL,
		CoverageStart: c.coverageStart.Format(time.DateOnly),
		CoverageEnd:   c.coverageEnd.Format(time.DateOnly),
		Status:        "published_planning_calendar",
	}
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
