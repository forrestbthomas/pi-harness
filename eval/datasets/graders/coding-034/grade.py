#!/usr/bin/env python3
"""Deterministic grader for coding-034 (regression twin of coding-029, grep).

Twin purpose: proves the grep-script fix did not break the sibling behavior —
when the file exists, the script must still print exactly the matching lines,
in order, to stdout (and exit 0), plus the missing-file fix contract.

Reads the candidate's bash script from stdin, writes it to a temp dir, and
runs it against a fixture file with several patterns. Exit 0 = pass,
exit 1 = fail.
"""

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


FIXTURE = """\
alpha line
TODO: first item
beta line
TODO: second item
gamma
"""


def run_script(script, args, cwd):
    return subprocess.run(
        ["bash", "script.sh"] + args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=30,
    )


def main():
    script = extract_bash(sys.stdin.read())
    if "grep" not in script:
        print("candidate does not use grep", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        Path(td, "script.sh").write_text(script, encoding="utf-8")
        Path(td, "notes.txt").write_text(FIXTURE, encoding="utf-8")

        # Sibling behavior: file exists, pattern matches.
        proc = run_script(script, ["TODO", "notes.txt"], td)
        expected = "TODO: first item\nTODO: second item\n"
        if proc.returncode != 0 or proc.stdout != expected:
            print(
                f"exists+match run failed (rc={proc.returncode}):\n"
                f"stdout={proc.stdout!r}\nstderr={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # Sibling behavior: pattern matching a single line.
        proc = run_script(script, ["alpha", "notes.txt"], td)
        if proc.returncode != 0 or proc.stdout != "alpha line\n":
            print(
                f"single-line run failed (rc={proc.returncode}):\n"
                f"stdout={proc.stdout!r}\nstderr={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # Sibling behavior: no matching lines -> no stdout, non-zero exit.
        proc = run_script(script, ["ZZZ", "notes.txt"], td)
        if proc.returncode == 0 or proc.stdout != "":
            print(
                f"no-match run failed (rc={proc.returncode}):\n"
                f"stdout={proc.stdout!r}\nstderr={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # Fix contract from the prompt: missing file -> clear stderr + exit 1.
        proc = run_script(script, ["TODO", "missing.txt"], td)
        if proc.returncode != 1 or proc.stdout != "" or "missing.txt" not in proc.stderr:
            print(
                f"missing-file run failed (rc={proc.returncode}):\n"
                f"stdout={proc.stdout!r}\nstderr={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # Defaults contract: pattern TODO, file notes.txt when no args given.
        proc = run_script(script, [], td)
        if proc.returncode != 0 or proc.stdout != expected:
            print(
                f"defaults run failed (rc={proc.returncode}):\n"
                f"stdout={proc.stdout!r}\nstderr={proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

    print("script greps and prints matching lines correctly (and handles missing file)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
