#!/usr/bin/env bash
# =============================================================================
# Arkheion — Backup Script
# =============================================================================
# Backs up:
#   1. PostgreSQL database (pg_dump)
#   2. MinIO data (via mc mirror)
#
# Backups are stored in ./backups/<timestamp>/
# Run via cron or manually.
#
# Usage:
#   ./scripts/backup.sh
#   BACKUP_DIR=/mnt/backup ./scripts/backup.sh
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Load .env
if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  source "$ROOT_DIR/.env"
  set +a
fi

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backups}/$TIMESTAMP"

log()  { echo "[backup] $*"; }
err()  { echo "[backup] ERROR: $*" >&2; exit 1; }

mkdir -p "$BACKUP_DIR"
log "Backup directory: $BACKUP_DIR"

# ---------------------------------------------------------------------------
# 1. PostgreSQL backup
# ---------------------------------------------------------------------------
PG_DUMP_FILE="$BACKUP_DIR/arkheion_${TIMESTAMP}.sql.gz"
log "Backing up PostgreSQL..."

docker compose exec -T postgres \
  pg_dump -U "${POSTGRES_USER:-arkheion}" "${POSTGRES_DB:-arkheion}" \
  | gzip > "$PG_DUMP_FILE"

PG_SIZE=$(du -sh "$PG_DUMP_FILE" | cut -f1)
log "PostgreSQL backup: $PG_DUMP_FILE ($PG_SIZE)"

# ---------------------------------------------------------------------------
# 2. MinIO data backup
# ---------------------------------------------------------------------------
MINIO_BACKUP_DIR="$BACKUP_DIR/minio"
mkdir -p "$MINIO_BACKUP_DIR"

log "Backing up MinIO bucket: ${MINIO_BUCKET:-arkheion}..."

docker compose exec -T minio sh -c "
  mc alias set local http://localhost:9000 '${MINIO_ACCESS_KEY:-minioadmin}' '${MINIO_SECRET_KEY}' --quiet
  mc mirror local/${MINIO_BUCKET:-arkheion} /tmp/backup --quiet
" || log "WARNING: MinIO backup failed — check MinIO is running"

# Copy from container's /tmp/backup
docker compose cp minio:/tmp/backup/. "$MINIO_BACKUP_DIR/" 2>/dev/null || \
  log "WARNING: Could not copy MinIO data from container"

MINIO_SIZE=$(du -sh "$MINIO_BACKUP_DIR" 2>/dev/null | cut -f1 || echo "unknown")
log "MinIO backup: $MINIO_BACKUP_DIR ($MINIO_SIZE)"

# ---------------------------------------------------------------------------
# 3. Write backup manifest
# ---------------------------------------------------------------------------
cat > "$BACKUP_DIR/manifest.txt" << EOF
Arkheion Backup
Timestamp: $TIMESTAMP
Date: $(date)

Files:
  - $(basename "$PG_DUMP_FILE") — PostgreSQL dump ($PG_SIZE)
  - minio/ — MinIO object storage ($MINIO_SIZE)

Restore instructions:
  PostgreSQL:
    zcat $PG_DUMP_FILE | psql \$DATABASE_URL

  MinIO:
    mc mirror $MINIO_BACKUP_DIR local/${MINIO_BUCKET:-arkheion}
EOF

log ""
log "Backup complete: $BACKUP_DIR"

# ---------------------------------------------------------------------------
# 4. (Optional) Prune old backups — keep last 7
# ---------------------------------------------------------------------------
KEEP_DAYS="${BACKUP_KEEP_DAYS:-7}"
log "Pruning backups older than $KEEP_DAYS days..."
find "${BACKUP_DIR%/*}" -maxdepth 1 -type d -mtime +"$KEEP_DAYS" -exec rm -rf {} + 2>/dev/null || true
log "Pruning complete."
