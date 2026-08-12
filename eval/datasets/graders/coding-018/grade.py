#!/usr/bin/env python3
"""Deterministic grader for coding-018 (harness-routing: --model-tier cheap).

The answer must explain that the cheap tier selects the provider's
cheap-tier model, which for openai is openai/gpt-5.1-mini.
Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "cheap" not in out:
        print("answer does not mention the cheap tier", file=sys.stderr)
        return 1
    if "gpt-5.1-mini" not in out:
        print("answer does not name the openai cheap-tier model (gpt-5.1-mini)", file=sys.stderr)
        return 1
    print("answer identifies the cheap-tier model for openai")
    return 0


if __name__ == "__main__":
    sys.exit(main())
