#!/usr/bin/env python3
"""Deterministic grader for coding-018 (harness-routing: --model-tier cheap).

The answer must explain the tier MECHANISM: --model-tier selects a model
WITHIN the explicit provider's tier table (openai cheap -> the provider's
cheap-tier model; no cross-provider fallback, no dynamic "globally cheapest
model" search). Paraphrases are accepted — real model answers phrase this as
"resolves the tier within OpenAI", "overriding the provider's default
model", or "does not switch to another provider". We deliberately do NOT
assert the exact cheap-tier model id (currently openai/gpt-5-mini): the tier
table may change (verified against the pi catalog when shipped), so pinning
the id here would break on a legitimate catalog update. The reference answer
may name the current id, but the grader checks the mechanism only.
Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "cheap" not in out:
        print("answer does not mention the cheap tier", file=sys.stderr)
        return 1
    # The prompt asks what --model-tier cheap does and which model it selects:
    # the core mechanism is WITHIN-PROVIDER tier resolution (no fallback to
    # another provider / no dynamic cheapest-model search). This is graded on
    # substance, not exact phrases: both a cheap and a strong model answered
    # the prompt correctly without the literal words "no fallback"/"usage
    # error", so requiring those exact phrases over-specified the rubric.
    within_provider = any(
        phrase in out
        for phrase in (
            "no fallback",
            "without fallback",
            "never falls back",
            "does not fall back",
            "doesn't fall back",
            "does not switch",
            "doesn't switch",
            "same provider",
            "within the",
            "within openai",
            "within a provider",
            "stays on",
            "remains on",
            "overrid",  # "overriding the provider's default model"
        )
    )
    if not within_provider:
        print(
            "answer must state that tier selection stays within the provider "
            "(no cross-provider fallback)",
            file=sys.stderr,
        )
        return 1
    print("answer explains the within-provider tier mechanism with no fallback")
    return 0


if __name__ == "__main__":
    sys.exit(main())
