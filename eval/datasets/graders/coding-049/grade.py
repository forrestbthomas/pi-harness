#!/usr/bin/env python3
"""Deterministic grader for coding-049 (harness-routing: provider key).

`--provider deepseek` with no DEEPSEEK_API_KEY anywhere must fail fast
(missing-key error, exit code 3) and must never silently fall back to the
OpenAI key. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()

    if "deepseek" not in out:
        print("answer does not mention the deepseek provider", file=sys.stderr)
        return 1

    key_mention = any(
        p in out
        for p in (
            "deepseek_api_key", "deepseek key", "deepseek's key",
            "key for deepseek", "api key for deepseek", "its key",
            "its api key",
        )
    )
    if not key_mention:
        print("answer does not identify the deepseek key as the missing piece", file=sys.stderr)
        return 1

    missing = any(
        w in out
        for w in ("missing", "not set", "unset", "absent", "no value", "not configured", "not available", "nowhere")
    )
    if not missing:
        print("answer does not say the deepseek key is missing/unset", file=sys.stderr)
        return 1

    fails = any(
        p in out
        for p in ("exit 3", "exit code 3", "aborts", "fails", "error", "refuses", "does not run", "never runs", "hard error")
    )
    if not fails:
        print("answer does not say the run fails instead of running", file=sys.stderr)
        return 1

    no_fallback = any(
        p in out
        for p in (
            "no fallback", "does not fall back", "doesn't fall back",
            "never falls back", "does not fall", "won't fall back",
            "doesn't switch", "does not switch", "doesn't use openai",
            "does not use openai", "hard error", "no silent",
        )
    )
    if not no_fallback:
        print("answer does not rule out falling back to OpenAI", file=sys.stderr)
        return 1

    print("answer describes fail-fast provider key resolution with no fallback")
    return 0


if __name__ == "__main__":
    sys.exit(main())
