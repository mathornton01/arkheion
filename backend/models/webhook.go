package models

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEvent represents the type of event that triggers a webhook.
type WebhookEvent string

const (
	EventBookCreated       WebhookEvent = "book.created"
	EventBookUpdated       WebhookEvent = "book.updated"
	EventBookDeleted       WebhookEvent = "book.deleted"
	EventBookTextExtracted WebhookEvent = "book.text_extracted"
)

// AllWebhookEvents is the complete set of supported event types.
var AllWebhookEvents = []WebhookEvent{
	EventBookCreated,
	EventBookUpdated,
	EventBookDeleted,
	EventBookTextExtracted,
}

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	URL         string         `json:"url" db:"url"`
	Secret      string         `json:"secret,omitempty" db:"secret"` // Omitted from list responses
	Events      []WebhookEvent `json:"events" db:"events"`
	Active      bool           `json:"active" db:"active"`
	Description string         `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// CreateWebhookRequest is the payload for POST /api/v1/webhooks.
type CreateWebhookRequest struct {
	URL         string         `json:"url" validate:"required,url"`
	Secret      string         `json:"secret" validate:"required,min=16"`
	Events      []WebhookEvent `json:"events" validate:"required,min=1"`
	Description string         `json:"description"`
}

// WebhookPayload is the JSON body sent to a webhook URL on event dispatch.
type WebhookPayload struct {
	Event     WebhookEvent `json:"event"`
	Timestamp time.Time    `json:"timestamp"`
	Book      interface{}  `json:"book"` // Full book object or minimal stub for book.deleted
}
