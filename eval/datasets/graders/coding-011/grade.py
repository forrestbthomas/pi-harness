#!/usr/bin/env python3
"""Deterministic grader for coding-011 (bash: count lines in .txt files).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
with two .txt files (3 + 2 lines) and a non-.txt file (7 lines). The output
must total 5 lines across the .txt files. Exit 0 = pass, exit 1 = fail.
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


def run_bash(script, cwd):
    """Run the candidate one-liner; fall back to its first line when prose
    after the command trips bash."""
    tries = [script]
    lines = [ln for ln in script.splitlines() if ln.strip()]
    if len(lines) > 1:
        tries.append(lines[0])
        tries.append(lines[-1])
    last = None
    for text in tries:
        proc = subprocess.run(
            ["bash", "-c", text],
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=30,
        )
        if proc.returncode == 0:
            return proc.stdout
        last = proc.stderr
    raise RuntimeError(f"candidate failed under bash:\n{last}")


def main():
    command = extract_shell(sys.stdin.read())
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        (root / "a.txt").write_text("alpha\nbeta\ngamma\n", encoding="utf-8")
        (root / "b.txt").write_text("one\ntwo\n", encoding="utf-8")
        (root / "data.log").write_text(
            "x1\nx2\nx3\nx4\nx5\nx6\nx7\n", encoding="utf-8"
        )
        try:
            out = run_bash(command, root)
        except RuntimeError as exc:
            print(exc, file=sys.stderr)
            return 1
        numbers = re.findall(r"\d+", out)
        if "5" not in numbers:
            print(
                f"output does not report 5 total lines across .txt files:\n{out}",
                file=sys.stderr,
            )
            return 1
    print("line count is correct")
    return 0


if __name__ == "__main__":
    sys.exit(main())
