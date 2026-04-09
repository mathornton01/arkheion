# Contributing to Arkheion

Thank you for your interest in contributing to Arkheion! This document outlines how to get involved, from reporting bugs to submitting pull requests.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Submitting Issues](#submitting-issues)
- [Submitting Pull Requests](#submitting-pull-requests)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)

---

## Code of Conduct

Arkheion follows the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) Code of Conduct. By participating, you agree to uphold a welcoming and respectful environment for all contributors.

In short: be kind, be constructive, assume good faith.

---

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/arkheion.git
   cd Arkheion
   ```
3. **Add the upstream remote:**
   ```bash
   git remote add upstream https://github.com/mathornton01/arkheion.git
   ```
4. **Create a feature branch:**
   ```bash
   git checkout -b feat/my-new-feature
   ```

---

## Development Setup

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker Engine 24+ and Docker Compose v2
- `make` (GNU Make)

### Starting the Development Environment

```bash
# Copy environment files
cp .env.example .env
cp frontend/.env.example frontend/.env

# Start all services with dev overrides (hot reload on both frontend and backend)
make dev
```

This starts:
- All infrastructure services (PostgreSQL, Meilisearch, Tika, MinIO) via Docker
- The backend with `air` for hot reload (Go)
- The frontend with Vite dev server (SvelteKit)

### Backend Only

```bash
cd backend
go mod download
air   # requires github.com/cosmtrek/air installed
# or
go run main.go
```

### Frontend Only

```bash
cd frontend
npm install
npm run dev
```

### Running Migrations

```bash
make migrate
# or
./scripts/migrate.sh
```

---

## Project Structure

```
Arkheion/
├── backend/          # Go + Fiber API server
│   ├── api/          # HTTP handlers and middleware
│   ├── config/       # Environment configuration
│   ├── db/           # PostgreSQL connection and migrations
│   ├── models/       # Data model structs
│   └── services/     # Business logic (ISBN, Tika, MinIO, etc.)
├── frontend/         # SvelteKit application
│   └── src/
│       ├── lib/      # Shared utilities and API client
│       └── routes/   # SvelteKit file-based routes
├── docs/             # Project documentation
└── scripts/          # Operational scripts
```

---

## Submitting Issues

Before opening an issue:
1. Search existing issues to avoid duplicates
2. Check the [FAQ in the README](README.md) for common questions

**Bug reports** should include:
- Arkheion version (or commit hash)
- Docker version and OS
- Steps to reproduce the issue
- Expected vs. actual behavior
- Relevant log output (`docker compose logs backend`)

**Feature requests** should include:
- A clear description of the problem you're solving
- Your proposed solution or behavior
- Whether you're willing to implement it

Use GitHub issue labels:
- `bug` — something is broken
- `enhancement` — new feature or improvement
- `documentation` — docs only change
- `question` — clarification needed

---

## Submitting Pull Requests

### Before You Start

For significant changes, **open an issue first** to discuss the approach. This avoids wasted effort on PRs that won't be accepted.

For small fixes (typos, obvious bugs), just open a PR directly.

### PR Checklist

- [ ] Feature branch created from `main`
- [ ] Code follows the coding standards below
- [ ] All tests pass: `make test`
- [ ] Backend: `go vet ./...` passes
- [ ] Frontend: `npm run check` passes
- [ ] New functionality has tests
- [ ] Documentation updated if needed
- [ ] Commit messages are clear and descriptive
- [ ] PR description explains the change and links to relevant issues

### PR Title Format

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add bulk CSV import for book metadata
fix: handle nil author list in ISBN lookup response
docs: add Grimoire webhook setup guide
refactor: extract ISBN validation to shared utility
test: add handler tests for book deletion
chore: update Meilisearch client to v0.27
```

---

## Coding Standards

### Go (Backend)

- Follow standard Go formatting: run `gofmt` or `goimports` before committing
- Use `golangci-lint` for linting: `make lint`
- Error handling: always handle errors explicitly — no `_` discards of errors in non-test code
- Use `context.Context` throughout for cancellation propagation
- Keep handlers thin — business logic belongs in `services/`, not `api/handlers/`
- All exported functions and types must have Go doc comments
- Use `pgxpool` for database connections — never open a raw connection in a handler

### SvelteKit (Frontend)

- Use TypeScript where possible (`.ts`, `.svelte` with `<script lang="ts">`)
- Format with Prettier: `npm run format`
- Lint with ESLint: `npm run lint`
- Keep components small and focused — extract reusable pieces to `src/lib/`
- Use Svelte stores for cross-component state
- API calls go through `src/lib/api.js` — never `fetch` directly from a component
- Avoid inline styles — use app.css or component `<style>` blocks

### SQL Migrations

- Each migration is a single numbered SQL file in `backend/db/migrations/`
- Migrations are **append-only** — never edit an existing migration
- Include both `-- up` and a comment explaining what the migration does
- Use UUIDs as primary keys (`gen_random_uuid()`)
- Add `created_at` and `updated_at` to all entity tables

---

## Testing

### Backend Tests

```bash
cd backend
go test ./...

# With coverage
go test -cover ./...

# Run specific package tests
go test ./services/...
go test ./api/handlers/...
```

Tests use `testify` for assertions and `pgxmock` for database mocking. Integration tests that require a real PostgreSQL instance are tagged with `//go:build integration` and require the `DATABASE_URL` environment variable.

### Frontend Tests

```bash
cd frontend
npm run test        # Unit tests (vitest)
npm run test:e2e    # End-to-end tests (playwright, requires running stack)
```

### Test Coverage Goals

- Backend handlers: 80%+ coverage
- Backend services: 70%+ coverage
- Frontend lib utilities: 80%+ coverage

---

## Documentation

Documentation lives in two places:

1. **Code comments** — Go doc comments on exported symbols, JSDoc on exported JS functions
2. **Markdown files** — `README.md`, `ARCHITECTURE.md`, `docs/*.md`

When you add a new API endpoint, update:
- `backend/docs/openapi.yaml` — the OpenAPI spec
- `docs/API.md` — the human-readable API reference

When you change a service interaction (new service dependency, changed data flow), update `ARCHITECTURE.md`.

---

## Release Process

Releases are managed by the maintainers. Versioning follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking API changes or major architectural changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, documentation, dependency updates

---

## Questions?

Open a [GitHub Discussion](https://github.com/mathornton01/arkheion/discussions) for questions that don't fit neatly into an issue. We're happy to help you get oriented in the codebase.

---

Thank you for contributing to Arkheion!
