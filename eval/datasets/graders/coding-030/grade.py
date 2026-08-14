#!/usr/bin/env python3
"""Deterministic grader for coding-030 (bug-fix: merge_dicts mutation).

Reads the candidate's fixed Python code from stdin, compiles it, and runs it
against hidden tests in a temp dir. merge_dicts must return a new dict with
b winning on conflicts and must NOT mutate either input. Exit 0 = pass,
exit 1 = fail.
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


_check("merge two dicts", _m.merge_dicts({"x": 1}, {"y": 2}) == {"x": 1, "y": 2})
_check("b wins on conflict", _m.merge_dicts({"x": 1}, {"x": 2, "z": 3}) == {"x": 2, "z": 3})
_check("both empty", _m.merge_dicts({}, {}) == {})

a = {"x": 1}
m = _m.merge_dicts(a, {"x": 2, "z": 3})
_check("merged result", m == {"x": 2, "z": 3})
_check("input a unchanged", a == {"x": 1})

b = {"y": 2}
m2 = _m.merge_dicts({}, b)
_check("input b unchanged", b == {"y": 2})
_check("result is a fresh dict", m2 is not b)

a2 = {"a": {"k": 1}}
m3 = _m.merge_dicts(a2, {"b": 2})
_check("nested value kept", m3 == {"a": {"k": 1}, "b": 2})
_check("input a unchanged (nested)", a2 == {"a": {"k": 1}})

a3 = {}
m4 = _m.merge_dicts(a3, {"x": 1})
_check("result not the input a object", m4 is not a3)
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def merge_dicts")
    if "def merge_dicts" not in code:
        print("candidate does not define a merge_dicts function", file=sys.stderr)
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
