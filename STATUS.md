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
| **Live-suite timeout sizing** (EVAL-12 enabler) | **SHIPPED — 2026-08-14** — #97: per-test 120→600s, job 30→60m so the 54-case nightly can finish |
| **GOV-2 — Governance relocation + spec archive** | **SHIPPED — 2026-08-14** — #119: 19 dated specs → `docs/governance/specs-archive/` + `decisions.md` index; `docs/governance/` = governance home; living PM docs stay at root (scope decision — session-entry contract + GOV-1 path reads + coding-054 grader); SCOPE.md archived to `docs/governance/scope-history/` |
| **OSS-2 — Contributor on-ramp v2** | **SHIPPED — 2026-08-14** — #122: CONTRIBUTING first-issue path (`good first issue`) + 7-day review SLA + MIT-in/MIT-out; PR-template bugfix carve-out; LICENSE identity → `forrestbthomas` |
| **GOV-3 — Wire drift guards into CI** | **SHIPPED — 2026-08-14** — #123: docs/pm drift guards run in python-quick on every push (tags fetched so tag↔changelog enforces); end-to-end drift caught in CI (#124); guard fresh-checkout false positive fixed |
| **W4 Project-management layer** | Active ritual — governance layer; now with EPIC-6 (repo maturity) owning governance/community/debt work |

## Next (shaped — next 1–2 cycles)

| Item | RICE | Tag |
|---|---|---|
| **v0.11.0 — The gate that can't lie** (EVAL-12 re-baseline → EVAL-16 harness gate) | — | Release |
| **EPIC-1 — Eval suite → research-grade measurement** (EVAL-12 standing trigger, EVAL-16 pilot, EVAL-17, EVAL-9, EVAL-11) | top EVAL-16 ~1.6 | EPIC-1 |
| **EPIC-6 — Repo maturity** (OWN-1+SECURITY bundle, TAX-1, PORT-0) | top OWN-1 ~1.5 | EPIC-6 |
| **EPIC-2 — Self-healing resilience** (HEAL-5, HEAL-2 data-gated, HEAL-3; HEAL-1 demoted) | top HEAL-5 1.20 | EPIC-2 |
| **EPIC-3 — Cost intelligence & routing** (COST-2; COST-1 deferred) | top COST-2 0.50 | EPIC-3 |
| **EPIC-4 — Portability & distribution** | **PARKED** — PORT-0 quarterly re-confirm only (charter non-goal until a consumer asks) | EPIC-4 |
| ~~EPIC-5 — Insight & DX~~ | **DISSOLVED** — DX-1/DX-2 → idea inbox | — |
| **v1.0.0 — The contract release** (gates, not dates: EPIC-1+6 DoDs, ≥14 green nightlies, EVAL-16 enforced, install-path CI-proven, consumer OR recorded earned-bar decision) | — | Release |

## Later (raw — ideas worth exploring)

- Watchdog liveness heartbeat (HEAL-1; re-score from `PI_SELF_HEAL` data) ·
  eval output gallery (EVAL-10; parked) · sandbox live runs (EVAL-7; parked) ·
  dataset growth 50 → 100 (capped until consumer/regression-catch signal) ·
  cloud eval backend · context-engine un-park (DX-2) · Windows support
  (PORT-1; charter non-goal) · model-catalog auto-refresh in CI · cost-aware
  routing (COST-1; deferred — re-open on a consumer signal)

## Shipped recently (one spelling: CHANGELOG)

- **v0.10.0** (2026-08-14): charter (#98/#99), cut list −1790 lines (#100),
  EPIC restructure (#101), OSS-1 canonical identity (#102), live-suite timeout
  fix (#97), version-truth stamp + ledger repair (#104)
- **v0.9.2** (2026-08-13): W2 live-eval baseline, pin pi-subagents, owner-only
  perms · **v0.9.1** (2026-08-13): W1 self-healing, non-interactive env, exit 9 ·
  **v0.9.0**: eval hardening + live eval v2 · **v0.8.0**: project-understand,
  MCP, OTel, permission modes, hooks, 17 providers
- **main (post-v0.10.0, v0.11.0 in flight)**: EVAL-13 cost gate (#105), EVAL-14
  provenance parity (#106), EVAL-15 split-seam (#107), EVAL-8 judge
  stabilization (#108), REL-3/4 release machinery (#109), TAX-2 audit (#110),
  EVAL-6 agentic slice 1 (#111), GOV-1 drift guard (#112), GOV-2 relocation +
  spec archive (#119), GOV-3 CI-wiring item + docs-drift cwd fix (#120), OSS-2
  contributor on-ramp (#122), GOV-3 drift guards wired into CI (#123)

## Open PRs / branches

- Open PRs: **none** (harness) · Upstream: pi-subagents **#1077 merged** (`a660ea3`) as per-tool-call timeout, issue **#1076** · Branches: `main` + 37 stale remote branches (prune per `docs/roadmap-workflow.md`) · Remotes: `github` only
