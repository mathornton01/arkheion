package models

import (
	"time"

	"github.com/google/uuid"
)

// Tag is a user-defined label applied to books.
// Tags use a URL-safe slug for filtering.
type Tag struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
