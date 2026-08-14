# STATUS — where the harness is right now

> One-screen snapshot. Regenerate at every cycle start per
> [`docs/roadmap-workflow.md`](docs/roadmap-workflow.md). Last updated: 2026-08-13.

## Now (committed — this cycle)

| Item | Status |
|---|---|
| **W5 — Upstream per-tool-call timeout** (upstream #1076/#1077; run-level trace #978/#979) | Part A (pin 0.48.0) landed; **Part B MERGED upstream** (`a660ea3`, 2026-08-13); Part C (observe via `PI_SELF_HEAL`) pending a release carrying `toolTimeoutMs` |
| **W6 — Scorecard self-heal observability** (surface `PI_SELF_HEAL` events + enable in CI) | **SHIPPED — 2026-08-14** — merged #83; manual nightly verified `self-heal events: 0` on a healthy run (gate failed on unrelated coding-005/010 variance) |
| **W7 — Flake-aware gate + evidence artifacts** (EVAL-1/EVAL-2) | **SHIPPED — 2026-08-14** — merged #87; next live nightly confirms end-to-end (upload on failure + flake warn-not-fail) |
| **W4 Project-management layer** (roadmap workflow, STATUS, RICE cycle) | Active ritual — the governance layer that keeps everything else true |

## Next (shaped — next 1–2 cycles)

| Item | RICE | Tag |
|---|---|---|
| **EPIC-1 — Eval suite → research-grade benchmark** (EVAL-1..11) | top EVAL-5 1.60 | EPIC-1 |
| **EPIC-2 — Self-healing resilience** (HEAL-1..5; heartbeat user-deferred) | top HEAL-5 1.20 | EPIC-2 |
| **EPIC-3 — Cost intelligence & routing** (COST-1..2) | top COST-1 0.80 | EPIC-3 |
| **EPIC-4 — Portability & distribution** (PORT-1..2; research-first) | top PORT-2 0.20 | EPIC-4 |
| **EPIC-5 — Insight & DX** (DX-1..2) | top DX-1 0.25 | EPIC-5 |

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

- Open PRs: **none** (harness) · Upstream: pi-subagents **#1077 merged** (`a660ea3`) as per-tool-call timeout, issue **#1076** · Branches: `main` only · Remotes: `github` only
