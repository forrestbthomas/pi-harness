# Decision — COST-1 re-opened: static task→tier routing (2026-08-16)

**Status:** Decided (7-persona debate, `.pi/debate/task-routing/`) → re-open
COST-1, re-scoped to **static task→tier routing**, as a ranked backlog item
under EPIC-3. **Not** promoted to an active ROADMAP workstream (owner to
decide promotion at a later cycle).

## Decision

Re-open **COST-1** with a corrected scope and effort, via a recorded
**owner override** of the 2026-08-14 deferral:

- **Mechanism (unchanged from COST-1's DoD):** a data-driven `taskTiers` table +
  a `--task` flag resolved through `resolveModelTier`, **within the user's
  explicit provider, never cross-provider**, fail-closed (unavailable tier =
  error, never fallback).
- **Scope re-correction:** the *cost-aware* half (auto-route from the cost
  ledger / budget / per-request cost) **stays deferred** — that was what priced
  COST-1 at 2 pw. Static task→tier is **not** cost-aware. Effort corrected
  **2 pw → ~0.75 pw**.
- **Status:** re-opened (was DEFERRED 2026-08-14), ranked under **EPIC-3**.
- **Hard dependency:** **COST-2** (model-catalog auto-refresh in CI) must land
  first — a task table pointing at a stale model is the 2026-08-12 failure class.

## Why (the trigger — the honest part)

The recorded re-open trigger for COST-1 said *"consumer signal (an external
user asking for cost routing)."* That did **not** literally fire — no external
user asked. The re-open is instead:

1. **The author, as the charter's named "only customer," invoking the owner
   override** — the harness's stated job is "know whether *your* agent setup is
   improving," and a routing knob that makes the author's own nightly/eval cost
   follow its workload is the north star applied to the loop's *input* side.
2. **An explicit, dated override of the trigger language**, not a silent
   reinterpretation of "external" — the author-as-consumer framing is *not*
   claimed as the authority; the **owner override** is. Recorded so a stranger
   reading the repo sees a deferred item re-opened on purpose, not drift.

## The debate (`.pi/debate/task-routing/`)

Round-1 7 personas, Round-2 rebuttals, unanimous convergence. Key conditions
the DoD must carry (all adopted):

1. **No shadow row** — one backlog row (COST-1), re-scoped; no new identity.
2. **Gated on COST-2** (catalog refresh) — sequence COST-2 → COST-1.
3. **Eval-driven (Dogfooder clause):** `--task`/tier recorded in scorecard
   provenance (no silent cross-tier contamination) + a graded **cheap-vs-
   balanced** case family asserting cheap is honestly ≤ balanced. *"The eval is
   what makes routing measurable."*
4. **Fail-closed AND fail-loud (Skeptic clause):** a `--task` mapped to a tier
   the provider lacks errors listing available tiers (existing
   `resolveModelTier` path) + hermetic test; **no-`--task` behavior is
   byte-identical to today** (strictly additive).
5. **Adoption data gate (Skeptic):** task usage observable via provenance /
   cost ledger; park if unused after one cycle.
6. **Cost-aware half stays deferred** — not re-litigated without new evidence.

**Out of scope (holds the deferral):** the cost-aware routing half; any
cross-provider routing (charter NOT-6); any README/feature promotion before a
stranger asks (OSS clause); **`--task` tier routing for `eval` / `ci-benchmark`
runs** — benchmark runs keep `--provider`/`--model` only, so task routing can
never change which model grades a case or which scorecard is compared (the
original COST-1 design's eval/benchmark exclusion, carried forward:
`docs/governance/specs-archive/2026-08-12-cost-aware-routing-design.md`).

## References
- Prior deferral: `docs/knowledge-base/2026-08-14-persona-debate-scope.md`
  (whats-next consensus) + `BACKLOG.md` prior row.
- COST-1 design: `docs/governance/specs-archive/2026-08-12-cost-aware-routing-design.md`.
- Debate: `.pi/debate/task-routing/` (round1/, round2/, synthesis.md).
- Epic: `EPICS.md` EPIC-3 (Cost intelligence & routing).

## Status of the re-open
- [x] Decision node written (this file)
- [x] `BACKLOG.md` COST-1 row updated (re-opened, effort/RICE/DoD)
- [x] `EPICS.md` EPIC-3 status/order + deferred-row updated
- [ ] Owner decides promotion to active ROADMAP workstream (pending, not assumed)
- [ ] COST-2 lands first (blocking prerequisite)
- [ ] Spec written (taskTiers schema, `--task` semantics, eval cases) — when work begins
- [ ] `scope-lock` → SCOPE.md → TDD → PR