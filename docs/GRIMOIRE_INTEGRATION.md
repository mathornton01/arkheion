# Grimoire Integration

This guide explains how to connect [Grimoire](https://github.com/example/grimoire) — a personal knowledge management system — to Arkheion.

---

## Overview

Grimoire can consume Arkheion data in two ways:

1. **Polling** — Grimoire periodically calls `GET /api/v1/books` to sync the full catalog
2. **Webhooks** — Arkheion notifies Grimoire in real time when books are added, updated, or have text extracted

Both methods can be used together for reliability.

---

## Setup

### 1. Generate an API Key in Arkheion

In Arkheion's `.env`, add a dedicated key for Grimoire:

```bash
ARKHEION_API_KEYS=your-admin-key,grimoire-dedicated-key-abc123
```

Restart the backend:
```bash
docker compose restart arkheion-backend
```

### 2. Configure Grimoire

In Grimoire's settings, add an Arkheion source:

```yaml
# grimoire config example
sources:
  - type: arkheion
    name: My Library
    base_url: https://arkheion.example.com/api/v1
    api_key: grimoire-dedicated-key-abc123
    sync_interval: 3600   # seconds (1 hour)
    include_text: false   # set true to pull extracted book text into Grimoire nodes
```

### 3. Register a Webhook in Arkheion

In Arkheion Admin → Webhooks, add a webhook pointing to Grimoire:

| Field | Value |
|-------|-------|
| URL | `https://grimoire.example.com/webhooks/arkheion` |
| Secret | (generate a strong secret, save it in Grimoire too) |
| Events | `book.created`, `book.updated`, `book.deleted`, `book.text_extracted` |
| Description | `Grimoire knowledge graph sync` |

Or via API:
```bash
curl -X POST https://arkheion.example.com/api/v1/webhooks \
  -H "X-API-Key: your-admin-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://grimoire.example.com/webhooks/arkheion",
    "secret": "your-shared-webhook-secret",
    "events": ["book.created", "book.updated", "book.deleted", "book.text_extracted"],
    "description": "Grimoire knowledge graph sync"
  }'
```

---

## What Grimoire Receives

### `book.created` / `book.updated` Payload

```json
{
  "event": "book.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "book": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "isbn": "9780345539434",
    "title": "Cosmos",
    "subtitle": "A Personal Voyage",
    "authors": [{"id": "...", "name": "Carl Sagan"}],
    "publisher": "Random House",
    "published_date": "1980-10-12",
    "categories": ["Science", "Astronomy"],
    "language": "en",
    "cover_url": "https://covers.openlibrary.org/b/id/12345-L.jpg",
    "tags": [{"name": "science", "slug": "science"}],
    "text_extracted": false
  }
}
```

### `book.text_extracted` Payload

Same as above but `text_extracted: true`. This is the event Grimoire should use to trigger pulling the full text for indexing:

```bash
# After receiving book.text_extracted, Grimoire fetches:
GET /api/v1/books/{book_id}
```

The full extracted text is not included in webhook payloads (can be very large). Grimoire should fetch it on demand.

### `book.deleted` Payload

```json
{
  "event": "book.deleted",
  "timestamp": "2024-01-15T14:00:00Z",
  "book": {"id": "550e8400-..."}
}
```

---

## Webhook Signature Verification

Grimoire should verify the `X-Arkheion-Signature` header:

```
X-Arkheion-Signature: sha256=abc123...
```

Python verification:
```python
import hmac, hashlib

def verify(body: bytes, header: str, secret: str) -> bool:
    expected = "sha256=" + hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, header)
```

Go verification:
```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func verify(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

---

## Polling API

For resilience, Grimoire can poll Arkheion on a schedule:

```bash
# Fetch all books updated in the last hour
curl "https://arkheion.example.com/api/v1/books?per_page=100" \
  -H "X-API-Key: grimoire-key"

# Paginate through all books
for page in 1 2 3 ...; do
  curl "https://arkheion.example.com/api/v1/books?page=$page&per_page=100" \
    -H "X-API-Key: grimoire-key"
done
```

---

## Grimoire Knowledge Graph Schema

Suggested Grimoire node types for Arkheion books:

```
Book Node:
  - id: Arkheion book UUID
  - title
  - authors → links to Author nodes
  - categories → links to Category nodes
  - tags
  - source: "arkheion"
  - arkheion_url: https://arkheion.example.com/library/{id}

Author Node:
  - name
  - books → links to Book nodes

Category Node:
  - name
  - books → links to Book nodes
```

This creates a graph where you can navigate from an author to all their books, from a category to all relevant books, and cross-link with other knowledge nodes in Grimoire.
