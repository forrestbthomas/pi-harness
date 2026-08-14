#!/usr/bin/env python3
"""Deterministic grader for coding-035 (regression twin of coding-030,
merge_dicts).

Twin purpose: proves the merge_dicts fix did not break the sibling behavior —
merge_dicts must still return the correct merged result (b's keys win)
without raising, while also honoring the no-mutation fix contract.

Reads the candidate's Python code from stdin, extracts the merge_dicts
function, and runs it against hidden tests. Exit 0 = pass, exit 1 = fail.
"""

import re
import subprocess
import sys
import tempfile
from pathlib import Path


def extract_python(candidate, marker):
    """Pull runnable Python out of the candidate: a fenced block if present,
    else the trailing text starting at ``marker``, keeping any leading import
    statements, trimmed to the longest prefix that compiles."""
    text = candidate.strip()
    fence = re.search(r"```[a-zA-Z0-9_+-]*\n(.*?)```", text, re.DOTALL)
    if fence:
        text = fence.group(1).strip()
    idx = text.find(marker)
    if idx != -1:
        head = text[:idx]
        keep = [
            line for line in head.splitlines()
            if line.startswith(("import ", "from "))
        ]
        text = "\n".join(keep + [text[idx:].strip()]).strip()
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


# Sibling behavior (twin of coding-030): correct merged result, no raising.
_check("merge_dicts({'a': 1}, {'b': 2}) == {'a': 1, 'b': 2}", _m.merge_dicts({"a": 1}, {"b": 2}) == {"a": 1, "b": 2})
_check("merge_dicts({}, {}) == {}", _m.merge_dicts({}, {}) == {})
_check("b wins on overlap", _m.merge_dicts({"x": 1}, {"x": 2}) == {"x": 2})
_check("mixed overlap", _m.merge_dicts({"a": 1, "b": 2}, {"b": 3, "c": 4}) == {"a": 1, "b": 3, "c": 4})
_check("empty b", _m.merge_dicts({"a": 1}, {}) == {"a": 1})
_check("empty a", _m.merge_dicts({}, {"a": 1}) == {"a": 1})
_check("string keys and nested values", _m.merge_dicts({"k": [1, 2]}, {"j": {"n": 1}}) == {"k": [1, 2], "j": {"n": 1}})
# Fix contract stated in the prompt: neither input dict is modified.
a = {"a": 1}
b = {"b": 2}
_m.merge_dicts(a, b)
_check("input a not mutated", a == {"a": 1})
_check("input b not mutated", b == {"b": 2})
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
