#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$PROJECT_DIR/.env"
PATCH_DIR="$PROJECT_DIR/db/patches"

if [[ ! -f "$ENV_FILE" ]]; then
  ENV_FILE="$PROJECT_DIR/.env.example"
fi

set -a
# shellcheck source=/dev/null
source "$ENV_FILE"
set +a

if [[ ! -d "$PATCH_DIR" ]]; then
  echo "patch directory not found: $PATCH_DIR" >&2
  exit 1
fi

shopt -s nullglob
patches=("$PATCH_DIR"/*.sql)

if (( ${#patches[@]} == 0 )); then
  echo "no patches found"
  exit 0
fi

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres sh -lc 'until pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" >/dev/null 2>&1; do sleep 1; done'

for patch_file in "${patches[@]}"; do
  echo "applying patch: $(basename "$patch_file")"
  docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
    psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f /dev/stdin < "$patch_file"
done

echo "all patches applied"