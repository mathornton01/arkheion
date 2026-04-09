# Arkheion API Reference

Base URL: `http://localhost:8080/api/v1` (development)

All requests require the `X-API-Key` header (except `/health`).

The complete machine-readable spec is in `backend/docs/openapi.yaml`.

---

## Authentication

```
X-API-Key: your-api-key-here
```

API keys are configured via the `ARKHEION_API_KEYS` environment variable (comma-separated list). Generate a key:

```bash
openssl rand -hex 32
```

---

## Books

### List Books

```
GET /api/v1/books
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default: 1) |
| `per_page` | int | Results per page (default: 20, max: 100) |
| `tag` | string | Filter by tag slug |
| `category` | string | Filter by category (exact match) |
| `language` | string | Filter by language code (e.g. `en`) |
| `text_extracted` | bool | Filter by extraction status |
| `q` | string | Basic metadata search |

**Example:**
```bash
curl "http://localhost:8080/api/v1/books?page=1&per_page=20&language=en" \
  -H "X-API-Key: your-key"
```

**Response:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "isbn": "9780345539434",
      "title": "Cosmos",
      "authors": [{"id": "...", "name": "Carl Sagan"}],
      "text_extracted": true,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 347,
    "total_pages": 18
  }
}
```

### Get Book

```
GET /api/v1/books/:id
```

### Create Book

```
POST /api/v1/books
Content-Type: application/json
```

```json
{
  "isbn": "9780345539434",
  "title": "Cosmos",
  "authors": ["Carl Sagan"],
  "publisher": "Random House",
  "published_date": "1980-10-12",
  "categories": ["Science", "Astronomy"],
  "language": "en",
  "tags": ["science", "classic"]
}
```

### Update Book

```
PUT /api/v1/books/:id
Content-Type: application/json
```

All fields optional — only provided fields are updated.

### Delete Book

```
DELETE /api/v1/books/:id
```

Returns `204 No Content`.

### Upload File

```
POST /api/v1/books/:id/upload
Content-Type: multipart/form-data
```

Field name: `file`. Accepted types: `.pdf`, `.epub`, `.txt`, `.docx`.

```bash
curl -X POST "http://localhost:8080/api/v1/books/550e8400.../upload" \
  -H "X-API-Key: your-key" \
  -F "file=@cosmos.pdf"
```

Text extraction starts automatically and fires `book.text_extracted` when done.

### Download File

```
GET /api/v1/books/:id/download
```

Returns the file with appropriate `Content-Type` and `Content-Disposition` headers.

---

## Catalog (ISBN Lookup)

### Lookup by ISBN

```
GET /api/v1/catalog/isbn/:isbn
```

```bash
curl "http://localhost:8080/api/v1/catalog/isbn/9780345539434" \
  -H "X-API-Key: your-key"
```

**Response:**
```json
{
  "isbn": "9780345539434",
  "title": "Cosmos",
  "authors": ["Carl Sagan"],
  "publisher": "Random House",
  "published_date": "1980-10-12",
  "categories": ["Science"],
  "language": "en",
  "cover_url": "https://covers.openlibrary.org/b/id/12345-L.jpg",
  "source": "openlibrary"
}
```

### Scan Barcode

```
POST /api/v1/catalog/scan
Content-Type: application/json

{"barcode": "9780345539434"}
```

Same response as ISBN lookup.

---

## Search

```
GET /api/v1/search?q=cosmos&page=1&per_page=20
```

Optional `filter` parameter uses Meilisearch syntax:

```bash
# Only searchable books in English
GET /api/v1/search?q=physics&filter=language="en"%20AND%20text_extracted=true

# Books with specific tag
GET /api/v1/search?q=quantum&filter=tags="science"
```

**Response:**
```json
{
  "query": "cosmos",
  "hits": [...],
  "total_hits": 42,
  "processing_time_ms": 3,
  "page": 1,
  "hits_per_page": 20
}
```

---

## Export

```
GET /api/v1/export?format=jsonl
GET /api/v1/export?format=jsonl&tag=philosophy
GET /api/v1/export?format=jsonl&category=Science&language=en
```

Returns a JSONL stream (one JSON object per line). Suitable for:
- Golem LLM training pipelines
- Hugging Face dataset creation
- Custom fine-tuning scripts

**Each line:**
```json
{"id":"uuid","title":"Book Title","authors":["Author"],"categories":["Science"],"language":"en","text":"Full extracted text here..."}
```

```bash
# Download all searchable books
curl "http://localhost:8080/api/v1/export?format=jsonl" \
  -H "X-API-Key: your-key" \
  -o training_data.jsonl

# Count records
wc -l training_data.jsonl
```

---

## Webhooks

### List Webhooks

```
GET /api/v1/webhooks
```

### Create Webhook

```
POST /api/v1/webhooks
Content-Type: application/json

{
  "url": "https://grimoire.example.com/webhooks/arkheion",
  "secret": "my-secret-at-least-16-chars",
  "events": ["book.created", "book.text_extracted"],
  "description": "Grimoire sync"
}
```

**Events:**
- `book.created`
- `book.updated`
- `book.deleted`
- `book.text_extracted`

### Delete Webhook

```
DELETE /api/v1/webhooks/:id
```

### Activate / Deactivate

```
PUT /api/v1/webhooks/:id/activate
PUT /api/v1/webhooks/:id/deactivate
```

---

## Webhook Payload Format

```json
{
  "event": "book.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "book": {
    "id": "550e8400-...",
    "title": "Cosmos",
    "authors": [{"name": "Carl Sagan"}],
    "text_extracted": false
  }
}
```

**Signature verification (Python example):**
```python
import hmac, hashlib

def verify_webhook(body: bytes, signature: str, secret: str) -> bool:
    expected = "sha256=" + hmac.new(
        secret.encode(), body, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)

# In your webhook handler:
sig = request.headers.get("X-Arkheion-Signature")
if not verify_webhook(request.body, sig, "your-secret"):
    return 401
```

---

## Error Format

All errors return consistent JSON:

```json
{
  "error": {
    "code": "BOOK_NOT_FOUND",
    "message": "No book found with id: 550e8400-...",
    "status": 404
  }
}
```

Common error codes:

| Code | Status | Description |
|------|--------|-------------|
| `MISSING_API_KEY` | 401 | No API key provided |
| `INVALID_API_KEY` | 403 | API key not recognized |
| `BOOK_NOT_FOUND` | 404 | Book ID does not exist |
| `WEBHOOK_NOT_FOUND` | 404 | Webhook ID does not exist |
| `ISBN_NOT_FOUND` | 404 | ISBN not in any catalog |
| `VALIDATION_ERROR` | 422 | Request body failed validation |
| `UNSUPPORTED_FILE_TYPE` | 400 | File extension not allowed |
| `UPLOAD_FAILED` | 500 | MinIO error during upload |
| `SEARCH_FAILED` | 502 | Meilisearch unavailable |
| `LOOKUP_FAILED` | 502 | ISBN lookup API unavailable |
