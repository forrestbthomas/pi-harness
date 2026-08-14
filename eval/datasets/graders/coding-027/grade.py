#!/usr/bin/env python3
"""Deterministic grader for coding-027 (bug-fix: flatten null handling).

Reads the candidate's JavaScript from stdin, compiles it together with hidden
tests, and runs the emitted flatten in node against nested arrays containing
null values, strings, and other falsy non-array values. Exit 0 = pass,
exit 1 = fail.
"""

import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


def extract_js(candidate):
    """Pull the candidate's JS out of a fenced block, else the whole text."""
    text = candidate.strip()
    fence = re.search(r"```(?:js|javascript)?\n(.*?)```", text, re.DOTALL)
    return fence.group(1).strip() if fence else text


HIDDEN_TESTS = """\
const assert = require("assert");
{CODE}
assert.deepStrictEqual(flatten([]), [], "empty input");
assert.deepStrictEqual(flatten([1, [2, [3, 4]], 5]), [1, 2, 3, 4, 5], "nested arrays to any depth");
assert.deepStrictEqual(flatten([null, [null, 1], 2]), [null, null, 1, 2], "null values must survive flattening");
assert.deepStrictEqual(flatten([[1, null], [2, [3, null]]]), [1, null, 2, 3, null], "nulls at any depth");
assert.deepStrictEqual(flatten([["a"], "b"]), ["a", "b"], "strings are atomic values, not recursed into");
assert.deepStrictEqual(flatten([[0, false, ""]]), [0, false, ""], "falsy non-array values preserved");
console.log("OK");
"""


def main():
    node = shutil.which("node")
    if not node:
        print("node runtime not found; cannot execute the JS candidate", file=sys.stderr)
        return 1
    code = extract_js(sys.stdin.read())
    if "flatten" not in code:
        print("candidate does not define a flatten function", file=sys.stderr)
        return 1
    harness = HIDDEN_TESTS.replace("{CODE}", code)
    with tempfile.TemporaryDirectory() as td:
        path = Path(td, "flatten_test.js")
        path.write_text(harness, encoding="utf-8")
        proc = subprocess.run([node, str(path)], capture_output=True, text=True, timeout=30)
    if proc.returncode != 0:
        print(
            f"hidden JS tests failed:\n{proc.stdout}\n{proc.stderr}",
            file=sys.stderr,
        )
        return 1
    print("flatten passes hidden null/nesting/string tests")
    return 0


if __name__ == "__main__":
    sys.exit(main())
