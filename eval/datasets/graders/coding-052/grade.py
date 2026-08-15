#!/usr/bin/env python3
"""Deterministic grader for coding-052 (agentic: list graders dir).

The agent must USE ls/find on eval/datasets/graders/ and report the real
count. The expected count is derived from the filesystem at grade time
(self-healing): a hardcoded count rotted here (45 → 49 → 50 as the dataset
grew) and punished agents for reporting reality instead of hallucinating the
stale number. The reference answer is enforced against this count by the
dataset lint's oracle test, so drift now fails CI loudly instead of mis-grading
a correct tool-grounded answer. Exit 0 = pass, exit 1 = fail.
"""

import re
import sys
from pathlib import Path


def main():
    out = sys.stdin.read().lower()
    # eval/datasets/graders/ — count subdirectories that actually carry a
    # grade.py (the directory contains only grader-task dirs).
    graders_dir = Path(__file__).resolve().parent.parent
    count = sum(1 for d in graders_dir.iterdir() if d.is_dir() and (d / "grade.py").is_file())
    expected = str(count)

    # The count must be recognizably the real number (word boundary, so
    # "50" is not satisfied by "150" or "1050").
    if not re.search(rf"\b{expected}\b", out):
        print(f"answer does not report the real count ({count} deterministic grader scripts)", file=sys.stderr)
        return 1
    if "grad" not in out and "grader" not in out:
        print("answer does not identify what was counted", file=sys.stderr)
        return 1
    print(f"answer reports the grader count ({count}, tool-grounded)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
