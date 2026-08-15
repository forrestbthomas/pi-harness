#!/usr/bin/env python3
"""Deterministic grader for coding-058 (agentic multi-turn: follow-up clarification).

Turn 2 supplies the missing numbers. The final answer must incorporate them
and conclude the case stayed green because 1-of-5 against a 0.2 baseline is
within variance (a flake, not a regression). Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    checks = {
        "turn-2 data used": "0.2" in out or "20%" in out,
        "flake conclusion": any(k in out for k in ("flake", "not a regression", "variance", "one extra failure")),
        "within variance": any(k in out for k in ("variance", "within", "expected", "noise", "one run")),
    }
    failed = [name for name, ok in checks.items() if not ok]
    if failed:
        print("missing: " + ", ".join(failed), file=sys.stderr)
        return 1
    print("answer incorporates turn-2 info and concludes flake-not-regression")
    return 0


if __name__ == "__main__":
    sys.exit(main())
