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

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres sh -lc 'until pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" >/dev/null 2>&1; do sleep 1; done'

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f /dev/stdin < "$PROJECT_DIR/db/init/001_schema.sql"

echo "seed sync completed from db/init/001_schema.sql"