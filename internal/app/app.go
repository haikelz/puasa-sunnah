package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"puasa-sunnah/internal/config"
	fastinghttp "puasa-sunnah/internal/domains/fasting/delivery/http"
	fastingrepository "puasa-sunnah/internal/domains/fasting/repository"
	fastingusecase "puasa-sunnah/internal/domains/fasting/usecase"
	swaggerhttp "puasa-sunnah/internal/domains/swagger/delivery/http"
)

// The largest request is a small classify JSON body.
const bodyLimit = 16 * 1024

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
	server.Use(recover.New())

	server.Get("/health", health(cfg.AppName))
	swaggerhttp.Register(server)

	kemenagCalendar := fastingrepository.NewKemenagCalendar()
	ummAlQuraCalendar := fastingrepository.NewUmmAlQuraCalendar()
	fastingService, err := fastingusecase.NewFastingUsecase(kemenagCalendar, ummAlQuraCalendar)
	if err != nil {
		return nil, fmt.Errorf("create fasting usecase: %w", err)
	}
	fastingHandler, err := fastinghttp.NewHandler(fastingService)
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
		return c.JSON(HealthResponse{Status: "ok", Service: serviceName})
	}
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
	if errors.As(err, &fiberError) {
		status = fiberError.Code
		message = fiberError.Message
	}

	return c.Status(status).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    status,
			"message": message,
		},
	})
}
