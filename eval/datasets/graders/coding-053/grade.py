#!/usr/bin/env python3
"""Deterministic grader for coding-053 (agentic: read providers.json).

The agent must USE the read tool on providers.json and report the real
provider count (17). A guessed count fails. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "17" not in out:
        print("answer does not report 17 providers", file=sys.stderr)
        return 1
    if "provider" not in out:
        print("answer does not identify what was counted", file=sys.stderr)
        return 1
    print("answer reports the provider count (tool-grounded)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
