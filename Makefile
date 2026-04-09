# =============================================================================
# Arkheion — Makefile
# =============================================================================

.PHONY: help dev build test lint migrate backup clean deploy logs \
        backend-build frontend-build backend-test frontend-test \
        docker-push setup

COMPOSE        = docker compose
COMPOSE_DEV    = docker compose -f docker-compose.yml -f docker-compose.dev.yml
REGISTRY       = ghcr.io/mathornton01
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
help:
	@echo ""
	@echo "  Arkheion — Self-hosted library management system"
	@echo ""
	@echo "  Usage: make <target>"
	@echo ""
	@echo "  Development"
	@echo "  -----------"
	@echo "  dev           Start all services in development mode (hot reload)"
	@echo "  dev-down      Stop development services"
	@echo "  logs          Follow logs for all services"
	@echo "  logs-backend  Follow backend logs only"
	@echo ""
	@echo "  Building"
	@echo "  --------"
	@echo "  build         Build production Docker images"
	@echo "  backend-build Build backend binary (local)"
	@echo "  frontend-build Build frontend (local)"
	@echo ""
	@echo "  Testing"
	@echo "  -------"
	@echo "  test          Run all tests"
	@echo "  backend-test  Run backend Go tests"
	@echo "  frontend-test Run frontend vitest tests"
	@echo "  lint          Lint backend and frontend"
	@echo ""
	@echo "  Operations"
	@echo "  ----------"
	@echo "  migrate       Run pending database migrations"
	@echo "  backup        Backup PostgreSQL and MinIO data"
	@echo "  setup         First-run setup (init DB, MinIO bucket)"
	@echo "  deploy        Build + push images + restart production stack"
	@echo "  clean         Remove build artifacts and dev containers"
	@echo ""

# =============================================================================
# Development
# =============================================================================

dev: check-env
	$(COMPOSE_DEV) up -d --build
	@echo ""
	@echo "  Arkheion development stack is running."
	@echo ""
	@echo "  Frontend:    http://localhost:5173"
	@echo "  Backend API: http://localhost:8080/api/v1"
	@echo "  MinIO:       http://localhost:9001"
	@echo "  Meilisearch: http://localhost:7700"
	@echo ""
	@echo "  Run 'make logs' to follow all logs."

dev-down:
	$(COMPOSE_DEV) down

logs:
	$(COMPOSE_DEV) logs -f

logs-backend:
	$(COMPOSE_DEV) logs -f arkheion-backend

logs-frontend:
	$(COMPOSE_DEV) logs -f arkheion-frontend

# =============================================================================
# Building
# =============================================================================

build:
	$(COMPOSE) build --no-cache
	@echo "Build complete. VERSION=$(VERSION)"

backend-build:
	cd backend && go build -o bin/arkheion -ldflags="-X main.Version=$(VERSION)" ./...
	@echo "Backend binary: backend/bin/arkheion"

frontend-build:
	cd frontend && npm ci && npm run build
	@echo "Frontend build: frontend/build/"

docker-push:
	docker tag arkheion-backend $(REGISTRY)/arkheion-backend:$(VERSION)
	docker tag arkheion-backend $(REGISTRY)/arkheion-backend:latest
	docker tag arkheion-frontend $(REGISTRY)/arkheion-frontend:$(VERSION)
	docker tag arkheion-frontend $(REGISTRY)/arkheion-frontend:latest
	docker push $(REGISTRY)/arkheion-backend:$(VERSION)
	docker push $(REGISTRY)/arkheion-backend:latest
	docker push $(REGISTRY)/arkheion-frontend:$(VERSION)
	docker push $(REGISTRY)/arkheion-frontend:latest

# =============================================================================
# Testing & Linting
# =============================================================================

test: backend-test frontend-test

backend-test:
	cd backend && go test -v -race -cover ./...

frontend-test:
	cd frontend && npm ci && npm run test

lint: backend-lint frontend-lint

backend-lint:
	cd backend && go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || \
	  echo "golangci-lint not installed — skipping (go vet passed)"

frontend-lint:
	cd frontend && npm ci && npm run lint && npm run check

# =============================================================================
# Operations
# =============================================================================

migrate:
	./scripts/migrate.sh

backup:
	./scripts/backup.sh

setup: check-env
	./scripts/setup.sh

deploy: build docker-push
	$(COMPOSE) pull
	$(COMPOSE) up -d --remove-orphans

# =============================================================================
# Cleanup
# =============================================================================

clean:
	$(COMPOSE_DEV) down -v --remove-orphans 2>/dev/null || true
	rm -f backend/bin/arkheion
	rm -rf frontend/build frontend/.svelte-kit

# =============================================================================
# Helpers
# =============================================================================

check-env:
	@test -f .env || (echo "ERROR: .env not found. Run: cp .env.example .env" && exit 1)
