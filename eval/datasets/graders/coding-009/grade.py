#!/usr/bin/env python3
"""Deterministic grader for coding-009 (bug-fix: load_config).

Reads the candidate's fixed Python code from stdin, applies it to a fixture
module, and asserts the corrected behavior against temp JSON config files.
Exit 0 = pass, exit 1 = fail.
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
import json
import tempfile

import candidate as _m


def _config(data):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False, encoding="utf-8") as f:
        json.dump(data, f)
        return f.name


def _check(name, cond):
    if not cond:
        raise AssertionError(name)


_check("full config", _m.load_config(_config({"timeout": 30, "retries": 5})) == (30, 5))
_check("missing timeout", _m.load_config(_config({"retries": 2})) == (None, 2))
_check("empty config", _m.load_config(_config({})) == (None, 3))
print("OK")
"""


def main():
    code = extract_python(sys.stdin.read(), "def load_config")
    if "def load_config" not in code:
        print("candidate does not define a load_config function", file=sys.stderr)
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
