#!/usr/bin/env python3
"""Deterministic grader for coding-054 (agentic: grep for a section).

The agent must USE grep/find to locate the 'Release Milestones' heading in
ROADMAP.md and name it. A model that guesses wrong (or claims it's in README)
fails. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "release milestones" not in out:
        print("answer does not name the 'Release Milestones' section", file=sys.stderr)
        return 1
    if "roadmap" not in out:
        print("answer does not identify ROADMAP.md as the file", file=sys.stderr)
        return 1
    print("answer names the Release Milestones heading in ROADMAP.md (tool-grounded)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
