package http

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"puasa-sunnah/internal/domains/fasting"
)

type Handler struct {
	usecase fasting.Usecase
}

var errUsecaseRequired = errors.New("fasting usecase is required")

// Vercel's CDN honors s-maxage and purges its cache on every deploy, so a
// response for a fixed date stays valid until the calendar data changes.
const (
	cacheFixedDate = "public, max-age=3600, s-maxage=86400, stale-while-revalidate=86400"
	// Without a date the anchor is today in Asia/Jakarta, so keep the copy short.
	cacheToday = "public, max-age=300, s-maxage=300"
)

func NewHandler(usecase fasting.Usecase) (*Handler, error) {
	if usecase == nil {
		return nil, errUsecaseRequired
	}
	return &Handler{usecase: usecase}, nil
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
// @Failure 422 {object} ErrorResponse "Calendar does not cover the requested range"
// @Failure 503 {object} ErrorResponse "Request timed out"
// @Failure 500 {object} ErrorResponse
// @Router /api/{language}/fasting/schedules [get]
func (h *Handler) schedule(c *fiber.Ctx) error {
	language := requestLanguage(c)
	format := c.Query("format", "json")
	if format != "json" && format != "ics" {
		message := "format must be json or ics"
		if language == fasting.LanguageIndonesian {
			message = "format harus bernilai json atau ics"
		}
		return fiber.NewError(fiber.StatusBadRequest, message)
	}
	var types []string
	if c.Query("types") != "" {
		for _, key := range strings.Split(c.Query("types"), ",") {
			types = append(types, strings.TrimSpace(key))
		}
	}
	isPilgrim := false
	if value := c.Query("is_pilgrim_at_arafah"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			message := "is_pilgrim_at_arafah must be true or false"
			if language == fasting.LanguageIndonesian {
				message = "is_pilgrim_at_arafah harus bernilai true atau false"
			}
			return fiber.NewError(fiber.StatusBadRequest, message)
		}
		isPilgrim = parsed
	}

	result, err := h.usecase.Schedule(c.UserContext(), fasting.ScheduleRequest{
		Period:            fasting.SchedulePeriod(c.Query("period")),
		Date:              c.Query("date"),
		IsPilgrimAtArafah: isPilgrim,
		Language:          language,
		Calendar:          fasting.Calendar(c.Query("calendar")),
		Types:             types,
	})
	if err != nil {
		switch {
		case errors.Is(err, fasting.ErrInvalidRequest):
			return fiber.NewError(fiber.StatusBadRequest, clientErrorMessage(err, fasting.ErrInvalidRequest))
		case errors.Is(err, fasting.ErrCalendarUnavailable):
			return fiber.NewError(fiber.StatusUnprocessableEntity, clientErrorMessage(err, fasting.ErrCalendarUnavailable))
		default:
			return err
		}
	}

	cacheControl := cacheFixedDate
	if c.Query("date") == "" {
		cacheControl = cacheToday
	}
	c.Set(fiber.HeaderCacheControl, cacheControl)

	if format == "ics" {
		c.Set(fiber.HeaderContentType, "text/calendar; charset=utf-8")
		c.Set(fiber.HeaderContentDisposition, "attachment; filename=puasa-sunnah.ics")
		return c.SendString(scheduleICS(result))
	}
	return c.JSON(fiber.Map{"data": result})
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
	var request fasting.ClassificationRequest
	// Decode JSON directly; BodyParser would also accept form and XML bodies.
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		message := "body must be valid JSON"
		if language == fasting.LanguageIndonesian {
			message = "body harus berupa JSON yang valid"
		}
		return fiber.NewError(fiber.StatusBadRequest, message)
	}
	request.Language = language

	result, err := h.usecase.Classify(c.UserContext(), request)
	if err != nil {
		if errors.Is(err, fasting.ErrInvalidRequest) {
			return fiber.NewError(fiber.StatusBadRequest, clientErrorMessage(err, fasting.ErrInvalidRequest))
		}
		return err
	}

	return c.JSON(fiber.Map{"data": result})
}

func requireSupportedLanguage(c *fiber.Ctx) error {
	if !requestLanguage(c).IsSupported() {
		return fiber.ErrNotFound
	}
	return c.Next()
}

func requestLanguage(c *fiber.Ctx) fasting.Language {
	return fasting.Language(c.Params("language"))
}

func clientErrorMessage(err, sentinel error) string {
	return strings.TrimPrefix(err.Error(), sentinel.Error()+": ")
}
