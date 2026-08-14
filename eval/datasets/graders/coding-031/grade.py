#!/usr/bin/env python3
"""Deterministic grader for coding-031 (regression twin of coding-026, sum_list).

Twin purpose: proves the sum_list fix did not break the sibling behavior —
sum_list must still sum non-empty lists correctly, including negative numbers
and floats (and return 0 for an empty list, per the fix contract).

Reads the candidate's Python code from stdin, extracts the sum_list function,
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


# Sibling behavior (twin of coding-026): sum_list must still sum non-empty
# lists correctly, including negatives and floats.
_check("sum_list([1, 2, 3]) == 6", _m.sum_list([1, 2, 3]) == 6)
_check("sum_list([-1, -2, -3]) == -6", _m.sum_list([-1, -2, -3]) == -6)
_check("sum_list([-1, 2]) == 1", _m.sum_list([-1, 2]) == 1)
_check("sum_list([1.5, 2.5]) == 4.0", _m.sum_list([1.5, 2.5]) == 4.0)
_check("sum_list([-2.5, 1]) == -1.5", _m.sum_list([-2.5, 1]) == -1.5)
_check("sum_list([7]) == 7", _m.sum_list([7]) == 7)
_check("sum_list([-3.0]) == -3.0", _m.sum_list([-3.0]) == -3.0)
_check("sum_list([1, 2.5, 3]) == 6.5", _m.sum_list([1, 2.5, 3]) == 6.5)
# Fix contract stated in the prompt: empty list sums to 0.
_check("sum_list([]) == 0", _m.sum_list([]) == 0)
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def sum_list")
    if "def sum_list" not in code:
        print("candidate does not define a sum_list function", file=sys.stderr)
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
