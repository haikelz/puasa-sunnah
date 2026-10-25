package fasting

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidRequest marks input the caller must correct.
	ErrInvalidRequest = errors.New("invalid request")
	// ErrCalendarUnavailable marks a request outside the selected calendar's coverage.
	ErrCalendarUnavailable = errors.New("calendar does not cover requested range")
	// ErrDateOutsideCalendar marks a date a calendar repository cannot convert.
	ErrDateOutsideCalendar = errors.New("date is outside calendar coverage")
)

// Request error codes. The delivery layer turns each into a localized message.
const (
	CodeDateFormat          = "date_format"
	CodeHijriRange          = "hijri_range"
	CodeUnsupportedType     = "unsupported_type"
	CodeUnsupportedCalendar = "unsupported_calendar"
	CodeUnsupportedPeriod   = "unsupported_period"
	CodeOutsideCoverage     = "outside_coverage"
)

// RequestError is a client error with a stable code and the values its message
// needs. Kind is ErrInvalidRequest or ErrCalendarUnavailable, so errors.Is
// classifies it. Error returns English for logs, never for clients.
type RequestError struct {
	Kind          error
	Code          string
	Field         string
	Calendar      Calendar
	CoverageStart string
	CoverageEnd   string
}

func (e *RequestError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Kind, e.Code, e.Field)
	}

	return fmt.Sprintf("%s: %s", e.Kind, e.Code)
}

func (e *RequestError) Unwrap() error {
	return e.Kind
}
