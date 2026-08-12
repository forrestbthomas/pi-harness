#!/usr/bin/env python3
"""Deterministic grader for coding-015 (negative-edge: invalid model tier).

`--model-tier ultra` is not a valid tier: the answer must reject it rather
than invent a model. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "ultra" not in out:
        print("answer does not address the 'ultra' tier", file=sys.stderr)
        return 1
    rejects = (
        "invalid",
        "not a valid",
        "unknown",
        "no such",
        "does not exist",
        "not supported",
        "usage error",
    )
    if not any(word in out for word in rejects):
        print("answer does not reject 'ultra' as an invalid tier", file=sys.stderr)
        return 1
    print("answer rejects the unknown tier instead of hallucinating a model")
    return 0


if __name__ == "__main__":
    sys.exit(main())
