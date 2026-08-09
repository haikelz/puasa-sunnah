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
// @description Production API for classifying and scheduling voluntary Islamic fasting days using the published Kementerian Agama RI Hijri calendar.
// @description Calendar output is planning information, not a fatwa. Ramadan, Shawwal, and Dhu al-Hijjah dates may be superseded by an official Sidang Isbat decision.
// @BasePath /
// @schemes http https
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	server := app.New(cfg)
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
