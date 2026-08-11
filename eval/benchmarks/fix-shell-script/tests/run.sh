#!/usr/bin/env bash
# Benchmark verification: exit 0 = task solved, any other exit = fail.
set -euo pipefail
out="$(bash src/build.sh)"
if [ "$out" != "hello pi-harness" ]; then
  echo "build.sh printed: $out" >&2
  exit 1
fi
echo "test_build: all assertions passed"
