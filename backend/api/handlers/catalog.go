package handlers

import (
	"strings"

	"github.com/mathornton01/arkheion/services"

	"github.com/gofiber/fiber/v2"
)

// CatalogHandler handles ISBN and barcode lookup routes.
type CatalogHandler struct {
	svc *services.Bundle
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(svc *services.Bundle) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

// LookupISBN handles GET /api/v1/catalog/isbn/:isbn
// Queries OpenLibrary first, falls back to Google Books.
// Returns normalized book metadata ready to populate the "Add Book" form.
func (h *CatalogHandler) LookupISBN(c *fiber.Ctx) error {
	isbn := normalizeISBN(c.Params("isbn"))
	if isbn == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ISBN", "ISBN must not be empty", fiber.StatusBadRequest))
	}

	result, err := h.svc.ISBN.Lookup(c.Context(), isbn)
	if err != nil {
		if err == services.ErrISBNNotFound {
			return c.Status(fiber.StatusNotFound).JSON(apiError("ISBN_NOT_FOUND",
				"No book found for ISBN: "+isbn, fiber.StatusNotFound))
		}
		return c.Status(fiber.StatusBadGateway).JSON(apiError("LOOKUP_FAILED",
			"ISBN lookup service unavailable: "+err.Error(), fiber.StatusBadGateway))
	}

	return c.JSON(result)
}

// ScanBarcode handles POST /api/v1/catalog/scan
// Accepts a JSON body with a "barcode" field (the raw string from ZXing).
// Normalizes it to ISBN format and performs the same lookup as LookupISBN.
func (h *CatalogHandler) ScanBarcode(c *fiber.Ctx) error {
	var body struct {
		Barcode string `json:"barcode"`
	}
	if err := c.BodyParser(&body); err != nil || body.Barcode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("MISSING_BARCODE",
			"Request body must include a 'barcode' field", fiber.StatusBadRequest))
	}

	isbn := normalizeISBN(body.Barcode)
	if isbn == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_BARCODE",
			"Barcode value could not be parsed as an ISBN", fiber.StatusBadRequest))
	}

	result, err := h.svc.ISBN.Lookup(c.Context(), isbn)
	if err != nil {
		if err == services.ErrISBNNotFound {
			return c.Status(fiber.StatusNotFound).JSON(apiError("ISBN_NOT_FOUND",
				"No book found for barcode: "+isbn, fiber.StatusNotFound))
		}
		return c.Status(fiber.StatusBadGateway).JSON(apiError("LOOKUP_FAILED",
			"ISBN lookup service unavailable: "+err.Error(), fiber.StatusBadGateway))
	}

	return c.JSON(result)
}

// normalizeISBN strips dashes and spaces from an ISBN string.
func normalizeISBN(s string) string {
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)
	// Basic sanity: ISBNs are 10 or 13 digits
	if len(s) != 10 && len(s) != 13 {
		return ""
	}
	return s
}
