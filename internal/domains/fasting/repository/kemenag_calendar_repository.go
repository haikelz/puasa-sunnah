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

	domain "puasa-sunnah/internal/domains/fasting"
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
			return nil, fmt.Errorf(
				"coverage gap between %s and %s",
				calendar.coverageEnd.Format(time.DateOnly),
				publication.CoverageStart,
			)
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
	lastStart := calendar.starts[len(calendar.starts)-1].date
	if lastLength := daysBetween(lastStart, calendar.coverageEnd) + 1; lastLength > 30 {
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
	isIncomplete := publication.Year < 1 ||
		publication.ID == "" ||
		publication.Name == "" ||
		publication.URL == "" ||
		len(publication.SHA256) != 64 ||
		len(publication.MonthStarts) == 0
	if isIncomplete {
		return kemenagPublication{}, errors.New("year, id, name, url, sha256, and month_starts are required")
	}

	publication.coverageStart, err = time.Parse(time.DateOnly, publication.CoverageStart)
	if err != nil {
		return kemenagPublication{}, fmt.Errorf("parse coverage_start: %w", err)
	}
	publication.coverageEnd, err = time.Parse(time.DateOnly, publication.CoverageEnd)
	if err != nil {
		return kemenagPublication{}, fmt.Errorf("parse coverage_end: %w", err)
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
				return nil, fmt.Errorf("parse %s month start: %w", publication.ID, err)
			}
			if entry.Month < 1 || entry.Month > 12 || entry.Year < 1 {
				return nil, fmt.Errorf(
					"%s month start %s has invalid Hijri month %d/%d",
					publication.ID,
					entry.Date,
					entry.Month,
					entry.Year,
				)
			}

			start := monthStart{
				date:  date,
				year:  entry.Year,
				month: entry.Month,
			}
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
	sort.Slice(starts, func(i, j int) bool {
		return starts[i].date.Before(starts[j].date)
	})

	for i := 1; i < len(starts); i++ {
		previous, current := starts[i-1], starts[i]
		wantYear, wantMonth := previous.year, previous.month+1
		if wantMonth > 12 {
			wantYear, wantMonth = wantYear+1, 1
		}

		if current.year != wantYear || current.month != wantMonth {
			return nil, fmt.Errorf(
				"month starting %s is %d/%d, want %d/%d",
				current.date.Format(time.DateOnly),
				current.month,
				current.year,
				wantMonth,
				wantYear,
			)
		}
		if length := daysBetween(previous.date, current.date); length != 29 && length != 30 {
			return nil, fmt.Errorf(
				"month starting %s has %d days, want 29 or 30",
				previous.date.Format(time.DateOnly),
				length,
			)
		}
	}

	return starts, nil
}

func (r *KemenagCalendar) HijriDate(ctx context.Context, value time.Time) (domain.HijriDate, error) {
	if err := ctx.Err(); err != nil {
		return domain.HijriDate{}, err
	}

	value = date(value.Year(), value.Month(), value.Day())
	if value.Before(r.coverageStart) || value.After(r.coverageEnd) {
		return domain.HijriDate{}, fmt.Errorf(
			"%w: Kemenag calendars cover %s through %s",
			domain.ErrDateOutsideCalendar,
			r.coverageStart.Format(time.DateOnly),
			r.coverageEnd.Format(time.DateOnly),
		)
	}

	for i := len(r.starts) - 1; i >= 0; i-- {
		start := r.starts[i]
		if !value.Before(start.date) {
			return domain.HijriDate{
				Year:  start.year,
				Month: start.month,
				Day:   daysBetween(start.date, value) + 1,
			}, nil
		}
	}

	return domain.HijriDate{}, fmt.Errorf(
		"%w: no Kemenag month boundary for %s",
		domain.ErrDateOutsideCalendar,
		value.Format(time.DateOnly),
	)
}

func (r *KemenagCalendar) Source(start, end time.Time) domain.CalendarSource {
	start = date(start.Year(), start.Month(), start.Day())
	end = date(end.Year(), end.Month(), end.Day())

	publications := make([]domain.CalendarPublication, 0, 2)
	seen := make(map[string]bool)
	for year := start.Year(); year <= end.Year(); year++ {
		publication, ok := r.publicationFor(year, start, end)
		if !ok || seen[publication.ID] {
			continue
		}

		seen[publication.ID] = true
		publications = append(publications, domain.CalendarPublication{
			ID:            publication.ID,
			Name:          publication.Name,
			URL:           publication.URL,
			CoverageStart: publication.CoverageStart,
			CoverageEnd:   publication.CoverageEnd,
		})
	}

	source := domain.CalendarSource{
		ID:            "kemenag-ri",
		Name:          "Kalender Hijriah Indonesia",
		Publisher:     "Ditjen Bimbingan Masyarakat Islam, Kementerian Agama RI",
		URL:           r.publications[len(r.publications)-1].URL,
		CoverageStart: r.coverageStart.Format(time.DateOnly),
		CoverageEnd:   r.coverageEnd.Format(time.DateOnly),
		Status:        domain.CalendarStatusPublished,
		Publications:  publications,
	}
	if len(publications) > 0 {
		source.URL = publications[0].URL
	}

	return source
}

// publicationFor prefers the PDF published for that Gregorian year; a
// neighbouring year's PDF only answers for its few spill-over days.
func (r *KemenagCalendar) publicationFor(
	year int,
	start time.Time,
	end time.Time,
) (kemenagPublication, bool) {
	for _, publication := range r.publications {
		if publication.Year == year {
			return publication, true
		}
	}

	yearStart := maxTime(start, date(year, time.January, 1))
	yearEnd := minTime(end, date(year, time.December, 31))
	for _, publication := range r.publications {
		overlaps := !publication.coverageStart.After(yearEnd) &&
			!publication.coverageEnd.Before(yearStart)
		if overlaps {
			return publication, true
		}
	}

	return kemenagPublication{}, false
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}

	return b
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
