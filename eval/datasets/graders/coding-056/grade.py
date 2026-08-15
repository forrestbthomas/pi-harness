#!/usr/bin/env python3
"""Deterministic grader for coding-056 (agentic multi-turn: gate tuning).

Turn 2 challenges the turn-1 proposal with the variance concern. The final
answer must reconcile strictness with variance: acknowledge that a flat
threshold false-fails on noise, and propose a variance-aware rule. Exit 0 =
pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    checks = {
        "variance-aware rule": any(
            k in out for k in ("variance-aware", "variance aware", "run-step", "run step", "one run", "more than one run", "flake")
        ),
        "noise acknowledged": any(
            k in out for k in ("noise", "variance", "by chance", "random")
        ),
        "false-fail avoided": any(
            k in out for k in ("false-fail", "false fail", "flake", "not a regression", "tolerance")
        ),
    }
    failed = [name for name, ok in checks.items() if not ok]
    if failed:
        print("missing: " + ", ".join(failed), file=sys.stderr)
        return 1
    print("answer reconciles strictness with variance")
    return 0


if __name__ == "__main__":
    sys.exit(main())
