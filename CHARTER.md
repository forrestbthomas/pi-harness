# pi-harness — Project Charter

> Purpose + in-scope + explicit out-of-scope (PMI charter shape). This is the
> project's boundary contract: if a proposed change serves none of the
> in-scope items and is not an earned exception, it is out of scope. Derived
> from the 2026-08-14 six-persona scope debate — see
> `docs/knowledge-base/decision/2026-08-14-persona-debate-scope.md` for the
> evidence and the raw transcripts (`.pi/debate/`, local only).

## North star

**The score is the product**: a versioned, reproducible, variance-aware,
self-healing measurement of *your* agent configuration (provider, model tier,
prompts, settings) over a versioned contract (`tasks.json` → `score_run.py` →
scorecard, including the harness's own health events) — we cannot cheat it,
and you can verify it. **This repository is the demo**: the discipline this
project applies to its own agent, packaged so you can apply it to yours.

**No moat, no new science** — stated plainly because honesty is the entire
premise. The measurement techniques here are commodity; every major agent
vendor has an internal eval team that could build this in a week. The
offering is the discipline made external and verifiable: the vendors'
measurement is welded to their agent (they measure *their* model on *their*
terms); a neutral, cross-provider seam for *your* config is an uncontested
space, not a protected one. Nobody is motivated to make measurement neutral
and verifiable for your config — this is the starter kit for that discipline.

**We do not** own a benchmark, a PM system, a spec library, an observability
platform, or a general MCP platform. **Distributable is earned**: macOS /
Homebrew is the shipped leg; any further platform or benchmark repo
(pi-bench) waits for a concrete consumer or a second owner team.

## Why this project exists

Coding agents are becoming the default way software is written, and teams are
locking into a single AI vendor because they cannot compare providers
honestly. The vendors all measure — but their measurement is welded to their
agent, private, and on their terms. This project exists so you can point one
harness at any provider, get a real agent, get an honest pass-rate and cost
over a versioned contract, and verify the score yourself — because the only
thing this project sells is that you don't have to trust the number, ours or
the vendors'. It is the discipline this project applies to its own agent,
packaged so you can apply it to yours (dogfood-first).

## What this project is (in scope)

1. **The harness runtime** — the Go CLI (`pi-run`): provider routing across
   many providers (explicit, never silent fallback), launching the Pi coding
   agent (`chat` / `print` / `resume`), honest cost attribution (cost ledger),
   health/self-audit (`doctor`, `config-check`, `project-understand`), and
   release machinery that ships the CLI as a versioned binary.
2. **Self-healing by default** — watchdog detection of no-output stalls,
   process-group kill, git-state auto-recovery, escalation packets with
   evidence (exit code 9), and observability of heal events (`PI_SELF_HEAL`).
3. **Measurability as a first-class property** — the versioned eval contract
   (`tasks.json` → `score_run.py` → scorecard JSON) is the seam that makes
   pass-rate/cost honest and reproducible. The eval suite lives in this repo
   today as the harness's honesty machinery (baseline gate, provenance,
   flake-aware gate, always-upload evidence artifacts).
4. **Honesty over optimism** — hermetic tests for contract, live eval for
   signal; a run that skipped itself is indistinguishable from one that
   passed, so nothing silently skips.
5. **Provider-agnostic, stdlib-only core** — the Go CLI stays dependency-free;
   everything else is optional, env-gated, or additive.
6. **A governance skeleton** — this charter, CONTRIBUTING, CODEOWNERS,
   SECURITY, and the roadmap/backlog that rank work by evidence (RICE).

## What this project is NOT (explicit out-of-scope)

1. **Not a general-purpose agent runtime** — we build the harness that runs
   and measures coding agents, not the agents themselves. We do not claim a
   better agent; the agent underneath is Pi's product.
2. **Not an eval framework or benchmark product** — the eval suite is the
   harness's measurement layer, not a standalone product today, and **not a
   benchmark corpus**: its 55 cases were written by one person as a demo of
   the discipline; treat their rates as evidence of the mechanism, not of
   agent quality. A separate benchmark repo (pi-bench) is an explicitly
   *triggered* future split, not today's scope.
3. **Not a PM system or spec library as a product surface** — project
   management artifacts (roadmap ritual, RICE, EPICS, scope-lock) are internal
   governance, not contributor-facing product features. The living ritual docs
   (`ROADMAP.md`, `BACKLOG.md`, `EPICS.md`, `STATUS.md`) stay at the repo root
   because they are the session-entry contract, are path-read by the GOV-1
   drift guard, and a live eval grader tests for `ROADMAP.md`; dated planning
   specs are archived under `docs/governance/` (`specs-archive/` + `decisions.md`
   index — GOV-2, 2026-08-14).
4. **Not an observability platform** — no OTel exporter or metrics backend is
   product scope (the consumerless OTel exporter was cut 2026-08-14).
5. **Not a general MCP platform** — the read-only MCP server was a harness
   feature with no external consumer and was cut 2026-08-14.
6. **No automatic cross-provider fallback** — the provider is explicit
   (`--provider` / `PI_PROVIDER`); unknown or unmapped tiers fail loudly.
7. **No multi-platform packaging beyond the shipped leg** — macOS/Homebrew is
   shipped; Windows/cloud are non-goals until a consumer asks (a non-goal is an
   invitation to open an issue, not a refusal).
8. **No support commitment** — a no-moat, single-owner OSS project has no SLA;
   the CONTRIBUTING review SLA is a best-effort commitment, not a contract.

## How scope changes

- **In:** a one-paragraph pitch + DoD + rough RICE in `BACKLOG.md`, ranked.
- **Promoted:** user approves moving to `ROADMAP.md` as an active workstream
  with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record
  why).
- **Scope rule:** any change that serves none of the above is out of scope
  unless it earns a backlog entry first.
- **Split trigger (recorded, not yet active):** the eval suite becomes
  pi-bench (own repo) when EPIC-1's DoD closes **and** an external consumer
  appears, or when a release is actually blocked by cross-layer coupling. The
  seam (`tasks.json` / `score_run.py` / scorecard) is the versioned contract
  that keeps this split cheap either way.
