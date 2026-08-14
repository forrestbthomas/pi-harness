# Scope Contract
**Task:** EVAL-15 — Split-seam verification (seam contract doc + hermetic dry-run) | **Plan:** BACKLOG EVAL-15 (EPIC-1, RICE ~1.5, 0.5 pw); ROADMAP §Release Milestones v0.11.0 | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged

## In Scope
- **Files:**
  - `docs/benchmark-seam.md` (NEW) — the versioned contract for the harness↔eval seam:
    - `eval/datasets/tasks.json` (schema + `datasetVersion` + lint rule)
    - `eval/scripts/score_run.py` CLI + exit codes + `SCHEMA_VERSION`
    - scorecard JSON shape (live nightly summary + Go ci-benchmark scorecard)
    - `eval/datasets/coding_samples.jsonl` + graders/ + references/ + eval/benchmarks/ layout
    - `eval/grader.py` / `eval/secret_backend.py` boundaries
    - known coupling today (recorded honestly: `score_run.repo_root()`, `.pi/heal` reads, 11 tests spawning `pi-run`) and the **split trigger** (EPIC-1 DoD AND external consumer)
  - `eval/tests/test_benchmark_seam.py` (NEW) — hermetic dry-run:
    1. **Contract pins** (hard pass/fail): tasks.json exists + has `datasetVersion`; score_run has `SCHEMA_VERSION`; coding_samples.jsonl has 50 lines; graders/references counts match the manifest; benchmark dirs present.
    2. **Self-containment dry-run** (records gaps, never fails the suite): scan `eval/` for harness-root coupling — `repo_root()`, `.pi/` reads, `cmd/pi-run`, `internal/cli`, subprocess `pi-run` spawns — and assert the *report* is written to `eval/live-results/seam-report.json` with a `couplings` list. The dry-run documents the gap: it does not pretend the seam is clean today.
  - `CHANGELOG.md`, `SCOPE.md`, decision record — updates.
- **Features:**
  1. The seam is pinned in one doc (a stranger or future splitter reads it to know exactly what leaves and what stays).
  2. A hermetic dry-run measures the seam's self-containment and records the coupling inventory — so the charter's "keeps this split cheap either way" is tested, not assumed.
  3. The dry-run's coupling list is the actionable input to a future pi-bench split.
- **Boundaries:**
  - No code changes to score_run/grader/conftest (this item *measures* coupling, it does not refactor it).
  - Dry-run never fails CI on coupling found — it writes the report (a failing-on-coupling test would block every PR today, which is not the point; the *split trigger* is consumer-gated).
  - No new Go changes; no workflow changes.

## Out of Scope
- Actually splitting pi-bench (triggered future — EPIC-1 DoD AND external consumer).
- Refactoring `score_run.repo_root()` / de-coupling (a future split workstream, gated by the dry-run's findings).
- GOV-2 (relocation), GOV-1 (drift guard) — separate EPIC-6 items.
- EVAL-6/EVAL-8 (separate v0.11.0 items).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] After merge: the seam-report.json coupling inventory is the input to the pi-bench split decision (EPIC-1 DoD AND external consumer trigger).
