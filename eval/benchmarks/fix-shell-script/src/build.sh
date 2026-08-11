#!/usr/bin/env bash
# Prints a greeting for $NAME (default: pi-harness).
set -euo pipefail
NAME="${NAME:-pi-harness}"
echo "hello ${NAM}"  # BUG: typo — ${NAM} is not the ${NAME} set above
