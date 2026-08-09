package config

import (
	"fmt"
	"os"
	"strconv"
)

const defaultPort = 8080

type Config struct {
	AppName string
	Port    int
}

func Load() (Config, error) {
	cfg := Config{
		AppName: getEnv("APP_NAME", "puasa-sunnah-api"),
		Port:    defaultPort,
	}

	if value := os.Getenv("PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("PORT must be an integer from 1 to 65535")
		}
		cfg.Port = port
	}

	return cfg, nil
}

func (c Config) HTTPAddress() string {
	return fmt.Sprintf(":%d", c.Port)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
