# Scope Contract
**Task:** Apply the maturity-scan change-set to the PM layer (EPICS.md + BACKLOG.md + STATUS.md) | **Plan:** `.pi/debate/maturity-scan/synthesis.md` (six-persona convergence, 2026-08-14) | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged

## In Scope
- **Files:**
  - `EPICS.md` — apply the converged epic shape: EPIC-1 trimmed (outcome "research-grade measurement", appetite in person-weeks, +EVAL-13..16, −EVAL-10, EVAL-7 parked, EVAL-6/12 updated); EPIC-2 (HEAL-1 demoted, HEAL-2 data-gated); EPIC-3 (COST-1 within-provider); EPIC-4 parked → PORT-0; EPIC-5 dissolved (DX-1/DX-2 → idea inbox); **EPIC-6 (new) "Repo maturity"** added.
  - `BACKLOG.md` — rebuild the ranked queue: remove shipped EVAL-1..5 rows, remove EVAL-10, park EVAL-7/EPIC-4 items, demote HEAL-1 to a deferred section, add OSS-1/GOV-1/GOV-2/OSS-2/OWN-1/TAX-1/TAX-2/PORT-0/SECURITY-line + EVAL-13/14/15/16; update EVAL-6 (RICE 1.4), EVAL-12 (DoD), EVAL-5 (hold at 50), HEAL-2, COST-1; expand idea inbox (EVAL-10, DX-1, DX-2, HEAL-1 design question).
  - `STATUS.md` — regenerate the one-screen snapshot to match the new epic/queue shape (EPIC-6 next; EPIC-4 parked; EPIC-5 dissolved).
  - `SCOPE.md` — this contract, closed on completion.
- **Features:**
  1. Five epic rows: EPIC-1, 2, 3, 4 (parked: PORT-0 line), 6 (new); EPIC-5 dissolved; DX-1/DX-2/HEAL-1/EVAL-10/EVAL-7/PORT-1/PORT-2 in idea-inbox or deferred with recorded triggers.
  2. EPIC-1 honest core ≈ 2.5–3 pw (EVAL-12 → EVAL-6 → EVAL-8 → EVAL-13/14/15/16); EVAL-5 held at 50.
  3. EPIC-6 "Repo maturity" closes (DoD: OSS-1 + GOV-1 + GOV-2 + OSS-2 + OWN-1 + TAX-1 + TAX-2 + SECURITY line shipped/parked with recorded decisions; backlog ≤ ~12 active rows).
- **Boundaries:**
  - PM docs only — no product code, no workflow changes, no eval/dataset changes.
  - RICE values are the debate's converged rough scores; documented as such.
  - GOV-2 ships before GOV-1 in execution order (can't guard a location that doesn't exist); RICE order governs the ranked queue.

## Out of Scope
- Executing any individual backlog item (each lands as a normal workstream later).
- The OSS-1 module-path/identity fix itself (that's a backlog item to execute next, not part of this docs restructure).
- Spec shelf-ware physical archive (that's GOV-2's execution, not this contract).
- Any change to code, CI, or the eval suite.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] After merge: execute OSS-1 (canonical install/identity) as the highest-value next workstream.
- [ ] After merge: bet the next ROADMAP workstream (EVAL-12 re-baseline pending tonight's nightly; then EPIC-6 GOV-2 relocation).
