package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ExportHandler handles the bulk text export endpoint for LLM training pipelines.
type ExportHandler struct {
	db *pgxpool.Pool
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(db *pgxpool.Pool) *ExportHandler {
	return &ExportHandler{db: db}
}

// exportRecord is a single JSONL line in the export output.
type exportRecord struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Authors    []string `json:"authors"`
	Categories []string `json:"categories"`
	Language   string   `json:"language"`
	Text       string   `json:"text"`
}

// Export handles GET /api/v1/export
//
// Query parameters:
//   format   - Output format. Currently only "jsonl" is supported.
//   tag      - Filter by tag slug (optional)
//   category - Filter by category string (optional)
//   language - Filter by language code (optional, e.g. "en")
//
// Response: application/x-ndjson — newline-delimited JSON.
// Each line: {"id":"...","title":"...","authors":[...],"text":"..."}
//
// Only books with text_extracted=true are included.
// Books without extracted text are silently skipped.
//
// This endpoint is designed for large exports; it streams results row-by-row
// rather than loading everything into memory.
func (h *ExportHandler) Export(c *fiber.Ctx) error {
	format := c.Query("format", "jsonl")
	if format != "jsonl" {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("UNSUPPORTED_FORMAT",
			"Only format=jsonl is currently supported", fiber.StatusBadRequest))
	}

	tag := c.Query("tag")
	category := c.Query("category")
	language := c.Query("language")

	// Build query with optional filters
	conditions := []string{"b.text_extracted = TRUE", "b.extracted_text IS NOT NULL", "b.extracted_text != ''"}
	args := []interface{}{}
	idx := 1

	if tag != "" {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM book_tags bt
				JOIN tags t ON t.id = bt.tag_id
				WHERE bt.book_id = b.id AND t.slug = $%d
			)`, idx))
		args = append(args, tag)
		idx++
	}
	if category != "" {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(b.categories)", idx))
		args = append(args, category)
		idx++
	}
	if language != "" {
		conditions = append(conditions, fmt.Sprintf("b.language = $%d", idx))
		args = append(args, language)
		idx++
	}

	query := fmt.Sprintf(`
		SELECT b.id::text, b.title, b.categories, b.language, b.extracted_text
		FROM books b
		WHERE %s
		ORDER BY b.created_at`, strings.Join(conditions, " AND "))

	ctx := context.Background()
	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		log.Error().Err(err).Msg("Export: query failed")
		return fiber.ErrInternalServerError
	}
	defer rows.Close()

	// Collect author names for each book in a second pass
	// For large exports this is N+1 but acceptable — bulk export is a batch operation.
	c.Set("Content-Type", "application/x-ndjson")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Content-Type-Options", "nosniff")

	w := bufio.NewWriterSize(c.Response().BodyWriter(), 64*1024)

	count := 0
	for rows.Next() {
		var rec exportRecord
		var text string
		if err := rows.Scan(&rec.ID, &rec.Title, &rec.Categories, &rec.Language, &text); err != nil {
			log.Error().Err(err).Msg("Export: scan failed")
			continue
		}
		rec.Text = text

		// Fetch authors for this book
		authors, err := h.fetchAuthors(ctx, rec.ID)
		if err != nil {
			log.Warn().Err(err).Str("book_id", rec.ID).Msg("Export: failed to fetch authors")
		}
		rec.Authors = authors

		line, err := json.Marshal(rec)
		if err != nil {
			log.Error().Err(err).Msg("Export: JSON marshal failed")
			continue
		}
		fmt.Fprintf(w, "%s\n", line)
		count++
	}

	if err := w.Flush(); err != nil {
		log.Error().Err(err).Msg("Export: flush failed")
	}

	log.Info().Int("records", count).Str("tag", tag).Str("category", category).Msg("Export complete")
	return nil
}

func (h *ExportHandler) fetchAuthors(ctx context.Context, bookID string) ([]string, error) {
	rows, err := h.db.Query(ctx, `
		SELECT a.name
		FROM authors a
		JOIN book_authors ba ON ba.author_id = a.id
		WHERE ba.book_id = $1::uuid
		ORDER BY ba.sort_order`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}
