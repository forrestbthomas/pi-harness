# STATUS — where the harness is right now

> One-screen snapshot. Regenerate at every cycle start per
> [`docs/roadmap-workflow.md`](docs/roadmap-workflow.md). Last updated: 2026-08-13.

## Now (committed — this cycle)

| Item | Status |
|---|---|
| **W4 Project-management layer** (roadmap workflow, this STATUS file) | Active ritual — the governance layer that keeps everything else true |

_No feature workstream is in flight. Next cycle starts with a prioritization
pass (RICE) to promote the next item into Now._

## Next (shaped — next 1–2 cycles)

| Item | Evidence |
|---|---|
| **BACKLOG #1 — per-tool-call timeout upstream** (pi-subagents #150) | Our v0.9.1 watchdog bounds runs but not mid-tool wedges; upstream is the right home; our #978/#979 merged upstream |
| **BACKLOG #2 — pin pi-subagents** | ✅ **SHIPPED 2026-08-13** (v0.9.2) — kept here until the cycle closes the row |
| **BACKLOG #3 — owner-only artifact perms** | ✅ **SHIPPED 2026-08-13** (v0.9.2) — kept here until the cycle closes the row |

## Later (raw — ideas worth exploring)

- Cloud eval backend · context-engine un-park (`pi-run context`) · model-catalog
  auto-refresh in CI · Windows support
- Idea inbox (unranked): doctor non-interactive-env guard · eval output gallery ·
  self-heal auto-issue · nightly event archiving · post-rebase hook ·
  `PI_SELF_HEAL` scorecard surfacing · harness self-evaluation

## Shipped recently (one spelling: CHANGELOG)

- **v0.9.2** (2026-08-13): W2 live-eval baseline, pin pi-subagents, owner-only
  perms · **v0.9.1** (2026-08-13): W1 self-healing, non-interactive env, exit 9 ·
  **v0.9.0**: eval hardening + live eval v2 · **v0.8.0**: project-understand,
  MCP, OTel, permission modes, hooks, 17 providers

## Open PRs / branches

- Open PRs: **none** · Branches: `main` only · Remotes: `github` only
