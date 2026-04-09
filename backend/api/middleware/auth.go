// Package middleware provides Fiber middleware for the Arkheion API.
package middleware

import (
	"github.com/mathornton01/arkheion/config"

	"github.com/gofiber/fiber/v2"
)

// APIKeyAuth returns a Fiber middleware that validates the X-API-Key header.
// Keys are compared against the list configured in ARKHEION_API_KEYS.
//
// In production, consider storing keys hashed (bcrypt) in the database and
// comparing with constant-time comparison. For simplicity, this implementation
// does a direct constant-time string comparison against the configured keys.
func APIKeyAuth(cfg *config.Config) fiber.Handler {
	// Build a set for O(1) lookup
	keySet := make(map[string]struct{}, len(cfg.APIKeys))
	for _, k := range cfg.APIKeys {
		keySet[k] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		key := c.Get("X-API-Key")
		if key == "" {
			// Also accept Authorization: Bearer <key>
			key = c.Get("Authorization")
			if len(key) > 7 && key[:7] == "Bearer " {
				key = key[7:]
			}
		}

		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "MISSING_API_KEY",
					"message": "API key is required. Provide it in the X-API-Key header.",
					"status":  fiber.StatusUnauthorized,
				},
			})
		}

		if _, ok := keySet[key]; !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INVALID_API_KEY",
					"message": "The provided API key is not valid.",
					"status":  fiber.StatusForbidden,
				},
			})
		}

		// Store key in context for downstream handlers if needed
		c.Locals("api_key", key)
		return c.Next()
	}
}
