#!/usr/bin/env python3
"""Deterministic grader for coding-007 (is_palindrome).

Reads the candidate's Python code from stdin, compiles it, and runs it
against hidden tests in a temp dir. Exit 0 = pass, exit 1 = fail.
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


_check("racecar", _m.is_palindrome("racecar") is True)
_check("case-insensitive", _m.is_palindrome("Racecar") is True)
_check("punctuation ignored", _m.is_palindrome("A man, a plan, a canal: Panama") is True)
_check("not a palindrome", _m.is_palindrome("hello") is False)
_check("empty string", _m.is_palindrome("") is True)
_check("digits", _m.is_palindrome("12321") is True)
_check("mixed alnum", _m.is_palindrome("12a21") is True)
_check("plain two chars", _m.is_palindrome("ab") is False)
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def is_palindrome")
    if "def is_palindrome" not in code:
        print("candidate does not define an is_palindrome function", file=sys.stderr)
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
