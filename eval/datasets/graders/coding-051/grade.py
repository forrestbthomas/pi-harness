#!/usr/bin/env python3
"""Deterministic grader for coding-051 (agentic: read go.mod).

The agent must USE the read tool on go.mod and report the real Go version
(1.26.5). A print-only model would guess a version; a tool-using agent reads
the file. Exit 0 = pass, exit 1 = fail.
"""

import re
import sys


def main():
    out = sys.stdin.read().lower()
    # Accept the exact version or a tolerant "go 1.26" form, but require it to
    # look like a real version number the agent could only have read.
    if "1.26" not in out:
        print("answer does not report Go 1.26.x from go.mod", file=sys.stderr)
        return 1
    if not re.search(r"go\s*(1\.26(\.\d+)?)", out) and "1.26.5" not in out:
        print("answer does not name the go.mod go directive version", file=sys.stderr)
        return 1
    print("answer reports the go.mod Go version (tool-grounded)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
