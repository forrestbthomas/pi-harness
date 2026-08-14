#!/usr/bin/env python3
"""Deterministic grader for coding-042 (bash: awk field filter + sort).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
against two data.txt fixtures (name,age,city rows). For each fixture the
output must be exactly the sorted list of cities whose age is > 30. This
rejects answers that forget the age filter (cut/plain awk) and answers that
forget the sort. Exit 0 = pass, exit 1 = fail.
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
    """Run the candidate one-liner; fall back to command-like lines when prose
    wraps the command (agents sometimes emit an intro sentence first)."""
    last = None
    for text in _command_candidates(script):
        proc = subprocess.run(
            ["bash", "-c", text],
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=30,
        )
        if proc.returncode == 0:
            if proc.stdout.strip():
                return proc.stdout.strip(), None
            last = proc.stderr or "candidate produced empty output"
            continue
        last = proc.stderr
    return None, last


# Command starters that mark a line as runnable shell even when it sits inside
# prose (real LLM output is prose-heavy: bullets and multi-option answers are
# the norm).
_SHELL_STARTERS = frozenset(
    (
        "find", "ls", "grep", "wc", "cat", "awk", "sed", "xargs", "echo",
        "head", "tail", "sort", "stat", "du", "touch", "cp", "mv", "rm",
        "mkdir", "chmod", "chown", "printf", "tr", "cut", "uniq", "tee",
        "diff", "ps", "kill", "tar", "curl", "wget", "python", "python3",
        "node", "bash", "sh", "export", "set", "cd", "for", "while", "if",
        "source", "alias", "shopt", "umask", "ulimit", "df", "file",
        "dirname", "basename", "readlink", "realpath", "date", "sleep",
        "test", "[", "true", "false", "exit", "return", "declare", "local",
    )
)


def _command_candidates(script):
    """Candidate bash commands, in priority order: the whole text first (a
    bare one-liner, or a fenced block already extracted by extract_shell),
    then each line that plausibly starts a shell command. Prose lines
    (bullets, sentence punctuation, parentheticals) are skipped so they are
    never executed."""
    tries = [script]
    for raw in script.splitlines():
        line = raw.strip()
        if not line or line.startswith(("-", "*", "(", ")", ">")):
            continue
        if line.startswith("#") and not line.startswith("#!"):
            continue
        first = line.split(None, 1)[0] if line.split() else ""
        if first in _SHELL_STARTERS:
            tries.append(line)
            continue
        if line.endswith((":", ".", ",", ";", ")")):
            continue  # prose sentence, not a command
        if any(ch in line for ch in ("|", "<", "$", "&", ";")):
            tries.append(line)
    return tries


FIXTURES = (
    # Rows over 30: bob(45)->rome, carol(31)->athens, eve(60)->zurich.
    (
        "alice,25,paris\nbob,45,rome\ncarol,31,athens\ndave,18,oslo\neve,60,zurich\n",
        ["athens", "rome", "zurich"],
    ),
    # Rows over 30: x(40)->beta, z(50)->gamma.
    ("x,40,beta\ny,10,alpha\nz,50,gamma\n", ["beta", "gamma"]),
)


def main():
    script = extract_shell(sys.stdin.read())
    if not script:
        print("no shell command found in candidate", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        for data, expected in FIXTURES:
            Path(td, "data.txt").write_text(data, encoding="utf-8")
            out, err = run_bash(script, td)
            if err is not None:
                print(f"candidate failed under bash:\n{err}", file=sys.stderr)
                return 1
            lines = out.splitlines()
            if lines != expected:
                print(
                    f"expected sorted cities {expected!r}, got {lines!r}",
                    file=sys.stderr,
                )
                return 1
    print("cities of people over 30 printed, sorted, for both fixtures")
    return 0


if __name__ == "__main__":
    sys.exit(main())
