package models

import (
	"time"

	"github.com/google/uuid"
)

// Author represents a book author record.
// Authors are normalized — the same person is stored once and linked to many books.
type Author struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
