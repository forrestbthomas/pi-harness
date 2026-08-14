#!/usr/bin/env python3
"""Deterministic grader for coding-002 (concept: `is` vs `==` in Python).

A correct answer distinguishes VALUE equality (==, __eq__) from IDENTITY
(is, same object in memory) and notes the None singleton convention.
Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    checks = [
        ("value equality", ("value", "equal", "equality")),
        ("identity", ("identity", "same object", "object in memory")),
        ("none singleton", ("none", "singleton")),
    ]
    for label, words in checks:
        if not any(w in out for w in words):
            print(f"answer does not mention {label}", file=sys.stderr)
            return 1
    if "==" not in out and "equals" not in out and "value" not in out:
        print("answer does not reference == (value comparison)", file=sys.stderr)
        return 1
    if "is" not in out or ("identity" not in out and "same object" not in out):
        print("answer does not reference `is` (identity comparison)", file=sys.stderr)
        return 1
    print("answer distinguishes is (identity) from == (value equality)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
