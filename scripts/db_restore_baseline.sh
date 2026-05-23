#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASELINE_DIR="$PROJECT_DIR/backup/dev/baseline"
MANIFEST_FILE="$PROJECT_DIR/backup/manifest/dev/baseline/manifest.md"
BASELINE_ARG="${1:-}"

baseline_files=()
while IFS= read -r baseline_file; do
  baseline_files+=("$baseline_file")
done < <(find "$BASELINE_DIR" -maxdepth 1 -type f -name '*.dump' -print | sort -r)

if [[ -n "$BASELINE_ARG" ]]; then
  if [[ "$BASELINE_ARG" = /* ]]; then
    BASELINE_FILE="$BASELINE_ARG"
  else
    BASELINE_FILE="$BASELINE_DIR/$BASELINE_ARG"
  fi
elif (( ${#baseline_files[@]} > 0 )); then
  BASELINE_FILE="${baseline_files[0]}"
else
  echo "baseline restore failed: no local baseline dump found under $BASELINE_DIR" >&2
  if [[ -f "$MANIFEST_FILE" ]]; then
    echo "baseline manifest: $MANIFEST_FILE" >&2
  fi
  echo "請先從已驗證的本機 PostgreSQL 或其他已確認來源建立最小 baseline dump 與對應 backup manifest；不得用假資料替代。" >&2
  exit 1
fi

if [[ ! -f "$BASELINE_FILE" ]]; then
  echo "baseline restore failed: baseline file not found: $BASELINE_FILE" >&2
  exit 1
fi

echo "baseline manifest: $MANIFEST_FILE"
echo "baseline restore target: $BASELINE_FILE"

"$PROJECT_DIR/scripts/db_restore.sh" "$BASELINE_FILE"