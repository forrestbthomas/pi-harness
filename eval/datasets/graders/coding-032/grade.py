#!/usr/bin/env python3
"""Deterministic grader for coding-032 (regression twin of coding-027, flatten).

Twin purpose: proves the flatten fix did not break the sibling behavior —
flatten must still flatten plain nested arrays (no nulls) correctly, at any
depth.

Reads the candidate's Python code from stdin, extracts the flatten function,
and runs it against hidden sibling-behavior tests. Exit 0 = pass, exit 1 = fail.
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


# Sibling behavior (twin of coding-027): flatten must still flatten plain
# nested arrays (no nulls) correctly, at any depth.
_check("flatten([]) == []", _m.flatten([]) == [])
_check("flatten([1, 2, 3]) == [1, 2, 3]", _m.flatten([1, 2, 3]) == [1, 2, 3])
_check("flatten([[1, 2], [3, 4]]) == [1, 2, 3, 4]", _m.flatten([[1, 2], [3, 4]]) == [1, 2, 3, 4])
_check("flatten([1, [2, 3]]) == [1, 2, 3]", _m.flatten([1, [2, 3]]) == [1, 2, 3])
_check("flatten([[1, [2, [3]]]]) == [1, 2, 3]", _m.flatten([[1, [2, [3]]]]) == [1, 2, 3])
_check("flatten([[[1]], 2, [3]]) == [1, 2, 3]", _m.flatten([[[1]], 2, [3]]) == [1, 2, 3])
_check("flatten([[1], [], [2, [3]]]) == [1, 2, 3]", _m.flatten([[1], [], [2, [3]]]) == [1, 2, 3])
_check("flatten(['a', ['b', 'c']]) == ['a', 'b', 'c']", _m.flatten(["a", ["b", "c"]]) == ["a", "b", "c"])
_check("flatten([0, [1, [2]]]) == [0, 1, 2]", _m.flatten([0, [1, [2]]]) == [0, 1, 2])
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def flatten")
    if "def flatten" not in code:
        print("candidate does not define a flatten function", file=sys.stderr)
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
