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
TARGET="${1:-}"

if [[ "$TARGET" = "all" ]]; then
  find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.dump' ! -name '.gitkeep' -delete
  echo "all dump backups deleted"
  exit 0
fi

backup_files=()
while IFS= read -r backup_file; do
  backup_files+=("$backup_file")
done < <(find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.dump' -print | sort -r)

if [[ -z "$TARGET" ]]; then
  if (( ${#backup_files[@]} == 0 )); then
    echo "no backups found for $APP_ENV" >&2
    exit 1
  fi
  TARGET="${backup_files[0]}"
elif [[ "$TARGET" != /* ]]; then
  TARGET="$BACKUP_DIR/$TARGET"
fi

if [[ ! -f "$TARGET" ]]; then
  echo "backup file not found: $TARGET" >&2
  exit 1
fi

rm -f "$TARGET"
echo "app_env: $APP_ENV"
echo "backup deleted: $TARGET"