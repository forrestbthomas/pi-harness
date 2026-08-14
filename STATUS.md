# STATUS — where the harness is right now

> One-screen snapshot. Regenerate at every cycle start per
> [`docs/roadmap-workflow.md`](docs/roadmap-workflow.md). Last updated: 2026-08-14.

## Now (committed — this cycle)

| Item | Status |
|---|---|
| **Charter + project identity** | **SHIPPED — 2026-08-14** — `CHARTER.md` (boundary contract: one product, the harness; "we do not" non-goals); north star adjusted; README single-product; AGENTS/.pi/SYSTEM carry scope for agents (#98/#99) |
| **Cut list** (MCP server, OTel, pdf2txt) | **SHIPPED — 2026-08-14** — −1790 lines (#100); Homebrew/release machinery kept (charter wins); spec shelf-ware deferred |
| **W5 — Upstream per-tool-call timeout** (upstream #1076/#1077) | Part A (pin 0.48.0) landed; **Part B MERGED upstream** (`a660ea3`, 2026-08-13); Part C (observe via `PI_SELF_HEAL`) pending a release carrying `toolTimeoutMs` |
| **W6 — Scorecard self-heal observability** | **SHIPPED — 2026-08-14** — #83; nightly verified `self-heal events: 0` on a healthy run |
| **W7 — Flake-aware gate + evidence artifacts** (EVAL-1/EVAL-2) | **SHIPPED — 2026-08-14** — #87; end-to-end confirmed by next live nightly |
| **W8 — Dataset versioning + provenance** (EVAL-3) | **SHIPPED — 2026-08-14** — #89; guarded `datasetVersion` + `provenance` in scorecard |
| **W9 — Self-heal events in provider scorecard** (EVAL-4) | **SHIPPED — 2026-08-14** — #91 |
| **W10 — Dataset growth 20 → 50** (EVAL-5) | **SHIPPED — 2026-08-14** — 50 live cases + 3 edit-based benchmark tasks; EVAL-12 re-baseline is the follow-up (tonight) |
| **Live-suite timeout sizing** (EVAL-12 enabler) | **SHIPPED — 2026-08-14** — #97: per-test 120→600s, job 30→60m so the 50-case nightly can finish |
| **W4 Project-management layer** | Active ritual — governance layer; now with EPIC-6 (repo maturity) owning governance/community/debt work |

## Next (shaped — next 1–2 cycles)

| Item | RICE | Tag |
|---|---|---|
| **v0.10.0 — Identity, boundary, truth** (ship NOW: main batch #98–#102 + REL-1 version stamp + REL-2 ledger repair + REL-5 baseline provenance + BREAKING banner; EVAL-12 rides as data release) | — | Release |
| **EPIC-1 — Eval suite → research-grade measurement** (EVAL-6, 12, 13–16, 8, 9, 11) | top EVAL-15 1.50 / EVAL-6 1.40 | EPIC-1 |
| **EPIC-6 — Repo maturity** (OSS-1, GOV-1, GOV-2, OSS-2, OWN-1, TAX-1/2, REL-1..5, SECURITY, PORT-0) | top OSS-1 ~18 | EPIC-6 |
| **EPIC-2 — Self-healing resilience** (HEAL-5, HEAL-2 data-gated, HEAL-3; HEAL-1 demoted) | top HEAL-5 1.20 | EPIC-2 |
| **EPIC-3 — Cost intelligence & routing** (COST-2, COST-1 within-provider) | top COST-1 0.80 | EPIC-3 |
| **EPIC-4 — Portability & distribution** | **PARKED** — PORT-0 quarterly re-confirm only (charter non-goal until a consumer asks) | EPIC-4 |
| ~~EPIC-5 — Insight & DX~~ | **DISSOLVED** — DX-1/DX-2 → idea inbox | — |
| **v1.0.0 — The contract release** (gates, not dates: EPIC-1+6 DoDs, ≥14 green nightlies, EVAL-16 enforced, install-path CI-proven, consumer OR recorded earned-bar decision) | — | Release |

## Later (raw — ideas worth exploring)

- Watchdog liveness heartbeat (HEAL-1; re-score from `PI_SELF_HEAL` data) ·
  eval output gallery (EVAL-10; parked) · sandbox live runs (EVAL-7; parked) ·
  dataset growth 50 → 100 (capped until consumer/regression-catch signal) ·
  cloud eval backend · context-engine un-park (DX-2) · Windows support
  (PORT-1; charter non-goal) · model-catalog auto-refresh in CI

## Shipped recently (one spelling: CHANGELOG)

- **v0.9.2** (2026-08-13): W2 live-eval baseline, pin pi-subagents, owner-only
  perms · **v0.9.1** (2026-08-13): W1 self-healing, non-interactive env, exit 9 ·
  **v0.9.0**: eval hardening + live eval v2 · **v0.8.0**: project-understand,
  MCP, OTel, permission modes, hooks, 17 providers · post-v0.9.2 (main):
  charter (#98/#99), cut list (#100), live-suite timeout fix (#97)

## Open PRs / branches

- Open PRs: **none** (harness) · Upstream: pi-subagents **#1077 merged** (`a660ea3`) as per-tool-call timeout, issue **#1076** · Branches: `main` only · Remotes: `github` only
