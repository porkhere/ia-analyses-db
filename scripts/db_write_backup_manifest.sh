#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_FILE="${1:-}"

if [[ -z "$BACKUP_FILE" ]]; then
  echo "usage: $0 <backup-file> [--restore-command <command>] [--restore-completed true|false] [--migrate-completed true|false] [--schema-drift-checked true|false] [--restart-completed true|false] [--validation-completed true|false]" >&2
  exit 1
fi
shift

if [[ "$BACKUP_FILE" != /* ]]; then
  BACKUP_FILE="$PROJECT_DIR/$BACKUP_FILE"
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

REL_PATH="${BACKUP_FILE#$PROJECT_DIR/}"
if [[ "$REL_PATH" == "$BACKUP_FILE" || "$REL_PATH" != backup/* ]]; then
  echo "backup file must be inside $PROJECT_DIR/backup: $BACKUP_FILE" >&2
  exit 1
fi

RESTORE_COMMAND=""
RESTORE_COMPLETED="false"
MIGRATE_COMPLETED="false"
SCHEMA_DRIFT_CHECKED="false"
RESTART_COMPLETED="false"
VALIDATION_COMPLETED="false"
STORAGE_TYPE="local"
AVAILABILITY_SCOPE="same-machine"
NOT_IN_GIT_REASON="依憲法附則第 1 條：DB backup 實體檔不入 git"
RULE_ADDENDUM="附則第 1 條：DB backup 實體檔不入 git"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --restore-command)
      RESTORE_COMMAND="$2"
      shift 2
      ;;
    --restore-completed)
      RESTORE_COMPLETED="$2"
      shift 2
      ;;
    --migrate-completed)
      MIGRATE_COMPLETED="$2"
      shift 2
      ;;
    --schema-drift-checked)
      SCHEMA_DRIFT_CHECKED="$2"
      shift 2
      ;;
    --restart-completed)
      RESTART_COMPLETED="$2"
      shift 2
      ;;
    --validation-completed)
      VALIDATION_COMPLETED="$2"
      shift 2
      ;;
    --storage-type)
      STORAGE_TYPE="$2"
      shift 2
      ;;
    --availability-scope)
      AVAILABILITY_SCOPE="$2"
      shift 2
      ;;
    --not-in-git-reason)
      NOT_IN_GIT_REASON="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

backup_name="$(basename "$BACKUP_FILE")"
backup_stem="${backup_name%.dump}"
backup_relative="${REL_PATH#backup/}"
manifest_file="$PROJECT_DIR/backup/manifest/${backup_relative%.dump}.md"
mkdir -p "$(dirname "$manifest_file")"

size_bytes="$(wc -c < "$BACKUP_FILE" | tr -d ' ')"
checksum="$(shasum -a 256 "$BACKUP_FILE" | awk '{print $1}')"
written_at="$(date '+%Y-%m-%d-%H:%M')"

if [[ "$backup_stem" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9]{2}-[0-9]{2}$ ]]; then
  created_at="$backup_stem"
else
  created_at="$(date -r "$BACKUP_FILE" '+%Y-%m-%d-%H:%M')"
fi

if [[ -z "$RESTORE_COMMAND" ]]; then
  case "$backup_relative" in
    dev/baseline/*)
      RESTORE_COMMAND="make dev-restore-baseline BASELINE_FILE=$backup_name"
      ;;
    dev/*)
      RESTORE_COMMAND="make dev-restore BACKUP_FILE=$backup_name"
      ;;
    prod/*)
      RESTORE_COMMAND="make prod-restore BACKUP_FILE=$backup_name"
      ;;
    *)
      RESTORE_COMMAND="make dev-restore BACKUP_FILE=$backup_name"
      ;;
  esac
fi

cat > "$manifest_file" <<EOF
# Backup Manifest

- backup_file_name: $backup_name
- backup_created_at: $created_at
- file_size_bytes: $size_bytes
- sha256: $checksum
- storage_type: $STORAGE_TYPE
- local_path: $REL_PATH
- availability_scope: $AVAILABILITY_SCOPE
- restore_command: $RESTORE_COMMAND
- not_in_git_reason: $NOT_IN_GIT_REASON
- rule_addendum: $RULE_ADDENDUM
- restore_completed: $RESTORE_COMPLETED
- migrate_completed: $MIGRATE_COMPLETED
- schema_drift_checked: $SCHEMA_DRIFT_CHECKED
- restart_completed: $RESTART_COMPLETED
- validation_completed: $VALIDATION_COMPLETED
- manifest_written_at: $written_at
EOF

echo "backup manifest written: ${manifest_file#$PROJECT_DIR/}"