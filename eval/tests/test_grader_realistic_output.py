"""Hermetic grader calibration tests: real LLM output shapes must pass.

Regression guard for the live-eval calibration investigation (nightly run
31652076075 graded 0/51 agent runs as FAIL). Root cause of the 0/51 was the
workflow installing the wrong `pi` npm package (an unrelated legacy CLI that
prints "3" for every invocation) — the agent output was literally "3\n", and
the graders correctly failed it. With REAL cheap-tier output the graders pass
10/12 tasks locally; the two genuine calibration bugs found are pinned here:

- coding-003 / coding-011 / coding-012 (shared run_bash): prose-heavy answers
  with bullet lists defeated the first-line/last-line fallback ("bash: - :
  invalid option"). The fallback now extracts command-like lines and skips
  prose.
- coding-018: the rubric demanded phrases ("no fallback", "usage error")
  beyond what the prompt asks; both a cheap and a strong model gave
  substantively correct answers that failed. The rubric now grades the
  within-provider mechanism substance.

These tests are hermetic: they run the graders against realistic LLM-style
candidate texts (prose + fenced code, prose-heavy bullets, terse answers) and
require no API key or network. Stdlib-only.
"""

import sys
from pathlib import Path

import pytest

EVAL_DIR = Path(__file__).resolve().parents[1]
if str(EVAL_DIR) not in sys.path:
    sys.path.insert(0, str(EVAL_DIR))

from grader import run_grader  # noqa: E402


def _task(task_id: str) -> dict:
    """Synthetic dataset row carrying the graderRef the harness resolves."""
    return {"id": task_id, "graderRef": f"graders/{task_id}/grade.py"}


def _grade(task_id: str, candidate: str) -> tuple[bool, str]:
    return run_grader(_task(task_id), candidate)


# -- coding-003: prose-heavy shell answer (the shape that broke the old
# first-line/last-line fallback) -------------------------------------------

PROSE_HEAVY_FIND = """\
Here are a couple simple options.

- Recursive (current directory and subdirectories), precise 24 hours (GNU find):
  find . -type f -mmin -1440 -print

- Recursive, portable (uses 24-hour day rounding):
  find . -type f -mtime -1 -print

Notes:
- Replace -type f with nothing (or with -type d) to include directories.
- Some BSD/macOS finds lack -mmin and/or -maxdepth; use -mtime -1 (less precise) on those systems.
"""


def test_coding003_prose_heavy_answer_passes():
    passed, detail = _grade("coding-003", PROSE_HEAVY_FIND)
    assert passed, f"prose-heavy find answer must pass: {detail}"


def test_coding003_fenced_answer_passes():
    candidate = "Use this:\n\n```bash\nfind . -type f -mtime -1\n```\n\nDone."
    passed, detail = _grade("coding-003", candidate)
    assert passed, f"fenced find answer must pass: {detail}"


def test_coding003_wrong_command_fails():
    passed, _ = _grade("coding-003", "find . -type f -mtime +30")
    assert not passed, "a command listing only the ancient file must fail"


# -- coding-011: fenced + bare-line answers ---------------------------------

def test_coding011_fenced_answer_passes():
    candidate = (
        "Use this (handles filenames with spaces):\n\n"
        "```bash\nfind . -maxdepth 1 -type f -name '*.txt' -print0 "
        "| xargs -0 cat -- | wc -l\n```\n"
    )
    passed, detail = _grade("coding-011", candidate)
    assert passed, f"fenced wc answer must pass: {detail}"


def test_coding011_bare_command_line_passes():
    passed, detail = _grade("coding-011", "wc -l *.txt\n")
    assert passed, f"bare wc one-liner must pass: {detail}"


def test_coding011_wrong_command_fails():
    passed, _ = _grade("coding-011", "wc -l *.log")
    assert not passed, "counting non-.txt files must fail"


# -- coding-012: dual GNU/BSD answer (empty-output pipeline regression) -----

DUAL_STAT_ANSWER = """\
Here are two portable one-liners (GNU/Linux and macOS/BSD).

GNU (Linux with GNU stat):
find . -maxdepth 1 -type f -print0 | xargs -0 stat --printf='%s %n\\n' | sort -nr | head -n1 | awk '{$1=""; sub(/^ /,""); print}'

macOS / BSD (stat -f):
find . -maxdepth 1 -type f -print0 | xargs -0 stat -f '%z %N' | sort -nr | head -n1 | awk '{$1=""; sub(/^ /,""); print}'

Notes:
- These handle filenames with spaces but not filenames containing newlines.
- For a very short (less robust) variant: ls -1S | head -n1 (includes directories).
"""


def test_coding012_dual_variant_answer_passes():
    passed, detail = _grade("coding-012", DUAL_STAT_ANSWER)
    assert passed, f"dual GNU/BSD answer must pass: {detail}"


def test_coding012_wrong_command_fails():
    passed, _ = _grade("coding-012", "ls small.txt")
    assert not passed, "an answer naming the wrong file must fail"


# -- coding-018: within-provider mechanism rubric ---------------------------

CHEAP_STYLE_ANSWER = """\
Short answer: it selects the model "openai/gpt-5-mini".

Why: in providers.json the OpenAI provider maps model tiers to concrete model
IDs. So running pi-run print --model-tier cheap with the provider set to
openai tells pi-run to use the "cheap" tier, which resolves to
openai/gpt-5-mini (overriding the provider's defaultModel). If you omit
--model-tier, pi-run will use the provider's defaultModel.
"""

STRONG_STYLE_ANSWER = """\
With OpenAI selected (explicitly via --provider openai, or implicitly as the
default provider), --model-tier cheap resolves the tier within OpenAI and
launches Pi with:

--model openai/gpt-5-mini

It does not switch to another provider or dynamically calculate the globally
cheapest model. The mapping is configured in providers.json.
"""


def test_coding018_cheap_style_answer_passes():
    passed, detail = _grade("coding-018", CHEAP_STYLE_ANSWER)
    assert passed, f"cheap-style mechanism answer must pass: {detail}"


def test_coding018_strong_style_answer_passes():
    passed, detail = _grade("coding-018", STRONG_STYLE_ANSWER)
    assert passed, f"strong-style mechanism answer must pass: {detail}"


def test_coding018_cross_provider_fallback_answer_fails():
    passed, _ = _grade(
        "coding-018",
        "With the cheap tier pi-run picks the cheapest model available across "
        "all providers, computed dynamically per request.",
    )
    assert not passed, "a cross-provider dynamic-cheapest answer must fail"


def test_coding018_vague_answer_fails():
    passed, _ = _grade("coding-018", "The cheap tier is a cost-saving option.")
    assert not passed, "a vague answer must fail"


# -- nightly symptom control: garbage agent output must never pass ----------

GARBAGE_OUTPUT = "3\n"  # exactly what the impostor `pi` npm CLI printed


@pytest.mark.parametrize(
    "task_id",
    ["coding-003", "coding-011", "coding-012", "coding-018"],
)
def test_garbage_output_fails_all_graders(task_id):
    passed, _ = _grade(task_id, GARBAGE_OUTPUT)
    assert not passed, f"{task_id}: garbage output {GARBAGE_OUTPUT!r} must fail"
