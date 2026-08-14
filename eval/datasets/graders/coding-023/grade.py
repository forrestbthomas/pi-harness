#!/usr/bin/env python3
"""Deterministic grader for coding-023 (JS full-array flatten).

Reads the candidate's JavaScript code from stdin, compiles it together with
hidden tests, and runs the emitted flatten in node against nesting/type/
immutability assertions. Exit 0 = pass, exit 1 = fail.
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

assert.deepStrictEqual(flatten([1, [2, 3], [[4]]]), [1, 2, 3, 4]);
assert.deepStrictEqual(flatten([]), []);
assert.deepStrictEqual(flatten([1, 2, 3]), [1, 2, 3]);
assert.deepStrictEqual(flatten([[1], [[2]], [[[3]]]]), [1, 2, 3]);
assert.deepStrictEqual(flatten(["a", ["b", ["c"]]]), ["a", "b", "c"]);
assert.deepStrictEqual(flatten([null, [undefined], 0]), [null, undefined, 0]);
assert.deepStrictEqual(flatten([[1, 2], [3], [4, 5]]), [1, 2, 3, 4, 5]);
assert.deepStrictEqual(flatten([[[]], [[], [7]]]), [7]);

const input = [1, [2]];
const out = flatten(input);
assert.strictEqual(input.length, 2, "input must not be mutated");
assert.strictEqual(Array.isArray(input[1]), true, "input must not be mutated");
assert.notStrictEqual(out, input, "must return a new array");
process.exit(0);
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
    print("flatten passes hidden nesting/type/immutability tests")
    return 0


if __name__ == "__main__":
    sys.exit(main())
