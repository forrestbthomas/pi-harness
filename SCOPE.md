# Scope Contract
**Task:** W9 — EVAL-4: self-heal events in provider scorecard | **Plan:** `docs/superpowers/specs/2026-08-14-self-heal-in-provider-scorecard.md` | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged (W9 shipped #91); **W10 (EVAL-5 dataset growth) is ACTIVE** per the approved 2026-08-14 bet — see `docs/superpowers/specs/2026-08-14-dataset-growth-to-50.md`

> Supersedes the W8 dataset versioning contract (CLOSED 2026-08-14; record preserved in git history). W10 (EVAL-5 dataset growth) is a separate contract drafted for the same bet and lands after this one closes.

## In Scope
- **Files — harness:**
  - `internal/cli/scorecard.go` — `scorecardSelfHeal { NEvents, ByKind }`; `SelfHeal` on `scorecard` (`omitempty`); `readSelfHealEvents(root)` best-effort reader; populate in `runScorecard`
  - `.github/workflows/provider-scorecard.yml` — `PI_SELF_HEAL: '1'` in job env
  - `internal/cli/scorecard_test.go` — reader + scorecard-JSON tests (golden fixture re-gen if schema grows)
  - `CHANGELOG.md`, `ROADMAP.md` (W9 row), `STATUS.md`, `BACKLOG.md` (EVAL-4 → active), `SCOPE.md`, spec
- **Features:**
  1. Provider scorecard JSON surfaces `selfHeal { nEvents, byKind }` (informational; 0/omitted when no events).
  2. Provider benchmark workflow records `.pi/heal/events.jsonl` (`PI_SELF_HEAL=1`).
  3. Hermetic Go tests.
- **Boundaries:**
  - Informational only; no gating on self-heal counts.
  - No change to ci-benchmark grading/baseline/budget logic.
  - Backward-compatible JSON (`omitempty`).

## Out of Scope
- Self-heal gating/thresholds.
- Python `score_run.py` selfHeal (shipped W6).
- EVAL-5 dataset growth (W10, separate contract).
- Any change to benchmark grading, baselines, or budget.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] — (none yet)
