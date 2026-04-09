// Package handlers contains the HTTP request handlers for the Arkheion API.
package handlers

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mathornton01/arkheion/models"
	"github.com/mathornton01/arkheion/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// BooksHandler handles all /api/v1/books routes.
type BooksHandler struct {
	db  *pgxpool.Pool
	svc *services.Bundle
}

// NewBooksHandler creates a new BooksHandler.
func NewBooksHandler(db *pgxpool.Pool, svc *services.Bundle) *BooksHandler {
	return &BooksHandler{db: db, svc: svc}
}

// ListBooks handles GET /api/v1/books
// Supports pagination and filtering by tag, category, language, text_extracted.
func (h *BooksHandler) ListBooks(c *fiber.Ctx) error {
	ctx := context.Background()

	page := max(1, c.QueryInt("page", 1))
	perPage := clampInt(c.QueryInt("per_page", 20), 1, 100)
	offset := (page - 1) * perPage

	filter := models.BookFilter{
		Tag:      c.Query("tag"),
		Category: c.Query("category"),
		Language: c.Query("language"),
		Query:    c.Query("q"),
		Page:     page,
		PerPage:  perPage,
	}
	if te := c.Query("text_extracted"); te != "" {
		b := te == "true"
		filter.TextExtracted = &b
	}

	// Build dynamic query
	where, args := buildBookWhereClause(filter)

	countQuery := fmt.Sprintf(`SELECT COUNT(DISTINCT b.id) FROM books b
		LEFT JOIN book_tags bt ON bt.book_id = b.id
		LEFT JOIN tags t ON t.id = bt.tag_id
		%s`, where)

	var total int
	if err := h.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		log.Error().Err(err).Msg("ListBooks: count query failed")
		return fiber.ErrInternalServerError
	}

	args = append(args, perPage, offset)
	listQuery := fmt.Sprintf(`
		SELECT DISTINCT b.id, b.isbn, b.title, b.subtitle, b.publisher, b.published_date,
		       b.description, b.page_count, b.categories, b.language, b.cover_url,
		       b.file_path, b.file_type, b.file_size_bytes, b.text_extracted,
		       b.physical_location, b.notes, b.created_at, b.updated_at
		FROM books b
		LEFT JOIN book_tags bt ON bt.book_id = b.id
		LEFT JOIN tags t ON t.id = bt.tag_id
		%s
		ORDER BY b.created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := h.db.Query(ctx, listQuery, args...)
	if err != nil {
		log.Error().Err(err).Msg("ListBooks: list query failed")
		return fiber.ErrInternalServerError
	}
	defer rows.Close()

	books := make([]models.Book, 0, perPage)
	for rows.Next() {
		var b models.Book
		if err := scanBook(rows, &b); err != nil {
			log.Error().Err(err).Msg("ListBooks: scan failed")
			return fiber.ErrInternalServerError
		}
		books = append(books, b)
	}

	// Populate authors and tags for each book
	for i := range books {
		if err := h.loadBookRelations(ctx, &books[i]); err != nil {
			log.Warn().Err(err).Str("book_id", books[i].ID.String()).Msg("Failed to load book relations")
		}
	}

	totalPages := (total + perPage - 1) / perPage

	return c.JSON(models.BookListResponse{
		Data: books,
		Pagination: models.Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetBook handles GET /api/v1/books/:id
func (h *BooksHandler) GetBook(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Book ID must be a valid UUID", fiber.StatusBadRequest))
	}

	book, err := h.getBookByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apiError("BOOK_NOT_FOUND",
				fmt.Sprintf("No book found with id: %s", id), fiber.StatusNotFound))
		}
		log.Error().Err(err).Str("id", id.String()).Msg("GetBook: query failed")
		return fiber.ErrInternalServerError
	}

	if err := h.loadBookRelations(ctx, book); err != nil {
		log.Warn().Err(err).Msg("GetBook: failed to load relations")
	}

	return c.JSON(book)
}

// CreateBook handles POST /api/v1/books
func (h *BooksHandler) CreateBook(c *fiber.Ctx) error {
	ctx := context.Background()

	var req models.CreateBookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_BODY", "Invalid JSON body", fiber.StatusBadRequest))
	}

	if strings.TrimSpace(req.Title) == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(apiError("VALIDATION_ERROR", "title is required", fiber.StatusUnprocessableEntity))
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	defer tx.Rollback(ctx)

	bookID := uuid.New()
	var publishedDate *time.Time
	if req.PublishedDate != "" {
		t, err := time.Parse("2006-01-02", req.PublishedDate)
		if err == nil {
			publishedDate = &t
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO books (id, isbn, title, subtitle, publisher, published_date, description,
		    page_count, categories, language, cover_url, physical_location, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		bookID, nullStr(req.ISBN), req.Title, nullStr(req.Subtitle),
		nullStr(req.Publisher), publishedDate, nullStr(req.Description),
		req.PageCount, req.Categories, strOrDefault(req.Language, "en"),
		nullStr(req.CoverURL), nullStr(req.PhysicalLocation), nullStr(req.Notes),
	)
	if err != nil {
		log.Error().Err(err).Msg("CreateBook: insert failed")
		return fiber.ErrInternalServerError
	}

	// Upsert and link authors
	if err := upsertAndLinkAuthors(ctx, tx, bookID, req.Authors); err != nil {
		log.Error().Err(err).Msg("CreateBook: author linking failed")
		return fiber.ErrInternalServerError
	}

	// Upsert and link tags
	if err := upsertAndLinkTags(ctx, tx, bookID, req.Tags); err != nil {
		log.Error().Err(err).Msg("CreateBook: tag linking failed")
		return fiber.ErrInternalServerError
	}

	if err := tx.Commit(ctx); err != nil {
		return fiber.ErrInternalServerError
	}

	book, err := h.getBookByID(ctx, bookID)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if err := h.loadBookRelations(ctx, book); err != nil {
		log.Warn().Err(err).Msg("CreateBook: failed to load relations")
	}

	// Sync to Meilisearch asynchronously
	go func() {
		if err := h.svc.Meilisearch.IndexBook(book); err != nil {
			log.Error().Err(err).Str("book_id", bookID.String()).Msg("Failed to index book in Meilisearch")
		}
	}()

	// Dispatch webhook
	go h.svc.Webhook.Dispatch(models.EventBookCreated, book)

	return c.Status(fiber.StatusCreated).JSON(book)
}

// UpdateBook handles PUT /api/v1/books/:id
func (h *BooksHandler) UpdateBook(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Book ID must be a valid UUID", fiber.StatusBadRequest))
	}

	// Verify book exists
	if _, err := h.getBookByID(ctx, id); err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apiError("BOOK_NOT_FOUND",
				fmt.Sprintf("No book found with id: %s", id), fiber.StatusNotFound))
		}
		return fiber.ErrInternalServerError
	}

	var req models.UpdateBookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_BODY", "Invalid JSON body", fiber.StatusBadRequest))
	}

	// Build dynamic SET clause from non-nil fields
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	addSet := func(col string, val interface{}) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}

	if req.ISBN != nil         { addSet("isbn", *req.ISBN) }
	if req.Title != nil        { addSet("title", *req.Title) }
	if req.Subtitle != nil     { addSet("subtitle", *req.Subtitle) }
	if req.Publisher != nil    { addSet("publisher", *req.Publisher) }
	if req.Description != nil  { addSet("description", *req.Description) }
	if req.PageCount != nil    { addSet("page_count", *req.PageCount) }
	if req.Language != nil     { addSet("language", *req.Language) }
	if req.CoverURL != nil     { addSet("cover_url", *req.CoverURL) }
	if req.PhysicalLocation != nil { addSet("physical_location", *req.PhysicalLocation) }
	if req.Notes != nil        { addSet("notes", *req.Notes) }
	if req.Categories != nil   { addSet("categories", req.Categories) }
	if req.PublishedDate != nil {
		if t, err := time.Parse("2006-01-02", *req.PublishedDate); err == nil {
			addSet("published_date", t)
		}
	}

	if len(setClauses) > 1 {
		args = append(args, id)
		query := fmt.Sprintf("UPDATE books SET %s WHERE id = $%d",
			strings.Join(setClauses, ", "), argIdx)
		if _, err := h.db.Exec(ctx, query, args...); err != nil {
			log.Error().Err(err).Msg("UpdateBook: update failed")
			return fiber.ErrInternalServerError
		}
	}

	// Update authors and tags if provided
	if req.Authors != nil {
		if err := replaceBookAuthors(ctx, h.db, id, req.Authors); err != nil {
			log.Error().Err(err).Msg("UpdateBook: author replace failed")
		}
	}
	if req.Tags != nil {
		if err := replaceBookTags(ctx, h.db, id, req.Tags); err != nil {
			log.Error().Err(err).Msg("UpdateBook: tag replace failed")
		}
	}

	book, err := h.getBookByID(ctx, id)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if err := h.loadBookRelations(ctx, book); err != nil {
		log.Warn().Err(err).Msg("UpdateBook: failed to load relations")
	}

	go func() {
		if err := h.svc.Meilisearch.IndexBook(book); err != nil {
			log.Error().Err(err).Str("book_id", id.String()).Msg("Failed to re-index book in Meilisearch")
		}
	}()
	go h.svc.Webhook.Dispatch(models.EventBookUpdated, book)

	return c.JSON(book)
}

// DeleteBook handles DELETE /api/v1/books/:id
func (h *BooksHandler) DeleteBook(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Book ID must be a valid UUID", fiber.StatusBadRequest))
	}

	book, err := h.getBookByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apiError("BOOK_NOT_FOUND",
				fmt.Sprintf("No book found with id: %s", id), fiber.StatusNotFound))
		}
		return fiber.ErrInternalServerError
	}

	// Delete file from MinIO if it exists
	if book.FilePath != "" {
		go func() {
			if err := h.svc.MinIO.DeleteFile(book.FilePath); err != nil {
				log.Error().Err(err).Str("path", book.FilePath).Msg("Failed to delete book file from MinIO")
			}
		}()
	}

	if _, err := h.db.Exec(ctx, "DELETE FROM books WHERE id = $1", id); err != nil {
		log.Error().Err(err).Msg("DeleteBook: delete failed")
		return fiber.ErrInternalServerError
	}

	// Remove from Meilisearch
	go func() {
		if err := h.svc.Meilisearch.DeleteBook(id.String()); err != nil {
			log.Error().Err(err).Str("book_id", id.String()).Msg("Failed to delete book from Meilisearch")
		}
	}()

	go h.svc.Webhook.Dispatch(models.EventBookDeleted, fiber.Map{"id": id.String()})

	return c.SendStatus(fiber.StatusNoContent)
}

// UploadFile handles POST /api/v1/books/:id/upload
// Accepts multipart/form-data with a "file" field.
func (h *BooksHandler) UploadFile(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Book ID must be a valid UUID", fiber.StatusBadRequest))
	}

	book, err := h.getBookByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apiError("BOOK_NOT_FOUND",
				fmt.Sprintf("No book found with id: %s", id), fiber.StatusNotFound))
		}
		return fiber.ErrInternalServerError
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("MISSING_FILE", "Multipart field 'file' is required", fiber.StatusBadRequest))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]string{
		".pdf":  "pdf",
		".epub": "epub",
		".txt":  "txt",
		".docx": "docx",
	}
	fileType, ok := allowedExts[ext]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("UNSUPPORTED_FILE_TYPE",
			fmt.Sprintf("Unsupported file type: %s. Allowed: pdf, epub, txt, docx", ext),
			fiber.StatusBadRequest))
	}

	// Open multipart file for streaming
	src, err := file.Open()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	defer src.Close()

	objectKey := fmt.Sprintf("books/%s/file%s", id.String(), ext)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	size, err := h.svc.MinIO.UploadFile(ctx, objectKey, src, file.Size, contentType)
	if err != nil {
		log.Error().Err(err).Msg("UploadFile: MinIO upload failed")
		return c.Status(fiber.StatusInternalServerError).JSON(apiError("UPLOAD_FAILED", "File upload failed", fiber.StatusInternalServerError))
	}

	// Update book record with file info
	if _, err := h.db.Exec(ctx, `
		UPDATE books SET file_path = $1, file_type = $2, file_size_bytes = $3, updated_at = NOW()
		WHERE id = $4`, objectKey, fileType, size, id); err != nil {
		log.Error().Err(err).Msg("UploadFile: update book failed")
		return fiber.ErrInternalServerError
	}

	book.FilePath = objectKey
	book.FileType = fileType
	book.FileSizeBytes = size

	// Trigger text extraction asynchronously
	go h.triggerTextExtraction(book)

	return c.JSON(fiber.Map{
		"message":         "File uploaded successfully. Text extraction started.",
		"file_path":       objectKey,
		"file_type":       fileType,
		"file_size_bytes": size,
	})
}

// DownloadFile handles GET /api/v1/books/:id/download
// Proxies the file download from MinIO through the backend (auth-gated).
func (h *BooksHandler) DownloadFile(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Book ID must be a valid UUID", fiber.StatusBadRequest))
	}

	book, err := h.getBookByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apiError("BOOK_NOT_FOUND",
				fmt.Sprintf("No book found with id: %s", id), fiber.StatusNotFound))
		}
		return fiber.ErrInternalServerError
	}

	if book.FilePath == "" {
		return c.Status(fiber.StatusNotFound).JSON(apiError("NO_FILE", "This book has no uploaded file", fiber.StatusNotFound))
	}

	reader, size, contentType, err := h.svc.MinIO.DownloadFile(ctx, book.FilePath)
	if err != nil {
		log.Error().Err(err).Str("path", book.FilePath).Msg("DownloadFile: MinIO download failed")
		return fiber.ErrInternalServerError
	}
	defer reader.Close()

	filename := sanitizeFilename(book.Title) + "." + book.FileType
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", strconv.FormatInt(size, 10))
	return c.SendStream(reader, int(size))
}

// triggerTextExtraction downloads a file from MinIO, extracts text via Tika,
// stores the result in PostgreSQL, updates Meilisearch, and fires a webhook.
// Runs in a background goroutine.
func (h *BooksHandler) triggerTextExtraction(book *models.Book) {
	ctx := context.Background()

	log.Info().Str("book_id", book.ID.String()).Str("file_type", book.FileType).Msg("Starting text extraction")

	reader, _, _, err := h.svc.MinIO.DownloadFile(ctx, book.FilePath)
	if err != nil {
		log.Error().Err(err).Msg("Text extraction: failed to download file from MinIO")
		return
	}
	defer reader.Close()

	text, err := h.svc.Tika.ExtractText(ctx, reader, book.FileType)
	if err != nil {
		log.Error().Err(err).Msg("Text extraction: Tika failed")
		return
	}

	if _, err := h.db.Exec(ctx, `
		UPDATE books SET extracted_text = $1, text_extracted = TRUE, updated_at = NOW()
		WHERE id = $2`, text, book.ID); err != nil {
		log.Error().Err(err).Msg("Text extraction: failed to store extracted text")
		return
	}

	book.ExtractedText = text
	book.TextExtracted = true

	if err := h.svc.Meilisearch.IndexBook(book); err != nil {
		log.Error().Err(err).Msg("Text extraction: failed to update Meilisearch index")
	}

	h.svc.Webhook.Dispatch(models.EventBookTextExtracted, book)
	log.Info().Str("book_id", book.ID.String()).Int("text_bytes", len(text)).Msg("Text extraction complete")
}

// --- private helpers ---------------------------------------------------------

func (h *BooksHandler) getBookByID(ctx context.Context, id uuid.UUID) (*models.Book, error) {
	var b models.Book
	row := h.db.QueryRow(ctx, `
		SELECT id, isbn, title, subtitle, publisher, published_date, description, page_count,
		       categories, language, cover_url, file_path, file_type, file_size_bytes,
		       text_extracted, physical_location, notes, created_at, updated_at
		FROM books WHERE id = $1`, id)
	if err := scanBook(row, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (h *BooksHandler) loadBookRelations(ctx context.Context, b *models.Book) error {
	// Authors
	rows, err := h.db.Query(ctx, `
		SELECT a.id, a.name, a.created_at
		FROM authors a
		JOIN book_authors ba ON ba.author_id = a.id
		WHERE ba.book_id = $1
		ORDER BY ba.sort_order`, b.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a models.Author
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			return err
		}
		b.Authors = append(b.Authors, a)
	}

	// Tags
	tagRows, err := h.db.Query(ctx, `
		SELECT t.id, t.name, t.slug, t.created_at
		FROM tags t
		JOIN book_tags bt ON bt.tag_id = t.id
		WHERE bt.book_id = $1
		ORDER BY t.name`, b.ID)
	if err != nil {
		return err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var t models.Tag
		if err := tagRows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return err
		}
		b.Tags = append(b.Tags, t)
	}

	return nil
}

// scanBook scans a book row from either pgx.Row or pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

func scanBook(row scannable, b *models.Book) error {
	return row.Scan(
		&b.ID, &b.ISBN, &b.Title, &b.Subtitle, &b.Publisher, &b.PublishedDate,
		&b.Description, &b.PageCount, &b.Categories, &b.Language, &b.CoverURL,
		&b.FilePath, &b.FileType, &b.FileSizeBytes, &b.TextExtracted,
		&b.PhysicalLocation, &b.Notes, &b.CreatedAt, &b.UpdatedAt,
	)
}

func buildBookWhereClause(f models.BookFilter) (string, []interface{}) {
	conditions := []string{}
	args := []interface{}{}
	idx := 1

	if f.Language != "" {
		conditions = append(conditions, fmt.Sprintf("b.language = $%d", idx))
		args = append(args, f.Language)
		idx++
	}
	if f.Category != "" {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(b.categories)", idx))
		args = append(args, f.Category)
		idx++
	}
	if f.TextExtracted != nil {
		conditions = append(conditions, fmt.Sprintf("b.text_extracted = $%d", idx))
		args = append(args, *f.TextExtracted)
		idx++
	}
	if f.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("t.slug = $%d", idx))
		args = append(args, f.Tag)
		idx++
	}
	if f.Query != "" {
		conditions = append(conditions, fmt.Sprintf(
			"to_tsvector('english', coalesce(b.title,'') || ' ' || coalesce(b.subtitle,'') || ' ' || coalesce(b.description,'')) @@ plainto_tsquery('english', $%d)", idx))
		args = append(args, f.Query)
		idx++
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func upsertAndLinkAuthors(ctx context.Context, tx pgx.Tx, bookID uuid.UUID, names []string) error {
	for i, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var authorID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO authors (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, name).Scan(&authorID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO book_authors (book_id, author_id, sort_order) VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`, bookID, authorID, i); err != nil {
			return err
		}
	}
	return nil
}

func upsertAndLinkTags(ctx context.Context, tx pgx.Tx, bookID uuid.UUID, names []string) error {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		slug := toSlug(name)
		var tagID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO tags (name, slug) VALUES ($1, $2)
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, name, slug).Scan(&tagID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO book_tags (book_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, bookID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func replaceBookAuthors(ctx context.Context, db *pgxpool.Pool, bookID uuid.UUID, names []string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "DELETE FROM book_authors WHERE book_id = $1", bookID); err != nil {
		return err
	}
	if err := upsertAndLinkAuthors(ctx, tx, bookID, names); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func replaceBookTags(ctx context.Context, db *pgxpool.Pool, bookID uuid.UUID, names []string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "DELETE FROM book_tags WHERE book_id = $1", bookID); err != nil {
		return err
	}
	if err := upsertAndLinkTags(ctx, tx, bookID, names); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func strOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func toSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	if result == "" {
		return "book"
	}
	return result
}

func apiError(code, message string, status int) fiber.Map {
	return fiber.Map{
		"error": fiber.Map{
			"code":    code,
			"message": message,
			"status":  status,
		},
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
