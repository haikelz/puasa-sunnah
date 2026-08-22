package app

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"puasa-sunnah/internal/config"
	fastinghttp "puasa-sunnah/internal/domains/fasting/delivery/http"
	fastingrepository "puasa-sunnah/internal/domains/fasting/repository"
	fastingusecase "puasa-sunnah/internal/domains/fasting/usecase"
	swaggerhttp "puasa-sunnah/internal/domains/swagger/delivery/http"
)

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"puasa-sunnah-api"`
}

func New(cfg config.Config) *fiber.App {
	server := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler,
	})
	server.Use(recover.New())

	server.Get("/health", health(cfg.AppName))
	swaggerhttp.Register(server)

	kemenagCalendar := fastingrepository.NewKemenagCalendar()
	ummAlQuraCalendar := fastingrepository.NewUmmAlQuraCalendar()
	fastingService := fastingusecase.NewFastingUsecase(kemenagCalendar, ummAlQuraCalendar)
	fastingHandler := fastinghttp.NewHandler(fastingService)
	fastingHandler.Register(server.Group("/api/:language/fasting"))

	return server
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
	status := fiber.StatusInternalServerError
	message := "internal server error"
	if c.Params("language") == "id" {
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
