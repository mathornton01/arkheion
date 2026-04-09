#!/usr/bin/env bash
# =============================================================================
# Arkheion — Database Migration Runner
# =============================================================================
# Applies all pending SQL migrations in order.
# Each migration file is idempotent (uses ON CONFLICT DO NOTHING on the
# schema_migrations table).
#
# Usage:
#   ./scripts/migrate.sh
#   DATABASE_URL=postgres://... ./scripts/migrate.sh
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="$ROOT_DIR/backend/db/migrations"

# Load .env if present
if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  source "$ROOT_DIR/.env"
  set +a
fi

DATABASE_URL="${DATABASE_URL:-}"
if [ -z "$DATABASE_URL" ]; then
  echo "ERROR: DATABASE_URL is not set." >&2
  exit 1
fi

log() { echo "[migrate] $*"; }

log "Running migrations from: $MIGRATIONS_DIR"
log "Database: ${DATABASE_URL%%@*}@..."

APPLIED=0
SKIPPED=0

for migration_file in $(ls "$MIGRATIONS_DIR"/*.sql | sort); do
  filename=$(basename "$migration_file")
  version="${filename%.sql}"

  # Check if migration already applied
  already_applied=$(psql "$DATABASE_URL" -tAc \
    "SELECT COUNT(*) FROM schema_migrations WHERE version = '$version'" 2>/dev/null || echo "0")

  if [ "${already_applied:-0}" = "1" ]; then
    log "  SKIP  $filename (already applied)"
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  log "  APPLY $filename"
  if psql "$DATABASE_URL" -f "$migration_file" -v ON_ERROR_STOP=1 -q; then
    log "  OK    $filename"
    APPLIED=$((APPLIED + 1))
  else
    echo "[migrate] FAILED: $filename" >&2
    exit 1
  fi
done

log ""
log "Done. Applied: $APPLIED, Skipped: $SKIPPED"
