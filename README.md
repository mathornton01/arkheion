# Arkheion

> **Self-hosted library management system** — catalog, search, read, and train on your entire book collection.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Go-1.22-blue.svg)](https://golang.org/)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-2.0-orange.svg)](https://kit.svelte.dev/)
[![Docker](https://img.shields.io/badge/Docker-Compose-blue.svg)](https://docs.docker.com/compose/)

---

## What is Arkheion?

Arkheion (from the Greek *ἀρχεῖον* — the place where records are kept) is a self-hosted, open-source library management system designed for individuals, small institutions, and research teams who want full control over their digital and physical book collections.

It is **not** a cloud service. Your books, your metadata, your full-text index, and your reading history live entirely on your own infrastructure.

---

## Screenshot

```
┌─────────────────────────────────────────────────────────────────────┐
│  Arkheion  [Search your library...]            [+ Add Book]  [⚙]   │
├─────────────────────────────────────────────────────────────────────┤
│  📚 Library   🔍 Search   📷 Scan   👤 Admin                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐      │
│  │ [cov]│  │ [cov]│  │ [cov]│  │ [cov]│  │ [cov]│  │ [cov]│      │
│  │      │  │      │  │      │  │      │  │      │  │      │      │
│  │Title │  │Title │  │Title │  │Title │  │Title │  │Title │      │
│  │Author│  │Author│  │Author│  │Author│  │Author│  │Author│      │
│  └──────┘  └──────┘  └──────┘  └──────┘  └──────┘  └──────┘      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

*(Real screenshot coming soon — contributions welcome)*

---

## Features

### Core Library Management
- **Book catalog** with rich metadata: ISBN, title, authors, publisher, description, categories, tags, physical location, and personal notes
- **ISBN/barcode lookup** via OpenLibrary API and Google Books API — scan or type an ISBN and metadata fills in automatically
- **Cover art** fetched automatically from OpenLibrary Covers API
- **Bulk import** from CSV or folder of files

### In-Browser Reading
- **PDF reading** powered by PDF.js — no plugins, no downloads
- **EPUB reading** powered by epub.js — full reflowable ebook support
- **Reading progress** tracked per-book per-user

### Full-Text Search
- **Meilisearch** powers lightning-fast search across book metadata and extracted text content
- **Apache Tika** extracts text from PDF and EPUB files automatically on upload
- Search across full book content, not just titles and authors

### Mobile-Friendly Barcode Scanning
- **ZXing-js** enables barcode scanning directly in the browser via phone camera
- Scan a physical book's ISBN barcode to add it to your catalog instantly
- Works on any modern mobile browser — no app install required

### External Integrations
- **REST API** with full OpenAPI 3.0 specification — integrate with anything
- **Webhook support** for `book.created`, `book.updated`, `book.deleted`, `book.text_extracted` events
- **Grimoire integration** — your personal knowledge management system can index your library
- **Golem / LLM training** — bulk text export endpoint returns JSONL for training or fine-tuning language models

### Storage & Infrastructure
- **MinIO** (S3-compatible) for file storage — store PDFs, EPUBs, and covers locally
- **PostgreSQL** for relational data with full ACID guarantees
- **Docker Compose** — the entire stack runs with a single command

---

## Quick Start

### Prerequisites
- Docker Engine 24+ and Docker Compose v2
- 4 GB RAM minimum (8 GB recommended for Tika + Meilisearch)

### Start Arkheion

```bash
git clone https://github.com/mathornton01/arkheion.git
cd Arkheion
cp .env.example .env
# Edit .env with your secrets
docker compose up -d
```

The services will be available at:

| Service         | URL                      |
|-----------------|--------------------------|
| Frontend (UI)   | http://localhost:3000     |
| Backend API     | http://localhost:8080     |
| MinIO Console   | http://localhost:9001     |
| Meilisearch UI  | http://localhost:7700     |

### First Run

```bash
# Run database migrations
./scripts/migrate.sh

# (Optional) import a folder of PDFs
curl -X POST http://localhost:8080/api/v1/books/bulk-import \
  -H "X-API-Key: your-api-key" \
  -F "files=@/path/to/books/"
```

---

## Full Stack

| Layer            | Technology           | Purpose                              |
|------------------|----------------------|--------------------------------------|
| Frontend         | SvelteKit 2          | Fast, reactive UI with SSR           |
| Backend          | Go 1.22 + Fiber v2   | High-performance REST API            |
| Database         | PostgreSQL 16        | Primary data store                   |
| Full-text search | Meilisearch 1.x      | Sub-millisecond search               |
| Text extraction  | Apache Tika 2.x      | PDF/EPUB text extraction             |
| File storage     | MinIO (RELEASE.2024) | S3-compatible object storage         |
| Container        | Docker Compose       | Single-command deployment            |

---

## Grimoire Integration

[Grimoire](https://github.com/example/grimoire) is a personal knowledge management tool. Arkheion exposes a REST API that Grimoire can poll or receive webhooks from to keep your knowledge graph in sync with your book catalog.

**Setup:**
1. Generate an API key in Arkheion Admin → Settings
2. In Grimoire, configure the Arkheion source with your API base URL and key
3. Grimoire will poll `GET /api/v1/books` and create nodes for each book
4. Enable the `book.created` webhook to push new books to Grimoire in real time

See [docs/GRIMOIRE_INTEGRATION.md](docs/GRIMOIRE_INTEGRATION.md) for full details.

---

## Golem / LLM Training Integration

Arkheion's bulk text export endpoint is designed for feeding extracted book text into LLM training pipelines like [Golem](https://github.com/example/golem).

```bash
# Export all books as JSONL (title + extracted text)
curl "http://localhost:8080/api/v1/export?format=jsonl" \
  -H "X-API-Key: your-api-key" \
  -o training_data.jsonl

# Filter by tag
curl "http://localhost:8080/api/v1/export?format=jsonl&tag=philosophy" \
  -H "X-API-Key: your-api-key" \
  -o philosophy.jsonl

# Filter by category
curl "http://localhost:8080/api/v1/export?format=jsonl&category=Science" \
  -H "X-API-Key: your-api-key" \
  -o science.jsonl
```

Each JSONL line contains:
```json
{"id": "uuid", "title": "Book Title", "authors": ["Author Name"], "text": "Full extracted text..."}
```

See [docs/GOLEM_INTEGRATION.md](docs/GOLEM_INTEGRATION.md) for pipeline configuration.

---

## API Overview

The Arkheion API is documented in [backend/docs/openapi.yaml](backend/docs/openapi.yaml) and summarized in [docs/API.md](docs/API.md).

**Authentication:** All API requests require an `X-API-Key` header.

**Base URL:** `/api/v1`

Key endpoints:

```
GET    /api/v1/books              List books (paginated, filterable)
POST   /api/v1/books              Create a book
GET    /api/v1/books/:id          Get book details
PUT    /api/v1/books/:id          Update book
DELETE /api/v1/books/:id          Delete book
POST   /api/v1/books/:id/upload   Upload file (PDF/EPUB)
GET    /api/v1/books/:id/download Download book file
POST   /api/v1/catalog/isbn/:isbn Look up ISBN metadata
GET    /api/v1/search             Full-text search
GET    /api/v1/export             Bulk JSONL export
POST   /api/v1/webhooks           Register webhook
GET    /api/v1/webhooks           List webhooks
DELETE /api/v1/webhooks/:id       Delete webhook
```

---

## Development

```bash
# Start all services in dev mode (with hot reload)
make dev

# Run backend tests
make test

# Build production images
make build

# Run database migrations
make migrate

# Format and lint
make lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

---

## Deployment

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for:
- Single-node Docker Compose deployment
- Nginx reverse proxy configuration
- SSL/TLS with Let's Encrypt
- Environment variable reference
- Backup and restore procedures

---

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for:
- System architecture diagram
- Service interaction diagram
- Book ingestion pipeline
- API design decisions

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

Built with ❤️ by the open source community.
