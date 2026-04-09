// Package models defines the core data structures for the Arkheion application.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Book represents a single item in the library catalog.
// It holds both bibliographic metadata and operational state (file info, extraction status).
type Book struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	ISBN             string     `json:"isbn,omitempty" db:"isbn"`
	Title            string     `json:"title" db:"title"`
	Subtitle         string     `json:"subtitle,omitempty" db:"subtitle"`
	Authors          []Author   `json:"authors,omitempty" db:"-"` // Populated via JOIN
	Publisher        string     `json:"publisher,omitempty" db:"publisher"`
	PublishedDate    *time.Time `json:"published_date,omitempty" db:"published_date"`
	Description      string     `json:"description,omitempty" db:"description"`
	PageCount        int        `json:"page_count,omitempty" db:"page_count"`
	Categories       []string   `json:"categories,omitempty" db:"categories"`
	Language         string     `json:"language,omitempty" db:"language"`
	CoverURL         string     `json:"cover_url,omitempty" db:"cover_url"`
	FilePath         string     `json:"file_path,omitempty" db:"file_path"`
	FileType         string     `json:"file_type,omitempty" db:"file_type"`
	FileSizeBytes    int64      `json:"file_size_bytes,omitempty" db:"file_size_bytes"`
	TextExtracted    bool       `json:"text_extracted" db:"text_extracted"`
	ExtractedText    string     `json:"-" db:"extracted_text"` // Hidden from API; used internally
	Tags             []Tag      `json:"tags,omitempty" db:"-"` // Populated via JOIN
	PhysicalLocation string     `json:"physical_location,omitempty" db:"physical_location"`
	Notes            string     `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateBookRequest is the payload for POST /api/v1/books.
type CreateBookRequest struct {
	ISBN             string   `json:"isbn"`
	Title            string   `json:"title" validate:"required,min=1,max=500"`
	Subtitle         string   `json:"subtitle"`
	Authors          []string `json:"authors"` // Author names (created/reused automatically)
	Publisher        string   `json:"publisher"`
	PublishedDate    string   `json:"published_date"` // ISO 8601 date string
	Description      string   `json:"description"`
	PageCount        int      `json:"page_count"`
	Categories       []string `json:"categories"`
	Language         string   `json:"language"`
	CoverURL         string   `json:"cover_url"`
	Tags             []string `json:"tags"` // Tag names (created/reused automatically)
	PhysicalLocation string   `json:"physical_location"`
	Notes            string   `json:"notes"`
}

// UpdateBookRequest is the payload for PUT /api/v1/books/:id.
// All fields are optional — only provided fields are updated.
type UpdateBookRequest struct {
	ISBN             *string  `json:"isbn"`
	Title            *string  `json:"title"`
	Subtitle         *string  `json:"subtitle"`
	Authors          []string `json:"authors"`
	Publisher        *string  `json:"publisher"`
	PublishedDate    *string  `json:"published_date"`
	Description      *string  `json:"description"`
	PageCount        *int     `json:"page_count"`
	Categories       []string `json:"categories"`
	Language         *string  `json:"language"`
	CoverURL         *string  `json:"cover_url"`
	Tags             []string `json:"tags"`
	PhysicalLocation *string  `json:"physical_location"`
	Notes            *string  `json:"notes"`
}

// BookListResponse wraps a list of books with pagination metadata.
type BookListResponse struct {
	Data       []Book     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination holds standard pagination metadata for list responses.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// BookFilter represents filtering options for the book list endpoint.
type BookFilter struct {
	Tag           string
	Category      string
	Language      string
	TextExtracted *bool
	Query         string // Basic metadata search (not full-text)
	Page          int
	PerPage       int
}

// MeilisearchBook is a flattened representation of a book sent to Meilisearch.
// It omits extracted_text from the full-text index document (too large) —
// a separate partial update adds searchable text content.
type MeilisearchBook struct {
	ID               string   `json:"id"`
	ISBN             string   `json:"isbn"`
	Title            string   `json:"title"`
	Subtitle         string   `json:"subtitle"`
	Authors          []string `json:"authors"`
	Publisher        string   `json:"publisher"`
	Description      string   `json:"description"`
	Categories       []string `json:"categories"`
	Language         string   `json:"language"`
	Tags             []string `json:"tags"`
	TextExtracted    bool     `json:"text_extracted"`
	// ExtractedTextSnippet holds the first 50,000 chars of extracted text.
	// Full text is in PostgreSQL; Meilisearch indexes a truncated version.
	ExtractedTextSnippet string `json:"extracted_text_snippet,omitempty"`
}

// ToMeilisearch converts a Book to its Meilisearch index representation.
func (b *Book) ToMeilisearch() MeilisearchBook {
	authorNames := make([]string, len(b.Authors))
	for i, a := range b.Authors {
		authorNames[i] = a.Name
	}
	tagNames := make([]string, len(b.Tags))
	for i, t := range b.Tags {
		tagNames[i] = t.Name
	}

	snippet := b.ExtractedText
	if len(snippet) > 50000 {
		snippet = snippet[:50000]
	}

	return MeilisearchBook{
		ID:                   b.ID.String(),
		ISBN:                 b.ISBN,
		Title:                b.Title,
		Subtitle:             b.Subtitle,
		Authors:              authorNames,
		Publisher:            b.Publisher,
		Description:          b.Description,
		Categories:           b.Categories,
		Language:             b.Language,
		Tags:                 tagNames,
		TextExtracted:        b.TextExtracted,
		ExtractedTextSnippet: snippet,
	}
}
