# Scope Contract
**Task:** W7 — EVAL-1 + EVAL-2: evidence artifacts on every outcome + flake-aware gate | **Plan:** `docs/superpowers/specs/2026-08-14-flake-aware-gate-and-evidence-artifacts.md` | **Date:** 2026-08-14 | **Status:** ACTIVE (approved 2026-08-14)

> Supersedes the W6 scorecard self-heal contract (CLOSED 2026-08-14; record preserved in git history). W5 Part C remains tracked in ROADMAP (upstream release gate).

## In Scope
- **Files — harness:**
  - `.github/workflows/nightly-live-eval.yml` — `if: always()` on "Upload live results"; add `.pi/heal/events.jsonl` to artifact path; `EVAL_RUNS_PER_CASE` 3→5; `--runs 5`
  - `.github/workflows/provider-scorecard.yml` — `if: always()` on "Upload scorecard"
  - `eval/scripts/score_run.py` — `nFailed` in `aggregate_case`; `flake` in `compare_case` (`regressed` only when `nFailed >= 2`); `flakes` in summary + compact + markdown; stderr flake line
  - `eval/tests/test_score_run.py` — nFailed, flake-vs-regression, flakes in summary/compact/markdown, cost-regression-unchanged
  - `eval/tests/test_docs_drift.py` — invariant: both workflows' upload steps carry `if: always()`
  - `CHANGELOG.md`, `ROADMAP.md` (W7 row), `STATUS.md`, `BACKLOG.md` (EVAL-1/2 → active), `SCOPE.md`, spec
- **Features:**
  1. EVAL-1: eval evidence (results + heal events) is uploaded on every gate outcome, nightly and weekly.
  2. EVAL-2: single-run flake warns (never fails); ≥2 failed runs = regression; cost regression unchanged; flakes reported in the scorecard; n=5 runs.
  3. Hermetic tests for both + the workflow invariant.
- **Boundaries:**
  - Flake rule fixed at 1-of-N = flake, ≥2 = regression; no configurable threshold; no "fail after N flakes".
  - No auto-quarantine (EVAL-9), no re-baseline in this PR, no change to `ci-benchmark` gate logic.
  - Upload events.jsonl only — not escalation packets (goal/ledger sensitivity).

## Out of Scope
- Flake thresholds / quarantine registry / auto-quarantine (EVAL-9).
- Baseline re-baseline (live-run act; runsPerCase informational).
- `ci-benchmark` scorecard gate semantics (Go; only the upload `if:` changes).
- Uploading `.pi/heal/*-report.json` escalation packets.
- Any change to cost-regression or incomplete-run gate behavior.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] — (none yet)
