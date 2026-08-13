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
    """Run the candidate one-liner; fall back to command-like lines when prose
    around the command trips bash (real LLM output is prose-heavy)."""
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
                return proc.stdout
            # Exit 0 with empty output is a vacuous pipeline (e.g. a GNU-only
            # `stat --printf` on macOS: errors go to stderr, awk still exits
            # 0). Empty output can never satisfy the grader's content checks,
            # so keep looking instead of shadowing the real command.
            last = proc.stderr or "candidate produced empty output"
            continue
        last = proc.stderr
    raise RuntimeError(f"candidate failed under bash:\n{last}")


# Command starters that mark a line as runnable shell even when it sits inside
# prose or contains parentheticals (real LLM answers are prose-heavy: bullets
# and multi-option answers are the norm, e.g. "- Recursive (GNU find):").
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
    never executed - the old first-line/last-line fallback ran bullets like
    "- Recursive ..." and died with a confusing "bash: - : invalid option"."""
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
