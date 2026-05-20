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

APP_ENV="${APP_ENV:-dev}"
BACKUP_DIR="$PROJECT_DIR/backup/$APP_ENV"
TIMESTAMP="$(date +%Y-%m-%d-%H-%M)"
BACKUP_FILE="$BACKUP_DIR/${TIMESTAMP}.dump"

mkdir -p "$BACKUP_DIR"

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  pg_dump -U "$PGUSER" -d "$PGDATABASE" -Fc > "$BACKUP_FILE"

"$PROJECT_DIR/scripts/db_write_backup_manifest.sh" "$BACKUP_FILE"

backup_files=()
while IFS= read -r backup_file; do
  backup_files+=("$backup_file")
done < <(find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.dump' -print | sort -r)

if (( ${#backup_files[@]} > 5 )); then
  for (( index=5; index<${#backup_files[@]}; index++ )); do
    rm -f "${backup_files[$index]}"
    backup_relative="${backup_files[$index]#$PROJECT_DIR/backup/}"
    rm -f "$PROJECT_DIR/backup/manifest/${backup_relative%.dump}.md"
    echo "backup pruned: ${backup_files[$index]}"
  done
fi

echo "app_env: $APP_ENV"
echo "backup created: $BACKUP_FILE"