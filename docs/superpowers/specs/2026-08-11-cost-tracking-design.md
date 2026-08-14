# `pi-run cost` + Budget Cap — Design

**Date:** 2026-08-11
**Status:** SHIPPED — cost ledger + budget caps (v0.5.0, PR #25; cost.go)
**Target release:** v0.5.0 (feature)

## 1. Context & Motivation

**Gap:** Claude Code has `--max-budget-usd`; pi-subagents reports per-run
token/cost in its results; but our harness CLI has **no cost visibility and no
budget cap**. For an anti-lock-in tool, "know what each provider actually
costs" is a headline feature — it lets users compare OpenAI vs DeepSeek vs a
local model on real spend, not just capability.

**Key finding (verified):** Pi session files already record per-message
`usage.cost` (input/output/total USD) for **every** provider/model we've used
(deepseek, openai, kimi, openrouter). So `pi-run cost` can aggregate **real
spend from existing session data** — no price tables, no estimation. The
budget cap can check cumulative cost from sessions + subagent artifacts before
launching.

## 2. Current State (verified)

- Session files `.pi/sessions/*.jsonl` contain assistant messages with
  `usage: { input, output, cacheRead, cacheWrite, reasoning, totalTokens, cost: { input, output, cacheRead, cacheWrite, total } }`
  and `provider`/`model` on each message.
- Subagent children write their own session files under `.pi/sessions/`
  (per-run/`parallel-N` dirs) plus transcript JSONL under `.pi-subagents/artifacts/`;
  artifact meta has `model`/`agent`/`durationMs` but **not** cost (cost lives in
  the child session files).
- `pi-run` already parses session paths (`repoRoot`, `sessionDir`), so locating
  session files is straightforward.
- The CLI is stdlib-only Go 1.26; JSON parsing is `encoding/json`.

## 3. Scope

### In scope

1. **`pi-run cost`** — aggregate and report usage/spend:
   - Reads all Pi session files under `.pi/sessions/` (including subagent child
     sessions) and sums `usage.cost.total` per provider/model/session.
   - Output: human-readable table (provider, model, tokens, cost, sessions) +
     total; `--json` for machine-readable.
   - `--since <date>` filter; `--reset` archives the current ledger state and
     starts fresh (moves a marker file; does not delete session files).
2. **Budget cap** — `pi-run chat|print --max-budget-usd <n>` (env
   `PI_MAX_BUDGET_USD` default):
   - Pre-flight: compute cumulative spend (sessions + subagents) before launch;
     if ≥ budget, refuse with a clear message and a **dedicated exit code (6 =
     budget exceeded)** rather than reusing the key-missing code (3).
   - After launch: record spend for the just-finished run; warn if budget was
     exceeded.
   - `pi-run cost` is the read path; `--max-budget-usd` is the enforcement path.
3. **Spend ledger** (append-only JSONL) so budget checks don't rescan huge
   session files every time:
   - `.pi/cost-ledger.jsonl` (gitignored), one line per run:
     `{ts, provider, model, inputTokens, outputTokens, costUsd, mode}`.
   - `pi-run cost` = ledger + (optionally) session-file scan for backfill;
     budget check = ledger total + session scan for anything not yet logged.
4. **Tests**: hermetic Go unit tests with fixture session files + ledger
   (aggregation, JSON output, budget pre-flight logic). No keys/network.

### Explicitly NOT in scope

- Live cost display during a session (would require deeper Pi integration) —
  pre-flight + post-run is the v1 contract.
- Price-table maintenance — we use Pi's own `usage.cost`; no per-model pricing
  data in our repo. (If a provider ever reports 0 cost, show "unknown" rather
  than estimate.)
- A dashboard/UI — `pi-run cost --json` is the contract.
- Cross-machine aggregation — per-project ledger only.

## 4. Design

### Ledger format (`.pi/cost-ledger.jsonl`)

```json
{"ts":"2026-08-11T10:00:00Z","provider":"openai","model":"gpt-5.6-terra","inputTokens":38962,"outputTokens":280,"costUsd":0.00553308,"mode":"print"}
```

### CLI

```
pi-run cost [--json] [--since <date>] [--reset]
pi-run chat  --max-budget-usd 5.00 "task"
pi-run print --max-budget-usd 5.00 "task"
```

### Go structure (stdlib-only)

```
internal/cli/cost.go          // ledger read/write, session scan, aggregation
internal/cli/cost_test.go     // hermetic unit tests
```

- `sessionCost(path)` — stream a session JSONL, sum `usage.cost.total` per
  provider/model (skip messages without `usage.cost`).
- `ledgerAppend/ledgerSum` — append-only JSONL, atomic append.
- `budgetCheck(capUSD)` — ledger + session scan; returns remaining.
- Wiring: `runLaunch` reads `--max-budget-usd` / `PI_MAX_BUDGET_USD`, calls
  `budgetCheck` before `execPi`, appends to ledger after.

## 5. Implementation Plan

1. `internal/cli/cost.go`: session-scan + aggregation + `--json` output.
2. Ledger: append-only JSONL + `--reset` marker; gitignore `.pi/cost-ledger.jsonl`.
3. Budget cap: `--max-budget-usd` flag + `PI_MAX_BUDGET_USD` env; pre-flight
   check in `runLaunch`; post-run ledger append.
4. Wire `pi-run cost` command + `--since` filter.
5. Tests: fixture session files + ledger (aggregation, JSON, budget pre-flight).
6. Update the `--exit-codes` help table (app.go) to document exit code 6 =
   budget exceeded; keep 0-5 meanings unchanged.
7. Docs: README "Cost & Budgets" section; CHANGELOG v0.5.0 entry.

## 6. Changelog Entry (draft)

```markdown
### Added
- `pi-run cost`: aggregate real spend from Pi session files (per provider/model,
  `--json`, `--since`, `--reset`).
- `pi-run chat|print --max-budget-usd <n>` (env `PI_MAX_BUDGET_USD`): pre-flight
  budget check before launching, plus an append-only spend ledger.
```

## 7. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Some providers report 0 cost in usage | Medium | Show "unknown"; ledger stores Pi's value verbatim; no estimation |
| Session files grow large → slow scans | Low-Med | Ledger is the fast path; session scan only for backfill/`--reset` |
| Budget check misses in-flight subagents | Low | Ledger captures post-run totals; pre-flight is best-effort (documented) |
| `.pi/cost-ledger.jsonl` committed accidentally | Low | gitignore entry + a `TestNoHardcodedUserPaths`-style check |

## 8. Decision

**Recommend proceeding.** Cost visibility is high-value, low-effort, and
directly reinforces the anti-lock-in positioning ("see what each provider
costs"). The verified `usage.cost` data makes it pure aggregation with no price
tables. Budget cap is a natural second half with clear UX (pre-flight refusal).
Scope is v1-minimal (per-project ledger, pre-flight + post-run enforcement)
with a clean extension path (live cost, cross-machine aggregation).
