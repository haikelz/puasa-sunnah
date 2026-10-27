package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"puasa-sunnah/internal/config"
	fastingHTTP "puasa-sunnah/internal/domains/fasting/delivery/http"
	fastingRepository "puasa-sunnah/internal/domains/fasting/repository"
	fastingUsecase "puasa-sunnah/internal/domains/fasting/usecase"
	swaggerHTTP "puasa-sunnah/internal/domains/swagger/delivery/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

const (
	// The largest request is a small classify JSON body.
	bodyLimit = 16 * 1024

	// A full-year schedule computes in milliseconds; the deadline bounds the use-case loop.
	requestTimeout = 5 * time.Second

	// Health checks and errors must never come from a cache.
	cacheNoStore = "no-store"
)

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"puasa-sunnah-api"`
}

func New(cfg config.Config) (*fiber.App, error) {
	server := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler,
		BodyLimit:    bodyLimit,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	})
	server.Use(
		requestid.New(),
		accessLog,
		recover.New(recover.Config{EnableStackTrace: true}),
		withRequestTimeout,
	)

	server.Get("/health", health(cfg.AppName))
	swaggerHTTP.Register(server)

	kemenagCalendar, err := fastingRepository.NewKemenagCalendarRepository()
	if err != nil {
		return nil, fmt.Errorf("load Kemenag calendar: %w", err)
	}

	ummAlQuraCalendar := fastingRepository.NewUmmAlQuraCalendarRepository()
	fastingService, err := fastingUsecase.NewFastingUsecase(kemenagCalendar, ummAlQuraCalendar)
	if err != nil {
		return nil, fmt.Errorf("create fasting usecase: %w", err)
	}

	fastingHandler, err := fastingHTTP.NewFastingHandler(fastingService)
	if err != nil {
		return nil, fmt.Errorf("create fasting HTTP handler: %w", err)
	}

	fastingHandler.Register(server.Group("/api/:language/fasting"))

	return server, nil
}

// health godoc
// @Summary Check service health
// @Description Returns the API process health and service name.
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func health(serviceName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, cacheNoStore)
		return c.JSON(HealthResponse{
			Status:  "ok",
			Service: serviceName,
		})
	}
}

// accessLog renders handler errors itself so the logged status is the one sent.
func accessLog(c *fiber.Ctx) error {
	start := time.Now()

	if err := c.Next(); err != nil {
		if handlerErr := c.App().Config().ErrorHandler(c, err); handlerErr != nil {
			slog.Error(
				"render error response",
				"request_id", c.Locals("requestid"),
				"error", handlerErr,
			)
			// The response already failed to render; the log above records it.
			_ = c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	slog.Info(
		"http request",
		"request_id", c.Locals("requestid"),
		"method", c.Method(),
		"route", c.Route().Path,
		"status", c.Response().StatusCode(),
		"latency", time.Since(start),
	)

	return nil
}

func withRequestTimeout(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), requestTimeout)
	defer cancel()

	c.SetUserContext(ctx)
	return c.Next()
}

func errorHandler(c *fiber.Ctx, err error) error {
	// Fiber also calls this for requests rejected before routing (oversized
	// body, malformed headers); c.Params panics there, so read the path.
	isIndonesian := strings.HasPrefix(c.Path(), "/api/id/")
	status := fiber.StatusInternalServerError
	message := "internal server error"
	if isIndonesian {
		message = "kesalahan internal server"
	}

	var fiberError *fiber.Error
	switch {
	case errors.As(err, &fiberError):
		status = fiberError.Code
		message = fiberError.Message
	case errors.Is(err, context.DeadlineExceeded):
		status = fiber.StatusServiceUnavailable
		message = "request timed out"
		if isIndonesian {
			message = "permintaan melebihi batas waktu"
		}
	}

	if status >= fiber.StatusInternalServerError {
		slog.Error(
			"request failed",
			"request_id", c.Locals("requestid"),
			"method", c.Method(),
			"route", c.Route().Path,
			"status", status,
			"error", err,
		)
	}

	c.Set(fiber.HeaderCacheControl, cacheNoStore)
	return c.Status(status).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    status,
			"message": message,
		},
	})
}
