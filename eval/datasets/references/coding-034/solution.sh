#!/usr/bin/env bash
set -euo pipefail
PATTERN="${1:-TODO}"
FILE="${2:-notes.txt}"
if [[ ! -f "$FILE" ]]; then
  echo "error: file not found: $FILE" >&2
  exit 1
fi
grep "$PATTERN" "$FILE"
