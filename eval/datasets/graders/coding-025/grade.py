#!/usr/bin/env python3
"""Deterministic grader for coding-025 (bash: sum integers in numbers.txt).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
against two numbers.txt fixtures (sums 16 and 75). Each run must print
exactly one line containing only the expected sum. Exit 0 = pass, exit 1 =
fail.
"""

import re
import subprocess
import sys
import tempfile
from pathlib import Path


def extract_shell(candidate):
    """Pull the candidate's shell out of a fenced block, else the whole text."""
    text = candidate.strip()
    fence = re.search(r"```(?:bash|sh)?\n(.*?)```", text, re.DOTALL)
    return fence.group(1).strip() if fence else text


FIXTURES = (
    ("3\n5\n-2\n10\n", "16"),
    ("100\n-50\n25\n", "75"),
)


def run_fixture(script, cwd, numbers_text):
    """Run the candidate; fall back to individual lines when prose wraps the
    command (agents sometimes emit an intro sentence before the command)."""
    attempts = [script] + [ln for ln in script.splitlines() if ln.strip()]
    last_err = None
    for text in attempts:
        proc = subprocess.run(
            ["bash", "-c", text],
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=30,
        )
        if proc.returncode != 0:
            last_err = proc.stderr
            continue
        tokens = [t for line in proc.stdout.splitlines() for t in line.split()]
        if tokens:
            return tokens, None
        last_err = proc.stderr or "candidate produced empty output"
    return None, last_err


def main():
    script = extract_shell(sys.stdin.read())
    if not script:
        print("no shell command found in candidate", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        for numbers_text, expected in FIXTURES:
            Path(td, "numbers.txt").write_text(numbers_text, encoding="utf-8")
            tokens, err = run_fixture(script, td, numbers_text)
            if err is not None:
                print(f"candidate failed under bash:\n{err}", file=sys.stderr)
                return 1
            if tokens != [expected]:
                print(
                    f"fixture sum {expected}: expected output {expected!r}, "
                    f"got tokens {tokens!r}",
                    file=sys.stderr,
                )
                return 1
    print("candidate printed the correct sum for both fixtures")
    return 0


if __name__ == "__main__":
    sys.exit(main())
