#!/usr/bin/env bash
set -euo pipefail
KEY="${1:?usage: $0 KEY}"

if [[ ! -f config.ini ]]; then
  echo "config file missing" >&2
  exit 2
fi

LINE=$(grep "^${KEY}=" config.ini || true)
if [[ -z "$LINE" ]]; then
  echo "key not found: $KEY" >&2
  exit 1
fi
echo "${LINE#*=}"
