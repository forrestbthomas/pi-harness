#!/usr/bin/env python3
"""Deterministic grader for coding-029 (bug-fix: bash missing-file guard).

Reads the candidate's corrected bash script from stdin, writes it to a temp
file, and runs it against three fixtures: an existing log with matches, an
existing log with zero matches, and a missing log file. In every scenario the
script must print the expected output to stdout and exit 0. Exit 0 = pass,
exit 1 = fail.
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


def run_scenario(script_path, log_path):
    env = dict(os.environ)
    env["LOG_FILE"] = log_path
    return subprocess.run(
        ["bash", str(script_path)],
        capture_output=True,
        text=True,
        timeout=30,
        env=env,
    )


def main():
    script = extract_bash(sys.stdin.read())
    if "LOG_FILE" not in script:
        print("candidate does not reference the LOG_FILE variable", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        td = Path(td)
        script_path = td / "fix.sh"
        script_path.write_text(script, encoding="utf-8")
        (td / "with_matches.log").write_text(
            "INFO started\nERROR boom\nWARN note\nERROR again\n", encoding="utf-8"
        )
        (td / "no_matches.log").write_text(
            "INFO started\nWARN note\n", encoding="utf-8"
        )

        proc = run_scenario(script_path, str(td / "with_matches.log"))
        if proc.returncode != 0 or proc.stdout.strip() != "2":
            print(
                f"with-matches scenario failed: rc={proc.returncode} "
                f"out={proc.stdout!r} err={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        proc = run_scenario(script_path, str(td / "no_matches.log"))
        if proc.returncode != 0 or proc.stdout.strip() != "0":
            print(
                f"no-matches scenario failed: rc={proc.returncode} "
                f"out={proc.stdout!r} err={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        proc = run_scenario(script_path, str(td / "missing.log"))
        if proc.returncode != 0 or "no log file" not in proc.stdout:
            print(
                f"missing-file scenario failed: rc={proc.returncode} "
                f"out={proc.stdout!r} err={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1
    print("script handles matches, zero matches, and a missing log file")
    return 0


if __name__ == "__main__":
    sys.exit(main())
