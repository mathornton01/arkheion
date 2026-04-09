-- Migration 003: Create authors table and book_authors join table

CREATE TABLE IF NOT EXISTS authors (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT    NOT NULL UNIQUE,   -- Normalized author name
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authors_name ON authors (name);

-- Many-to-many: a book can have multiple authors, an author can have many books
CREATE TABLE IF NOT EXISTS book_authors (
    book_id    UUID NOT NULL REFERENCES books(id)   ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,           -- Display order (primary author first)
    PRIMARY KEY (book_id, author_id)
);

CREATE INDEX IF NOT EXISTS idx_book_authors_book   ON book_authors (book_id);
CREATE INDEX IF NOT EXISTS idx_book_authors_author ON book_authors (author_id);

INSERT INTO schema_migrations (version) VALUES ('003_authors')
    ON CONFLICT (version) DO NOTHING;
