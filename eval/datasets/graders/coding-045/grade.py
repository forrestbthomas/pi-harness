#!/usr/bin/env python3
"""Deterministic grader for coding-045 (bash: most frequent word pipeline).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
against two words.txt fixtures. Each run must print exactly the count and
the most frequent word, count first (e.g. "3 banana"), on a single line.
This rejects the plausible `sort | uniq -c | head -1` answer (first
sorted run, not the maximum) and `uniq -c | sort -rn | head -1` without the
pre-sort (adjacent duplicates only). Exit 0 = pass, exit 1 = fail.
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
    # banana:3, apple:2, cherry:1 -> "3 banana".
    ("apple\nbanana\nbanana\napple\nbanana\ncherry\n", ["3", "banana"]),
    # y:3, x:2, z:1 -> "3 y".
    ("x\ny\nx\ny\nz\ny\n", ["3", "y"]),
)


def main():
    script = extract_shell(sys.stdin.read())
    if not script:
        print("no shell command found in candidate", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        for data, expected in FIXTURES:
            Path(td, "words.txt").write_text(data, encoding="utf-8")
            out, err = run_bash(script, td)
            if err is not None:
                print(f"candidate failed under bash:\n{err}", file=sys.stderr)
                return 1
            tokens = out.split()
            if tokens != expected:
                print(
                    f"expected {expected!r}, got {tokens!r}",
                    file=sys.stderr,
                )
                return 1
    print("most frequent word with count is correct for both fixtures")
    return 0


if __name__ == "__main__":
    sys.exit(main())
