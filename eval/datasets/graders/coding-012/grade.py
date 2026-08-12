#!/usr/bin/env python3
"""Deterministic grader for coding-012 (bash: name of the largest file).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
with files of 10 / 50 / 100 bytes. The output must name the 100-byte file.
Exit 0 = pass, exit 1 = fail.
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
        (root / "small.txt").write_text("s" * 10, encoding="utf-8")
        (root / "medium.txt").write_text("m" * 50, encoding="utf-8")
        (root / "big.txt").write_text("b" * 100, encoding="utf-8")
        try:
            out = run_bash(command, root)
        except RuntimeError as exc:
            print(exc, file=sys.stderr)
            return 1
        if "big.txt" not in out:
            print(
                f"output does not name the largest file (big.txt):\n{out}",
                file=sys.stderr,
            )
            return 1
    print("largest file correctly identified")
    return 0


if __name__ == "__main__":
    sys.exit(main())
