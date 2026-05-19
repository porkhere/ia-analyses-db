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

files=()
while IFS= read -r backup_file; do
  files+=("$backup_file")
done < <(find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.dump' -print | sort -r)

if (( ${#files[@]} == 0 )); then
  echo "no backups found for $APP_ENV"
  exit 0
fi

echo "app_env: $APP_ENV"
printf "%-4s %-22s %12s\n" "no" "backup_file" "size_mb"

for (( index=0; index<${#files[@]}; index++ )); do
  file="${files[$index]}"
  size_bytes="$(wc -c < "$file" | tr -d ' ')"
  size_mb="$(awk -v bytes="$size_bytes" 'BEGIN { printf "%.2f", bytes / 1024 / 1024 }')"
  printf "%-4s %-22s %12s\n" "$((index + 1))" "$(basename "$file")" "$size_mb"
done