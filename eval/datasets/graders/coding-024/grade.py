#!/usr/bin/env python3
"""Deterministic grader for coding-024 (Levenshtein edit distance).

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


_check("kitten -> sitting == 3", _m.levenshtein("kitten", "sitting") == 3)
_check("flaw -> lawn == 2", _m.levenshtein("flaw", "lawn") == 2)
_check("saturday -> sunday == 3", _m.levenshtein("saturday", "sunday") == 3)
_check("book -> back == 2", _m.levenshtein("book", "back") == 2)
_check("empty -> empty == 0", _m.levenshtein("", "") == 0)
_check("abc -> empty == 3", _m.levenshtein("abc", "") == 3)
_check("empty -> abc == 3", _m.levenshtein("", "abc") == 3)
_check("identical == 0", _m.levenshtein("abc", "abc") == 0)
_check("long identical == 0", _m.levenshtein("a" * 10, "a" * 10) == 0)
_result = _m.levenshtein("abc", "xyz")
_check("returns int, not float/bool", isinstance(_result, int) and not isinstance(_result, bool))
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def levenshtein")
    if "def levenshtein" not in code:
        print("candidate does not define a levenshtein function", file=sys.stderr)
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
