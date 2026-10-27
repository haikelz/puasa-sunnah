package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	domain "puasa-sunnah/internal/domains/fasting"

	"github.com/gofiber/fiber/v2"
)

// FastingHandler serves the fasting routes and maps use-case errors to
// localized HTTP responses.
type FastingHandler struct {
	usecase domain.Usecase
}

var errUsecaseRequired = errors.New("fasting usecase is required")

// Vercel's CDN honors s-maxage and purges its cache on every deploy, so a
// response for a fixed date stays valid until the calendar data changes.
const (
	cacheFixedDate = "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"
	// Without a date the anchor is today in Asia/Jakarta, so keep the copy short.
	cacheToday = "public, max-age=300, s-maxage=300"
	// Responses built from a caller's fasting history stay out of every cache.
	cachePrivate = "no-store"
)

const (
	formatJSON = "json"
	formatICS  = "ics"
)

// NewFastingHandler returns an error when usecase is nil.
func NewFastingHandler(usecase domain.Usecase) (*FastingHandler, error) {
	if usecase == nil {
		return nil, errUsecaseRequired
	}

	return &FastingHandler{
		usecase: usecase,
	}, nil
}

func (h *FastingHandler) Register(router fiber.Router) {
	router.Use(requireSupportedLanguage)
	router.Get("/rules", h.rules)
	router.Get("/calendars", h.calendars)
	router.Get("/schedules", h.schedule)
	router.Get("/days/:date", h.day)
	router.Post("/days/:date", h.dayWithHistory)
	router.Post("/classify", h.classify)
}

// rules godoc
// @Summary List supported fasting rules
// @Description Returns recommended patterns, prohibited days, evidence links, calculations, and jurisprudential caveats.
// @Tags Fasting
// @Produce json
// @Param language path string true "Response language" Enums(en, id)
// @Success 200 {object} RulesResponse
// @Header 200 {string} Cache-Control "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/rules [get]
func (h *FastingHandler) rules(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, cacheFixedDate)
	return c.JSON(fiber.Map{"data": h.usecase.Rules(c.UserContext(), requestLanguage(c))})
}

// calendars godoc
// @Summary List calendars and their coverage
// @Description Returns each selectable calendar with its full coverage, publications, Sidang Isbat decisions, status, and caveat. The default calendar answers requests that omit calendar.
// @Tags Fasting
// @Produce json
// @Param language path string true "Response language" Enums(en, id)
// @Success 200 {object} CalendarsResponse
// @Header 200 {string} Cache-Control "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/calendars [get]
func (h *FastingHandler) calendars(c *fiber.Ctx) error {
	language := requestLanguage(c)
	res, err := h.usecase.Calendars(c.UserContext(), language)
	if err != nil {
		return usecaseError(err, language)
	}

	// The calendar data changes only with a deploy, which purges the CDN.
	c.Set(fiber.HeaderCacheControl, cacheFixedDate)
	return c.JSON(fiber.Map{"data": res})
}

// schedule godoc
// @Summary Get a fasting schedule
// @Description Returns recommended voluntary fasting opportunities for the Gregorian week, month, or year containing the anchor date, or for an explicit range. period=range takes start (default today) with end or days, up to 366 days; omit start for a rolling feed a calendar app can subscribe to. Select the published Indonesian Kemenag calendar or the calculated Saudi Umm al-Qura calendar. Dates use Asia/Jakarta timezone. format=ics returns a text/calendar file with all-day events instead of the JSON envelope.
// @Tags Fasting
// @Produce json,text/calendar
// @Param language path string true "Response language" Enums(en, id)
// @Param period query string true "Schedule period" Enums(week, month, year, range)
// @Param date query string false "Anchor date in YYYY-MM-DD for week, month, and year; defaults to today in Asia/Jakarta"
// @Param start query string false "First date in YYYY-MM-DD for period=range; defaults to today in Asia/Jakarta"
// @Param end query string false "Last date in YYYY-MM-DD for period=range; use end or days"
// @Param days query integer false "Number of days for period=range, from 1 to 366; use end or days" minimum(1) maximum(366)
// @Param calendar query string false "Hijri calendar source; defaults to id" Enums(id, umm-al-qura) default(id)
// @Param types query []string false "Filter recommendation keys (OR); omitted means all" collectionFormat(csv) Enums(monday_thursday, white_days, ashura, six_shawwal_candidate, first_nine_dhul_hijjah, arafah)
// @Param format query string false "Response representation" Enums(json, ics) default(json)
// @Param reminder query string false "ICS only: add an alarm at HH:MM on the evening before each fast"
// @Param is_pilgrim_at_arafah query boolean false "Apply the pilgrim-at-Arafah exception" default(false)
// @Success 200 {object} ScheduleResponse
// @Header 200 {string} Cache-Control "CDN-cacheable for a day with an explicit date or start; 5 minutes when the range is relative to today"
// @Failure 400 {object} ErrorResponse "Invalid query"
// @Failure 422 {object} ErrorResponse "Calendar does not cover the requested range; calendar=id errors name calendar=umm-al-qura as an alternative"
// @Failure 503 {object} ErrorResponse "Request timed out"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/schedules [get]
func (h *FastingHandler) schedule(c *fiber.Ctx) error {
	language := requestLanguage(c)

	format := c.Query("format", formatJSON)
	if format != formatJSON && format != formatICS {
		return badRequest(
			language,
			"format must be json or ics",
			"format harus bernilai json atau ics",
		)
	}

	var types []string
	if c.Query("types") != "" {
		for _, key := range strings.Split(c.Query("types"), ",") {
			types = append(types, strings.TrimSpace(key))
		}
	}

	var days *int
	if value := c.Query("days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return badRequest(
				language,
				"days must be an integer",
				"days harus berupa bilangan bulat",
			)
		}

		days = &parsed
	}

	reminder, err := parseReminder(c.Query("reminder"), format, language)
	if err != nil {
		return err
	}

	isPilgrim, err := parsePilgrim(c, language)
	if err != nil {
		return err
	}

	res, err := h.usecase.Schedule(c.UserContext(), domain.ScheduleRequest{
		Period:            domain.SchedulePeriod(c.Query("period")),
		Date:              c.Query("date"),
		Start:             c.Query("start"),
		End:               c.Query("end"),
		Days:              days,
		IsPilgrimAtArafah: isPilgrim,
		Language:          language,
		Calendar:          domain.Calendar(c.Query("calendar")),
		Types:             types,
	})
	if err != nil {
		return usecaseError(err, language)
	}

	cacheControl := cacheFixedDate
	if c.Query("date") == "" && c.Query("start") == "" {
		cacheControl = cacheToday
	}
	c.Set(fiber.HeaderCacheControl, cacheControl)

	if format == formatICS {
		c.Set(fiber.HeaderContentType, "text/calendar; charset=utf-8")
		c.Set(fiber.HeaderContentDisposition, "attachment; filename=puasa-sunnah.ics")
		return c.SendString(scheduleICS(res, reminder))
	}

	return c.JSON(fiber.Map{"data": res})
}

// day godoc
// @Summary Look up one date
// @Description Classifies one Gregorian date with a Hijri date derived from the selected calendar. Unlike schedules, the result includes restrictions such as Ramadan, Eid, and Tashriq, so it answers whether to fast on that day.
// @Tags Fasting
// @Produce json
// @Param language path string true "Response language" Enums(en, id)
// @Param date path string true "Gregorian date in YYYY-MM-DD, or today for Asia/Jakarta"
// @Param calendar query string false "Hijri calendar source; defaults to id" Enums(id, umm-al-qura) default(id)
// @Param is_pilgrim_at_arafah query boolean false "Apply the pilgrim-at-Arafah exception" default(false)
// @Success 200 {object} DayResponse
// @Header 200 {string} Cache-Control "CDN-cacheable for a day with an explicit date; 5 minutes for today"
// @Failure 400 {object} ErrorResponse "Invalid date or query"
// @Failure 422 {object} ErrorResponse "Calendar does not cover the date"
// @Failure 503 {object} ErrorResponse "Request timed out"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/days/{date} [get]
func (h *FastingHandler) day(c *fiber.Ctx) error {
	return h.respondDay(c, nil)
}

// dayWithHistory godoc
// @Summary Look up one date with personal patterns
// @Description Same as GET, plus patterns that need the caller's history: progress through six days of Shawwal, Dawud alternation (with observes_dawud), and the Friday-alone caution. fasting_dates lists days the caller fasted or plans to fast. The API evaluates the body for this request only; it never stores it, and the response is not cacheable. Restricted days and a pilgrim at Arafah return no patterns.
// @Tags Fasting
// @Accept json
// @Produce json
// @Param language path string true "Response language" Enums(en, id)
// @Param date path string true "Gregorian date in YYYY-MM-DD, or today for Asia/Jakarta"
// @Param calendar query string false "Hijri calendar source; defaults to id" Enums(id, umm-al-qura) default(id)
// @Param is_pilgrim_at_arafah query boolean false "Apply the pilgrim-at-Arafah exception" default(false)
// @Param history body fasting.DtoFastingHistory true "Fasting dates (at most 366) and the Dawud opt-in"
// @Success 200 {object} DayResponse
// @Header 200 {string} Cache-Control "no-store"
// @Failure 400 {object} ErrorResponse "Invalid body, date, or query"
// @Failure 422 {object} ErrorResponse "Calendar does not cover the date"
// @Failure 503 {object} ErrorResponse "Request timed out"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/days/{date} [post]
func (h *FastingHandler) dayWithHistory(c *fiber.Ctx) error {
	var req domain.DtoFastingHistory
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	// A misspelled field would otherwise read as an empty history and wrong patterns.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return badRequest(
			requestLanguage(c),
			"body must be JSON with only fasting_dates and observes_dawud",
			"body harus berupa JSON yang hanya berisi fasting_dates dan observes_dawud",
		)
	}

	return h.respondDay(c, &req)
}

func (h *FastingHandler) respondDay(c *fiber.Ctx, history *domain.DtoFastingHistory) error {
	language := requestLanguage(c)

	isPilgrim, err := parsePilgrim(c, language)
	if err != nil {
		return err
	}

	res, err := h.usecase.Day(c.UserContext(), domain.DtoDayRequest{
		Date:              c.Params("date"),
		Calendar:          domain.Calendar(c.Query("calendar")),
		IsPilgrimAtArafah: isPilgrim,
		Language:          language,
		History:           history,
	})
	if err != nil {
		return usecaseError(err, language)
	}

	switch {
	case history != nil:
		c.Set(fiber.HeaderCacheControl, cachePrivate)
	case c.Params("date") == domain.DateToday:
		c.Set(fiber.HeaderCacheControl, cacheToday)
	default:
		c.Set(fiber.HeaderCacheControl, cacheFixedDate)
	}

	return c.JSON(fiber.Map{"data": res})
}

// classify godoc
// @Summary Classify a fasting date
// @Description Classifies one Gregorian date using a caller-supplied, locally accepted Hijri date. Restrictions take precedence over voluntary recommendations.
// @Tags Fasting
// @Accept json
// @Produce json
// @Param language path string true "Response language" Enums(en, id)
// @Param request body fasting.ClassificationRequest true "Gregorian and accepted Hijri date"
// @Success 200 {object} ClassificationResponse
// @Failure 400 {object} ErrorResponse "Invalid JSON or date"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/classify [post]
func (h *FastingHandler) classify(c *fiber.Ctx) error {
	language := requestLanguage(c)
	var req domain.ClassificationRequest
	// Decode JSON directly; BodyParser would also accept form and XML bodies.
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest(
			language,
			"body must be valid JSON",
			"body harus berupa JSON yang valid",
		)
	}
	req.Language = language

	res, err := h.usecase.Classify(c.UserContext(), req)
	if err != nil {
		return usecaseError(err, language)
	}

	return c.JSON(fiber.Map{"data": res})
}

func parsePilgrim(c *fiber.Ctx, language domain.Language) (bool, error) {
	value := c.Query("is_pilgrim_at_arafah")
	if value == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, badRequest(
			language,
			"is_pilgrim_at_arafah must be true or false",
			"is_pilgrim_at_arafah harus bernilai true atau false",
		)
	}

	return parsed, nil
}

// parseReminder returns how long before the all-day event the alarm fires:
// 20:00 on the evening before is 4 hours before midnight. Zero means no alarm.
func parseReminder(
	value string,
	format string,
	language domain.Language,
) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	if format != formatICS {
		return 0, badRequest(
			language,
			"reminder applies only to format=ics",
			"reminder hanya berlaku untuk format=ics",
		)
	}

	at, err := time.Parse("15:04", value)
	if err != nil {
		return 0, badRequest(
			language,
			"reminder must use HH:MM",
			"reminder harus menggunakan format HH:MM",
		)
	}

	return 24*time.Hour - time.Duration(at.Hour())*time.Hour - time.Duration(at.Minute())*time.Minute, nil
}

func badRequest(
	language domain.Language,
	english string,
	indonesian string,
) error {
	return fiber.NewError(fiber.StatusBadRequest, localized(language, english, indonesian))
}

func requireSupportedLanguage(c *fiber.Ctx) error {
	if !requestLanguage(c).IsSupported() {
		return fiber.ErrNotFound
	}

	return c.Next()
}

func requestLanguage(c *fiber.Ctx) domain.Language {
	return domain.Language(c.Params("language"))
}
