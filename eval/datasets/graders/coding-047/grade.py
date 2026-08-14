#!/usr/bin/env python3
"""Deterministic grader for coding-047 (negative-edge: process_file).

Reads the candidate's Python code from stdin, compiles it, and runs it
against hidden validation tests in a temp dir. process_file must return
stripped lines for an existing file and must exit 1 with the exact message
"error: no such file: <path>" on stderr for a missing file, an empty path,
or None — never a traceback. Exit 0 = pass, exit 1 = fail.
"""

import re
import subprocess
import sys
import tempfile
from pathlib import Path


def extract_python(candidate, marker):
    """Pull runnable Python out of the candidate: a fenced block if present,
    else the trailing text starting at ``marker``, trimmed to the longest
    prefix that compiles (agents often wrap code in prose)."""
    text = candidate.strip()
    fence = re.search(r"```[a-zA-Z0-9_+-]*\n(.*?)```", text, re.DOTALL)
    if fence:
        text = fence.group(1).strip()
    if marker:
        idx = text.find(marker)
        if idx != -1:
            text = text[idx:].strip()
    lines = text.splitlines()
    for cut in range(len(lines), 0, -1):
        chunk = "\n".join(lines[:cut])
        try:
            compile(chunk, "<candidate>", "exec")
            return chunk
        except SyntaxError:
            continue
    return text


HIDDEN_TESTS = """\
import subprocess
import sys

import candidate as _m


def _raises_error_message(path_arg):
    # Run the candidate in a subprocess so sys.exit(1) is observable.
    code = (
        "import candidate; candidate.process_file("
        + repr(path_arg)
        + ")"
    )
    return subprocess.run(
        [sys.executable, "-c", code], capture_output=True, text=True
    )


def _check(name, cond):
    if not cond:
        raise AssertionError(name)


# 1. existing file -> list of stripped lines, no exit
with open("data.txt", "w", encoding="utf-8") as fh:
    fh.write("alpha\\nbeta\\ngamma\\n")
_check("existing file returns stripped lines",
       _m.process_file("data.txt") == ["alpha", "beta", "gamma"])

# 2. missing file -> exit 1, exact stderr message
proc = _raises_error_message("missing.txt")
_check("missing file exits 1", proc.returncode == 1)
_check("missing file prints 'error: no such file: missing.txt' to stderr",
       "error: no such file: missing.txt" in proc.stderr)

# 3. empty path -> exit 1, error message on stderr (no traceback)
proc = _raises_error_message("")
_check("empty path exits 1", proc.returncode == 1)
_check("empty path prints error to stderr",
       "error: no such file" in proc.stderr and "Traceback" not in proc.stderr)

# 4. None path -> exit 1, error message on stderr (no traceback)
proc = _raises_error_message(None)
_check("None path exits 1", proc.returncode == 1)
_check("None path prints error to stderr",
       "error: no such file" in proc.stderr and "Traceback" not in proc.stderr)

print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def process_file")
    if "def process_file" not in code:
        print("candidate does not define a process_file function", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        Path(td, "candidate.py").write_text(code, encoding="utf-8")
        Path(td, "run_tests.py").write_text(HIDDEN_TESTS, encoding="utf-8")
        proc = subprocess.run(
            [sys.executable, str(Path(td, "run_tests.py"))],
            cwd=td,
            capture_output=True,
            text=True,
            timeout=30,
        )
    if proc.returncode != 0:
        print(
            f"hidden tests failed:\n{proc.stdout}\n{proc.stderr}",
            file=sys.stderr,
        )
        return 1
    print("all hidden tests passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
