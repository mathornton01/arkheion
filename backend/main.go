package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mathornton01/arkheion/api"
	"github.com/mathornton01/arkheion/config"
	"github.com/mathornton01/arkheion/db"
	"github.com/mathornton01/arkheion/services"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Version is set at build time via -ldflags="-X main.Version=x.y.z"
var Version = "dev"

func main() {
	// Load .env file if present (development convenience; in production env vars
	// are injected by Docker Compose).
	if err := godotenv.Load(); err != nil {
		// Not fatal — .env is optional in containerized environments.
		log.Debug().Msg("No .env file found — using environment variables")
	}

	// Load and validate configuration.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Configure structured logging.
	setupLogger(cfg.LogLevel)

	log.Info().
		Str("version", Version).
		Str("log_level", cfg.LogLevel).
		Msg("Starting Arkheion backend")

	// Connect to PostgreSQL.
	pool, err := db.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer pool.Close()
	log.Info().Str("host", cfg.PostgresHost).Msg("Connected to PostgreSQL")

	// Initialize all services.
	svcBundle, err := services.NewBundle(cfg, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize services")
	}
	log.Info().Msg("All services initialized")

	// Build the Fiber application.
	app := fiber.New(fiber.Config{
		AppName:       "Arkheion v" + Version,
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  60 * time.Second,
		IdleTimeout:   120 * time.Second,
		BodyLimit:     500 * 1024 * 1024, // 500 MB — allow large book uploads
		ErrorHandler:  api.ErrorHandler,
	})

	// Register all routes.
	api.RegisterRoutes(app, cfg, pool, svcBundle)

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start the server in a goroutine.
	go func() {
		addr := fmt.Sprintf(":%d", cfg.BackendPort)
		log.Info().Str("address", addr).Msg("HTTP server listening")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for shutdown signal.
	<-quit
	log.Info().Msg("Shutdown signal received — draining connections")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("Error during graceful shutdown")
	}

	log.Info().Msg("Arkheion backend stopped")
}

// setupLogger configures zerolog based on the log level string from config.
func setupLogger(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	// Pretty-print in development (when stdout is a terminal).
	if fi, _ := os.Stdout.Stat(); (fi.Mode() & os.ModeCharDevice) != 0 {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		// JSON output for production / Docker.
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}
}
