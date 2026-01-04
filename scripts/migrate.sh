#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MIGRATIONS_DIR="$ROOT_DIR/internal/storage/pgstorage/migrations"

DEFAULT_DATABASE_URL="postgresql://cs_parser:cs_parser@localhost:5432/cs_parser?sslmode=disable"
DATABASE_URL="${DATABASE_URL:-$DEFAULT_DATABASE_URL}"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql client is required to run migrations" >&2
  exit 1
fi

echo "Applying migrations from: $MIGRATIONS_DIR"
echo "Database URL: $DATABASE_URL"
echo ""

for migration in $(ls "$MIGRATIONS_DIR"/*.sql | sort); do
  echo "Applying migration: $(basename $migration)"
  psql "$DATABASE_URL" -f "$migration"
  if [ $? -eq 0 ]; then
    echo "✓ Successfully applied $(basename $migration)"
  else
    echo "✗ Failed to apply $(basename $migration)" >&2
    exit 1
  fi
  echo ""
done

echo "All migrations applied successfully!"

