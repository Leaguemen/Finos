package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv              string
	HTTPPort            string
	DatabaseURL         string
	JWTSecret           string
	JWTTTL              time.Duration
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	CloudinaryFolder    string
}

func Load() (Config, error) {
	jwtTTL, err := time.ParseDuration(getEnv("JWT_TTL", "8h"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_TTL must be a valid duration: %w", err)
	}

	cfg := Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		HTTPPort:            getEnv("HTTP_PORT", "8080"),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:           strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTTTL:              jwtTTL,
		CloudinaryCloudName: strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME")),
		CloudinaryAPIKey:    strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY")),
		CloudinaryAPISecret: strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET")),
		CloudinaryFolder:    getEnv("CLOUDINARY_FOLDER", "finos/development/receipts"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len([]byte(cfg.JWTSecret)) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if cfg.JWTTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be greater than zero")
	}
	if cfg.CloudinaryCloudName == "" {
		return Config{}, fmt.Errorf("CLOUDINARY_CLOUD_NAME is required")
	}
	if cfg.CloudinaryAPIKey == "" {
		return Config{}, fmt.Errorf("CLOUDINARY_API_KEY is required")
	}
	if cfg.CloudinaryAPISecret == "" {
		return Config{}, fmt.Errorf("CLOUDINARY_API_SECRET is required")
	}

	port, err := strconv.Atoi(cfg.HTTPPort)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("HTTP_PORT must be a number between 1 and 65535")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
