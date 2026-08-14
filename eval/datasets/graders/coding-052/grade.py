#!/usr/bin/env python3
"""Deterministic grader for coding-052 (agentic: list graders dir).

The agent must USE ls/find on eval/datasets/graders/ and report the real
count (45). A hallucinated count fails. Exit 0 = pass, exit 1 = fail.
"""

import re
import sys


def main():
    out = sys.stdin.read().lower()
    # The count may be stated as "45", "45 graders", "forty-five", or as a
    # range around the true value — but it must be recognizably 45.
    if "45" not in out:
        print("answer does not report 45 graders", file=sys.stderr)
        return 1
    if "grad" not in out and "grader" not in out:
        print("answer does not identify what was counted", file=sys.stderr)
        return 1
    print("answer reports the grader count (tool-grounded)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
