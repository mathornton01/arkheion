-- Migration 005: Create webhooks table
-- Stores registered webhook endpoints and their configuration.

CREATE TABLE IF NOT EXISTS webhooks (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    url         TEXT    NOT NULL,
    secret      TEXT    NOT NULL,       -- HMAC-SHA256 signing secret for this webhook
    events      TEXT[]  NOT NULL,       -- e.g. {'book.created', 'book.text_extracted'}
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,                   -- Human-readable label
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks (active);

-- Valid webhook events (enforced in application layer):
--   book.created
--   book.updated
--   book.deleted
--   book.text_extracted

DROP TRIGGER IF EXISTS webhooks_updated_at ON webhooks;
CREATE TRIGGER webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Webhook delivery log for debugging and retry tracking
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id   UUID        NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event        TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    response_code INTEGER,
    response_body TEXT,
    attempts     INTEGER     NOT NULL DEFAULT 0,
    succeeded    BOOLEAN     NOT NULL DEFAULT FALSE,
    last_attempt TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries (webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created ON webhook_deliveries (created_at DESC);

INSERT INTO schema_migrations (version) VALUES ('005_webhooks')
    ON CONFLICT (version) DO NOTHING;
