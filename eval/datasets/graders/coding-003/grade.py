#!/usr/bin/env python3
"""Deterministic grader for coding-003 (bash find: files modified in 24h).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
containing a recently modified file and an ancient file. The output must
list the recent file and omit the ancient one. Exit 0 = pass, exit 1 = fail.
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
    after the command trips bash (e.g. '...\\n\\nThis lists recent files')."""
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
        (root / "new.txt").write_text("recent", encoding="utf-8")
        old = root / "old.txt"
        old.write_text("ancient", encoding="utf-8")
        subprocess.run(["touch", "-t", "200001010000", str(old)], check=True, timeout=30)
        try:
            out = run_bash(command, root)
        except RuntimeError as exc:
            print(exc, file=sys.stderr)
            return 1
        if "new.txt" not in out:
            print(f"output does not list the recently modified file:\n{out}", file=sys.stderr)
            return 1
        if "old.txt" in out:
            print(f"output lists a file older than 24h:\n{out}", file=sys.stderr)
            return 1
    print("find command lists only the recently modified file")
    return 0


if __name__ == "__main__":
    sys.exit(main())
