package services

import (
	"fmt"

	"github.com/mathornton01/arkheion/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Bundle aggregates all Arkheion services into a single struct for dependency injection.
// It is initialized once at startup and passed to all HTTP handlers.
type Bundle struct {
	ISBN        *ISBNService
	Tika        *TikaService
	Meilisearch *MeilisearchService
	MinIO       *MinIOService
	Webhook     *WebhookService
}

// NewBundle initializes all services and returns the Bundle.
// Returns an error if any service fails to initialize (e.g. connectivity issues).
func NewBundle(cfg *config.Config, db *pgxpool.Pool) (*Bundle, error) {
	ms, err := NewMeilisearchService(cfg)
	if err != nil {
		return nil, fmt.Errorf("meilisearch service: %w", err)
	}

	minioSvc, err := NewMinIOService(cfg)
	if err != nil {
		return nil, fmt.Errorf("minio service: %w", err)
	}

	return &Bundle{
		ISBN:        NewISBNService(cfg),
		Tika:        NewTikaService(cfg),
		Meilisearch: ms,
		MinIO:       minioSvc,
		Webhook:     NewWebhookService(cfg, db),
	}, nil
}
