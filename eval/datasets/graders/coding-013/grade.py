#!/usr/bin/env python3
"""Deterministic grader for coding-013 (concept: list vs tuple in Python).

A correct answer notes lists are MUTABLE (written with []) and tuples are
IMMUTABLE (written with ()) and gives a use-case hint for each.
Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "mutable" not in out and "change" not in out and "modified" not in out:
        print("answer does not describe list mutability", file=sys.stderr)
        return 1
    if "immutable" not in out and "cannot change" not in out and "cannot be modified" not in out:
        print("answer does not describe tuple immutability", file=sys.stderr)
        return 1
    if "list" not in out or "tuple" not in out:
        print("answer does not name both list and tuple", file=sys.stderr)
        return 1
    print("answer distinguishes mutable list from immutable tuple")
    return 0


if __name__ == "__main__":
    sys.exit(main())
