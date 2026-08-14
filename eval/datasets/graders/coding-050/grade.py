#!/usr/bin/env python3
"""Deterministic grader for coding-050 (harness-routing: config-check).

pi-run config-check runs deterministic harness-config checks with no API
key and no network access; exit 0 = all checks passed, exit 1 = failures
found. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()

    if not any(p in out for p in ("config-check", "config check")):
        print("answer does not name pi-run config-check", file=sys.stderr)
        return 1

    no_key = any(
        p in out
        for p in (
            "no key", "no keys", "without a key", "no api key",
            "no credentials", "doesn't need a key", "does not need a key",
        )
    )
    if not no_key:
        print("answer does not state that no API key is required", file=sys.stderr)
        return 1

    no_net = any(
        p in out
        for p in (
            "no network", "without network", "no network access",
            "offline", "doesn't need network", "does not need network",
            "no internet",
        )
    )
    if not no_net:
        print("answer does not state that no network access is required", file=sys.stderr)
        return 1

    if "exit" not in out or "0" not in out:
        print("answer does not explain the success exit code", file=sys.stderr)
        return 1

    if not any(p in out for p in ("exit code 1", "exit 1", "nonzero", "non-zero", "1 means", "failures")):
        print("answer does not explain the failure exit code", file=sys.stderr)
        return 1

    print("answer names config-check with no-key/no-network and exit codes")
    return 0


if __name__ == "__main__":
    sys.exit(main())
