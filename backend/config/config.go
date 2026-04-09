// Package config loads and validates the application configuration from
// environment variables. All fields are documented in .env.example.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the Arkheion backend.
type Config struct {
	// General
	BackendPort         int
	APIKeys             []string // Raw keys (hashed on first use)
	CORSAllowedOrigins  []string
	LogLevel            string

	// PostgreSQL
	PostgresHost              string
	PostgresPort              int
	PostgresDB                string
	PostgresUser              string
	PostgresPassword          string
	DatabaseURL               string
	DBMaxOpenConns            int
	DBMaxIdleConns            int
	DBConnMaxLifetimeMinutes  int

	// Meilisearch
	MeilisearchURL        string
	MeilisearchMasterKey  string
	MeilisearchBooksIndex string

	// Apache Tika
	TikaURL            string
	TikaTimeoutSeconds int

	// MinIO
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
	MinioPublicURL string

	// Webhooks
	WebhookDefaultSecret          string
	WebhookMaxRetries             int
	WebhookRetryInitialDelaySecs  int
	WebhookTimeoutSeconds         int

	// External APIs
	GoogleBooksAPIKey     string
	OpenLibraryBaseURL    string
	GoogleBooksBaseURL    string
}

// Load reads configuration from environment variables and validates required fields.
func Load() (*Config, error) {
	cfg := &Config{}
	var errs []string

	// General
	cfg.BackendPort = getEnvInt("BACKEND_PORT", 8080)
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.CORSAllowedOrigins = splitComma(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"))

	rawKeys := getEnv("ARKHEION_API_KEYS", "")
	if rawKeys == "" {
		errs = append(errs, "ARKHEION_API_KEYS is required")
	} else {
		cfg.APIKeys = splitComma(rawKeys)
	}

	// PostgreSQL
	cfg.DatabaseURL = getEnv("DATABASE_URL", "")
	if cfg.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	cfg.PostgresHost = getEnv("POSTGRES_HOST", "postgres")
	cfg.PostgresPort = getEnvInt("POSTGRES_PORT", 5432)
	cfg.PostgresDB = getEnv("POSTGRES_DB", "arkheion")
	cfg.PostgresUser = getEnv("POSTGRES_USER", "arkheion")
	cfg.PostgresPassword = getEnv("POSTGRES_PASSWORD", "")
	cfg.DBMaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	cfg.DBMaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", 5)
	cfg.DBConnMaxLifetimeMinutes = getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)

	// Meilisearch
	cfg.MeilisearchURL = getEnv("MEILISEARCH_URL", "http://meilisearch:7700")
	cfg.MeilisearchMasterKey = getEnv("MEILISEARCH_MASTER_KEY", "")
	if cfg.MeilisearchMasterKey == "" {
		errs = append(errs, "MEILISEARCH_MASTER_KEY is required")
	}
	cfg.MeilisearchBooksIndex = getEnv("MEILISEARCH_BOOKS_INDEX", "books")

	// Tika
	cfg.TikaURL = getEnv("TIKA_URL", "http://tika:9998")
	cfg.TikaTimeoutSeconds = getEnvInt("TIKA_TIMEOUT_SECONDS", 120)

	// MinIO
	cfg.MinioEndpoint = getEnv("MINIO_ENDPOINT", "minio:9000")
	cfg.MinioAccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinioSecretKey = getEnv("MINIO_SECRET_KEY", "")
	if cfg.MinioSecretKey == "" {
		errs = append(errs, "MINIO_SECRET_KEY is required")
	}
	cfg.MinioBucket = getEnv("MINIO_BUCKET", "arkheion")
	cfg.MinioUseSSL = getEnvBool("MINIO_USE_SSL", false)
	cfg.MinioPublicURL = getEnv("MINIO_PUBLIC_URL", "http://localhost:9000")

	// Webhooks
	cfg.WebhookDefaultSecret = getEnv("WEBHOOK_DEFAULT_SECRET", "")
	cfg.WebhookMaxRetries = getEnvInt("WEBHOOK_MAX_RETRIES", 3)
	cfg.WebhookRetryInitialDelaySecs = getEnvInt("WEBHOOK_RETRY_INITIAL_DELAY_SECONDS", 5)
	cfg.WebhookTimeoutSeconds = getEnvInt("WEBHOOK_TIMEOUT_SECONDS", 10)

	// External APIs
	cfg.GoogleBooksAPIKey = getEnv("GOOGLE_BOOKS_API_KEY", "")
	cfg.OpenLibraryBaseURL = getEnv("OPENLIBRARY_BASE_URL", "https://openlibrary.org")
	cfg.GoogleBooksBaseURL = getEnv("GOOGLE_BOOKS_BASE_URL", "https://www.googleapis.com/books/v1")

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return cfg, nil
}

// --- helpers -----------------------------------------------------------------

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return defaultVal
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// Validate checks that a config struct is internally consistent.
// Called internally by Load; exported for testing.
func (c *Config) Validate() error {
	if len(c.APIKeys) == 0 {
		return errors.New("at least one API key must be configured")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL must not be empty")
	}
	return nil
}
