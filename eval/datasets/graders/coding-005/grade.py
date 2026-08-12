#!/usr/bin/env python3
"""Deterministic grader for coding-005 (JS debounce).

Reads the candidate's JavaScript code from stdin, compiles it together with
hidden tests, and runs the emitted debounce in node against timing/argument/
this-binding assertions. Exit 0 = pass, exit 1 = fail.
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
let calls = 0;
const debounced = debounce(() => { calls += 1; }, 20);
debounced();
debounced();
debounced();
setTimeout(() => {
  assert.strictEqual(calls, 1, "expected exactly one trailing call, got " + calls);
  let lastArgs = null;
  const fn2 = debounce(function (a, b) { lastArgs = [a, b]; }, 10);
  fn2(1, 2);
  setTimeout(() => {
    assert.deepStrictEqual(lastArgs, [1, 2], "arguments were not preserved");
    const obj = { v: 42, run: null };
    obj.run = debounce(function () { this.got = this.v; }, 10);
    obj.run();
    setTimeout(() => {
      assert.strictEqual(obj.got, 42, "this binding was lost");
      process.exit(0);
    }, 40);
  }, 40);
}, 80);
"""


def main():
    node = shutil.which("node")
    if not node:
        print("node runtime not found; cannot execute the JS candidate", file=sys.stderr)
        return 1
    code = extract_js(sys.stdin.read())
    if "debounce" not in code:
        print("candidate does not define a debounce function", file=sys.stderr)
        return 1
    harness = HIDDEN_TESTS.replace("{CODE}", code)
    with tempfile.TemporaryDirectory() as td:
        path = Path(td, "debounce_test.js")
        path.write_text(harness, encoding="utf-8")
        proc = subprocess.run([node, str(path)], capture_output=True, text=True, timeout=30)
    if proc.returncode != 0:
        print(
            f"hidden JS tests failed:\n{proc.stdout}\n{proc.stderr}",
            file=sys.stderr,
        )
        return 1
    print("debounce passes hidden timing/args/this tests")
    return 0


if __name__ == "__main__":
    sys.exit(main())
