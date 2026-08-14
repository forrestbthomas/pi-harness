#!/usr/bin/env python3
"""Deterministic grader for coding-043 (bash: config parse + exit codes).

Reads the candidate's bash script from stdin, writes it to a temp file, and
runs it with `bash <script> KEY` against fixture configs. The script must
resolve KEY=VALUE entries (ignoring blanks and # comments) and obey three
exit codes: 0 with the value on stdout for a present key (including empty
values), 1 with "key not found" on stderr for an absent key (a key that is
only a prefix of another key must NOT match), and 2 with "config file
missing" on stderr when config.ini does not exist. Exit 0 = pass, exit 1 =
fail.
"""

import re
import subprocess
import sys
import tempfile
from pathlib import Path

# Comment + blank lines must be ignored; user= has an empty value.
CONFIG = "# comment line\n# port=9999\n\nport=8080\ntimeout=30\nuser=\n"


def extract_script(candidate):
    """Pull the candidate's script out of a fenced block, else the whole text."""
    text = candidate.strip()
    fence = re.search(r"```(?:bash|sh)?\n(.*?)```", text, re.DOTALL)
    return fence.group(1).strip() if fence else text


def run_script(script, cwd, key):
    """Run the candidate script as `bash solve.sh KEY` in cwd."""
    path = Path(cwd, "solve.sh")
    path.write_text(script, encoding="utf-8")
    return subprocess.run(
        ["bash", str(path), key],
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=30,
    )


def main():
    script = extract_script(sys.stdin.read())
    if not script:
        print("no bash script found in candidate", file=sys.stderr)
        return 1
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        (root / "config.ini").write_text(CONFIG, encoding="utf-8")

        # 1. Present key -> value on stdout, exit 0.
        proc = run_script(script, root, "port")
        if proc.returncode != 0 or proc.stdout.strip() != "8080":
            print(
                f"present key 'port': expected exit 0 + '8080', got "
                f"exit {proc.returncode} stdout {proc.stdout!r} stderr {proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # 2. Present key with empty value -> empty stdout, exit 0.
        proc = run_script(script, root, "user")
        if proc.returncode != 0 or proc.stdout.strip() != "":
            print(
                f"empty value for 'user': expected exit 0 + empty stdout, got "
                f"exit {proc.returncode} stdout {proc.stdout!r} stderr {proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # 3. Absent key -> exit 1, "key not found" on stderr.
        proc = run_script(script, root, "nokey")
        if proc.returncode != 1 or "key not found" not in proc.stderr:
            print(
                f"absent key 'nokey': expected exit 1 + 'key not found' on "
                f"stderr, got exit {proc.returncode} stdout {proc.stdout!r} "
                f"stderr {proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # 4. Prefix collision: 'time' must NOT match timeout=30.
        proc = run_script(script, root, "time")
        if proc.returncode != 1 or "key not found" not in proc.stderr:
            print(
                f"prefix key 'time': expected exit 1 + 'key not found' (must "
                f"not match timeout=30), got exit {proc.returncode} stdout "
                f"{proc.stdout!r} stderr {proc.stderr!r}",
                file=sys.stderr,
            )
            return 1

        # 5. Missing config file -> exit 2, "config file missing" on stderr.
        (root / "config.ini").unlink()
        proc = run_script(script, root, "port")
        if proc.returncode != 2 or "config file missing" not in proc.stderr:
            print(
                f"missing config.ini: expected exit 2 + 'config file missing' "
                f"on stderr, got exit {proc.returncode} stdout {proc.stdout!r} "
                f"stderr {proc.stderr!r}",
                file=sys.stderr,
            )
            return 1
    print("config parsing and exit-code contract (0/1/2) verified")
    return 0


if __name__ == "__main__":
    sys.exit(main())
