#!/usr/bin/env python3
"""Deterministic grader for coding-018 (harness-routing: --model-tier cheap).

The answer must explain the tier MECHANISM: --model-tier selects a model
WITHIN the explicit provider, and an unavailable/unknown tier is a hard usage
error with no fallback. We deliberately do NOT assert the exact cheap-tier
model id (currently openai/gpt-5-mini): the tier table may change (verified
against the pi catalog when shipped), so pinning the id here would break on a
legitimate catalog update. The reference answer may name the current id, but
the grader checks the mechanism only.
Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "cheap" not in out:
        print("answer does not mention the cheap tier", file=sys.stderr)
        return 1
    if "no fallback" not in out and "without fallback" not in out:
        print("answer must state that tier selection never falls back", file=sys.stderr)
        return 1
    if "usage error" not in out and "exit 2" not in out and "exit code 2" not in out:
        print("answer must state an unknown tier is a usage error", file=sys.stderr)
        return 1
    print("answer explains the within-provider tier mechanism with no fallback")
    return 0


if __name__ == "__main__":
    sys.exit(main())
