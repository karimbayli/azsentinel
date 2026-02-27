#!/usr/bin/env bash
# ============================================================
# Sentinel V2 — Database Backup Script
# Creates a compressed pg_dump of the sentinel database
# ============================================================
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/sentinel}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-sentinel}"
DB_USER="${DB_USER:-sentinel}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/sentinel_${TIMESTAMP}.sql.gz"

echo "📦 Starting database backup..."

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Run pg_dump
PGPASSWORD="${DB_PASSWORD:-sentinel_secret}" pg_dump \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --no-owner \
    --no-privileges \
    | gzip > "$BACKUP_FILE"

echo "✅ Backup created: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1))"

# Clean old backups
echo "🧹 Cleaning backups older than ${RETENTION_DAYS} days..."
find "$BACKUP_DIR" -name "sentinel_*.sql.gz" -mtime +"$RETENTION_DAYS" -delete

echo "📋 Current backups:"
ls -lh "$BACKUP_DIR"/sentinel_*.sql.gz 2>/dev/null || echo "   (none)"
