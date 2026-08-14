#!/usr/bin/env python3
"""score_run.py — baseline gate + cost metrics for the nightly live eval.

Stdlib-only port of the Autonoma run_evals pattern (live-agent-eval-v2 design,
spec §4.4). Consumes a pytest-json-report JSON produced by the live suite
(eval/tests/test_live_suite.py) and compares per-case aggregates against a
committed baseline (eval/baselines/live-baseline.json).

Usage:
    score_run.py --report <pytest-json-report.json> --baseline <live-baseline.json>
                 [--tolerance 0.05] [--runs 3] [--budget-usd 2.0] [--out <path>]
                 [--update-baseline] [--allow] [--json-summary <path>]

Gate math (spec §4.3/§4.4):
  * per-case passRate = MEAN over runs (binary runs, so mean = proportion)
  * per-case cost / latency / tokens = MEDIAN over runs (robust to outliers)
  * totals: costPerTaskUsd = total(agent + judge) / nCases (ALL-IN);
    agentCostPerTaskUsd reported separately; costPerSuccessfulTaskUsd guards
    division by zero (0.0 when nPassed == 0); tokensPerTask = total / nCases
  * per-case regression: passRate < baseline - tolerance
  * cost regression: costPerTaskUsd > 2 * baseline costPerTaskUsd
  * unbaselined cases are recorded, never failed
  * incomplete runs (fewer than --runs completed per case, or any errored run)
    fail the gate; exceeding --budget-usd fails the gate

Exit codes:
  0  pass
  1  any gate failure (pass-rate regression, cost regression, incomplete,
     budget, or a non-clean pytest report)
  2  usage error

Fully hermetic: reads only the report + baseline JSONs, writes the summary
JSON and (optionally) rewrites the baseline. No keys, no network.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import statistics
import sys
from datetime import datetime, timezone
from pathlib import Path

SCHEMA_VERSION = 1
DEFAULT_RUNS = 3
DEFAULT_TOLERANCE = 0.05
DEFAULT_BUDGET_USD = 2.0
BASELINE_REL = Path("eval/baselines/live-baseline.json")
LIVE_RESULTS_REL = Path("eval/live-results")
LIVE_SUITE_FILE = "test_live_suite.py"

# The scorer only consumes parametrized nodeids from the live suite
# (tests/test_live_suite.py::test_case[coding-001] -> coding-001). Nodeids
# from other live files (test_live_metrics.py, test_agent_task_completion.py)
# are ignored so metric parametrization can never pollute the case registry.
NODEID_CASE_RE = re.compile(r"\[([^\]]+)\]$")
COMPLETED_OUTCOMES = {"passed", "failed", "xpassed", "xfailed"}


def repo_root() -> Path:
    """Absolute path to the repository root (this file lives at <root>/eval/scripts/)."""
    return Path(__file__).resolve().parents[2]


def default_baseline_path() -> Path:
    return repo_root() / BASELINE_REL


def default_out_path() -> Path:
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S")
    return repo_root() / LIVE_RESULTS_REL / f"live-{stamp}.json"


def utc_now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


# ---------------------------------------------------------------------------
# Parsing
# ---------------------------------------------------------------------------


def _as_bool(value):
    if isinstance(value, bool):
        return value
    if isinstance(value, (int, float)):
        return bool(value)
    if isinstance(value, str):
        return value.strip().lower() in ("1", "true", "yes", "pass", "passed")
    return bool(value)


def _as_float(value):
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _as_int(value):
    try:
        return int(value)
    except (TypeError, ValueError):
        return 0


def _property(metadata, name):
    """Read a record_property value (scalar or list) from test metadata.

    pytest-json-report stores record_property values under test.metadata; a
    property recorded multiple times becomes a list (we take the last value).
    """
    if not isinstance(metadata, dict):
        return None
    value = metadata.get(name)
    if isinstance(value, list):
        return value[-1] if value else None
    return value


def _ordered_properties(test: dict) -> dict[str, list]:
    """Return {property name: values in recorded order} from user_properties.

    pytest-json-report 1.5.0 serializes record_property under
    ``test["user_properties"]`` as a FLAT, ORDERED list of single-key dicts:
    ``[{str(key): val}, ...]`` (plugin.py: ``user_properties =
    [{str(key): val} for key, val in report.user_properties]``). The live
    suite (eval/tests/test_live_suite.py) records each repeat as one set of
    properties (pass, costUsd, judgeCostUsd, tokens, latencyMs), so the list
    reads: [pass0, cost0, judge0, tokens0, latency0, pass1, cost1, ...]. We
    bucket by name preserving order so callers can zip values back into
    per-run rows.
    """
    out: dict[str, list] = {}
    for item in test.get("user_properties") or []:
        if not isinstance(item, dict):
            continue
        for key, value in item.items():
            out.setdefault(key, []).append(value)
    return out


def _zip_runs(props: dict[str, list], outcome: str) -> list[dict]:
    """Zip bucketed property lists into per-run rows (one row per repeat).

    The live suite records every property once per repeat in lockstep, so the
    number of runs is the length of any property list (they are all equal). A
    missing/None property for a run means the metric was not recorded for that
    repeat; downstream aggregation treats it as 0/absent without failing the
    case (the suite itself only emits complete rows).
    """
    n = max((len(values) for values in props.values()), default=0)
    runs = []
    for i in range(n):
        runs.append(
            {
                "outcome": outcome,
                "pass": props.get("pass", [None] * n)[i],
                "costUsd": props.get("costUsd", [None] * n)[i],
                "judgeCostUsd": props.get("judgeCostUsd", [None] * n)[i],
                "tokens": props.get("tokens", [None] * n)[i],
                "latencyMs": props.get("latencyMs", [None] * n)[i],
            }
        )
    return runs


def extract_case_id(nodeid: str) -> str | None:
    """Recover the case id from a live-suite parametrized nodeid.

    tests/test_live_suite.py::test_case[coding-001] -> "coding-001"
    Returns None for nodeids that are not live-suite parametrized cases.
    """
    if not nodeid or LIVE_SUITE_FILE not in nodeid:
        return None
    match = NODEID_CASE_RE.search(nodeid)
    return match.group(1) if match else None


def parse_report(path) -> dict:
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def collect_cases(report: dict) -> dict[str, list[dict]]:
    """Map case id -> list of per-run dicts from report['tests'].

    Each live-suite test entry carries ALL EVAL_RUNS_PER_CASE repeats inside
    one ``user_properties`` list (flat, ordered); we zip the property values
    back into one row per repeat (see _zip_runs).
    """
    cases: dict[str, list[dict]] = {}
    for test in report.get("tests", []):
        case_id = extract_case_id(test.get("nodeid", ""))
        if case_id is None:
            continue
        props = _ordered_properties(test)
        runs = _zip_runs(props, test.get("outcome", ""))
        cases.setdefault(case_id, []).extend(runs)
    return cases


# ---------------------------------------------------------------------------
# Aggregation (spec §4.3 gate math)
# ---------------------------------------------------------------------------


def _completed(run: dict) -> bool:
    return run["outcome"] in COMPLETED_OUTCOMES


def _run_pass(run: dict) -> bool:
    recorded = run["pass"]
    if recorded is not None:
        return _as_bool(recorded)
    return run["outcome"] in ("passed", "xpassed")


def _median(values: list[float]) -> float:
    return statistics.median(values) if values else 0.0


def aggregate_case(runs: list[dict], expected_runs: int) -> dict:
    """Rebuild one case's aggregates: mean pass rate, median cost/latency/tokens."""
    completed = [r for r in runs if _completed(r)]
    errored = any(r["outcome"] == "error" for r in runs)

    pass_flags = [_run_pass(r) for r in completed]
    pass_rate = statistics.fmean(pass_flags) if pass_flags else 0.0

    agent_costs = [_as_float(r["costUsd"]) for r in completed]
    judge_costs = [_as_float(r["judgeCostUsd"]) for r in completed]
    cost_per_run = [a + j for a, j in zip(agent_costs, judge_costs)]
    tokens = [_as_int(r["tokens"]) for r in completed]
    latencies = [_as_float(r["latencyMs"]) for r in completed]

    return {
        "nRuns": len(completed),
        "nFailed": sum(1 for r in completed if not _run_pass(r)),
        "errored": errored,
        "incomplete": errored or len(completed) < expected_runs,
        "passRate": round(pass_rate, 6),
        "agentCostUsd": round(_median(agent_costs), 9),
        "judgeCostUsd": round(_median(judge_costs), 9),
        "costPerTaskUsd": round(_median(cost_per_run), 9),
        "tokensPerTask": _as_int(round(_median(tokens))),
        "latencyMs": round(_median(latencies), 3),
    }


def compute_totals(cases_raw: dict[str, list[dict]], case_aggs: dict[str, dict]) -> dict:
    """Totals across the whole run.

    totalCostUsd = sum of (agent + judge) over every completed run (spec §4.4).
    costPerTaskUsd is ALL-IN (agent + judge); the Go scorecard's costPerTaskUsd
    is agent-only and must not be compared directly — agentCostPerTaskUsd is
    reported separately for cross-surface comparison.
    """
    total_agent = 0.0
    total_judge = 0.0
    total_tokens = 0
    for runs in cases_raw.values():
        for run in runs:
            if not _completed(run):
                continue
            total_agent += _as_float(run["costUsd"])
            total_judge += _as_float(run["judgeCostUsd"])
            total_tokens += _as_int(run["tokens"])

    n_cases = len(case_aggs)
    pass_rates = [agg["passRate"] for agg in case_aggs.values()]
    # A case counts as "passed" for the cost-per-successful-task denominator
    # only when ALL of its runs passed (passRate == 1.0). A tolerance-passing
    # case (e.g. 0.67) is deliberately excluded: it did not fully succeed, so
    # its spend should not be counted as "successful task" economics. This is
    # the strict definition; document if it ever needs to become partial-credit.
    n_passed = sum(1 for rate in pass_rates if rate == 1.0)
    overall = statistics.fmean(pass_rates) if pass_rates else 0.0
    total_cost = total_agent + total_judge

    return {
        "nCases": n_cases,
        "nPassed": n_passed,
        "nFailed": n_cases - n_passed,
        "overallPassRate": round(overall, 6),
        "totalCostUsd": round(total_cost, 9),
        "agentCostUsd": round(total_agent, 9),
        "judgeCostUsd": round(total_judge, 9),
        "costPerTaskUsd": round(total_cost / n_cases, 9) if n_cases else 0.0,
        "agentCostPerTaskUsd": round(total_agent / n_cases, 9) if n_cases else 0.0,
        "costPerSuccessfulTaskUsd": round(total_cost / n_passed, 9) if n_passed else 0.0,
        "totalTokens": total_tokens,
        "tokensPerTask": round(total_tokens / n_cases) if n_cases else 0,
    }


# ---------------------------------------------------------------------------
# Baseline comparison + gate (spec §4.4)
# ---------------------------------------------------------------------------


def load_baseline(path) -> dict:
    """Load a baseline; a missing file is treated as an empty baseline."""
    baseline_path = Path(path)
    if not baseline_path.is_file():
        print(f"score_run: warning: baseline not found at {baseline_path}; all cases unbaselined", file=sys.stderr)
        return {"schemaVersion": SCHEMA_VERSION, "cases": {}}
    with open(baseline_path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def compare_case(case_id: str, agg: dict, baseline_cases: dict, tolerance: float) -> dict:
    """Compare one case against its baseline entry (unbaselined cases never fail)."""
    base = (baseline_cases or {}).get(case_id)
    if base is None:
        return {
            "unbaselined": True,
            "flake": False,
            "regressed": False,
            "costRegressed": False,
            "incomplete": agg["incomplete"],
        }
    base_pass = _as_float(base.get("passRate"))
    base_cost = _as_float(base.get("costPerTaskUsd"))
    n_failed = int(agg.get("nFailed", 0))
    # Flake-aware gate (EVAL-2): a single failed run is a flake, not a
    # regression; recurring failure (>= 2 runs) is a real signal. Flakes are
    # reported in the scorecard but never fail the gate.
    below_tolerance = agg["passRate"] < base_pass - tolerance
    return {
        "unbaselined": False,
        "flake": below_tolerance and n_failed == 1,
        "regressed": below_tolerance and n_failed >= 2,
        "costRegressed": agg["costPerTaskUsd"] > 2.0 * base_cost,
        "incomplete": agg["incomplete"],
        "baselinePassRate": base_pass,
        "baselineCostPerTaskUsd": base_cost,
        "baselineTokensPerTask": _as_int(base.get("tokensPerTask")),
        "baselineLatencyMs": _as_float(base.get("latencyMs")),
        "deltaPassRate": round(agg["passRate"] - base_pass, 6),
        "deltaCostPerTaskUsd": round(agg["costPerTaskUsd"] - base_cost, 9),
    }


def evaluate_gate(case_comps: dict[str, dict], totals: dict, budget_usd: float,
                  report_exitcode: int) -> dict:
    """Evaluate all gate conditions; return {passed, failures}."""
    failures: list[str] = []
    for case_id, comp in sorted(case_comps.items()):
        if comp["incomplete"]:
            failures.append(
                f"case {case_id}: incomplete run (fewer than --runs completed per case or errored)"
            )
        elif not comp["unbaselined"]:
            if comp["regressed"]:
                failures.append(
                    f"case {case_id}: passRate {comp.get('deltaPassRate', 0):+.3f} vs baseline "
                    f"(below baseline - tolerance)"
                )
            if comp["costRegressed"]:
                failures.append(
                    f"case {case_id}: costPerTaskUsd regression (> 2x baseline)"
                )
    if totals["nCases"] == 0:
        failures.append("no live case tests found in report")
    if report_exitcode != 0:
        failures.append(f"pytest run did not complete cleanly (report exitcode {report_exitcode})")
    if budget_usd is not None and budget_usd > 0 and totals["totalCostUsd"] >= budget_usd:
        failures.append(
            f"budget exceeded: totalCostUsd {totals['totalCostUsd']:.4f} >= budget "
            f"{budget_usd:.4f} (PI_MAX_BUDGET_USD cap surfaced as a gate failure)"
        )
    return {"passed": not failures, "failures": failures}


# ---------------------------------------------------------------------------
# Output
# ---------------------------------------------------------------------------


def merge_case_entries(case_aggs: dict[str, dict], case_comps: dict[str, dict]) -> dict[str, dict]:
    """Merge per-case aggregates with their baseline comparison into one entry."""
    entries = {}
    for case_id in case_aggs:
        entry = dict(case_aggs[case_id])
        entry.update(case_comps[case_id])
        entries[case_id] = entry
    return entries


def parse_self_heal_events(path) -> dict:
    """Read .pi/heal/events.jsonl (best-effort) and count events by kind.

    Missing files and malformed lines are tolerated (0 events): this is
    observability data for the scorecard and must never change the gate.
    """
    by_kind: dict[str, int] = {}
    n_events = 0
    try:
        with Path(path).open("r", encoding="utf-8") as handle:
            for line in handle:
                line = line.strip()
                if not line:
                    continue
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if not isinstance(event, dict):
                    continue
                kind = event.get("kind")
                if not isinstance(kind, str) or not kind:
                    continue
                n_events += 1
                by_kind[kind] = by_kind.get(kind, 0) + 1
    except OSError:
        return {"nEvents": 0, "byKind": {}}
    return {"nEvents": n_events, "byKind": by_kind}


def load_dataset_version() -> str:
    """Read datasetVersion from the tasks manifest (best-effort, EVAL-3).

    Missing/malformed manifests yield "unknown" so provenance is never fatal;
    the schema lint enforces that the manifest actually carries a version.
    """
    try:
        tasks_path = repo_root() / "eval" / "datasets" / "tasks.json"
        tasks = json.loads(tasks_path.read_text(encoding="utf-8"))
        version = tasks.get("datasetVersion")
        return version if isinstance(version, str) and version else "unknown"
    except (OSError, ValueError):
        return "unknown"


def build_provenance() -> dict:
    """Attribution for a scorecard run: dataset, agent tier, judge, pi version."""
    return {
        "datasetVersion": load_dataset_version(),
        "agentModel": os.environ.get("PI_MODEL_TIER") or "unknown",
        "judgeModel": os.environ.get("OPENAI_MODEL_NAME") or "unknown",
        "piVersion": os.environ.get("PI_VERSION") or "unknown",
    }


def build_summary(args, totals: dict, gate: dict,
                  case_entries: dict[str, dict], exit_code: int,
                  self_heal: dict | None = None) -> dict:
    return {
        "schemaVersion": SCHEMA_VERSION,
        "generated": utc_now(),
        "run": {
            "report": args.report,
            "baseline": args.baseline,
            "runs": args.runs,
            "tolerance": args.tolerance,
            "budgetUsd": args.budget_usd,
            "exitcode": exit_code,
        },
        "totals": totals,
        "gate": gate,
        "unbaselined": sorted(
            case_id for case_id, entry in case_entries.items() if entry["unbaselined"]
        ),
        "flakes": sorted(
            case_id for case_id, entry in case_entries.items() if entry.get("flake")
        ),
        "provenance": build_provenance(),
        "cases": case_entries,
        "selfHeal": self_heal if self_heal is not None else {"nEvents": 0, "byKind": {}},
    }


def build_compact_summary(summary: dict) -> dict:
    """Small machine-readable summary (--json-summary): totals + gate only."""
    return {
        "schemaVersion": summary["schemaVersion"],
        "generated": summary["generated"],
        "run": summary["run"],
        "totals": summary["totals"],
        "gate": summary["gate"],
        "unbaselined": summary["unbaselined"],
        "flakes": summary["flakes"],
        "provenance": summary["provenance"],
        "selfHeal": summary["selfHeal"],
    }


def render_markdown(case_entries: dict[str, dict], totals: dict, gate: dict,
                    self_heal: dict | None = None) -> str:
    """Markdown table for GITHUB_STEP_SUMMARY: current / baseline / delta."""
    lines = ["## Live eval gate", ""]
    lines.append(f"- gate: **{'PASS' if gate['passed'] else 'FAIL'}**")
    lines.append(
        f"- overall pass rate: {totals['overallPassRate']:.1%} "
        f"({totals['nPassed']}/{totals['nCases']} cases)"
    )
    lines.append(
        f"- total cost: ${totals['totalCostUsd']:.4f} · "
        f"cost/task: ${totals['costPerTaskUsd']:.4f} · "
        f"cost/successful task: ${totals['costPerSuccessfulTaskUsd']:.4f}"
    )
    if self_heal is not None:
        by_kind = self_heal.get("byKind") or {}
        detail = ", ".join(f"{kind}: {count}" for kind, count in sorted(by_kind.items()))
        lines.append(f"- self-heal events: {self_heal.get('nEvents', 0)}" + (f" ({detail})" if detail else ""))
    flakes = sorted(case_id for case_id, entry in case_entries.items() if entry.get("flake"))
    if flakes:
        lines.append(f"- flakes: {len(flakes)} — single-run failures (not regressions): {', '.join(flakes)}")
    lines.append("")
    lines.append("| Case | passRate | baseline | Δ pass | cost/task | baseline cost | Δ cost | status |")
    lines.append("|---|---|---|---|---|---|---|---|")
    for case_id in sorted(case_entries):
        entry = case_entries[case_id]
        if entry["unbaselined"]:
            status = "unbaselined"
            base_pass = base_cost = delta_pass = delta_cost = "-"
        else:
            base_pass = f"{entry['baselinePassRate']:.2f}"
            base_cost = f"${entry['baselineCostPerTaskUsd']:.4f}"
            delta_pass = f"{entry['deltaPassRate']:+.2f}"
            delta_cost = f"${entry['deltaCostPerTaskUsd']:+.4f}"
            if entry["regressed"] or entry["costRegressed"]:
                status = ":x: REGRESSED"
            elif entry.get("flake"):
                status = ":warning: flake"
            elif entry["incomplete"]:
                status = "incomplete"
            else:
                status = "ok"
        lines.append(
            f"| {case_id} | {entry['passRate']:.2f} | {base_pass} | {delta_pass} | "
            f"${entry['costPerTaskUsd']:.4f} | {base_cost} | {delta_cost} | {status} |"
        )
    return "\n".join(lines) + "\n"


def write_json(path, payload) -> None:
    out = Path(path)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def write_baseline(baseline_path, case_aggs: dict[str, dict], runs: int) -> dict:
    """Rewrite the committed baseline from the current report (deliberate act).

    Incomplete cases are omitted — a baseline must never enshrine a partial run.
    """
    cases = {}
    for case_id in sorted(case_aggs):
        agg = case_aggs[case_id]
        if agg["incomplete"]:
            print(f"score_run: skipping incomplete case {case_id} from baseline", file=sys.stderr)
            continue
        cases[case_id] = {
            "passRate": agg["passRate"],
            "costPerTaskUsd": agg["costPerTaskUsd"],
            "tokensPerTask": agg["tokensPerTask"],
            "latencyMs": agg["latencyMs"],
        }
    baseline = {
        "schemaVersion": SCHEMA_VERSION,
        "generated": utc_now(),
        "runsPerCase": runs,
        # NOTE: agentModel records the --model-tier tier name (e.g. "cheap"),
        # not the resolved model id — the resolved id is not plumbed into the
        # report. It is informational (re-baseline provenance), so a tier name
        # is acceptable; do not treat it as the exact model string.
        "agentModel": os.environ.get("PI_MODEL_TIER") or "unknown",
        "judgeModel": os.environ.get("OPENAI_MODEL_NAME") or "unknown",
        "cases": cases,
    }
    path = Path(baseline_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(baseline, indent=2) + "\n", encoding="utf-8")
    print(f"score_run: baseline updated at {path} "
          f"({len(cases)} cases, {len(case_aggs) - len(cases)} incomplete omitted)", file=sys.stderr)
    return baseline


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def build_arg_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="score_run.py",
        description=(
            "Baseline gate + cost metrics for the nightly live eval. Consumes a "
            "pytest-json-report JSON and compares per-case aggregates against a "
            "committed baseline. Exit 0 = pass, 1 = gate failure, 2 = usage error."
        ),
    )
    parser.add_argument("--report", required=True, help="pytest-json-report JSON from the live suite")
    parser.add_argument(
        "--baseline",
        default=str(default_baseline_path()),
        help="committed baseline JSON (default: eval/baselines/live-baseline.json)",
    )
    parser.add_argument(
        "--tolerance", type=float, default=DEFAULT_TOLERANCE,
        help="per-case pass-rate tolerance (default: 0.05)",
    )
    parser.add_argument(
        "--runs", type=int, default=DEFAULT_RUNS,
        help="expected runs per case (default: 3); fewer completed runs fail the gate as incomplete",
    )
    parser.add_argument(
        "--budget-usd", type=float, default=DEFAULT_BUDGET_USD,
        help="total run spend cap; exceeding it fails the gate (default: 2.0; <=0 disables)",
    )
    parser.add_argument(
        "--out", default=None,
        help="summary JSON path (default: eval/live-results/live-<run>.json)",
    )
    parser.add_argument(
        "--update-baseline", action="store_true",
        help="rewrite the baseline from the current report (deliberate act; requires --allow)",
    )
    parser.add_argument(
        "--allow", action="store_true",
        help="acknowledge that re-baselining is deliberate; required with --update-baseline",
    )
    parser.add_argument(
        "--json-summary", default=None,
        help="optional compact machine-readable summary (totals + gate, no per-case detail)",
    )
    parser.add_argument(
        "--heal-events", default=str(repo_root() / ".pi" / "heal" / "events.jsonl"),
        help="self-heal events JSONL to surface in the scorecard (default: <repo>/.pi/heal/events.jsonl)",
    )
    return parser


def main(argv=None) -> int:
    parser = build_arg_parser()
    args = parser.parse_args(argv)

    if args.update_baseline and not args.allow:
        parser.error("--update-baseline requires --allow (re-baselining is a deliberate, reviewed act)")
    if args.allow and not args.update_baseline:
        parser.error("--allow is only meaningful together with --update-baseline")
    if args.runs < 1:
        parser.error("--runs must be >= 1")
    if args.tolerance < 0:
        parser.error("--tolerance must be >= 0")

    try:
        report = parse_report(args.report)
        baseline = load_baseline(args.baseline)
    except (OSError, ValueError) as exc:
        print(f"score_run: cannot read report/baseline: {exc}", file=sys.stderr)
        return 2

    cases_raw = collect_cases(report)
    case_aggs = {case_id: aggregate_case(runs, args.runs) for case_id, runs in cases_raw.items()}
    totals = compute_totals(cases_raw, case_aggs)
    baseline_cases = baseline.get("cases", {}) if isinstance(baseline, dict) else {}
    case_comps = {
        case_id: compare_case(case_id, case_aggs[case_id], baseline_cases, args.tolerance)
        for case_id in case_aggs
    }
    case_entries = merge_case_entries(case_aggs, case_comps)
    gate = evaluate_gate(case_comps, totals, args.budget_usd, report.get("exitcode", 0))
    exit_code = 0 if gate["passed"] else 1

    summary = build_summary(args, totals, gate, case_entries, exit_code,
                            parse_self_heal_events(args.heal_events))

    try:
        out_path = Path(args.out) if args.out else default_out_path()
        write_json(out_path, summary)
        if args.json_summary:
            write_json(args.json_summary, build_compact_summary(summary))
        if args.update_baseline:
            write_baseline(args.baseline, case_aggs, args.runs)
        step_summary = os.environ.get("GITHUB_STEP_SUMMARY")
        if step_summary:
            with open(step_summary, "a", encoding="utf-8") as handle:
                handle.write(render_markdown(case_entries, totals, gate, summary["selfHeal"]))
    except OSError as exc:
        print(f"score_run: cannot write outputs: {exc}", file=sys.stderr)
        return 1

    status = "PASS" if gate["passed"] else "FAIL"
    print(
        f"score_run: gate {status} — {totals['nPassed']}/{totals['nCases']} cases passed, "
        f"overall pass rate {totals['overallPassRate']:.1%}, "
        f"total cost ${totals['totalCostUsd']:.4f}, "
        f"cost/task ${totals['costPerTaskUsd']:.4f}, "
        f"cost/successful ${totals['costPerSuccessfulTaskUsd']:.4f}"
    )
    for failure in gate["failures"]:
        print(f"score_run: gate failure: {failure}", file=sys.stderr)
    for case_id in sorted(case_entries):
        entry = case_entries[case_id]
        if entry.get("flake"):
            base = entry.get("baselinePassRate", 0.0)
            print(
                f"score_run: flake (not gate failure): case {case_id}: "
                f"passRate {entry['passRate']:.3f} vs baseline {base:.3f} (single failed run)",
                file=sys.stderr,
            )
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
