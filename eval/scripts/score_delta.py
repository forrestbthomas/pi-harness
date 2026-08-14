"""score_delta.py — EVAL-16 pilot: harness-change eval delta (report-only).

The loop must measure changes to itself (EPIC-1 outcome; EVAL-16). A change
to the eval/harness surface can silently shift every scorecard — the
zero-token silent-success class — and nothing measures the delta today. This
module provides the two hermetic pilot mechanics:

1. `--classify <path...>` — classify a PR's changed paths into eval surfaces
   (dataset / grader / gate / benchmark / harness-scorecard / workflow /
   eval-tests / baseline / other) and flag whether the change needs nightly
   verification (surface list mirrors docs/benchmark-seam.md).

2. `--diff <candidate> <baseline> [--tolerance 0.05]` — compare two
   scorecards per case using the same tolerance model as the nightly gate
   (EVAL-2 flake-aware pass rate; EVAL-13 cost-variance) and emit a compact
   delta report (JSON + optional markdown).

Both modes are fully hermetic (stdlib only; no keys, no network, no live
runs). The pilot is report-only by design; promotion to an enforced gate is a
later, evidence-gated decision (first caught regression, or validated
delta-vs-noise mechanics).
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

# ---------------------------------------------------------------------------
# Classification (eval-surface inventory mirrors docs/benchmark-seam.md §5)
# ---------------------------------------------------------------------------

# Ordered surface patterns; the first match wins. Specific subpaths (graders/
# references/ live under eval/datasets/) must come before the broader
# eval/datasets/ rule.
SURFACE_RULES: list[tuple[str, tuple[str, ...]]] = [
    ("grader", ("eval/grader.py", "eval/datasets/graders/", "eval/datasets/references/")),
    ("dataset", ("eval/datasets/",)),
    ("gate", ("eval/scripts/score_run.py", "eval/scripts/score_delta.py")),
    ("baseline", ("eval/baselines/",)),
    ("benchmark", ("eval/benchmarks/",)),
    ("eval-tests", ("eval/tests/",)),
    ("harness-scorecard", ("internal/cli/scorecard",)),
    ("workflow", (".github/workflows/nightly-live-eval.yml", ".github/workflows/provider-scorecard.yml")),
]

# Any surface other than "other" changes live measurement semantics.
VERIFY_SURFACES = {
    "dataset", "grader", "gate", "baseline", "benchmark",
    "eval-tests", "harness-scorecard", "workflow",
}


def classify_path(path: str) -> str:
    """Return the eval surface for a changed path (first matching rule)."""
    for surface, prefixes in SURFACE_RULES:
        if any(path.startswith(prefix) for prefix in prefixes):
            return surface
    return "other"


def classify_paths(paths: list[str]) -> dict:
    """Classify a set of changed paths into a compact report."""
    touched: dict[str, list[str]] = {}
    for path in sorted(set(paths)):
        touched.setdefault(classify_path(path), []).append(path)
    eval_surfaces = [s for s in touched if s in VERIFY_SURFACES]
    return {
        "generated": utc_now(),
        "touched": touched,
        "evalTouching": bool(eval_surfaces),
        "needsNightlyVerification": bool(eval_surfaces),
        "evalSurfaces": sorted(eval_surfaces),
    }


# ---------------------------------------------------------------------------
# Scorecard delta (mirrors score_run.py's gate tolerance model)
# ---------------------------------------------------------------------------

DEFAULT_TOLERANCE = 0.05
COST_MULTIPLIER = 2.0  # EVAL-13: median > 2x baseline is a cost regression


def _as_float(value) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def load_cases(path: str) -> tuple[str, dict[str, dict]]:
    """Load a scorecard's per-case map; tolerate missing files."""
    p = Path(path)
    if not p.is_file():
        print(f"score_delta: warning: scorecard not found at {p}; treated as empty", file=sys.stderr)
        return str(p), {}
    with open(p, "r", encoding="utf-8") as handle:
        payload = json.load(handle)
    cases = payload.get("cases", {}) if isinstance(payload, dict) else {}
    return str(p), cases


def compare_cases(candidate: dict[str, dict], baseline: dict[str, dict],
                  tolerance: float) -> dict[str, dict]:
    """Per-case delta using the EVAL-2 flake + EVAL-13 cost model.

    Rules mirror score_run.compare_case: a pass-rate drop below
    baseline - tolerance with a single failed run is a flake (never a
    regression); >= 2 failed runs (or unknown run counts) is a regression.
    Cost > 2x baseline median is a cost regression. Missing baseline entries
    are recorded unbaselined (never failed).
    """
    deltas: dict[str, dict] = {}
    for case_id in sorted(set(candidate) | set(baseline)):
        cand = candidate.get(case_id)
        base = baseline.get(case_id)
        if cand is None:
            deltas[case_id] = {"missingInCandidate": True, "unbaselined": base is None}
            continue
        cand_rate = _as_float(cand.get("passRate"))
        cand_cost = _as_float(cand.get("costPerTaskUsd"))
        if base is None:
            deltas[case_id] = {
                "unbaselined": True, "passRate": cand_rate,
                "costPerTaskUsd": cand_cost,
            }
            continue
        base_rate = _as_float(base.get("passRate"))
        base_cost = _as_float(base.get("costPerTaskUsd"))
        below = cand_rate < base_rate - tolerance
        n_failed = cand.get("nFailed")
        n_cost_over = 0
        if base_cost > 0:
            n_cost_over = sum(
                1 for c in cand.get("costsPerRun", []) if _as_float(c) > COST_MULTIPLIER * base_cost
            )
        median_over = base_cost > 0 and cand_cost > COST_MULTIPLIER * base_cost
        flake = below and n_failed == 1
        regressed = below and (n_failed is None or n_failed >= 2)
        deltas[case_id] = {
            "unbaselined": False,
            "passRate": cand_rate,
            "baselinePassRate": base_rate,
            "deltaPassRate": round(cand_rate - base_rate, 6),
            "costPerTaskUsd": cand_cost,
            "baselineCostPerTaskUsd": base_cost,
            "deltaCostPerTaskUsd": round(cand_cost - base_cost, 9),
            "flake": flake,
            "costFlake": (not median_over) and n_cost_over == 1,
            "regressed": regressed,
            "costRegressed": median_over or n_cost_over >= 2,
        }
    return deltas


def summarize(deltas: dict[str, dict]) -> dict:
    keys = ("regressed", "costRegressed", "flake", "costFlake", "unbaselined")
    counts = {k: sum(1 for d in deltas.values() if d.get(k)) for k in keys}
    counts["missingInCandidate"] = sum(1 for d in deltas.values() if d.get("missingInCandidate"))
    counts["nCases"] = len(deltas)
    return counts


def render_markdown(deltas: dict[str, dict], summary: dict, candidate: str, baseline: str) -> str:
    lines = [
        "# Scorecard delta (EVAL-16 pilot, report-only)",
        "",
        f"- candidate: `{candidate}`",
        f"- baseline: `{baseline}`",
        f"- cases: {summary['nCases']} | regressions: {summary['regressed']} | "
        f"cost regressions: {summary['costRegressed']} | flakes: {summary['flake']} | "
        f"cost flakes: {summary['costFlake']} | unbaselined: {summary['unbaselined']}",
        "",
        "| case | Δ passRate | Δ cost | status |",
        "|---|---|---|---|",
    ]
    for case_id, d in sorted(deltas.items()):
        if d.get("missingInCandidate"):
            status = "missing in candidate"
        elif d.get("unbaselined"):
            status = "unbaselined"
        elif d.get("regressed") or d.get("costRegressed"):
            status = "REGRESSION"
        elif d.get("flake") or d.get("costFlake"):
            status = "flake"
        else:
            status = "ok"
        drate = f"{d.get('deltaPassRate', 0):+.3f}" if "deltaPassRate" in d else "—"
        dcost = f"{d.get('deltaCostPerTaskUsd', 0):+.6f}" if "deltaCostPerTaskUsd" in d else "—"
        lines.append(f"| {case_id} | {drate} | {dcost} | {status} |")
    return "\n".join(lines) + "\n"


def utc_now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="score_delta.py",
        description="EVAL-16 pilot: hermetic eval-surface classification + scorecard delta (report-only).",
    )
    parser.add_argument("--classify", nargs="*", default=None, metavar="PATH",
                        help="classify changed paths as eval surfaces; use '-' to read paths from stdin")
    parser.add_argument("--diff", nargs=2, default=None, metavar=("CANDIDATE", "BASELINE"),
                        help="compare two scorecards per case (candidate vs baseline)")
    parser.add_argument("--tolerance", type=float, default=DEFAULT_TOLERANCE,
                        help="per-case pass-rate tolerance (default: 0.05)")
    parser.add_argument("--out", default=None, help="write JSON report to this path (stdout default)")
    parser.add_argument("--markdown", default=None, help="also write a markdown report to this path")
    return parser


def main(argv=None) -> int:
    args = build_parser().parse_args(argv)
    if args.classify is None and args.diff is None:
        build_parser().print_usage(sys.stderr)
        return 2
    if args.classify is not None:
        paths = args.classify
        if paths == ["-"]:
            paths = [line.strip() for line in sys.stdin if line.strip()]
        report = classify_paths(paths)
    else:
        candidate_path, baseline_path = args.diff
        candidate, candidate_cases = load_cases(candidate_path)
        _, baseline_cases = load_cases(baseline_path)
        deltas = compare_cases(candidate_cases, baseline_cases, args.tolerance)
        summary = summarize(deltas)
        report = {
            "generated": utc_now(),
            "tolerance": args.tolerance,
            "candidate": candidate_path,
            "baseline": baseline_path,
            "summary": summary,
            "deltas": deltas,
        }
        if args.markdown:
            Path(args.markdown).write_text(
                render_markdown(deltas, summary, candidate_path, baseline_path),
                encoding="utf-8",
            )
    text = json.dumps(report, indent=2) + "\n"
    if args.out:
        Path(args.out).write_text(text, encoding="utf-8")
    else:
        sys.stdout.write(text)
    return 0


if __name__ == "__main__":
    sys.exit(main())
