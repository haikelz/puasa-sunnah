package http

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"puasa-sunnah/internal/domains/fasting"
)

// scheduleICS renders one all-day event per date with an exclusive end date.
func scheduleICS(schedule fasting.Schedule) string {
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
	sourceLines := []string{schedule.CalendarSource.Name, schedule.CalendarSource.Publisher}
	for _, publication := range schedule.CalendarSource.Publications {
		sourceLines = append(sourceLines, publication.Name+": "+publication.URL)
	}
	sourceDescription := strings.Join(append(sourceLines,
		schedule.CalendarSource.Status,
		schedule.CalendarSource.Caveat,
		schedule.Disclaimer,
	), "\n")
	line("X-WR-CALDESC:" + calendarText(sourceDescription))
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, date := range schedule.Dates {
		day, _ := time.Parse(time.DateOnly, date.Date) // Validated by Schedule.
		names := make([]string, 0, len(date.Recommendations))
		for _, recommendation := range date.Recommendations {
			names = append(names, recommendation.Name)
		}
		uid := sha256.Sum256([]byte(string(schedule.Calendar) + ":" + schedule.Audience + ":" + date.Date))
		description := sourceDescription + "\n" + strings.Join(date.Notes, "\n")
		line("BEGIN:VEVENT")
		line(fmt.Sprintf("UID:%x@puasa-sunnah", uid))
		line("DTSTAMP:" + stamp)
		line("DTSTART;VALUE=DATE:" + day.Format("20060102"))
		line("DTEND;VALUE=DATE:" + day.AddDate(0, 0, 1).Format("20060102"))
		line("SUMMARY:" + calendarText(strings.Join(names, ", ")))
		line("DESCRIPTION:" + calendarText(description))
		line("TRANSP:TRANSPARENT")
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return output.String()
}

func calendarText(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\r\n", "\\n", "\n", "\\n", "\r", "\\n", ";", "\\;", ",", "\\,").Replace(value)
}
