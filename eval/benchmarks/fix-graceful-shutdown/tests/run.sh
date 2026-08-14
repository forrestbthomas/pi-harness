#!/usr/bin/env bash
# Benchmark verification: exit 0 = task solved, any other exit = fail.
set -euo pipefail
PYTHONPATH=src python3 tests/test_echo_shutdown.py
