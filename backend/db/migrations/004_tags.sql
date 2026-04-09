-- Migration 004: Create tags table and book_tags join table

CREATE TABLE IF NOT EXISTS tags (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT    NOT NULL UNIQUE,
    slug        TEXT    NOT NULL UNIQUE,  -- URL-safe lowercase version of name
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tags_slug ON tags (slug);

-- Many-to-many: books can have many tags
CREATE TABLE IF NOT EXISTS book_tags (
    book_id  UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    tag_id   UUID NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (book_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_book_tags_book ON book_tags (book_id);
CREATE INDEX IF NOT EXISTS idx_book_tags_tag  ON book_tags (tag_id);

INSERT INTO schema_migrations (version) VALUES ('004_tags')
    ON CONFLICT (version) DO NOTHING;
