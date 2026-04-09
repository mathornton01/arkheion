#!/usr/bin/env bash
# =============================================================================
# Arkheion — First-Run Setup Script
# =============================================================================
# Runs database migrations and initializes MinIO bucket.
# Run this once after starting the stack for the first time:
#   ./scripts/setup.sh
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Load .env
if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  source "$ROOT_DIR/.env"
  set +a
else
  echo "ERROR: .env not found. Copy .env.example to .env first."
  exit 1
fi

log() { echo "[setup] $*"; }
err() { echo "[setup] ERROR: $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# 1. Wait for PostgreSQL to be ready
# ---------------------------------------------------------------------------
log "Waiting for PostgreSQL to be ready..."
for i in $(seq 1 30); do
  if docker compose exec -T postgres pg_isready -U "${POSTGRES_USER:-arkheion}" -q 2>/dev/null; then
    log "PostgreSQL is ready."
    break
  fi
  echo -n "."
  sleep 2
  if [ "$i" -eq 30 ]; then
    err "PostgreSQL did not become ready in time."
  fi
done

# ---------------------------------------------------------------------------
# 2. Run database migrations
# ---------------------------------------------------------------------------
log "Running database migrations..."
"$SCRIPT_DIR/migrate.sh"
log "Migrations complete."

# ---------------------------------------------------------------------------
# 3. Initialize MinIO bucket
# ---------------------------------------------------------------------------
log "Initializing MinIO bucket..."
MINIO_ENDPOINT="${MINIO_ENDPOINT:-minio:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-}"
BUCKET="${MINIO_BUCKET:-arkheion}"

# Wait for MinIO
for i in $(seq 1 20); do
  if docker compose exec -T minio mc ready local --quiet 2>/dev/null; then
    break
  fi
  echo -n "."
  sleep 3
  if [ "$i" -eq 20 ]; then
    log "WARNING: MinIO health check timed out. The bucket may need to be created manually."
  fi
done

# Create bucket (MinIO mc via docker exec)
docker compose exec -T minio sh -c "
  mc alias set local http://localhost:9000 '${MINIO_ACCESS_KEY}' '${MINIO_SECRET_KEY}' --quiet
  mc mb --ignore-existing local/${BUCKET}
  echo 'Bucket ${BUCKET} ready'
" 2>/dev/null || log "WARNING: Could not initialize MinIO bucket automatically. Create it manually at the MinIO console."

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo ""
log "==========================================="
log " Arkheion setup complete!"
log "==========================================="
log ""
log "  Frontend: http://localhost:${FRONTEND_PORT:-3000}"
log "  Backend:  http://localhost:${BACKEND_PORT:-8080}/api/v1"
log "  MinIO:    http://localhost:9001 (user: ${MINIO_ACCESS_KEY})"
log ""
log "API key: ${ARKHEION_API_KEYS%%,*}"
log ""
