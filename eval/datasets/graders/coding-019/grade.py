#!/usr/bin/env python3
"""Deterministic grader for coding-019 (harness-routing: OTel best-effort).

The answer must describe the best-effort telemetry contract: a down collector
must never change the run's exit code. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "best" not in out or "effort" not in out:
        print("answer does not describe the export as best-effort", file=sys.stderr)
        return 1
    if not any(word in out for word in ("exit", "fail", "fatal")):
        print("answer does not state that telemetry failures never affect the run", file=sys.stderr)
        return 1
    print("answer describes the best-effort OTel contract")
    return 0


if __name__ == "__main__":
    sys.exit(main())
