# STATUS — where the harness is right now

> One-screen snapshot. Regenerate at every cycle start per
> [`docs/roadmap-workflow.md`](docs/roadmap-workflow.md). Last updated: 2026-08-13.

## Now (committed — this cycle)

| Item | Status |
|---|---|
| **W5 — Upstream per-tool-call timeout** (pi-subagents #150) | In research (Phase 1) — then spec → scope-lock → implement → validate |
| **W4 Project-management layer** (roadmap workflow, STATUS, RICE cycle) | Active ritual — the governance layer that keeps everything else true |

## Next (shaped — next 1–2 cycles)

| Item | RICE | Tag |
|---|---|---|
| **BACKLOG #1 — surface PI_SELF_HEAL events in scorecard + enable in CI** | 1.40 | Enabler |
| **BACKLOG #2 — post-rebase hook** (auto-continue wedged rebase) | 1.20 | Enabler |
| **BACKLOG #3 — nightly archives watchdog/git-state events** | 1.00 | Enabler |

## Later (raw — ideas worth exploring)

- Model-catalog auto-refresh in CI · auto-open issue on N self-heals (needs #3) ·
  harness self-evaluation · cloud eval backend · eval output gallery ·
  context-engine un-park · Windows support

## Shipped recently (one spelling: CHANGELOG)

- **v0.9.2** (2026-08-13): W2 live-eval baseline, pin pi-subagents, owner-only
  perms · **v0.9.1** (2026-08-13): W1 self-healing, non-interactive env, exit 9 ·
  **v0.9.0**: eval hardening + live eval v2 · **v0.8.0**: project-understand,
  MCP, OTel, permission modes, hooks, 17 providers

## Open PRs / branches

- Open PRs: **none** · Branches: `main` only · Remotes: `github` only
