#!/usr/bin/env python3
"""Deterministic grader for coding-010 (bug-fix: bash variable typo).

Reads the candidate's corrected bash script from stdin, writes it to a temp
file, and runs it with and without an explicit NAME env var. The output must
greet with the NAME value in both modes. Exit 0 = pass, exit 1 = fail.
"""

import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path


def extract_bash(candidate):
    """Pull the candidate's script out of a fenced block, else the whole text."""
    text = candidate.strip()
    fence = re.search(r"```(?:bash|sh)?\n(.*?)```", text, re.DOTALL)
    return fence.group(1).strip() if fence else text


def main():
    script = extract_bash(sys.stdin.read())
    if "NAME" not in script:
        print("candidate does not reference the NAME variable", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        path = Path(td, "fix.sh")
        path.write_text(script, encoding="utf-8")
        env_default = {k: v for k, v in os.environ.items() if k != "NAME"}
        proc = subprocess.run(
            ["bash", str(path)],
            capture_output=True,
            text=True,
            timeout=30,
            env=env_default,
        )
        if proc.returncode != 0 or "hello pi-harness" not in proc.stdout:
            print(
                f"default-mode run failed:\n{proc.stdout}\n{proc.stderr}",
                file=sys.stderr,
            )
            return 1
        env_custom = dict(env_default, NAME="world")
        proc = subprocess.run(
            ["bash", str(path)],
            capture_output=True,
            text=True,
            timeout=30,
            env=env_custom,
        )
        if proc.returncode != 0 or "hello world" not in proc.stdout:
            print(
                f"custom-mode run failed:\n{proc.stdout}\n{proc.stderr}",
                file=sys.stderr,
            )
            return 1
    print("script greets with NAME in both default and custom modes")
    return 0


if __name__ == "__main__":
    sys.exit(main())
