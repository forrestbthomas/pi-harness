# Decision — Observability-first resilience: OBS-1 + OBS-2 (2026-08-16)

**Status:** Decided (7-persona debate, `.pi/debate/resilience-observability/`)
→ adopt an observability-first slice under **EPIC-2**, promoted to an active
workstream (owner-approved).

## Decision

Add **OBS-1** and **OBS-2** as ranked, promoted EPIC-2 workstream items, and
sequence them ahead of the data-gated HEAL items:

- **OBS-1 — `pi-run sessions` live fleet view.** Reader over the existing
  `.pi/sessions/*.jsonl` v3 schema; lists active (recent mtime) + recent
  sessions; parseable output; `--recent` default view. CLI surface only.
  ~0.25 pw, stdlib-only.
- **OBS-2 — chat-path heal events.** Aggregate repeated connection/transport
  failures in interactive runs → one `connection-flap` event; **hermetic-
  tested aggregation boundary** (≥N failures/window → exactly 1 event; <N →
  0); **seam parity** (writes to existing `.pi/heal/events.jsonl` and surfaces
  in scorecard `selfHeal {nEvents, byKind}` — no parallel ledger); **canary
  acceptance** (test drives N failures → 1 `connection-flap` in `selfHeal`).
  ~0.3–0.5 pw, stdlib-only. Lands with the EVAL-17 pump.

## Why (the trigger)

The observability audit showed **asymmetric observability**: cost and
transcripts are rich, but the heal surface is gated to the non-interactive
watchdog path. Heal events on disk: **0**. Three concurrent chat sessions
across providers suffered repeated `Connection error.` episodes while the
harness measured `self-heal events: 0` — a **false statement about chat
mode**. The most failure-prone mode was the least-observed mode. The episode
is the evidence: the loop's "measure" step must cover all modes, or the
"skipped eval indistinguishable from passed" class reappears on the input
side.

## Why EPIC-2 and not a new epic

OBS-1/OBS-2 are the EPIC-2 enabler ("every hang/wedge class detected,
recovered, observable") — specifically the *observation* half. They also
**un-gate the deferred HEAL items**: HEAL-2's data requirement (≥1 week
non-zero wedge coverage) can finally be satisfied by chat-path data once OBS-2
exists, which in turn makes HEAL-1 (liveness) and HEAL-4 (stuck detection)
calibratable on evidence rather than guesses.

## Non-negotiables (all persona conditions adopted)

1. **CLI-only, no product surface** — no dashboard, metrics endpoint, or OTel
   (charter non-goal #4 preserved). Internal self-healing instrumentation, the
   same class as W6/W9.
2. **Seam parity** — OBS-2 writes through the existing `events.jsonl` +
   `selfHeal`; a parallel ledger would be a second observability system.
3. **Hermetic-tested aggregation boundary** — no magic numbers (the thing that
   killed HEAL-4's thresholds before).
4. **Counter ≠ score** — `selfHeal {nEvents}` going 0→nonzero is a signal to
   improve the healer, not a metric to inflate.
5. **HEAL-1/HEAL-4 stay deferred** until their recorded triggers fire (real
   wedge data), not "we have a counter now."
6. **README gets one row** for `pi-run sessions`; no feature paragraph.
7. **Promotion is a user decision** — owner approved active-workstream
   promotion on 2026-08-16; spec → scope-lock → TDD follow when work begins.

## References
- Debate: `.pi/debate/resilience-observability/` (round1/, round2/, synthesis.md).
- Audit inputs: `internal/cli/` watchdog/self-heal/escalation; `.pi/sessions/*.jsonl`;
  `.pi/heal/events.jsonl` (0 events at audit time); `.pi/cost-ledger.jsonl`.
- Epic: `EPICS.md` EPIC-2 (self-healing resilience).
- Prior decision: `docs/knowledge-base/decision/2026-08-16-reopen-cost1-task-routing.md`
  (same debate ritual, same session).

## Status of the slice
- [x] Debate + synthesis (`.pi/debate/resilience-observability/`)
- [x] `BACKLOG.md` OBS-1/OBS-2 rows added (ranked, promoted)
- [x] `EPICS.md` EPIC-2 sequence + table updated
- [x] `STATUS.md` EPIC-2 line updated
- [x] Decision node written (this file)
- [x] Owner-approved promotion to active EPIC-2 workstream
- [ ] Spec (when work begins): `pi-run sessions` command + OBS-2 event contract
- [ ] `scope-lock` → SCOPE.md → TDD → PR
- [ ] README one-row doc of `pi-run sessions`