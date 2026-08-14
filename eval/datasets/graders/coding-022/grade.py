#!/usr/bin/env python3
"""Deterministic grader for coding-022 (merge intervals).

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


_check(
    "basic merge",
    _m.merge_intervals([[1, 3], [2, 6], [8, 10], [15, 18]]) == [[1, 6], [8, 10], [15, 18]],
)
_check("adjacent touch merges", _m.merge_intervals([[1, 4], [4, 5]]) == [[1, 5]])
_check("contained interval", _m.merge_intervals([[1, 4], [2, 3]]) == [[1, 4]])
_check("chain of contained", _m.merge_intervals([[1, 10], [2, 3], [4, 8]]) == [[1, 10]])
_check("empty input", _m.merge_intervals([]) == [])
_check("single point", _m.merge_intervals([[5, 5]]) == [[5, 5]])
_check("unsorted input", _m.merge_intervals([[6, 8], [1, 9], [2, 4]]) == [[1, 9]])
_check("no overlap", _m.merge_intervals([[1, 2], [3, 4]]) == [[1, 2], [3, 4]])

_input = [[1, 3], [2, 6]]
_result = _m.merge_intervals(_input)
_check("input not mutated", _input == [[1, 3], [2, 6]])
_check("new list returned", _result is not _input and _result == [[1, 6]])
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def merge_intervals")
    if "def merge_intervals" not in code:
        print("candidate does not define a merge_intervals function", file=sys.stderr)
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
