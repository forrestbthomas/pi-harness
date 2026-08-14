"""EVAL-16 pilot — score_delta.py hermetic unit tests (no keys, no network)."""

import json
import sys

import pytest

sys.path.insert(0, str(__import__("pathlib").Path(__file__).resolve().parents[1] / "scripts"))

import score_delta  # noqa: E402


# ---------------------------------------------------------------------------
# Classification
# ---------------------------------------------------------------------------


def test_classify_dataset_path():
    assert score_delta.classify_path("eval/datasets/tasks.json") == "dataset"
    assert score_delta.classify_path("eval/datasets/coding_samples.jsonl") == "dataset"


def test_classify_grader_and_reference():
    assert score_delta.classify_path("eval/grader.py") == "grader"
    assert score_delta.classify_path("eval/datasets/graders/coding-054/grade.py") == "grader"
    assert score_delta.classify_path("eval/datasets/references/coding-054/ref.py") == "grader"


def test_classify_gate_baseline_benchmark_workflow():
    assert score_delta.classify_path("eval/scripts/score_run.py") == "gate"
    assert score_delta.classify_path("eval/scripts/score_delta.py") == "gate"
    assert score_delta.classify_path("eval/baselines/live-baseline.json") == "baseline"
    assert score_delta.classify_path("eval/benchmarks/rate-limiter/task.json") == "benchmark"
    assert score_delta.classify_path("internal/cli/scorecard.go") == "harness-scorecard"
    assert score_delta.classify_path(".github/workflows/nightly-live-eval.yml") == "workflow"
    assert score_delta.classify_path("eval/tests/test_live_suite.py") == "eval-tests"


def test_classify_other_and_flags():
    report = score_delta.classify_paths(["README.md", "internal/cli/app.go", "docs/benchmark-seam.md"])
    assert report["evalTouching"] is False
    assert report["needsNightlyVerification"] is False
    assert report["evalSurfaces"] == []
    assert report["touched"]["other"] == ["README.md", "docs/benchmark-seam.md", "internal/cli/app.go"]


def test_classify_mixed_flags_nightly_verification():
    report = score_delta.classify_paths(
        ["eval/scripts/score_run.py", "README.md", "eval/datasets/tasks.json"]
    )
    assert report["evalTouching"] is True
    assert report["needsNightlyVerification"] is True
    assert report["evalSurfaces"] == ["dataset", "gate"]


# ---------------------------------------------------------------------------
# Scorecard delta
# ---------------------------------------------------------------------------


def _scorecard(cases: dict) -> dict:
    return {"schemaVersion": "1.0", "generated": "t", "cases": cases}


def test_diff_reports_pass_rate_delta():
    candidate = {"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}}
    baseline = {"coding-001": {"passRate": 0.8, "costPerTaskUsd": 0.01}}
    deltas = score_delta.compare_cases(candidate, baseline, tolerance=0.05)
    assert deltas["coding-001"]["deltaPassRate"] == 0.2
    assert deltas["coding-001"]["regressed"] is False
    assert deltas["coding-001"]["flake"] is False


def test_diff_single_failed_run_is_flake_not_regression():
    # EVAL-2: one failed run below tolerance is a flake, never a regression.
    candidate = {"coding-001": {"passRate": 0.6, "nFailed": 1, "costPerTaskUsd": 0.01}}
    baseline = {"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}}
    deltas = score_delta.compare_cases(candidate, baseline, tolerance=0.05)
    assert deltas["coding-001"]["flake"] is True
    assert deltas["coding-001"]["regressed"] is False


def test_diff_two_failed_runs_is_regression():
    candidate = {"coding-001": {"passRate": 0.6, "nFailed": 2, "costPerTaskUsd": 0.01}}
    baseline = {"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}}
    deltas = score_delta.compare_cases(candidate, baseline, tolerance=0.05)
    assert deltas["coding-001"]["regressed"] is True


def test_diff_cost_regression_over_2x():
    # EVAL-13: median > 2x baseline cost is a cost regression.
    candidate = {"coding-010": {"passRate": 1.0, "costPerTaskUsd": 0.05, "costsPerRun": [0.05]}}
    baseline = {"coding-010": {"passRate": 1.0, "costPerTaskUsd": 0.02}}
    deltas = score_delta.compare_cases(candidate, baseline, tolerance=0.05)
    assert deltas["coding-010"]["costRegressed"] is True


def test_diff_single_cost_spike_is_cost_flake():
    # EVAL-13: a single run over 2x with a median at/under 2x is a cost flake.
    candidate = {
        "coding-010": {"passRate": 1.0, "costPerTaskUsd": 0.025, "costsPerRun": [0.02, 0.02, 0.05]}
    }
    baseline = {"coding-010": {"passRate": 1.0, "costPerTaskUsd": 0.02}}
    deltas = score_delta.compare_cases(candidate, baseline, tolerance=0.05)
    assert deltas["coding-010"]["costFlake"] is True
    assert deltas["coding-010"]["costRegressed"] is False


def test_diff_unbaselined_and_missing():
    candidate = {"coding-001": {"passRate": 1.0}, "coding-099": {"passRate": 0.5}}
    baseline = {"coding-001": {"passRate": 1.0}}
    deltas = score_delta.compare_cases(candidate, baseline, tolerance=0.05)
    assert deltas["coding-099"]["unbaselined"] is True
    # A baseline-only case missing from the candidate.
    deltas2 = score_delta.compare_cases({"coding-001": {"passRate": 1.0}},
                                        {"coding-001": {"passRate": 1.0}, "coding-100": {"passRate": 0.9}},
                                        tolerance=0.05)
    assert deltas2["coding-100"]["missingInCandidate"] is True


def test_summary_counts():
    deltas = {
        "a": {"regressed": True, "costRegressed": False, "flake": False, "costFlake": False, "unbaselined": False},
        "b": {"regressed": False, "costRegressed": True, "flake": False, "costFlake": False, "unbaselined": False},
        "c": {"regressed": False, "costRegressed": False, "flake": True, "costFlake": False, "unbaselined": False},
        "d": {"regressed": False, "costRegressed": False, "flake": False, "costFlake": False, "unbaselined": True},
    }
    summary = score_delta.summarize(deltas)
    assert summary["regressed"] == 1
    assert summary["costRegressed"] == 1
    assert summary["flake"] == 1
    assert summary["unbaselined"] == 1
    assert summary["nCases"] == 4


def test_render_markdown_has_table():
    deltas = {"coding-001": {"deltaPassRate": 0.2, "deltaCostPerTaskUsd": 0.0, "regressed": False,
                             "costRegressed": False, "flake": False, "costFlake": False, "unbaselined": False}}
    summary = score_delta.summarize(deltas)
    md = score_delta.render_markdown(deltas, summary, "cand.json", "base.json")
    assert "# Scorecard delta" in md
    assert "| case |" in md
    assert "coding-001" in md


def test_cli_diff_roundtrip(tmp_path, capsys):
    cand = tmp_path / "cand.json"
    base = tmp_path / "base.json"
    cand.write_text(json.dumps(_scorecard({"coding-001": {"passRate": 1.0, "costPerTaskUsd": 0.01}})))
    base.write_text(json.dumps(_scorecard({"coding-001": {"passRate": 0.9, "costPerTaskUsd": 0.01}})))
    rc = score_delta.main(["--diff", str(cand), str(base), "--tolerance", "0.05"])
    assert rc == 0
    report = json.loads(capsys.readouterr().out)
    assert report["deltas"]["coding-001"]["deltaPassRate"] == 0.1
    assert report["summary"]["nCases"] == 1


def test_cli_classify_stdin(capsys, monkeypatch):
    import io
    monkeypatch.setattr(sys, "stdin", io.StringIO("eval/scripts/score_run.py\nREADME.md\n"))
    rc = score_delta.main(["--classify", "-"])
    assert rc == 0
    report = json.loads(capsys.readouterr().out)
    assert report["evalTouching"] is True
    assert report["evalSurfaces"] == ["gate"]
