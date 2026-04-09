// Package api wires together the Fiber application, middleware, and all HTTP handlers.
package api

import (
	"github.com/mathornton01/arkheion/api/handlers"
	"github.com/mathornton01/arkheion/api/middleware"
	"github.com/mathornton01/arkheion/config"
	"github.com/mathornton01/arkheion/services"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes sets up all middleware and routes on the Fiber app.
func RegisterRoutes(app *fiber.App, cfg *config.Config, pool *pgxpool.Pool, svc *services.Bundle) {
	// Global middleware
	app.Use(middleware.RequestLogger())
	app.Use(middleware.CORS(cfg))
	app.Use(middleware.Recover())

	// Health check — no auth required
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "arkheion-backend"})
	})

	// OpenAPI spec — no auth required
	app.Static("/api/v1/docs", "./docs")

	// --- Authenticated API routes ---
	v1 := app.Group("/api/v1", middleware.APIKeyAuth(cfg))

	// Books
	bookHandler := handlers.NewBooksHandler(pool, svc)
	books := v1.Group("/books")
	books.Get("/", bookHandler.ListBooks)
	books.Post("/", bookHandler.CreateBook)
	books.Get("/:id", bookHandler.GetBook)
	books.Put("/:id", bookHandler.UpdateBook)
	books.Delete("/:id", bookHandler.DeleteBook)
	books.Post("/:id/upload", bookHandler.UploadFile)
	books.Get("/:id/download", bookHandler.DownloadFile)

	// Catalog (ISBN/barcode lookup)
	catalogHandler := handlers.NewCatalogHandler(svc)
	catalog := v1.Group("/catalog")
	catalog.Get("/isbn/:isbn", catalogHandler.LookupISBN)
	catalog.Post("/scan", catalogHandler.ScanBarcode) // accepts raw barcode string in body

	// Full-text search
	searchHandler := handlers.NewSearchHandler(svc)
	v1.Get("/search", searchHandler.Search)

	// Bulk export (for LLM training / Golem)
	exportHandler := handlers.NewExportHandler(pool)
	v1.Get("/export", exportHandler.Export)

	// Webhooks
	webhookHandler := handlers.NewWebhooksHandler(pool)
	webhooks := v1.Group("/webhooks")
	webhooks.Get("/", webhookHandler.ListWebhooks)
	webhooks.Post("/", webhookHandler.CreateWebhook)
	webhooks.Get("/:id", webhookHandler.GetWebhook)
	webhooks.Delete("/:id", webhookHandler.DeleteWebhook)
	webhooks.Put("/:id/activate", webhookHandler.ActivateWebhook)
	webhooks.Put("/:id/deactivate", webhookHandler.DeactivateWebhook)

	// 404 fallback
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(errorResponse(
			"NOT_FOUND",
			"The requested endpoint does not exist",
			fiber.StatusNotFound,
		))
	})
}

// ErrorHandler is the global Fiber error handler. It converts fiber.Error and
// any other error types into the standard Arkheion error JSON format.
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "An internal server error occurred"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		msg = e.Message
	}

	return c.Status(code).JSON(errorResponse("SERVER_ERROR", msg, code))
}

func errorResponse(code, message string, status int) fiber.Map {
	return fiber.Map{
		"error": fiber.Map{
			"code":    code,
			"message": message,
			"status":  status,
		},
	}
}
