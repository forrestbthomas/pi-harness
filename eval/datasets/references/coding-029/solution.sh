#!/usr/bin/env bash
set -euo pipefail
LOG_FILE="${LOG_FILE:-/var/log/app.log}"
if [[ -f "$LOG_FILE" ]]; then
    count=$(grep -c "ERROR" "$LOG_FILE" || true)
    echo "$count"
else
    echo "no log file"
    exit 0
fi
