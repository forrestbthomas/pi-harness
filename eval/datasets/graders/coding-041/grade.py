#!/usr/bin/env python3
"""Deterministic grader for coding-041 (bash: env-var default with :-).

Reads the candidate's bash one-liner from stdin and runs it in a temp dir
under three environment modes: PORT unset, PORT=9000, PORT="" (empty). Each
run must print exactly the resolved value (8080 / 9000 / 8080) and exit 0.
This rejects the plausible ${PORT-8080} trap (empty PORT must still fall
back to the default) and a bare $PORT with no default at all. Exit 0 =
pass, exit 1 = fail.
"""

import os
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


def run_bash(script, cwd, env):
    """Run the candidate one-liner with a specific environment; fall back to
    command-like lines when prose around the command trips bash."""
    last = None
    for text in _command_candidates(script):
        proc = subprocess.run(
            ["bash", "-c", text],
            cwd=cwd,
            env=env,
            capture_output=True,
            text=True,
            timeout=30,
        )
        if proc.returncode == 0:
            if proc.stdout.strip():
                return proc.stdout.strip(), None
            # Exit 0 with empty output can never satisfy the exact-value
            # checks (e.g. `${PORT-8080}` with PORT set-but-empty), so keep
            # looking instead of shadowing the real command.
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


def main():
    script = extract_shell(sys.stdin.read())
    if not script:
        print("no shell command found in candidate", file=sys.stderr)
        return 1
    # (label, PORT value or None for unset, expected output)
    modes = [
        ("PORT unset", None, "8080"),
        ("PORT=9000", "9000", "9000"),
        ("PORT empty", "", "8080"),
    ]
    with tempfile.TemporaryDirectory() as td:
        for label, port, expected in modes:
            env = {k: v for k, v in os.environ.items() if k != "PORT"}
            if port is not None:
                env["PORT"] = port
            out, err = run_bash(script, td, env)
            if err is not None:
                print(f"[{label}] candidate failed under bash:\n{err}", file=sys.stderr)
                return 1
            if out != expected:
                print(
                    f"[{label}] expected {expected!r}, got {out!r}",
                    file=sys.stderr,
                )
                return 1
    print("PORT resolved to 8080/9000/8080 across unset, set, and empty modes")
    return 0


if __name__ == "__main__":
    sys.exit(main())
