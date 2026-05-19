#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$PROJECT_DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  ENV_FILE="$PROJECT_DIR/.env.example"
fi

set -a
# shellcheck source=/dev/null
source "$ENV_FILE"
set +a

SIZE_MB="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -Atc "SELECT ROUND(pg_database_size(current_database())::numeric / 1024 / 1024, 2);")"

echo "db_name: $PGDATABASE"
echo "size_mb: $SIZE_MB"