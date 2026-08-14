# Spec — EVAL-1 + EVAL-2: Evidence Artifacts on Every Outcome + Flake-Aware Gate — W7

**Date:** 2026-08-14 · **Status:** SHIPPED — W7 (2026-08-14, PR #87)
**Source:** BACKLOG EVAL-1 (1.30) + EVAL-2 (1.40), EPIC-1 — bet on by user (2026-08-14).

## Goal

Make every eval run leave a complete, retrievable evidence artifact regardless of gate outcome (EVAL-1), and stop the gate from failing on single-run flake while still catching real regressions — with the scorecard reporting flake rate (EVAL-2).

## Problem Statement

1. **Evidence is lost on failure.** The nightly's "Upload live results" step and the weekly scorecard's "Upload scorecard" step run only on job success (no `if:`). When the gate fails — as it did 2026-08-14 (`coding-005` single-run dip) — the report, summary, and `.pi/heal/events.jsonl` stay on the dead runner. We had to reconstruct evidence from logs.
2. **Single-run flake reads as a regression.** With n=3, one failed run drops a case from passRate 1.0 → 0.67, a -0.33 swing, and the gate fails. That conflates noise with signal and blocks the nightly on flaky cases.

## Context From Code

- `nightly-live-eval.yml`: "Upload live results" step (`actions/upload-artifact`, `path: eval/live-results/`, retention 90) has **no `if:`**. Gate step: `score_run.py --runs 3 --tolerance 0.05 --budget-usd 2.0`.
- `provider-scorecard.yml`: "Upload scorecard" step (`path: eval/benchmark-results/scorecard-*.json` + `-latest.json`) also has **no `if:`**.
- `score_run.py`:
  - `aggregate_case` (line ~214) returns `nRuns`, `errored`, `incomplete`, `passRate`, cost/latency/tokens medians — **no `nFailed`**.
  - `compare_case` (line 314): `regressed = agg["passRate"] < base_pass - tolerance` (line 328); `costRegressed = > 2x base` (line 329).
  - `evaluate_gate` (line 340): fails on `incomplete`, `regressed`, `costRegressed`.
  - `render_markdown` (line ~470): `":x: REGRESSED"` when `regressed or costRegressed`; no flake concept.
  - `build_summary`/`build_compact_summary`: no flake field.
- Tests: `test_score_run.py` has boundary tests using n_runs=20 (2+ fails → still regressed under the new rule); `test_docs_drift.py` is the deterministic docs/workflow guard pattern.

## In Scope

- **EVAL-1 — evidence on every outcome:**
  - `nightly-live-eval.yml`: add `if: always()` to "Upload live results"; extend `path` to include `.pi/heal/events.jsonl` (the W6 events ledger) alongside `eval/live-results/`.
  - `provider-scorecard.yml`: add `if: always()` to "Upload scorecard".
  - Guard: extend `test_docs_drift.py` with a deterministic invariant asserting both workflows' upload steps carry `if: always()` (text-based, stdlib-only, keyless).
- **EVAL-2 — flake-aware gate:**
  - `aggregate_case`: add `nFailed` (completed runs where pass=False).
  - `compare_case`: add `flake`; `regressed` becomes `passRate < base - tolerance AND nFailed >= 2`; `flake` = `passRate < base - tolerance AND nFailed == 1`.
  - `evaluate_gate`: unchanged — flakes are **warn-only**, never gate failures.
  - Summary + compact: add `flakes` (list of case ids) to `build_summary`/`build_compact_summary`; `render_markdown` shows a `- flakes:` line and marks flake cases as `⚠️ flake` (not REGRESSED); `main()` prints a `score_run: flake (not gate failure)` stderr line.
  - Nightly: `EVAL_RUNS_PER_CASE: '3' → '5'` and `--runs 3 → 5` (cheaper per-run than the flake it prevents; ~$0.27/night).
- **Tests (hermetic):** `test_score_run.py` — nFailed aggregation; single-fail case = flake (gate passes, summary lists it); double-fail = regressed (gate fails); cost regression still fails; compact + markdown include flakes; existing boundary tests stay green (n_runs=20 cases have ≥2 fails).
- **Docs/records:** CHANGELOG `[Unreleased]`; ROADMAP W7 row; STATUS; BACKLOG EVAL-1/EVAL-2 → active; SCOPE.md contract; spec.

## Out Of Scope

- Flake thresholds beyond the fixed rule (1-of-N = flake; ≥2 = regression) — no configurable threshold, no "fail after N flakes".
- Auto-quarantine / known-flaky registry (EVAL-9, separate ticket).
- Re-baselining the committed baseline (a live-run act; runsPerCase stays informational; passRate proportions remain comparable).
- Provider-scorecard gate logic (`ci-benchmark` is Go; only the upload `if:` changes there).
- Uploading escalation packets (`.pi/heal/*-report.json`) — they carry goals/ledger; keep events.jsonl only.

## Constraints

- Stdlib-only Python in `eval/`; no PyYAML — the workflow guard is text-based like the existing docs-drift tests.
- Keyless deterministic suite: new tests must run without provider keys (they do — score_run and docs-drift are hermetic).
- Baseline gate math must stay honest: flakes are reported, never hidden; regressions still fail.

## Assumptions

- n=5 keeps per-night cost trivial (~$0.27) and is worth the better granularity.
- A single failed run (1-of-5) is assumed flake unless it recurs (2-of-5) — a pragmatic first pass; thresholds can be tuned later (EPIC-1 sequence allows it).
- `.pi/heal/events.jsonl` is safe to upload as an artifact (consistent with uploading cost summaries; events reference goals but not keys).

## Blocking Questions

None — bet approved by user; decisions above are the first-pass rule.

## Acceptance Criteria

Gherkin:

```gherkin
Scenario: Single-run flake warns, does not fail (EVAL-2)
  Given a case with baseline passRate 1.0 and exactly 1 of 5 failed runs
  When score_run.py runs
  Then the gate passes (exit 0)
  And the summary and compact summary contain the case id under "flakes"
  And the step summary shows a flake line, not REGRESSED

Scenario: Recurring failure fails (EVAL-2)
  Given a case with baseline passRate 1.0 and 2 of 5 failed runs
  When score_run.py runs
  Then the gate fails (exit 1) with the case reported as a regression

Scenario: Cost regression still fails (EVAL-2)
  Given a case with costPerTaskUsd > 2x baseline
  When score_run.py runs
  Then the gate fails with a cost regression

Scenario: Artifacts survive gate failure (EVAL-1)
  Given the nightly or provider-scorecard gate fails
  When the job finishes
  Then the upload step still runs (if: always())
  And the artifact contains the eval results and .pi/heal/events.jsonl
```

Non-behavioral:

- Hermetic tests cover both scenarios and the workflow invariant; all run in the deterministic CI job.
- `go build/vet/test`, the deterministic pytest set, and docs-drift stay green.

## Edge Cases

- Case with baseline passRate < 1.0 and 1 failed run: only a flake if the pass rate still drops below baseline - tolerance (0.67 vs 0.33 = improvement, no flake).
- 0 failed runs: never a pass regression (passRate 1.0 ≥ baseline).
- Errored/incomplete runs: unchanged (still gate failures).
- Unbaselined cases: unchanged (recorded, never failed).
- n=5 run with a single flake on a different case than Friday's: still warn-only.

## Validation Plan

- `eval/.venv/bin/python -m pytest eval/tests/test_score_run.py eval/tests/test_docs_drift.py -v`.
- Full deterministic CI set locally (the `deterministic` job pytest list).
- `go build ./... && go vet ./... && go test ./...`.
- Manual hermetic run: craft a 1-of-5 report + baseline → exit 0 with `flakes`; 2-of-5 → exit 1.
- Docs-drift test green after workflow edits.

## Execution Plan (ordered)

1. RED: add failing tests in `test_score_run.py` (nFailed, flake-vs-regression, flakes in summary/compact/markdown) and the `test_docs_drift.py` workflow `if: always()` invariant.
2. GREEN: implement `score_run.py` (nFailed, flake, summary/compact/markdown, stderr) and add `if: always()` + `.pi/heal/events.jsonl` + n=5 to the workflows.
3. Update docs: CHANGELOG `[Unreleased]`; ROADMAP W7 row; STATUS; BACKLOG (EVAL-1/2 active).
4. Verification-before-completion (Go + pytest + docs-drift).
5. Land via PR (`BACKLOG EVAL-1/EVAL-2 — evidence artifacts + flake-aware gate`), ff-only sync.
6. Post-merge docs reconciliation + W7 DoD close (mark SHIPPED; note the next live nightly verifies end-to-end).

## Open Questions

- None blocking. (Tuning the 1-of-N rule or bumping to n=5 later is an EPIC-1 follow-up, not this ticket.)

## Self-Check

Goal user-visible ✅ · problem stated ✅ · context from code present ✅ · in/out scope non-conflicting ✅ · constraints actionable ✅ · assumptions visible ✅ · blocking questions none ✅ · acceptance criteria observable/testable ✅ · edge cases covered ✅ · validation plan concrete ✅ · execution order matches EVAL-1→EVAL-2 bet ✅
