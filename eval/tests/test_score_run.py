"""Hermetic unit tests for eval/scripts/score_run.py (spec §4.4, §6).

Fixtures are inline dicts written to tmp files — no keys, no network, no
hardcoded user paths. Tests cover aggregation math (mean pass rate, median
cost/latency/tokens), the tolerance boundary (baseline - 0.05 exact edge),
cost >2x baseline regression, the costPerSuccessfulTaskUsd div-by-zero guard,
incomplete-run detection, --update-baseline --allow output shape, the
GITHUB_STEP_SUMMARY markdown table, and exit codes 0/1/2.
"""

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[2]
SCORE_RUN_PY = REPO_ROOT / "eval" / "scripts" / "score_run.py"


def _load_score_run():
    spec = importlib.util.spec_from_file_location("score_run_module", SCORE_RUN_PY)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


score_run = _load_score_run()


# ---------------------------------------------------------------------------
# Fixture builders
# ---------------------------------------------------------------------------


def std_run(passed, cost=0.01, judge=0.0, tokens=1000, latency=100.0, outcome=None):
    """One live-suite run: recorded properties + pytest outcome."""
    if outcome is None:
        outcome = "passed" if passed else "failed"
    return {
        "pass": passed,
        "costUsd": cost,
        "judgeCostUsd": judge,
        "tokens": tokens,
        "latencyMs": latency,
        "outcome": outcome,
    }


def make_report(cases, exitcode=0, nodeid_fn=None):
    """Build a pytest-json-report dict matching the REAL plugin output.

    cases: dict case_id -> list of std_run() dicts.
    nodeid_fn: optional callable(case_id, index) -> nodeid override.

    pytest-json-report 1.5.0 serializes one TEST ENTRY PER CASE (the live
    suite is parametrized per case and runs all EVAL_RUNS_PER_CASE repeats
    inside one test), with record_property values under test["user_properties"]
    as a FLAT, ORDERED list of single-key dicts:
    [{pass}, {costUsd}, {judgeCostUsd}, {tokens}, {latencyMs}, {pass}, ...].
    """
    tests = []
    for case_id, runs in cases.items():
        user_properties = []
        for run in runs:
            for key, value in run.items():
                if key != "outcome" and value is not None:
                    user_properties.append({key: value})
        # The plugin records the LAST pytest outcome for the whole test entry;
        # with all repeats inside one entry, use the last run's outcome (the
        # per-run pass flags come from user_properties, not the entry outcome).
        outcome = runs[-1]["outcome"] if runs else "passed"
        if nodeid_fn:
            nodeid = nodeid_fn(case_id, 0)
        else:
            nodeid = f"tests/test_live_suite.py::test_case[{case_id}]"
        tests.append(
            {
                "nodeid": nodeid,
                "outcome": outcome,
                "user_properties": user_properties,
                "duration": 0.1,
            }
        )
    return {"created": 0, "duration": 1.0, "exitcode": exitcode, "tests": tests}


def make_baseline(cases):
    return {
        "schemaVersion": 1,
        "generated": "2026-08-12T00:00:00Z",
        "runsPerCase": 3,
        "cases": cases,
    }


def write_json(tmp_path, name, payload) -> Path:
    path = tmp_path / name
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


def run_cli(tmp_path, report, baseline=None, *extra_args, env=None):
    """Run score_run.py in a subprocess against fixture files in tmp_path."""
    report_path = write_json(tmp_path, "report.json", report)
    args = [sys.executable, str(SCORE_RUN_PY), "--report", str(report_path)]
    if baseline is not None:
        baseline_path = write_json(tmp_path, "baseline.json", baseline)
        args += ["--baseline", str(baseline_path)]
    args += ["--out", str(tmp_path / "summary.json")]
    args += list(extra_args)
    run_env = dict(os.environ)
    run_env.pop("GITHUB_STEP_SUMMARY", None)
    if env:
        run_env.update(env)
    return subprocess.run(args, capture_output=True, text=True, timeout=120, env=run_env)


# ---------------------------------------------------------------------------
# Parsing
# ---------------------------------------------------------------------------


def test_extract_case_id_from_nodeid():
    assert score_run.extract_case_id("tests/test_live_suite.py::test_case[coding-001]") == "coding-001"
    assert (
        score_run.extract_case_id("tests/test_live_suite.py::TestLiveSuite::test_case[coding-020]")
        == "coding-020"
    )


def test_extract_case_id_ignores_other_files():
    # Metric parametrization and the legacy E2E test must never count as cases.
    assert (
        score_run.extract_case_id("tests/test_live_metrics.py::test_metrics[coding-001-AnswerRelevancyMetric]")
        is None
    )
    assert (
        score_run.extract_case_id("tests/test_agent_task_completion.py::test_agent_produces_expected_factorial")
        is None
    )
    assert score_run.extract_case_id("") is None


def test_collect_cases_only_consumes_live_suite_nodeids():
    report = make_report(
        {
            # metrics nodeid is NOT a live-suite case -> ignored entirely
            "coding-001-AnswerRelevancyMetric": [std_run(True)],
            # live-suite nodeid -> kept
            "coding-002": [std_run(True), std_run(True)],
        },
        nodeid_fn=lambda case_id, index: (
            "tests/test_live_metrics.py::test_metrics[coding-001-AnswerRelevancyMetric]"
            if case_id.startswith("coding-001-")
            else f"tests/test_live_suite.py::test_case[{case_id}]"
        ),
    )
    cases = score_run.collect_cases(report)
    assert list(cases) == ["coding-002"]
    assert len(cases["coding-002"]) == 2  # both repeats zipped from user_properties


# ---------------------------------------------------------------------------
# Aggregation math (spec §4.3 gate math)
# ---------------------------------------------------------------------------


def test_aggregate_mean_pass_rate_and_median_cost():
    runs = [
        std_run(True, cost=0.01, judge=0.001, tokens=1000, latency=100.0),
        std_run(False, cost=0.03, judge=0.001, tokens=3000, latency=300.0),
        std_run(False, cost=0.02, judge=0.001, tokens=2000, latency=200.0),
    ]
    agg = score_run.aggregate_case(runs, expected_runs=3)
    assert agg["passRate"] == round(1 / 3, 6)  # mean over runs
    assert agg["costPerTaskUsd"] == 0.021  # median of (agent + judge) per run
    assert agg["agentCostUsd"] == 0.02  # median of agent-only cost
    assert agg["judgeCostUsd"] == 0.001  # median of judge-only cost
    assert agg["tokensPerTask"] == 2000  # median
    assert agg["latencyMs"] == 200.0  # median
    assert agg["nRuns"] == 3
    assert agg["incomplete"] is False


def test_totals_math():
    cases_raw = {
        "coding-001": [
            std_run(True, cost=0.01, tokens=1000, latency=100.0),
            std_run(True, cost=0.02, tokens=2000, latency=200.0),
            std_run(True, cost=0.03, tokens=3000, latency=300.0),
        ],
        "coding-002": [
            std_run(True, cost=0.005, tokens=500, latency=50.0),
            std_run(False, cost=0.005, tokens=500, latency=50.0),
            std_run(False, cost=0.005, tokens=500, latency=50.0),
            std_run(False, cost=0.005, tokens=500, latency=50.0),
        ],
    }
    case_aggs = {cid: score_run.aggregate_case(runs, 3) for cid, runs in cases_raw.items()}
    totals = score_run.compute_totals(cases_raw, case_aggs)
    assert totals["nCases"] == 2
    assert totals["nPassed"] == 1  # only coding-001 has passRate == 1.0
    assert totals["nFailed"] == 1
    assert totals["overallPassRate"] == round((1.0 + 0.25) / 2, 6)  # mean of per-case rates
    assert totals["totalCostUsd"] == round(0.06 + 4 * 0.005, 9)  # sum of agent + judge
    assert totals["costPerTaskUsd"] == round(0.08 / 2, 9)  # ALL-IN (agent + judge)
    assert totals["agentCostPerTaskUsd"] == round(0.08 / 2, 9)
    assert totals["costPerSuccessfulTaskUsd"] == round(0.08 / 1, 9)
    assert totals["tokensPerTask"] == round((6000 + 2000) / 2)
    assert totals["totalTokens"] == 8000


def test_cost_per_successful_guard_division_by_zero():
    cases_raw = {
        "coding-001": [std_run(False), std_run(False), std_run(False)],
        "coding-002": [std_run(False), std_run(False), std_run(False)],
    }
    case_aggs = {cid: score_run.aggregate_case(runs, 3) for cid, runs in cases_raw.items()}
    totals = score_run.compute_totals(cases_raw, case_aggs)
    assert totals["nPassed"] == 0
    assert totals["costPerSuccessfulTaskUsd"] == 0.0  # guarded, never a division error


# ---------------------------------------------------------------------------
# Baseline comparison
# ---------------------------------------------------------------------------


def _agg(pass_frac, cost, tokens=1000, latency=100.0, n_runs=3):
    passed = round(pass_frac * n_runs)
    runs = [std_run(i < passed, cost=cost, tokens=tokens, latency=latency) for i in range(n_runs)]
    return score_run.aggregate_case(runs, n_runs)


def test_tolerance_boundary_at_edge_passes():
    # baseline passRate 1.0, tolerance 0.05: exactly baseline - 0.05 passes.
    at_edge = _agg(0.95, cost=0.01, n_runs=20)  # 19/20 -> 0.95
    comp = score_run.compare_case("coding-001", at_edge, {"coding-001": {"passRate": 1.0}}, 0.05)
    assert comp["regressed"] is False
    assert comp["unbaselined"] is False


def test_tolerance_boundary_below_fails():
    below = _agg(0.9, cost=0.01, n_runs=20)  # 18/20 -> 0.90 < 0.95
    comp = score_run.compare_case("coding-001", below, {"coding-001": {"passRate": 1.0}}, 0.05)
    assert comp["regressed"] is True


def test_cost_regression_boundary():
    baseline = {"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}}
    over_2x = _agg(1.0, cost=0.0200001)  # 0.0200001 > 2 * 0.01
    assert score_run.compare_case("coding-001", over_2x, baseline, 0.05)["costRegressed"] is True
    exactly_2x = _agg(1.0, cost=0.02)  # 0.02 is NOT > 2 * 0.01
    assert score_run.compare_case("coding-001", exactly_2x, baseline, 0.05)["costRegressed"] is False


def test_unbaselined_cases_recorded_not_failed():
    agg = _agg(0.0, cost=0.01)
    comp = score_run.compare_case("coding-001", agg, {}, 0.05)
    assert comp["unbaselined"] is True
    assert comp["regressed"] is False
    assert comp["costRegressed"] is False


def test_incomplete_run_detection_fewer_runs():
    runs = [std_run(True), std_run(True)]  # only 2 of expected 3
    agg = score_run.aggregate_case(runs, expected_runs=3)
    assert agg["incomplete"] is True
    assert agg["errored"] is False


def test_incomplete_run_detection_errored():
    runs = [std_run(True), std_run(True), std_run(True, outcome="error")]
    agg = score_run.aggregate_case(runs, expected_runs=3)
    assert agg["incomplete"] is True
    assert agg["errored"] is True


# ---------------------------------------------------------------------------
# End-to-end CLI behavior (exit codes 0/1/2)
# ---------------------------------------------------------------------------


def test_exit_code_zero_on_pass(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 0, result.stderr
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["gate"]["passed"] is True
    assert summary["totals"]["nPassed"] == 1


def test_exit_code_one_on_regression(tmp_path):
    report = make_report({"coding-001": [std_run(True), std_run(False), std_run(False)]})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 1
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["gate"]["passed"] is False
    assert any("coding-001" in failure for failure in summary["gate"]["failures"])


def test_exit_code_one_on_cost_regression(tmp_path):
    report = make_report({"coding-001": [std_run(True, cost=0.03)] * 3})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 1
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert any("costPerTaskUsd regression" in failure for failure in summary["gate"]["failures"])


def test_exit_code_one_on_incomplete(tmp_path):
    report = make_report({"coding-001": [std_run(True), std_run(True)]})  # 2 of --runs 3
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 1
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert any("incomplete" in failure for failure in summary["gate"]["failures"])


def test_exit_code_one_on_budget_exceeded(tmp_path):
    report = make_report({"coding-001": [std_run(True, cost=0.03)] * 3})  # total 0.09
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.03}})
    result = run_cli(tmp_path, report, baseline, "--budget-usd", "0.05")
    assert result.returncode == 1
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert any("budget exceeded" in failure for failure in summary["gate"]["failures"])


def test_exit_code_two_usage_errors(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3})
    # --update-baseline without --allow is a usage error.
    result = run_cli(tmp_path, report, make_baseline({}), "--update-baseline")
    assert result.returncode == 2
    # --allow without --update-baseline is a usage error.
    result = run_cli(tmp_path, report, make_baseline({}), "--allow")
    assert result.returncode == 2
    # Unknown flag.
    result = run_cli(tmp_path, report, make_baseline({}), "--nonsense")
    assert result.returncode == 2
    # --runs must be >= 1.
    result = run_cli(tmp_path, report, make_baseline({}), "--runs", "0")
    assert result.returncode == 2


def test_exit_code_two_unreadable_report(tmp_path):
    bad = tmp_path / "bad.json"
    bad.write_text("{not json", encoding="utf-8")
    result = subprocess.run(
        [sys.executable, str(SCORE_RUN_PY), "--report", str(bad), "--out", str(tmp_path / "o.json")],
        capture_output=True,
        text=True,
        timeout=120,
    )
    assert result.returncode == 2


def test_unbaselined_gate_passes_with_empty_baseline(tmp_path):
    report = make_report({"coding-001": [std_run(True), std_run(False), std_run(False)]})
    baseline = make_baseline({})  # empty committed bootstrap: nothing to regress against
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 0
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["unbaselined"] == ["coding-001"]
    assert summary["gate"]["passed"] is True


def test_nonzero_report_exitcode_fails_gate(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3}, exitcode=3)
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 1
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert any("exitcode 3" in failure for failure in summary["gate"]["failures"])


# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------


def test_update_baseline_requires_allow_and_writes_shape(tmp_path):
    report = make_report(
        {
            "coding-001": [std_run(True), std_run(True), std_run(True)],
            "coding-002": [std_run(True), std_run(False)],  # incomplete: 2 of 3 runs
        }
    )
    baseline_path = write_json(tmp_path, "baseline.json", make_baseline({}))
    result = run_cli(
        tmp_path, report, None,  # baseline passed explicitly below
        "--baseline", str(baseline_path), "--update-baseline", "--allow",
    )
    # coding-002 is incomplete (2 of 3 runs), so the gate fails — but the
    # deliberate re-baseline still writes, omitting the incomplete case.
    assert result.returncode == 1, result.stderr
    assert "incomplete" in result.stderr
    new_baseline = json.loads(baseline_path.read_text(encoding="utf-8"))
    assert new_baseline["schemaVersion"] == 1
    assert new_baseline["runsPerCase"] == 3
    assert "generated" in new_baseline
    assert "agentModel" in new_baseline and "judgeModel" in new_baseline
    # Incomplete cases must never be enshrined in the baseline.
    assert list(new_baseline["cases"]) == ["coding-001"]
    case = new_baseline["cases"]["coding-001"]
    assert set(case) == {"passRate", "costPerTaskUsd", "tokensPerTask", "latencyMs"}
    assert case["passRate"] == 1.0


def test_json_summary_is_compact(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    compact_path = tmp_path / "compact.json"
    result = run_cli(tmp_path, report, baseline, "--json-summary", str(compact_path))
    assert result.returncode == 0
    compact = json.loads(compact_path.read_text(encoding="utf-8"))
    assert "totals" in compact and "gate" in compact and "unbaselined" in compact
    assert "cases" not in compact  # compact summary has no per-case detail
    assert "run" in compact


def test_markdown_step_summary(tmp_path):
    report = make_report(
        {
            "coding-001": [std_run(True)] * 3,
            "coding-002": [std_run(True), std_run(False), std_run(False)],
        }
    )
    baseline = make_baseline(
        {
            "coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01},
            "coding-002": {"passRate": 1.0, "costPerTaskUsd": 0.01},
        }
    )
    markdown_path = tmp_path / "summary.md"
    result = run_cli(
        tmp_path, report, baseline, env={"GITHUB_STEP_SUMMARY": str(markdown_path)}
    )
    assert result.returncode == 1  # coding-002 regressed
    assert markdown_path.is_file()
    table = markdown_path.read_text(encoding="utf-8")
    assert "| Case | passRate | baseline |" in table
    assert "coding-001" in table and "coding-002" in table
    assert ":x: REGRESSED" in table  # coding-002 flagged


# ---------------------------------------------------------------------------
# Self-heal event surfacing (W6 — BACKLOG #1)
# ---------------------------------------------------------------------------


def write_events(tmp_path, lines):
    path = tmp_path / "events.jsonl"
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return path


def test_parse_self_heal_events_counts_by_kind(tmp_path):
    path = write_events(
        tmp_path,
        [
            json.dumps({"ts": "2026-08-13T00:00:00Z", "kind": "group-kill", "detail": "wall-clock timeout"}),
            json.dumps({"ts": "2026-08-13T00:00:01Z", "kind": "group-kill", "detail": "output stall"}),
            json.dumps({"ts": "2026-08-13T00:00:02Z", "kind": "recovery", "detail": "rebase continued"}),
        ],
    )
    assert score_run.parse_self_heal_events(path) == {
        "nEvents": 3,
        "byKind": {"group-kill": 2, "recovery": 1},
    }


def test_parse_self_heal_events_missing_file_is_empty(tmp_path):
    assert score_run.parse_self_heal_events(tmp_path / "does-not-exist.jsonl") == {
        "nEvents": 0,
        "byKind": {},
    }


def test_parse_self_heal_events_skips_malformed_lines(tmp_path):
    path = write_events(
        tmp_path,
        [
            "not json",
            json.dumps({"ts": "2026-08-13T00:00:00Z", "kind": "group-kill", "detail": "ok"}),
            "{broken",
            "",
            json.dumps({"ts": "2026-08-13T00:00:01Z", "kind": "recovery", "detail": "ok"}),
        ],
    )
    assert score_run.parse_self_heal_events(path) == {
        "nEvents": 2,
        "byKind": {"group-kill": 1, "recovery": 1},
    }


def test_summary_includes_self_heal_block(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    events = write_events(
        tmp_path,
        [json.dumps({"ts": "2026-08-13T00:00:00Z", "kind": "group-kill", "detail": "stall"})],
    )
    result = run_cli(tmp_path, report, baseline, "--heal-events", str(events))
    assert result.returncode == 0
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["selfHeal"] == {"nEvents": 1, "byKind": {"group-kill": 1}}


def test_compact_summary_includes_self_heal_block(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    events = write_events(
        tmp_path,
        [json.dumps({"ts": "2026-08-13T00:00:00Z", "kind": "recovery", "detail": "rebase"})],
    )
    compact_path = tmp_path / "compact.json"
    result = run_cli(
        tmp_path, report, baseline,
        "--heal-events", str(events), "--json-summary", str(compact_path),
    )
    assert result.returncode == 0
    compact = json.loads(compact_path.read_text(encoding="utf-8"))
    assert compact["selfHeal"] == {"nEvents": 1, "byKind": {"recovery": 1}}


def test_self_heal_events_do_not_affect_gate(tmp_path):
    report = make_report(
        {
            "coding-001": [std_run(True)] * 3,
            "coding-002": [std_run(True), std_run(False), std_run(False)],
        }
    )
    baseline = make_baseline(
        {
            "coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01},
            "coding-002": {"passRate": 1.0, "costPerTaskUsd": 0.01},
        }
    )
    events = write_events(
        tmp_path,
        [
            json.dumps({"ts": "2026-08-13T00:00:00Z", "kind": "group-kill", "detail": "stall"}),
            json.dumps({"ts": "2026-08-13T00:00:01Z", "kind": "group-kill", "detail": "stall"}),
        ],
    )
    with_events = run_cli(tmp_path, report, baseline, "--heal-events", str(events))
    assert with_events.returncode == 1  # coding-002 regression still fails the gate
    with_summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert with_summary["gate"]["passed"] is False
    assert with_summary["selfHeal"] == {"nEvents": 2, "byKind": {"group-kill": 2}}

    without_events = run_cli(tmp_path, report, baseline)
    assert without_events.returncode == 1
    without_summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert without_summary["selfHeal"] == {"nEvents": 0, "byKind": {}}


def test_markdown_step_summary_includes_self_heal_line(tmp_path):
    report = make_report({"coding-001": [std_run(True)] * 3})
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    events = write_events(
        tmp_path,
        [json.dumps({"ts": "2026-08-13T00:00:00Z", "kind": "group-kill", "detail": "stall"})],
    )
    markdown_path = tmp_path / "summary.md"
    result = run_cli(
        tmp_path, report, baseline,
        "--heal-events", str(events), env={"GITHUB_STEP_SUMMARY": str(markdown_path)},
    )
    assert result.returncode == 0
    table = markdown_path.read_text(encoding="utf-8")
    assert "self-heal events" in table
    assert "group-kill" in table


# ---------------------------------------------------------------------------
# Flake-aware gate (W7 — EVAL-2)
# ---------------------------------------------------------------------------


def test_aggregate_case_reports_failed_run_count():
    runs = [std_run(True), std_run(False), std_run(True), std_run(True), std_run(False)]
    agg = score_run.aggregate_case(runs, expected_runs=5)
    assert agg["nFailed"] == 2
    assert agg["nRuns"] == 5
    assert agg["passRate"] == round(3 / 5, 6)


def test_single_failed_run_is_flake_not_regression():
    # 1 of 5 failed: passRate 0.8 vs baseline 1.0 -> below tolerance, but a
    # single failure is a flake, not a regression.
    runs = [std_run(True), std_run(False), std_run(True), std_run(True), std_run(True)]
    agg = score_run.aggregate_case(runs, expected_runs=5)
    comp = score_run.compare_case("coding-001", agg, {"coding-001": {"passRate": 1.0}}, 0.05)
    assert comp["flake"] is True
    assert comp["regressed"] is False


def test_two_failed_runs_is_regression():
    runs = [std_run(True), std_run(False), std_run(True), std_run(False), std_run(True)]
    agg = score_run.aggregate_case(runs, expected_runs=5)
    comp = score_run.compare_case("coding-001", agg, {"coding-001": {"passRate": 1.0}}, 0.05)
    assert comp["flake"] is False
    assert comp["regressed"] is True


def test_flake_does_not_fail_gate(tmp_path):
    report = make_report(
        {"coding-001": [std_run(True), std_run(False), std_run(True), std_run(True), std_run(True)]},
        nodeid_fn=lambda case_id, index: f"tests/test_live_suite.py::test_case[{case_id}]",
    )
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 0, result.stderr
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["gate"]["passed"] is True
    assert summary["flakes"] == ["coding-001"]


def test_two_failed_runs_still_fails_gate(tmp_path):
    report = make_report(
        {"coding-001": [std_run(True), std_run(False), std_run(True), std_run(False), std_run(True)]},
        nodeid_fn=lambda case_id, index: f"tests/test_live_suite.py::test_case[{case_id}]",
    )
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert result.returncode == 1
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["gate"]["passed"] is False
    assert summary["flakes"] == []


def test_flake_listed_in_compact_summary(tmp_path):
    report = make_report(
        {"coding-001": [std_run(True), std_run(False), std_run(True), std_run(True), std_run(True)]},
        nodeid_fn=lambda case_id, index: f"tests/test_live_suite.py::test_case[{case_id}]",
    )
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    compact_path = tmp_path / "compact.json"
    result = run_cli(tmp_path, report, baseline, "--json-summary", str(compact_path))
    assert result.returncode == 0
    compact = json.loads(compact_path.read_text(encoding="utf-8"))
    assert compact["flakes"] == ["coding-001"]


def test_markdown_marks_flake_not_regressed(tmp_path):
    report = make_report(
        {"coding-001": [std_run(True), std_run(False), std_run(True), std_run(True), std_run(True)]},
        nodeid_fn=lambda case_id, index: f"tests/test_live_suite.py::test_case[{case_id}]",
    )
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    markdown_path = tmp_path / "summary.md"
    result = run_cli(
        tmp_path, report, baseline, env={"GITHUB_STEP_SUMMARY": str(markdown_path)}
    )
    assert result.returncode == 0
    table = markdown_path.read_text(encoding="utf-8")
    assert "flake" in table
    assert ":x: REGRESSED" not in table


def test_flake_note_on_stderr(tmp_path):
    report = make_report(
        {"coding-001": [std_run(True), std_run(False), std_run(True), std_run(True), std_run(True)]},
        nodeid_fn=lambda case_id, index: f"tests/test_live_suite.py::test_case[{case_id}]",
    )
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, report, baseline)
    assert "flake" in result.stderr


# ---------------------------------------------------------------------------
# Scorecard provenance (W8 — EVAL-3)
# ---------------------------------------------------------------------------


def _tasks_dataset_version():
    tasks_path = REPO_ROOT / "eval" / "datasets" / "tasks.json"
    return json.loads(tasks_path.read_text(encoding="utf-8"))["datasetVersion"]


def _one_case_report():
    return make_report(
        {"coding-001": [std_run(True)] * 5},
        nodeid_fn=lambda case_id, index: f"tests/test_live_suite.py::test_case[{case_id}]",
    )


def test_summary_records_dataset_version(tmp_path):
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(tmp_path, _one_case_report(), baseline)
    assert result.returncode == 0
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["provenance"]["datasetVersion"] == _tasks_dataset_version()


def test_summary_records_env_provenance(tmp_path):
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(
        tmp_path, _one_case_report(), baseline,
        env={"PI_MODEL_TIER": "cheap", "OPENAI_MODEL_NAME": "gpt-4.1-mini", "PI_VERSION": "v0.9.2"},
    )
    assert result.returncode == 0
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["provenance"]["agentModel"] == "cheap"
    assert summary["provenance"]["judgeModel"] == "gpt-4.1-mini"
    assert summary["provenance"]["piVersion"] == "v0.9.2"


def test_provenance_defaults_to_unknown_when_unset(tmp_path):
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    result = run_cli(
        tmp_path, _one_case_report(), baseline,
        env={"PI_MODEL_TIER": "", "OPENAI_MODEL_NAME": "", "PI_VERSION": ""},
    )
    assert result.returncode == 0
    summary = json.loads((tmp_path / "summary.json").read_text(encoding="utf-8"))
    assert summary["provenance"]["agentModel"] == "unknown"
    assert summary["provenance"]["judgeModel"] == "unknown"
    assert summary["provenance"]["piVersion"] == "unknown"


def test_compact_summary_records_provenance(tmp_path):
    baseline = make_baseline({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})
    compact_path = tmp_path / "compact.json"
    result = run_cli(
        tmp_path, _one_case_report(), baseline, "--json-summary", str(compact_path),
        env={"PI_MODEL_TIER": "cheap", "PI_VERSION": "v0.9.2"},
    )
    assert result.returncode == 0
    compact = json.loads(compact_path.read_text(encoding="utf-8"))
    assert compact["provenance"]["datasetVersion"] == _tasks_dataset_version()
    assert compact["provenance"]["agentModel"] == "cheap"
