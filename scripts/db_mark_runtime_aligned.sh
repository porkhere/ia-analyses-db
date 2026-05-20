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
BACKUP_ARG="${1:-}"

if [[ -z "$BACKUP_ARG" ]]; then
  echo "usage: $0 <backup-file>" >&2
  exit 1
fi

if [[ "$BACKUP_ARG" = /* ]]; then
  BACKUP_FILE="$BACKUP_ARG"
elif [[ -f "$BACKUP_ARG" ]]; then
  BACKUP_FILE="$BACKUP_ARG"
elif [[ "$BACKUP_ARG" == backup/* ]]; then
  BACKUP_FILE="$PROJECT_DIR/$BACKUP_ARG"
else
  BACKUP_FILE="$PROJECT_DIR/backup/$APP_ENV/$BACKUP_ARG"
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

"$PROJECT_DIR/scripts/db_write_backup_manifest.sh" "$BACKUP_FILE" \
  --restore-completed true \
  --migrate-completed true \
  --schema-drift-checked true \
  --restart-completed true \
  --validation-completed true

echo "runtime alignment marked in manifest: ${BACKUP_FILE#$PROJECT_DIR/}"