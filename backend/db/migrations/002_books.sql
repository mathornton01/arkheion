-- Migration 002: Create books table
-- The books table is the central entity. All metadata about a book lives here.

CREATE TABLE IF NOT EXISTS books (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    isbn             TEXT        UNIQUE,                -- ISBN-10 or ISBN-13 (normalized, no dashes)
    title            TEXT        NOT NULL,
    subtitle         TEXT,
    publisher        TEXT,
    published_date   DATE,
    description      TEXT,
    page_count       INTEGER,
    categories       TEXT[]      NOT NULL DEFAULT '{}', -- e.g. {"Science", "History"}
    language         TEXT        NOT NULL DEFAULT 'en', -- ISO 639-1 language code
    cover_url        TEXT,                              -- URL to cover image (external or MinIO)
    file_path        TEXT,                              -- Object key in MinIO (e.g. books/{id}/file.pdf)
    file_type        TEXT,                              -- 'pdf', 'epub', 'txt', etc.
    file_size_bytes  BIGINT,
    text_extracted   BOOLEAN     NOT NULL DEFAULT FALSE,
    extracted_text   TEXT,                              -- Full text from Tika (may be very large)
    physical_location TEXT,                             -- Shelf/box label for physical copies
    notes            TEXT,                              -- User notes, free-form
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Full-text search on title, subtitle, description (PostgreSQL FTS, complementary to Meilisearch)
CREATE INDEX IF NOT EXISTS idx_books_fts
    ON books USING GIN (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(subtitle, '') || ' ' || coalesce(description, ''))
    );

-- Common filter columns
CREATE INDEX IF NOT EXISTS idx_books_language        ON books (language);
CREATE INDEX IF NOT EXISTS idx_books_text_extracted  ON books (text_extracted);
CREATE INDEX IF NOT EXISTS idx_books_created_at      ON books (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_books_categories      ON books USING GIN (categories);

-- Trigger: auto-update updated_at on row modification
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS books_updated_at ON books;
CREATE TRIGGER books_updated_at
    BEFORE UPDATE ON books
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO schema_migrations (version) VALUES ('002_books')
    ON CONFLICT (version) DO NOTHING;
