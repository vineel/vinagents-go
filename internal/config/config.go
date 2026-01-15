package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env        string
	Port       int
	APIPrefix  string
	DatabaseURL string

	JWTSecret          string
	JWTExpiresIn       time.Duration
	JWTRefreshSecret   string
	JWTRefreshExpiresIn time.Duration

	CORSOrigins []string

	RateLimitWindowMS   int
	RateLimitMaxRequests int

	AnthropicAPIKey string
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Env:        getEnv("NODE_ENV", "development"),
		Port:       getEnvInt("PORT", 3000),
		APIPrefix:  getEnv("API_PREFIX", "/api/v1"),
		DatabaseURL: mustGetEnv("DATABASE_URL"),

		JWTSecret:          mustGetEnv("JWT_SECRET"),
		JWTExpiresIn:       parseDuration(getEnv("JWT_EXPIRES_IN", "1h")),
		JWTRefreshSecret:   mustGetEnv("JWT_REFRESH_SECRET"),
		JWTRefreshExpiresIn: parseDuration(getEnv("JWT_REFRESH_EXPIRES_IN", "7d")),

		CORSOrigins: strings.Split(getEnv("CORS_ORIGIN", "http://localhost:5173"), ","),

		RateLimitWindowMS:    getEnvInt("RATE_LIMIT_WINDOW_MS", 900000),
		RateLimitMaxRequests: getEnvInt("RATE_LIMIT_MAX_REQUESTS", 100),

		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
	}

	return cfg, nil
}

func LoadTest() (*Config, error) {
	// Load .env.test file
	_ = godotenv.Load(".env.test")
	return Load()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable not set: " + key)
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	// Handle formats like "1h", "7d", "30m"
	if strings.HasSuffix(s, "d") {
		days, _ := strconv.Atoi(strings.TrimSuffix(s, "d"))
		return time.Duration(days) * 24 * time.Hour
	}
	d, _ := time.ParseDuration(s)
	return d
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsTest() bool {
	return c.Env == "test"
}
