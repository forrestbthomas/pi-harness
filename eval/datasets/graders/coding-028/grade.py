#!/usr/bin/env python3
"""Deterministic grader for coding-028 (bug-fix: parse_csv_line quotes).

Reads the candidate's fixed Python code from stdin, compiles it, and runs it
against hidden CSV parsing tests in a temp dir: quoted fields with commas,
escaped quotes, and empty fields must be handled exactly. Exit 0 = pass,
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


# {Q} is replaced with a double quote at runtime so the template never
# contains three consecutive quotes, which would close the triple-quoted literal.
HIDDEN_TESTS = """\
import candidate as _m


def _check(name, cond):
    if not cond:
        raise AssertionError(name)


_check("plain fields", _m.parse_csv_line("a,b,c") == ["a", "b", "c"])
_check("quoted comma", _m.parse_csv_line('{Q}a,b{Q},c') == ["a,b", "c"])
_check("middle quoted", _m.parse_csv_line('x,{Q}y,z{Q},w') == ["x", "y,z", "w"])
_check("escaped quote", _m.parse_csv_line('{Q}say {Q}{Q}hi{Q}{Q}{Q},ok') == ['say {Q}hi{Q}', "ok"])
_check("quoted empty", _m.parse_csv_line('{Q}{Q},{Q}{Q}') == ["", ""])
_check("trailing empty", _m.parse_csv_line("a,") == ["a", ""])
_check("leading empty", _m.parse_csv_line(",a") == ["", "a"])
_check("all empty", _m.parse_csv_line(",") == ["", ""])
_check("spaces inside quotes", _m.parse_csv_line('{Q} a , b {Q},c') == [" a , b ", "c"])
_check("single field", _m.parse_csv_line("solo") == ["solo"])
print("OK")
""".format(Q=chr(34))


def main():
    code = extract_python(sys.stdin.read(), "def parse_csv_line")
    if "def parse_csv_line" not in code:
        print("candidate does not define a parse_csv_line function", file=sys.stderr)
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
