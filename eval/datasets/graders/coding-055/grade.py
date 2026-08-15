#!/usr/bin/env python3
"""Deterministic grader for coding-055 (bug-fix: root-cause a failed nightly).

The agent must identify the true root cause of a failing nightly eval — an
index/list mismatch in eval/tests/test_live_metrics.py where the judge-only
subset (_DATASET) and the full-dataset fixture (sample_cases) are indexed by
the same idx, so the judge grades each case's answer against a different
case's input/expected_output. It must also resist the red-herring versions
(deepeval/pi/Node differences — not the cause).

Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()

    # (1) The file must be named.
    if "test_live_metrics" not in out:
        print("answer does not name eval/tests/test_live_metrics.py", file=sys.stderr)
        return 1

    # (2) The data-structure mismatch must be described: the judge-only subset
    # vs the full-dataset fixture, indexed by the same idx.
    if "sample_cases" not in out:
        print("answer does not name the sample_cases fixture", file=sys.stderr)
        return 1
    if "_dataset" not in out and "judge-only" not in out and "judge only" not in out:
        print("answer does not name the judge-only subset (_DATASET)", file=sys.stderr)
        return 1
    if not any(k in out for k in ("idx", "index", "misalign", "mismatch", "different case")):
        print("answer does not describe the index/misalignment mechanism", file=sys.stderr)
        return 1

    # (3) The consequence: the judge grades against another case's input /
    # expected_output.
    if "expected_output" not in out and "another case" not in out and "wrong case" not in out:
        print("answer does not state the mis-grading consequence", file=sys.stderr)
        return 1

    # (4) The version differences must be explicitly marked as NOT the cause
    # (the red-herring-resistance check).
    if not any(k in out for k in ("red herring", "distraction", "not the cause", "not the root cause")):
        print("answer does not dismiss the version differences as red herrings", file=sys.stderr)
        return 1

    print("answer names the root cause (idx mismatch in test_live_metrics) and dismisses the version red herrings")
    return 0


if __name__ == "__main__":
    sys.exit(main())
