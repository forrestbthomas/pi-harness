# Scope Contract
**Task:** EVAL-13 — Cost-variance tolerance in the nightly baseline gate (flake-aware cost, mirrors EVAL-2) | **Plan:** BACKLOG EVAL-13 (EPIC-1, RICE ~1.0, 0.5 pw); ROADMAP §Release Milestones v0.11.0 "The gate that can't lie" | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged

## In Scope
- **Files:**
  - `eval/scripts/score_run.py` — cost gate becomes flake-aware:
    - `aggregate_case`: expose per-run costs (`costsPerRun` list) so the comparator can count over-threshold runs.
    - `compare_case`: `costFlake` = exactly 1 run over 2× baseline (median under); `costRegressed` = median > 2× baseline **OR** ≥2 runs over 2× baseline. Mirrors EVAL-2's pass-rate logic (`n_failed == 1` flake, `>= 2` regression).
    - `build_summary` + `build_compact_summary`: add `costFlakes` (sorted list), parallel to `flakes`.
    - `render_markdown`: report cost-flakes line; status column distinguishes `costFlake` from `costRegressed`.
    - `evaluate_gate`: cost-regression failure message unchanged (fires only on `costRegressed`, which is now median-or-≥2).
  - `eval/tests/test_score_run.py` — hermetic tests:
    - single over-threshold run (n=5) → `costFlake` True, `costRegressed` False, gate passes.
    - ≥2 over-threshold runs → `costRegressed` True, gate fails.
    - median over 2× (even if 1 run) → `costRegressed` True (genuine median shift).
    - exactly-2× boundary stays False (existing test preserved).
    - summary JSON carries `costFlakes`.
- **Features:**
  1. A single-run cost spike is a reported flake, never a gate failure (kills the 2026-08-14 coding-010 false-cost-alarm class).
  2. A genuine cost regression (median shift OR recurring ≥2 over-runs) still fails.
  3. Scorecard JSON + markdown surface `costFlakes` for observability.
- **Boundaries:**
  - No change to pass-rate gate math (EVAL-2), budget cap, or incomplete-run handling.
  - No change to the Go scorecard (`ci-benchmark`) — EVAL-13 is the Python nightly gate only (EVAL-14 covers benchmark parity separately).
  - No CLI/flag changes (score_run args unchanged).
  - Baseline JSON schema unchanged.

## Out of Scope
- EVAL-14 (benchmark provenance parity) — separate item.
- EVAL-8 (judge stabilization), EVAL-15 (split-seam), EVAL-6 (agentic cases) — separate items in v0.11.0.
- Changing the 2× cost threshold or adding cost thresholds to the Go side.
- `budget-usd` behavior (already a hard cap, unchanged).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] After merge: confirm the next live nightly (with EVAL-13) no longer false-fails on a single-run cost spike.
