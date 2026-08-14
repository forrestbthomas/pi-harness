#!/usr/bin/env python3
"""Deterministic grader for coding-004 (concept: binary search complexity).

Correct answers state O(log n) (or an equivalent big-O logarithmic form).
Exit 0 = pass, exit 1 = fail.
"""

import re
import sys


def main():
    out = sys.stdin.read().lower()
    # Accept O(log n), O(log2 n), O(logn), O(log_2 n), theta/log big-O forms.
    if re.search(r"o\s*\(\s*log", out):
        print("answer states logarithmic complexity")
        return 0
    if "logarithmic" in out or "log n" in out or "logn" in out:
        print("answer states logarithmic complexity (textual)")
        return 0
    print("answer does not state O(log n) complexity", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
