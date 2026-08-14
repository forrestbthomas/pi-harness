# Scope Contract
**Task:** W6 — Surface `PI_SELF_HEAL` events in scorecard + enable in CI | **Plan:** `docs/superpowers/specs/2026-08-13-surface-self-heal-events-scorecard.md` | **Date:** 2026-08-13 | **Status:** CLOSED — 0 changes logged (shipped #83, verified 2026-08-14)

> Supersedes the W5 per-tool-call timeout contract (record preserved in git history; W5 Part C remains tracked in ROADMAP and resumes when an upstream release carries `toolTimeoutMs`).

## In Scope
- **Files — harness:**
  - `.github/workflows/nightly-live-eval.yml` — add `PI_SELF_HEAL: '1'` to the `live` job env
  - `eval/scripts/score_run.py` — self-heal event parsing (`--heal-events` override, default `<repo>/.pi/heal/events.jsonl`), `selfHeal { nEvents, byKind }` block in full + compact summary, one-line `GITHUB_STEP_SUMMARY` addition; informational only (no gate change)
  - `eval/tests/test_score_run.py` — hermetic tests: missing file → 0 events; count by kind; malformed lines skipped; override honored; summary + compact carry the block
  - `CHANGELOG.md` — `[Unreleased]` entry
  - `ROADMAP.md`, `STATUS.md`, `BACKLOG.md`, `SCOPE.md` — W6 promotion + reconciliation
  - `docs/superpowers/specs/2026-08-13-surface-self-heal-events-scorecard.md` — this plan
- **Features:**
  1. CI enables `PI_SELF_HEAL=1` for the nightly live job.
  2. Scorecard reports self-heal event counts by kind without affecting the gate/baseline.
  3. Hermetic unit coverage in the deterministic CI job.
- **Boundaries:**
  - Informational only; no gate, threshold, or baseline change.
  - Events read only from the repo-root `.pi/heal/events.jsonl` (the live suite cwd contract) unless `--heal-events` overrides.
  - No watchdog/self-heal behavior changes and no new event kinds.

## Out of Scope
- Auto-open GitHub issue on N self-heals (BACKLOG #5).
- Nightly archive upload of events (BACKLOG #3).
- Model-catalog auto-refresh triggers (BACKLOG #4).
- Any change to the self-healing layer itself (W1 shipped).
- Reading events from other locations or other processes' cwds.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] — (none yet)
