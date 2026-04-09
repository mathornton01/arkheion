# Arkheion Backend

Go + Fiber REST API for Arkheion.

## Requirements

- Go 1.22+
- PostgreSQL 16 (via Docker or local)
- Meilisearch 1.x
- MinIO (or S3-compatible)
- Apache Tika 2.x

## Development

```bash
# Copy environment
cp ../.env.example ../.env

# Start infrastructure services
docker compose -f ../docker-compose.yml up -d postgres meilisearch minio tika

# Run migrations
../scripts/migrate.sh

# Start with hot reload (requires air: go install github.com/cosmtrek/air@latest)
air

# Or run directly
go run main.go
```

## Project Structure

```
backend/
├── main.go              Entry point — setup, wiring, graceful shutdown
├── config/config.go     Environment variable loading and validation
├── api/
│   ├── router.go        Route registration
│   ├── middleware/      Auth and CORS middleware
│   └── handlers/        HTTP request handlers
├── models/              Data model structs
├── db/
│   ├── db.go            PostgreSQL connection pool
│   └── migrations/      SQL migration files (001_init.sql → 005_webhooks.sql)
├── services/
│   ├── bundle.go        Service container (dependency injection)
│   ├── isbn.go          OpenLibrary + Google Books ISBN lookup
│   ├── tika.go          Apache Tika text extraction
│   ├── meilisearch.go   Meilisearch index management and search
│   ├── minio.go         MinIO file upload/download
│   └── webhook.go       Webhook dispatch with retries
└── docs/openapi.yaml    Full OpenAPI 3.0 specification
```

## Running Tests

```bash
go test -v -race -cover ./...

# Integration tests (requires real PostgreSQL)
DATABASE_URL=postgres://arkheion:password@localhost:5432/arkheion_test \
  go test -tags=integration ./...
```

## Building

```bash
go build -o bin/arkheion -ldflags="-X main.Version=1.0.0" ./...
```

## API Documentation

The OpenAPI spec is at `docs/openapi.yaml` and is served at
`/api/v1/docs/openapi.yaml` when the server is running.

Import into Insomnia, Postman, or any OpenAPI-compatible tool.
