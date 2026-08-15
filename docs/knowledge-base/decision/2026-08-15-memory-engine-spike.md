---
id: "kb-20260815-memory-engine-spike"
type: "decision"
status: "active"
scope: "project"
components:
  - "pi-run"
  - "pi-config"
tools: []
trigger_tags:
  - "type:decision"
  - "memory"
  - "scope"
related_files:
  - "README.md"
  - "docs/benchmark-seam.md"
  - "BACKLOG.md"
created_at: "2026-08-15T16:30:00+00:00"
updated_at: "2026-08-15T16:30:00+00:00"
---

# Memory-engine hook — spike decision (MEM-1)

## Summary

A seven-persona debate (`.pi/debate/memory-engine/`) unanimously concluded:
**the harness ships no memory engine and no pluggable memory abstraction.**
Default = none-in-harness. Memory is user-chosen at the pi config layer.
The one binding invariant: **eval agent runs are memory-free** (a memory
package entering the eval spawn path is a re-baseline trigger, not a silent
change).

## Decision

1. **No harness memory abstraction.** No `pi-run memory` command, no driver
   registry, no config knob, no standalone interface spec. A future
   `PI_MEMORY_*`-shaped env-gated knob is a *sketch only* (below), consumer-
   gated, and inert in eval mode.
2. **Default = none-in-harness.** The harness runs memory-free (verified
   after removing `@loreai/pi`, #136: config-check + print smoke green).
   "Memory MD files" is not a default engine — the docs are the drift-guarded
   **source of truth**. context-engine is an optional external session layer
   (MCP, a separate project), never a harness dependency.
3. **User choice lives at the pi config layer.** Any memory engine is loaded
   by the user via `.pi/settings.json` `packages` (the same surface
   `@loreai/pi` was removed from). The harness is already memory-agnostic by
   construction.
4. **Eval runs are memory-free (binding).** The seam's measurement contract
   assumes memory-free agent runs; adding a stateful/memory package to the
   eval spawn path is a **re-baseline trigger**, not a silent change
   (recorded in `docs/benchmark-seam.md`). Rationale: an agent carrying
   memory into an eval case could answer from memory instead of from the
   task, silently corrupting the honest pass-rate the north star sells (the
   `coding-055` class).

## Why (debate evidence)

- The choice surface already exists (`.pi/settings.json` packages) — a hook
  would be an abstraction over a knob one layer down.
- Zero consumers → the COST-1 deferral pattern (2 pw, zero consumers →
  deferred; re-open on a consumer signal).
- The `@loreai/pi` lesson cuts against building: the failure was a
  *default-loaded engine* (vulnerability chain, branding liability), not the
  absence of an abstraction. Memory must be additive, never default-bundled.
- `docs/roadmap-workflow.md` already names the memory layer: "the context
  engine is the memory layer under this; the docs are the source of truth."

## Future sketch (not implemented; consumer-gated)

If a consumer asks for a harness-level memory knob, the starting shape is a
`PI_MEMORY_*` env-gated provider selection passed to spawned agents,
**inert in eval mode** (the same discipline as `--offline` /
`--cost-mode live-eval`), with one reference implementation before
"pluggable" can be claimed. This is a note, not a spec; writing a versioned
contract for a consumerless interface is load-bearing tax (this repo's own
drift-guard discipline).

## Re-open triggers (data-gated, never calendar-gated)

1. A **consumer signal** — a user asking how to plug a memory engine into
   pi-run (the COST-1 unpark pattern).
2. A **measured contamination** — an eval pass-rate shift caused by a loaded
   memory package; then the seam line becomes a mechanical guard
   (config-check / hermetic test).
3. **EVAL-17 slice-2 data** showing memory-sensitive behavior is a measurable
   eval category.
4. DX-2 unpark (`pi-run context` session-stats) creating a genuine
   harness-internal memory need.

## Follow-ups

- [x] README memory statement (one line, front-door honesty).
- [x] Seam-contract line in `docs/benchmark-seam.md` (memory-free eval runs).
- [x] MEM-1 backlog row closed with this decision.
