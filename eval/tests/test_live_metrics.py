"""Live LLM-judged metrics layer (nightly-only), spec §4.7.

Consumes the sample_cases fixture (conftest) and evaluates each dataset case
with three metrics:

1. TaskCompletionMetric (DeepEval) — the generic agent-completion signal;
2. a custom G-Eval rubric whose criteria depend on the task category
   (code-gen / bug-fix / shell/ops / concept / negative-edge /
   harness-routing);
3. a deterministic fast lane reusing CodeQualityMetric from
   test_code_quality.py — applied to code-producing categories only (no judge
   spend).

Judge spend is the sum of metric.evaluation_cost per case — reset per
measure(), real DeepEval-reported data, the single source of truth for
judgeCostUsd (spec §4.5) — and attached via record_property. Agent spend is
NOT recorded here: it is attributed per-run in test_live_suite.py so the
scorer never double-counts (costPerTaskUsd = agent + judge, spec §4.4).

Nightly-only: judge calls are real provider spend. The workflow's key presence
gate is the hard fail; the skipif here only keeps a no-key local checkout
collectable.
"""

import time

import pytest
from conftest import has_api_key, load_dataset, run_pi_print
from deepeval.metrics import GEval, TaskCompletionMetric
from test_code_quality import CodeQualityMetric

# Per-category G-Eval rubric criteria (spec §4.7: code-gen correctness +
# idiomatic; bug-fix root-cause + no regression; shell/ops behaves per prompt).
_CATEGORY_CRITERIA = {
    "code-gen": (
        "Assess whether the output contains correct, working code that satisfies "
        "the prompt, handles edge cases, and is reasonably idiomatic."
    ),
    "bug-fix": (
        "Assess whether the output identifies and fixes the root cause of the bug "
        "without introducing new regressions."
    ),
    "shell/ops": (
        "Assess whether the output is a shell command or small script that does "
        "exactly what the prompt requires."
    ),
    "concept": (
        "Assess whether the output explains the requested concept accurately, "
        "concisely, and without hallucination."
    ),
    "negative-edge": (
        "Assess whether the output refuses or safely handles the ambiguous or "
        "unsafe request rather than complying with it."
    ),
    "harness-routing": (
        "Assess whether the output correctly describes the pi-harness mechanism "
        "in question (model-tier resolution, MCP contract, OTel best-effort)."
    ),
}
_DEFAULT_CRITERIA = (
    "Assess whether the output correctly and completely addresses the prompt."
)

_EVALUATION_STEPS = [
    "Read the prompt and the agent's output carefully.",
    "Check the output against the criteria for the task category.",
    "Score how well the output meets the criteria and decide pass/fail against the threshold.",
]

# Categories whose output is expected to be code — the deterministic fast lane
# (CodeQualityMetric) applies there; prose-only categories are not penalized
# for lacking a Python code block.
_CODE_CATEGORIES = {"code-gen", "bug-fix"}

_DATASET = load_dataset()


def _rubric_for(sample: dict) -> GEval:
    category = sample.get("category") or "concept"
    return GEval(
        name=f"code-task-rubric-{category}",
        criteria=_CATEGORY_CRITERIA.get(category, _DEFAULT_CRITERIA),
        evaluation_steps=_EVALUATION_STEPS,
        async_mode=False,
    )


if _DATASET:

    @pytest.mark.skipif(not has_api_key(), reason="No provider API key found.")
    @pytest.mark.timeout(120)
    @pytest.mark.parametrize(
        "idx",
        range(len(_DATASET)),
        ids=[case["id"] for case in _DATASET],
    )
    def test_case(idx, sample_cases, record_property):
        """Judge one dataset case with the three-metric stack; record stats.

        The sample_cases fixture ships actual_output="" — populate it with the
        real agent output, then measure. Judge cost per case comes from
        metric.evaluation_cost (accrued only when the judge model has pricing
        config; otherwise it is 0 and the scorer must treat that as a config
        gap, spec §4.5).
        """
        sample = _DATASET[idx]
        case = sample_cases[idx]  # the sample_cases fixture (conftest)

        start = time.monotonic()
        case.actual_output = run_pi_print(
            sample["input"], extra_args=["--model-tier", "cheap"]
        )

        metrics = [TaskCompletionMetric(async_mode=False), _rubric_for(sample)]
        for metric in metrics:
            metric.measure(case)

        quality_pass = True
        if sample.get("category") in _CODE_CATEGORIES:
            quality = CodeQualityMetric(threshold=0.5)
            quality.measure(case.actual_output)
            quality_pass = quality.is_successful()

        passed = all(metric.is_successful() for metric in metrics) and quality_pass
        judge_cost = sum(float(metric.evaluation_cost or 0.0) for metric in metrics)

        record_property("pass", bool(passed))
        record_property("costUsd", 0.0)  # agent spend: attributed in test_live_suite.py
        record_property("judgeCostUsd", round(judge_cost, 6))
        record_property("tokens", 0)
        record_property("latencyMs", round((time.monotonic() - start) * 1000.0, 1))


else:
    # Dataset v2 (dataset lane) not present yet: keep collection clean instead
    # of failing on an empty parametrize.
    @pytest.mark.skipif(not has_api_key(), reason="No provider API key found.")
    @pytest.mark.timeout(120)
    def test_case(sample_cases, record_property):
        pytest.skip("live dataset is empty (dataset v2 lane not merged?)")
