package handlers

import (
	"github.com/mathornton01/arkheion/services"

	"github.com/gofiber/fiber/v2"
)

// SearchHandler handles full-text search via Meilisearch.
type SearchHandler struct {
	svc *services.Bundle
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(svc *services.Bundle) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Search handles GET /api/v1/search
//
// Query parameters:
//   q        - Search query string (required)
//   page     - Page number (default 1)
//   per_page - Results per page (default 20, max 100)
//   filter   - Meilisearch filter expression (optional)
//              e.g. filter=language="en"
//              e.g. filter=categories="Science" AND text_extracted=true
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("MISSING_QUERY",
			"Query parameter 'q' is required", fiber.StatusBadRequest))
	}

	page := max(1, c.QueryInt("page", 1))
	perPage := clampInt(c.QueryInt("per_page", 20), 1, 100)
	filter := c.Query("filter")

	results, err := h.svc.Meilisearch.Search(c.Context(), q, page, perPage, filter)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(apiError("SEARCH_FAILED",
			"Search service error: "+err.Error(), fiber.StatusBadGateway))
	}

	return c.JSON(results)
}
