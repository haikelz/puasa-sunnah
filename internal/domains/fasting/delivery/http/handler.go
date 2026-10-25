package http

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	domain "puasa-sunnah/internal/domains/fasting"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase domain.Usecase
}

var errUsecaseRequired = errors.New("fasting usecase is required")

// Vercel's CDN honors s-maxage and purges its cache on every deploy, so a
// response for a fixed date stays valid until the calendar data changes.
const (
	cacheFixedDate = "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"
	// Without a date the anchor is today in Asia/Jakarta, so keep the copy short.
	cacheToday = "public, max-age=300, s-maxage=300"
)

const (
	formatJSON = "json"
	formatICS  = "ics"
)

func NewHandler(usecase domain.Usecase) (*Handler, error) {
	if usecase == nil {
		return nil, errUsecaseRequired
	}

	return &Handler{
		usecase: usecase,
	}, nil
}

func (h *Handler) Register(router fiber.Router) {
	router.Use(requireSupportedLanguage)
	router.Get("/rules", h.rules)
	router.Get("/schedules", h.schedule)
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
func (h *Handler) rules(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, cacheFixedDate)
	return c.JSON(fiber.Map{"data": h.usecase.Rules(c.UserContext(), requestLanguage(c))})
}

// schedule godoc
// @Summary Get a fasting schedule
// @Description Returns recommended voluntary fasting opportunities for the Gregorian week, month, or year containing the anchor date. Select the published Indonesian Kemenag calendar or the calculated Saudi Umm al-Qura calendar. Dates use Asia/Jakarta timezone. format=ics returns a downloadable text/calendar file with all-day events instead of the JSON envelope.
// @Tags Fasting
// @Produce json,text/calendar
// @Param language path string true "Response language" Enums(en, id)
// @Param period query string true "Schedule period" Enums(week, month, year)
// @Param date query string false "Anchor date in YYYY-MM-DD; defaults to today in Asia/Jakarta"
// @Param calendar query string false "Hijri calendar source; defaults to id" Enums(id, umm-al-qura) default(id)
// @Param types query []string false "Filter recommendation keys (OR); omitted means all" collectionFormat(csv) Enums(monday_thursday, white_days, ashura, six_shawwal_candidate, first_nine_dhul_hijjah, arafah)
// @Param format query string false "Response representation" Enums(json, ics) default(json)
// @Param is_pilgrim_at_arafah query boolean false "Apply the pilgrim-at-Arafah exception" default(false)
// @Success 200 {object} ScheduleResponse
// @Header 200 {string} Cache-Control "CDN-cacheable for a day with an explicit date; 5 minutes when date is omitted"
// @Failure 400 {object} ErrorResponse "Invalid query"
// @Failure 422 {object} ErrorResponse "Calendar does not cover the requested range; calendar=id errors name calendar=umm-al-qura as an alternative"
// @Failure 503 {object} ErrorResponse "Request timed out"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/schedules [get]
func (h *Handler) schedule(c *fiber.Ctx) error {
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

	isPilgrim, err := parsePilgrim(c, language)
	if err != nil {
		return err
	}

	res, err := h.usecase.Schedule(c.UserContext(), domain.ScheduleRequest{
		Period:            domain.SchedulePeriod(c.Query("period")),
		Date:              c.Query("date"),
		IsPilgrimAtArafah: isPilgrim,
		Language:          language,
		Calendar:          domain.Calendar(c.Query("calendar")),
		Types:             types,
	})
	if err != nil {
		return usecaseError(err, language)
	}

	cacheControl := cacheFixedDate
	if c.Query("date") == "" {
		cacheControl = cacheToday
	}
	c.Set(fiber.HeaderCacheControl, cacheControl)

	if format == formatICS {
		c.Set(fiber.HeaderContentType, "text/calendar; charset=utf-8")
		c.Set(fiber.HeaderContentDisposition, "attachment; filename=puasa-sunnah.ics")
		return c.SendString(scheduleICS(res))
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
func (h *Handler) classify(c *fiber.Ctx) error {
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
