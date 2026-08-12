"""Live agent-run suite (nightly-only): pi-run print against the live dataset.

Every deterministic task in the dataset is run via ``pi-run print
--model-tier cheap`` EVAL_RUNS_PER_CASE times (default 1) and graded with the
shared grading harness (eval/grader.py, dataset lane). Per-run stats are
attached via record_property — pass/costUsd/judgeCostUsd/tokens/latencyMs —
so eval/scripts/score_run.py can rebuild per-case aggregates from the
pytest-json-report (spec §4.3–§4.4).

Agent spend is attributed from the cost ledger (.pi/cost-ledger.jsonl) as the
delta around each invocation — real ledger data, never an estimate (spec
§4.5). The persistence coupling is required: pi-run print only records real
usage.cost.total when a budget cap is set (PI_MAX_BUDGET_USD), which the
nightly workflow sets. Without it the ledger delta is ~0 and cost reports as
0.0.

Nightly-only: these tests exercise real provider spend. The workflow's key
presence gate is the hard fail; the skipif here only keeps a no-key local
checkout collectable.
"""

import json
import os
import time
from pathlib import Path

import pytest

from conftest import has_api_key, load_dataset, run_pi_print

REPO_ROOT = Path(__file__).resolve().parents[2]
DATASET_DIR = REPO_ROOT / "eval" / "datasets"
LEDGER_PATH = REPO_ROOT / ".pi" / "cost-ledger.jsonl"

# Number of agent runs per case (the nightly sets EVAL_RUNS_PER_CASE=3).
RUNS_PER_CASE = int(os.environ.get("EVAL_RUNS_PER_CASE", "1"))


def _ledger_entries() -> list[dict]:
    """Read the spend ledger (.pi/cost-ledger.jsonl); best-effort on bad lines."""
    if not LEDGER_PATH.exists():
        return []
    entries = []
    try:
        with LEDGER_PATH.open("r", encoding="utf-8") as handle:
            for line in handle:
                line = line.strip()
                if not line:
                    continue
                try:
                    entries.append(json.loads(line))
                except json.JSONDecodeError:
                    continue  # partially-written line; best effort
    except OSError:
        return []
    return entries


def _grade(sample: dict, output: str) -> tuple[bool, str]:
    """Grade a deterministic case via the shared harness eval/grader.py.

    The harness invokes the task's grade.py (stdin = candidate text, exit 0 =
    pass). It is a dataset-lane artifact; if it is missing the case is skipped
    so the report marks the run incomplete rather than silently ungraded.
    """
    try:
        from grader import run_grader
    except ImportError as exc:
        pytest.skip(f"eval/grader.py is not importable ({exc}) — dataset lane not merged?")

    grader_ref = sample.get("graderRef")
    if not grader_ref:
        pytest.fail(f"deterministic case {sample.get('id')!r} has no graderRef")
    task_dir = str((DATASET_DIR / grader_ref).parent)
    return run_grader(task_dir, output)


def _run_agent_once(sample: dict) -> dict:
    """Run `pi-run print --model-tier cheap` once and attribute real spend.

    Returns one per-run stats dict (pass/costUsd/judgeCostUsd/tokens/
    latencyMs) for record_property. judgeCostUsd is 0.0 here: deterministic
    grading has no LLM judge — judge spend is collected in test_live_metrics.py.
    """
    before = len(_ledger_entries())
    start = time.monotonic()
    output = run_pi_print(sample["input"], extra_args=["--model-tier", "cheap"])
    latency_ms = (time.monotonic() - start) * 1000.0

    added = _ledger_entries()[before:]
    cost_usd = sum(float(entry.get("costUsd") or 0.0) for entry in added)
    tokens = sum(
        int(entry.get("inputTokens") or 0) + int(entry.get("outputTokens") or 0)
        for entry in added
    )
    passed, _detail = _grade(sample, output)
    return {
        "pass": passed,
        "costUsd": cost_usd,
        "judgeCostUsd": 0.0,
        "tokens": tokens,
        "latencyMs": round(latency_ms, 1),
    }


_DETERMINISTIC_CASES = [
    sample for sample in load_dataset() if sample.get("grader") == "deterministic"
]


if _DETERMINISTIC_CASES:

    @pytest.mark.skipif(not has_api_key(), reason="No provider API key found.")
    @pytest.mark.timeout(120)
    @pytest.mark.parametrize(
        "sample",
        _DETERMINISTIC_CASES,
        ids=[case["id"] for case in _DETERMINISTIC_CASES],
    )
    def test_case(sample, record_property):
        """Run the agent on the case EVAL_RUNS_PER_CASE times; record per-run stats.

        Each repeat emits one set of properties (pass/costUsd/judgeCostUsd/
        tokens/latencyMs); score_run.py pairs repeated property names in order
        to rebuild per-run values for the mean pass rate / median cost gate.
        """
        for _ in range(RUNS_PER_CASE):
            stats = _run_agent_once(sample)
            record_property("pass", stats["pass"])
            record_property("costUsd", stats["costUsd"])
            record_property("judgeCostUsd", stats["judgeCostUsd"])
            record_property("tokens", stats["tokens"])
            record_property("latencyMs", stats["latencyMs"])


else:
    # Dataset v2 (dataset lane) not present yet: keep collection clean instead
    # of failing on an empty parametrize.
    @pytest.mark.skipif(not has_api_key(), reason="No provider API key found.")
    @pytest.mark.timeout(120)
    def test_case(record_property):
        pytest.skip("live dataset has no grader==deterministic cases")
