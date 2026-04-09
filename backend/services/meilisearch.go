package services

import (
	"context"
	"fmt"

	"github.com/mathornton01/arkheion/config"
	"github.com/mathornton01/arkheion/models"

	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog/log"
)

// MeilisearchService wraps the Meilisearch client for book indexing and search.
type MeilisearchService struct {
	cfg    *config.Config
	client meilisearch.ServiceManager
	index  meilisearch.IndexManager
}

// SearchResult is the response from a Meilisearch search query.
type SearchResult struct {
	Hits               []interface{} `json:"hits"`
	TotalHits          int64         `json:"total_hits"`
	Page               int           `json:"page"`
	HitsPerPage        int           `json:"hits_per_page"`
	TotalPages         int64         `json:"total_pages"`
	EstimatedTotalHits int64         `json:"estimated_total_hits"`
	ProcessingTimeMs   int64         `json:"processing_time_ms"`
	Query              string        `json:"query"`
}

// NewMeilisearchService creates a new MeilisearchService and ensures the books
// index exists with the correct settings.
func NewMeilisearchService(cfg *config.Config) (*MeilisearchService, error) {
	client := meilisearch.New(cfg.MeilisearchURL, meilisearch.WithAPIKey(cfg.MeilisearchMasterKey))

	// Verify connectivity
	if _, err := client.Health(); err != nil {
		return nil, fmt.Errorf("Meilisearch health check failed: %w", err)
	}

	svc := &MeilisearchService{
		cfg:    cfg,
		client: client,
	}

	if err := svc.ensureIndex(); err != nil {
		return nil, fmt.Errorf("ensure Meilisearch index: %w", err)
	}

	svc.index = client.Index(cfg.MeilisearchBooksIndex)
	return svc, nil
}

// ensureIndex creates the books index if it doesn't exist and configures
// searchable, filterable, and sortable attributes.
func (s *MeilisearchService) ensureIndex() error {
	idx := s.client.Index(s.cfg.MeilisearchBooksIndex)

	// Create index (no-op if already exists)
	task, err := s.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        s.cfg.MeilisearchBooksIndex,
		PrimaryKey: "id",
	})
	if err != nil {
		// Index may already exist — not fatal
		log.Debug().Err(err).Msg("Meilisearch: create index (may already exist)")
	} else {
		log.Info().Int64("task_uid", task.TaskUID).Msg("Meilisearch: books index created")
	}

	// Configure searchable attributes (order = relevance weight)
	if _, err := idx.UpdateSearchableAttributes(&[]string{
		"title",
		"subtitle",
		"authors",
		"description",
		"publisher",
		"categories",
		"tags",
		"extracted_text_snippet",
	}); err != nil {
		return fmt.Errorf("update searchable attributes: %w", err)
	}

	// Configure filterable attributes for faceted search
	if _, err := idx.UpdateFilterableAttributes(&[]string{
		"language",
		"categories",
		"tags",
		"text_extracted",
		"authors",
	}); err != nil {
		return fmt.Errorf("update filterable attributes: %w", err)
	}

	// Configure sortable attributes
	if _, err := idx.UpdateSortableAttributes(&[]string{
		"title",
	}); err != nil {
		return fmt.Errorf("update sortable attributes: %w", err)
	}

	log.Info().Str("index", s.cfg.MeilisearchBooksIndex).Msg("Meilisearch index settings configured")
	return nil
}

// IndexBook adds or updates a book in the Meilisearch index.
func (s *MeilisearchService) IndexBook(book *models.Book) error {
	doc := book.ToMeilisearch()
	task, err := s.index.AddDocuments([]models.MeilisearchBook{doc}, "id")
	if err != nil {
		return fmt.Errorf("Meilisearch index book: %w", err)
	}
	log.Debug().Int64("task_uid", task.TaskUID).Str("book_id", book.ID.String()).Msg("Book queued for indexing")
	return nil
}

// DeleteBook removes a book from the Meilisearch index by ID.
func (s *MeilisearchService) DeleteBook(id string) error {
	task, err := s.index.DeleteDocument(id)
	if err != nil {
		return fmt.Errorf("Meilisearch delete book: %w", err)
	}
	log.Debug().Int64("task_uid", task.TaskUID).Str("book_id", id).Msg("Book deletion queued")
	return nil
}

// Search performs a full-text search and returns paginated results.
func (s *MeilisearchService) Search(_ context.Context, query string, page, perPage int, filter string) (*SearchResult, error) {
	params := &meilisearch.SearchRequest{
		Query:       query,
		Page:        int64(page),
		HitsPerPage: int64(perPage),
	}
	if filter != "" {
		params.Filter = filter
	}

	raw, err := s.index.Search(query, params)
	if err != nil {
		return nil, fmt.Errorf("Meilisearch search: %w", err)
	}

	return &SearchResult{
		Hits:               raw.Hits,
		TotalHits:          raw.TotalHits,
		Page:               page,
		HitsPerPage:        perPage,
		TotalPages:         raw.TotalPages,
		EstimatedTotalHits: raw.EstimatedTotalHits,
		ProcessingTimeMs:   raw.ProcessingTimeMs,
		Query:              query,
	}, nil
}

// ReindexAll fetches all books from the database and rebuilds the Meilisearch index.
// This is an administrative operation that should be called via the admin API or CLI.
// The caller must provide a function to fetch all books (to avoid circular dependencies).
func (s *MeilisearchService) ReindexAll(fetchBooks func() ([]models.Book, error)) error {
	books, err := fetchBooks()
	if err != nil {
		return fmt.Errorf("fetch books for reindex: %w", err)
	}

	docs := make([]models.MeilisearchBook, len(books))
	for i, b := range books {
		docs[i] = b.ToMeilisearch()
	}

	task, err := s.index.AddDocuments(docs, "id")
	if err != nil {
		return fmt.Errorf("Meilisearch bulk index: %w", err)
	}

	log.Info().Int64("task_uid", task.TaskUID).Int("count", len(docs)).Msg("Full reindex queued")
	return nil
}
