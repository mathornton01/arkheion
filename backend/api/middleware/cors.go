package middleware

import (
	"strings"

	"github.com/mathornton01/arkheion/config"

	"github.com/gofiber/fiber/v2"
)

// CORS returns a Fiber middleware that adds CORS headers based on configuration.
// Only origins listed in CORS_ALLOWED_ORIGINS are permitted.
func CORS(cfg *config.Config) fiber.Handler {
	allowedSet := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, o := range cfg.CORSAllowedOrigins {
		allowedSet[strings.TrimRight(o, "/")] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")
		if origin == "" {
			return c.Next()
		}

		if _, ok := allowedSet[origin]; ok {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
			c.Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization, Accept")
			c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Set("Access-Control-Max-Age", "3600")
		}

		// Handle preflight
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

// RequestLogger returns a minimal structured request logging middleware.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		// Zerolog is used in main; here we just pass through.
		// A production implementation would log method, path, status, latency.
		return err
	}
}

// Recover returns a middleware that recovers from panics and returns a 500.
func Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    "PANIC",
						"message": "An unexpected error occurred",
						"status":  fiber.StatusInternalServerError,
					},
				})
			}
		}()
		return c.Next()
	}
}
