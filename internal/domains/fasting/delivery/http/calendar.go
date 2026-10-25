package http

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"
)

// scheduleICS renders one all-day event per date with an exclusive end date.
// A positive reminder adds an alarm that long before the event starts.
func scheduleICS(schedule domain.Schedule, reminder time.Duration) string {
	var output strings.Builder
	line := func(value string) {
		// RFC 5545 folds at 75 octets without splitting UTF-8 characters.
		limit := 75
		for len(value) > limit {
			cut := limit
			for cut > 0 && value[cut]&0xc0 == 0x80 {
				cut--
			}

			output.WriteString(value[:cut] + "\r\n ")
			value = value[cut:]
			limit = 74
		}

		output.WriteString(value + "\r\n")
	}

	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Puasa Sunnah API//Fasting Schedule//EN")
	line("CALSCALE:GREGORIAN")
	line("X-WR-CALNAME:Puasa Sunnah")
	// Subscribed feeds (period=range without start) roll forward; ask apps to refetch daily.
	line("REFRESH-INTERVAL;VALUE=DURATION:P1D")
	line("X-PUBLISHED-TTL:P1D")

	sourceLines := []string{
		schedule.CalendarSource.Name,
		schedule.CalendarSource.Publisher,
	}
	for _, publication := range schedule.CalendarSource.Publications {
		sourceLines = append(sourceLines, publication.Name+": "+publication.URL)
	}
	sourceLines = append(
		sourceLines,
		schedule.CalendarSource.Status,
		schedule.CalendarSource.Caveat,
		schedule.Disclaimer,
	)
	sourceDescription := strings.Join(sourceLines, "\n")
	line("X-WR-CALDESC:" + calendarText(sourceDescription))

	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, date := range schedule.Dates {
		day, _ := time.Parse(time.DateOnly, date.Date) // Validated by Schedule.
		names := make([]string, 0, len(date.Recommendations))
		for _, recommendation := range date.Recommendations {
			names = append(names, recommendation.Name)
		}
		uid := sha256.Sum256([]byte(string(schedule.Calendar) + ":" + schedule.Audience + ":" + date.Date))
		summary := calendarText(strings.Join(names, ", "))
		description := sourceDescription + "\n" + strings.Join(date.Notes, "\n")

		line("BEGIN:VEVENT")
		line(fmt.Sprintf("UID:%x@puasa-sunnah", uid))
		line("DTSTAMP:" + stamp)
		line("DTSTART;VALUE=DATE:" + day.Format("20060102"))
		line("DTEND;VALUE=DATE:" + day.AddDate(0, 0, 1).Format("20060102"))
		line("SUMMARY:" + summary)
		line("DESCRIPTION:" + calendarText(description))
		line("TRANSP:TRANSPARENT")
		if reminder > 0 {
			line("BEGIN:VALARM")
			line("ACTION:DISPLAY")
			line("DESCRIPTION:" + summary)
			line("TRIGGER:" + icsBefore(reminder))
			line("END:VALARM")
		}
		line("END:VEVENT")
	}

	line("END:VCALENDAR")
	return output.String()
}

// icsBefore formats a negative RFC 5545 duration, such as -PT2H30M.
func icsBefore(d time.Duration) string {
	value := "-PT"
	if hours := int(d / time.Hour); hours > 0 {
		value += strconv.Itoa(hours) + "H"
	}
	if minutes := int(d % time.Hour / time.Minute); minutes > 0 {
		value += strconv.Itoa(minutes) + "M"
	}

	return value
}

func calendarText(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\r\n", "\\n", "\n", "\\n", "\r", "\\n", ";", "\\;", ",", "\\,").Replace(value)
}
