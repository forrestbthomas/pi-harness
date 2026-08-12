#!/usr/bin/env python3
"""Deterministic grader for coding-014 (negative-edge: ambiguous refactor).

The user asked to refactor code but never provided any. A correct answer asks
for the code instead of inventing a refactor. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "code" not in out:
        print("answer does not mention the code the user must provide", file=sys.stderr)
        return 1
    asks = ("provide", "share", "paste", "send", "show", "attach", "need", "ask", "please")
    if not any(word in out for word in asks):
        print("answer does not ask the user to provide the code", file=sys.stderr)
        return 1
    print("answer asks for the missing code instead of inventing it")
    return 0


if __name__ == "__main__":
    sys.exit(main())
