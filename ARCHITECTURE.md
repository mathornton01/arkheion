# Arkheion — Architecture

This document describes the system architecture, service interactions, data flows, and design decisions behind Arkheion.

---

## System Overview

```
                         ┌─────────────────────────────────────────────┐
                         │              Docker Network: arkheion        │
                         │                                             │
   Browser / Client      │   ┌─────────────┐      ┌──────────────┐   │
  ┌──────────────────┐   │   │             │      │              │   │
  │                  │──────▶│  Frontend   │      │   Backend    │   │
  │  SvelteKit UI    │   │   │  (SvelteKit)│─────▶│  (Go/Fiber) │   │
  │  PDF.js / epub.js│◀──────│  :3000      │      │  :8080       │   │
  │  ZXing scanner   │   │   │             │      │              │   │
  └──────────────────┘   │   └─────────────┘      └──────┬───────┘   │
                         │                               │           │
   External Tools        │           ┌──────────────────┼──────┐    │
  ┌──────────────────┐   │           │                  │      │    │
  │  Grimoire        │──────────────▶│           ┌──────▼──────┐    │
  │  Golem/LLM pipe  │   │   REST    │           │  PostgreSQL │    │
  │  CI/CD scripts   │   │   API     │           │  :5432      │    │
  └──────────────────┘   │           │           └─────────────┘    │
                         │           │                  │            │
                         │           │           ┌──────▼──────┐    │
                         │           │           │ Meilisearch │    │
                         │           │           │  :7700      │    │
                         │           │           └─────────────┘    │
                         │           │                  │            │
                         │           │           ┌──────▼──────┐    │
                         │           │           │    MinIO    │    │
                         │           │           │  :9000      │    │
                         │           │           └─────────────┘    │
                         │           │                  │            │
                         │           │           ┌──────▼──────┐    │
                         │           └───────────│ Apache Tika │    │
                         │                       │  :9998      │    │
                         │                       └─────────────┘    │
                         └─────────────────────────────────────────────┘
```

---

## Services

### Frontend — SvelteKit (Port 3000)

The frontend is a SvelteKit application that uses server-side rendering for initial page loads and transitions to a SPA model for subsequent navigation.

**Key responsibilities:**
- Book catalog grid and list views
- In-browser PDF and EPUB reading via PDF.js and epub.js
- Barcode scanning UI using ZXing-js (camera access)
- Meilisearch-powered instant search UI
- Admin panel for webhooks, API keys, and bulk export

**Key design choices:**
- No authentication UI — Arkheion is designed for single-user or trusted-network use; API key auth is sufficient
- PDF and EPUB files are streamed from the backend download endpoint, not served directly from MinIO (avoids exposing MinIO publicly)
- Barcode scanning runs entirely client-side; the ISBN is sent to the backend catalog endpoint

### Backend — Go + Fiber (Port 8080)

The backend is a Go application using the [Fiber](https://gofiber.io/) framework. It is the sole orchestrator of all service interactions.

**Key responsibilities:**
- REST API for all frontend and external client operations
- Authentication via `X-API-Key` header middleware
- Database CRUD via `pgxpool` connection pool
- Meilisearch index management (index books, run searches)
- MinIO file upload/download proxying
- Apache Tika text extraction (triggered post-upload)
- Webhook dispatch on book lifecycle events
- OpenAPI spec served at `/api/v1/docs`

**Why Fiber?**
Fiber is a fast, Express-inspired Go framework built on fasthttp. For a library management system expected to handle concurrent uploads and search requests, raw performance matters. Fiber's middleware system is clean and composable.

### PostgreSQL (Port 5432)

Primary relational data store. Holds:
- `books` table — all book metadata
- `authors` table — normalized author records
- `book_authors` join table
- `tags` table — user-defined tags
- `book_tags` join table
- `webhooks` table — registered webhook URLs and secrets

Migrations are in `backend/db/migrations/` and run with `scripts/migrate.sh`.

**Why PostgreSQL over SQLite?**
Multi-user access, full ACID transactions, and the ability to run complex filtered queries across large collections. Arkheion is designed to scale to tens of thousands of books.

### Meilisearch (Port 7700)

Full-text search engine. The backend maintains a `books` index in Meilisearch that contains:
- Book ID (for cross-referencing)
- Title, subtitle, authors, publisher
- Description, categories, tags
- Extracted full text (from Tika)

Meilisearch is NOT the source of truth — PostgreSQL is. Meilisearch is updated after every write operation and is re-indexed on demand via the admin API.

### Apache Tika (Port 9998)

Apache Tika is used exclusively for text extraction from uploaded files. The backend sends the file to Tika's REST interface (`PUT /tika`) and stores the extracted plain text in PostgreSQL. This text is then pushed to Meilisearch.

Supported formats:
- PDF (all versions)
- EPUB
- DOCX, DOC
- TXT, RTF, HTML (passthrough)

### MinIO (Port 9000 API, 9001 Console)

S3-compatible object storage for book files (PDFs, EPUBs) and cover images. All files are stored in a bucket named `arkheion`.

Object naming convention:
```
books/{book_id}/file.{ext}
books/{book_id}/cover.{ext}
```

The backend uses the MinIO Go SDK for all operations. Files are never served directly from MinIO to the browser — the backend proxies downloads to allow auth checking.

---

## Book Ingestion Pipeline

```
  User uploads PDF/EPUB
          │
          ▼
  ┌───────────────┐
  │  POST /api/v1 │
  │ /books/:id    │
  │ /upload       │
  └───────┬───────┘
          │
          ▼
  ┌───────────────┐    ┌─────────────────────────────┐
  │  Backend      │───▶│  MinIO                      │
  │  (Go/Fiber)   │    │  Store file in              │
  │               │    │  books/{id}/file.{ext}      │
  └───────┬───────┘    └─────────────────────────────┘
          │
          │  (async goroutine)
          ▼
  ┌───────────────┐    ┌─────────────────────────────┐
  │  Tika Service │───▶│  Apache Tika                │
  │               │    │  PUT /tika                  │
  │               │◀───│  Returns plain text         │
  └───────┬───────┘    └─────────────────────────────┘
          │
          ▼
  ┌───────────────┐
  │  Store text   │
  │  in PostgreSQL│
  │  book.text_   │
  │  extracted=T  │
  └───────┬───────┘
          │
          ▼
  ┌───────────────┐    ┌─────────────────────────────┐
  │  Meilisearch  │───▶│  Meilisearch                │
  │  Service      │    │  Update books index with    │
  │               │    │  full text                  │
  └───────┬───────┘    └─────────────────────────────┘
          │
          ▼
  ┌───────────────┐    ┌─────────────────────────────┐
  │  Webhook      │───▶│  All registered webhook     │
  │  Dispatch     │    │  URLs receive               │
  │               │    │  book.text_extracted event  │
  └───────────────┘    └─────────────────────────────┘
```

### ISBN Lookup Flow

```
  User scans barcode (ZXing-js in browser)
          │
          ▼ ISBN string
  ┌───────────────────────────────────┐
  │  POST /api/v1/catalog/isbn/:isbn  │
  └───────────────┬───────────────────┘
                  │
       ┌──────────┴────────────┐
       │                       │
       ▼                       ▼
  OpenLibrary API         Google Books API
  (primary)               (fallback)
       │                       │
       └──────────┬────────────┘
                  │ Merged metadata
                  ▼
  ┌───────────────────────────────────┐
  │  Return BookMetadata to client    │
  │  Client pre-fills the Add Book    │
  │  form, user confirms and saves    │
  └───────────────────────────────────┘
```

---

## API Design

### Base Path

All API routes are prefixed with `/api/v1`. This allows future major version bumps without breaking existing integrations.

### Authentication

All API endpoints (except `/api/v1/health`) require an `X-API-Key` header. API keys are stored hashed in the database. There is no user account system — Arkheion is intended for personal or small-team use.

```
GET /api/v1/books HTTP/1.1
Host: arkheion.example.com
X-API-Key: ark_live_abc123xyz
```

### Pagination

List endpoints support `?page=1&per_page=20` query parameters. Responses include a `pagination` object:

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 347,
    "total_pages": 18
  }
}
```

### Filtering

The `GET /api/v1/books` endpoint supports:
- `?tag=fiction` — filter by tag slug
- `?category=Science` — filter by category
- `?language=en` — filter by language code
- `?text_extracted=true` — filter by extraction status
- `?q=search+term` — basic metadata search (use `/api/v1/search` for full-text)

### Error Responses

All errors follow a consistent JSON structure:

```json
{
  "error": {
    "code": "BOOK_NOT_FOUND",
    "message": "No book found with id: 550e8400-e29b-41d4-a716-446655440000",
    "status": 404
  }
}
```

### Webhooks

Webhooks are dispatched asynchronously. Each webhook request includes:
- `X-Arkheion-Event` header with the event type
- `X-Arkheion-Signature` header with HMAC-SHA256 of the body using the webhook secret
- JSON body with event type and full book object

Webhook URLs should respond with 2xx within 10 seconds. Failed deliveries are retried 3 times with exponential backoff.

---

## Database Schema

```
┌─────────────────┐      ┌─────────────┐      ┌──────────────┐
│     books       │      │   authors   │      │     tags     │
├─────────────────┤      ├─────────────┤      ├──────────────┤
│ id (uuid, PK)   │      │ id (uuid)   │      │ id (uuid)    │
│ isbn            │      │ name        │      │ name         │
│ title           │      │ created_at  │      │ slug         │
│ subtitle        │      └──────┬──────┘      │ created_at   │
│ publisher       │             │             └──────┬───────┘
│ published_date  │      ┌──────▼──────┐             │
│ description     │      │book_authors │      ┌──────▼───────┐
│ page_count      │      ├─────────────┤      │  book_tags   │
│ categories[]    │◀─────│ book_id(FK) │      ├──────────────┤
│ language        │      │ author_id   │      │ book_id (FK) │
│ cover_url       │      └─────────────┘      │ tag_id (FK)  │
│ file_path       │                           └──────────────┘
│ file_type       │
│ file_size_bytes │      ┌─────────────────┐
│ text_extracted  │      │    webhooks     │
│ extracted_text  │      ├─────────────────┤
│ physical_loc    │      │ id (uuid, PK)   │
│ notes           │      │ url             │
│ created_at      │      │ secret          │
│ updated_at      │      │ events[]        │
└─────────────────┘      │ active          │
                         │ created_at      │
                         └─────────────────┘
```

---

## Concurrency Model

The backend uses Go's goroutine model for background tasks:

1. **Text extraction** runs in a background goroutine after file upload. The HTTP response returns immediately with the book's current state (`text_extracted: false`). When extraction completes, the book record is updated and the `book.text_extracted` webhook fires.

2. **Webhook dispatch** runs in a goroutine pool (max 10 concurrent). Each webhook URL is notified independently — a slow or failing webhook does not block others.

3. **Meilisearch sync** is performed synchronously on write operations (create, update, delete) but asynchronously after text extraction.

---

## Security Considerations

- API keys are stored as bcrypt hashes in PostgreSQL
- Webhook signatures use HMAC-SHA256 for payload verification
- MinIO credentials are never exposed to the frontend
- File downloads are proxied through the backend (no public MinIO access)
- CORS is restricted to the configured frontend origin
- All SQL uses parameterized queries (no string interpolation)
- Uploaded files are validated by MIME type before storage

---

## Scaling Considerations

For large collections (10,000+ books):

- PostgreSQL: Add indexes on `categories`, `language`, `text_extracted`; consider partitioning by year
- Meilisearch: Increase RAM allocation; the full-text index can grow large with extracted text
- Tika: Run multiple Tika instances and load-balance extraction jobs
- MinIO: Switch to multi-node MinIO or an actual S3 bucket for file storage
- Backend: Stateless — can run multiple replicas behind a load balancer
