# Scope Contract
**Task:** W11 — Observability-first resilience slice (OBS-1 `pi-run sessions` + OBS-2 chat-path heal events)
**Plan:** `docs/governance/specs-archive/2026-08-16-observability-first-resilience.md`
**Date:** 2026-08-16 · **Status:** ACTIVE

## In Scope
- **Files:**
  - `internal/cli/sessions.go` (new) — `runSessions` (fleet view + `--heal` mode)
  - `internal/cli/sessions_test.go` (new) — hermetic tests
  - `internal/cli/app.go` — dispatch `case "sessions"` + usage row
  - `README.md` — one row for `pi-run sessions`
  - `docs/reference.md` — command row
  - `CHANGELOG.md` — `[Unreleased]` entry
  - `ROADMAP.md`, `STATUS.md`, `BACKLOG.md` — W11/OBS rows status updates
  - `.pi/heal/events.jsonl` — runtime output (written by `--heal`; not committed)
- **Features (from plan):**
  - `pi-run sessions [--recent <dur>] [--active] [--json] [--heal] [--help]`
    — reader over existing `.pi/sessions/*.jsonl` v3 schema; live = recent mtime.
  - `--heal` mode: connection-class classifier on `stopReason:"error"`
    `errorMessage`; aggregation boundary ≥N/W per session → exactly 1
    `connection-flap` event; write through existing `events.jsonl` seam format.
  - Defaults N=3, W=10m; env-tunable `PI_HEAL_FLAP_THRESHOLD` /
    `PI_HEAL_FLAP_WINDOW`.
- **Boundaries:**
  - CLI surface only (no dashboard/metrics/OTel).
  - Backward-compatible: new command only; no existing flag/exit-code changes.
  - Unknown `sessions` flags → exit 2 usage error (house style).
  - Malformed/unreadable session lines skipped, never fatal.

## Out of Scope
- Teeing the chat TUI, Pi extension, live tail/daemon/watch loop.
- HEAL-1 (liveness heartbeat), HEAL-4 (stuck detection), HEAL-3 (auto-resume),
  HEAL-2 (watchdog tuning) — remain deferred; OBS-2 feeds their future data gate.
- Any change to `score_run.py`, the scorecard schema, the watchdog, or the chat
  spawn path.
- Dashboard / observability product surface (charter non-goal #4).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| — | — | — | — | — | — |

# Follow-up Tasks
- [ ] Tune aggregation defaults (N/W) from real OBS-2 data after ≥1 week of
      wedge coverage (data-gate discipline) — not part of this contract.
- [ ] Re-evaluate HEAL-1/HEAL-4 re-promotion when OBS-2 data accumulates —
      recorded trigger, not "we have a counter now."
