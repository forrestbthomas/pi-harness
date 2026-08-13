# Roadmap Workflow — how the roadmap stays true

This is the high-level workflow that keeps the roadmap coherent while work fans
out across branches, subagents, and PRs. It exists because docs drift when
nothing forces them to track reality (W1 sat at "In design" after it shipped;
tags diverged from main twice). Follow it every session.

## Sources of truth (who owns what)

| File | Owns | Does NOT own |
|---|---|---|
| `ROADMAP.md` | Active workstreams + DoD, parked items, recurring rituals | Shipped history, per-item detail |
| `BACKLOG.md` | Ranked RICE queue, deferred, idea inbox | Workstream status |
| `STATUS.md` | One-screen Now/Next/Later snapshot ("where are we?") | The detailed plans |
| `SCOPE.md` | scope-lock boundary contract for the current workstream (superseded contracts stay as history) | Roadmap decisions |
| `CHANGELOG.md` | **Shipped has exactly one spelling**: dated `## [x.y.z]` entries | Future plans |

## Cycle cadence (weekly, at cycle start — not ad hoc)

1. **Reconcile statuses with reality** — for every merged PR since last cycle,
   check its workstream row: statuses must match what actually shipped (this is
   the W1-class drift guard). Fix before adding anything new.
2. **Close DoD rows** — workstream/backlog items whose DoD boxes are closed move
   to `CHANGELOG.md` (dated entry) and are marked SHIPPED in `ROADMAP.md`.
3. **Prune the idea inbox** — kill items with no evidence after 3 months
   (roadmap-planning: Later is pruned, not accumulated).
4. **Re-rank RICE** in `BACKLOG.md` (from `productskills/feature-prioritization`).
5. **Regenerate `STATUS.md`** from the reconciled ROADMAP/BACKLOG/CHANGELOG so
   the one-screen snapshot is never stale.

Limit "Now" to 1–3 items (roadmap-planning). If more than three workstreams are
active, stop and cut scope before continuing.

## Traceability (no orphan work)

- Every PR body must cite the roadmap item it serves: `ROADMAP W# — <name>` or
  `BACKLOG #N — <name>`. If it serves neither, say so and justify (the scope
  rule in `BACKLOG.md` applies: out-of-scope work needs a backlog entry first).
- Work happens on **short-lived branches off main**; each branch maps to exactly
  one workstream/backlog item. No long-lived parallel branches (30 stale
  branches were deleted 2026-08-13 — keep it that way).
- `SCOPE.md` is the boundary contract for the active workstream; deviations are
  flagged, never silently absorbed.

## Release rule

Land every release commit (incl. CHANGELOG notes) on main via PR **first**, then
tag the **fetched main tip** — `scripts/tag-release.sh vX.Y.Z`. Never tag a
local commit that hasn't merged (squash merges rewrite hashes; see
`CONTRIBUTING.md` "Releases" and `release.yml`'s ancestry guard).

## Session entry sequence

At the start of any session: read `STATUS.md` (one-screen) → `ROADMAP.md`
(workstreams) → `BACKLOG.md` (queue) → follow the rituals above. The context
engine is the memory layer under this; the docs are the source of truth.
