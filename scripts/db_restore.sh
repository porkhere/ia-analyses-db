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
BACKUP_ARG="${1:-}"

backup_files=()
while IFS= read -r backup_file; do
  backup_files+=("$backup_file")
done < <(find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.dump' -print | sort -r)

if [[ -z "$BACKUP_ARG" ]]; then
  if (( ${#backup_files[@]} == 0 )); then
    echo "no backups found for $APP_ENV" >&2
    exit 1
  fi

  echo "可還原備份："
  for (( index=0; index<${#backup_files[@]}; index++ )); do
    size_bytes="$(wc -c < "${backup_files[$index]}" | tr -d ' ')"
    size_mb="$(awk -v bytes="$size_bytes" 'BEGIN { printf "%.2f", bytes / 1024 / 1024 }')"
    printf "  %d) %s (%s MB)\n" "$((index + 1))" "$(basename "${backup_files[$index]}")" "$size_mb"
  done
  echo "  n) 取消"

  while true; do
    read -r -p "請選擇要還原的備份：" selection
    if [[ "$selection" = "n" || "$selection" = "N" ]]; then
      echo "restore cancelled"
      exit 0
    fi

    if [[ "$selection" =~ ^[0-9]+$ ]] && (( selection >= 1 && selection <= ${#backup_files[@]} )); then
      BACKUP_FILE="${backup_files[$((selection - 1))]}"
      break
    fi

    echo "無效選擇，請輸入數字或 n"
  done
elif [[ "$BACKUP_ARG" = /* ]]; then
  BACKUP_FILE="$BACKUP_ARG"
elif [[ -f "$BACKUP_ARG" ]]; then
  BACKUP_FILE="$BACKUP_ARG"
else
  BACKUP_FILE="$BACKUP_DIR/$BACKUP_ARG"
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

echo "high-risk restore detected; creating pre-restore backup first"
"$PROJECT_DIR/scripts/db_backup.sh"

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres sh -lc 'until pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" >/dev/null 2>&1; do sleep 1; done'

docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  pg_restore -U "$PGUSER" -d "$PGDATABASE" --clean --if-exists --no-owner --no-privileges < "$BACKUP_FILE"

restore_validation="$(docker compose --project-directory "$PROJECT_DIR" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$PGUSER" -d "$PGDATABASE" -Atc "SELECT current_database() || '|' || COUNT(*) FROM pg_tables WHERE schemaname = 'public';")"

"$PROJECT_DIR/scripts/db_write_backup_manifest.sh" "$BACKUP_FILE" \
  --restore-completed true \
  --migrate-completed false \
  --schema-drift-checked false \
  --restart-completed false \
  --validation-completed false

echo "app_env: $APP_ENV"
echo "restore completed: $BACKUP_FILE"
echo "restore validation: $restore_validation"