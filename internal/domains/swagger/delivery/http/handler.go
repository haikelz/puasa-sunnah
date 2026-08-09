package http

import (
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/gofiber/swagger"

	_ "puasa-sunnah/internal/domains/swagger/generated"
)

func Register(server *fiber.App) {
	server.Get("/swagger/*", fiberSwagger.New(fiberSwagger.Config{
		InstanceName:           "swagger",
		Title:                  "Puasa Sunnah API Documentation",
		DeepLinking:            true,
		DisplayRequestDuration: true,
		DocExpansion:           "list",
		Filter:                 fiberSwagger.FilterConfig{Enabled: true},
		PersistAuthorization:   true,
		TryItOutEnabled:        true,
	}))
}
