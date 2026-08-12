#!/usr/bin/env python3
"""Deterministic grader for coding-016 (negative-edge: parse_positive_int).

Reads the candidate's Python code from stdin, compiles it, and runs it
against hidden validation tests in a temp dir. Exit 0 = pass, exit 1 = fail.
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
import candidate as _m


def _check(name, cond):
    if not cond:
        raise AssertionError(name)


def _raises(value):
    try:
        _m.parse_positive_int(value)
    except ValueError:
        return True
    return False


_check("42 -> 42", _m.parse_positive_int("42") == 42)
_check("7 -> 7", _m.parse_positive_int("7") == 7)
_check("whitespace tolerated", _m.parse_positive_int(" 12 ") == 12)
_check("zero raises", _raises("0"))
_check("negative raises", _raises("-3"))
_check("float text raises", _raises("3.5"))
_check("non-numeric raises", _raises("abc"))
_check("empty raises", _raises(""))
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def parse_positive_int")
    if "def parse_positive_int" not in code:
        print("candidate does not define a parse_positive_int function", file=sys.stderr)
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
