# STATUS — where the harness is right now

> One-screen snapshot. Regenerate at every cycle start per
> [`docs/roadmap-workflow.md`](docs/roadmap-workflow.md). Last updated: 2026-08-13.

## Now (committed — this cycle)

| Item | Status |
|---|---|
| **W5 — Upstream per-tool-call timeout** (upstream #1076/#1077; run-level trace #978/#979) | Part A (pin 0.48.0) landed; **Part B submitted upstream** — issue #1076, PR #1077 open, awaiting maintainer review; Part C (observe via `PI_SELF_HEAL`) starts after a release carries `toolTimeoutMs` |
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

- Open PRs: **none** (harness) · Upstream: pi-subagents **#1077** (per-tool-call timeout) open, issue **#1076** · Branches: `main` only · Remotes: `github` only
