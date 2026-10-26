package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"puasa-sunnah/internal/app"
	"puasa-sunnah/internal/config"
)

// @title Puasa Sunnah API
// @version 1.0
// @description Production API for classifying and scheduling voluntary Islamic fasting days using an explicitly selected Indonesian or Umm al-Qura Hijri calendar.
// @description Calendar output is planning information, not a fatwa. Dates can differ between accepted authorities and localities.
// @BasePath /
// @schemes http https
func main() {
	// One JSON object per line lets Vercel's log search filter on request_id and status.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	server, err := app.New(cfg)
	if err != nil {
		slog.Error("create application", "error", err)
		os.Exit(1)
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "address", cfg.HTTPAddress())
		errCh <- server.Listen(cfg.HTTPAddress())
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.ShutdownWithContext(shutdownCtx); err != nil {
			slog.Error("graceful shutdown", "error", err)
			os.Exit(1)
		}
	}
}
