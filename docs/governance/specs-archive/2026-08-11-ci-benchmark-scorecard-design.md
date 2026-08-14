# Provider Scorecard in CI (`pi-run ci-benchmark`) — Design

**Date:** 2026-08-11
**Status:** SHIPPED — `pi-run ci-benchmark` + provider scorecard (v0.6.0 era; scorecard.go)
**Target release:** v0.6.0 (feature)

## 1. Context & Motivation

**Gap:** ModelBench/pi-bench-style tooling compares models **interactively** — a
developer sits in a live Pi TUI, asks two models the same question, and eyeballs
the answers. Nobody in the Q2 2026 landscape offers a **repeatable, budgeted,
gated provider comparison in CI**: a scheduled job that runs the same scored
task suite against two or more providers, records a scorecard (pass rate, cost,
latency, tokens), fails the build on regression or budget breach, and leaves a
machine-readable artifact. SWE-bench and terminal-bench are research benchmarks,
not CI gates; Claude Code and Codex cap spend interactively but never compare
providers as a repeatable artifact.

**Opportunity:** our moat is the **choose → measure → control** loop:

- **choose** — provider-agnostic routing (7 providers, data-driven
  `providers.json`);
- **measure** — `pi-run eval --benchmark` scores the same Docker-isolated tasks
  per provider;
- **control** — `pi-run cost` + `--max-budget-usd` caps spend.

`pi-run ci-benchmark` makes that loop a **CI artifact**: every scheduled run
answers "did any provider regress, and what does this comparison cost?" on a
budget, with a green/red exit code. No other project combines a Docker-isolated
task runner, real-cost attribution, and a budgeted gate in one stdlib-only
binary — this is the natural next P0 from the gap analysis.

## 2. Current State (verified)

- `pi-run eval --benchmark` (internal/cli/benchmark.go, shipped v0.5.0):
  Docker-isolated task runner with **5 seed tasks**
  (`eval/benchmarks/{add-verbose-flag,fix-divide-by-zero,fix-json-parsing,fix-shell-script,implement-factorial}`),
  per-task pass/fail grading via `tests/run.sh` exit code, `--provider`/`--model`
  selection, `--benchmark-dry-run` hermetic validation, JSON report
  `eval/benchmark-results/<run-id>.json` (gitignored) via
  `writeBenchmarkResults`, and exit code **7** when Docker is unavailable.
  `runBenchmarkLive(tasks, opts, root)` orchestrates one provider;
  `benchmarkRunResult` already carries per-task `durationSecs` and
  `summary{total,passed,failed,errors,score}`.
- `pi-run cost` + budget cap (internal/cli/cost.go, shipped v0.5.0): aggregates
  **real spend** from Pi session files (`usage.cost.total`, no price tables) per
  provider/model; append-only spend ledger `.pi/cost-ledger.jsonl` (gitignored)
  with modes `chat|print|resume|backfill`; `--max-budget-usd` / `PI_MAX_BUDGET_USD`
  cap with exit code **6**. The plumbing we need already exists:
  `resolveBudgetCap` (cap parsing), `currentSpend` (pre-run snapshot),
  `recordRunSpend(root, start, mode, provider, model, preSpend)` (per-run delta
  attribution + ledger write).
- CLI is stdlib-only Go 1.26; hermetic tests exist for both features
  (benchmark_test.go, cost_test.go — no Docker, no keys).
- Exit-code table (app.go `--exit-codes`): 0 ok · 1 generic · 2 usage · 3 missing
  key · 4 node/pi missing · 5 eval venv missing · 6 budget exceeded · 7 docker
  unavailable. A gate needs one new code (8 = scorecard gate failed).
- `eval/benchmark-results/` is already gitignored, so scorecard files need no new
  ignore rule.

## 3. Scope

### In scope

1. **New `pi-run ci-benchmark` command** (recommended over `eval --ci-scorecard`;
   see §4.1): run the same benchmark suite against **2+ providers/models**,
   aggregate a per-provider scorecard, compare against a baseline/threshold, and
   exit non-zero on regression or budget breach.
2. **`--providers openai,deepseek` / `--models ...`** — comma-separated provider
   list (≥2, order-significant); `--models` optionally overrides each provider's
   default model (same order; defaults to the `providers.json` `defaultModel`).
3. **Gates**: `--fail-below <pass-rate>` (any provider below the rate fails) and
   `--max-budget-usd <n>` (total run cost at/above the cap fails, reusing exit
   code 6).
4. **`--baseline <path>`** — compare against a previous scorecard or per-provider
   run JSON; a per-provider pass-rate drop beyond `--baseline-tolerance` (default
   0.05) fails. File-based baseline is v1 (no cross-machine baseline store).
5. **Scorecard output**: `eval/benchmark-results/scorecard-<run>.json`
   (gitignored, schema in §4.3) + a human-readable table on stdout.
6. **`--runs <n>`** (default 1): repeat each provider suite n times and gate on
   the **median** pass rate (flakiness mitigation); `--quick-profile` caps the
   per-task agent timeout at 60 s for cheap scheduled smoke runs (cost
   mitigation).
7. **CI wiring**: a documented GitHub Actions job (§4.6) that runs the scorecard
   across 2 providers and uploads the JSON as an artifact.
8. **Tests**: hermetic scorecard aggregation + gate-logic tests (fixtures, no
   Docker/keys) and flag-parsing tests.

### Explicitly NOT in scope

- A live TUI dashboard / interactive model comparison — that is ModelBench/pi-bench
  territory; `ci-benchmark` stays non-interactive.
- A public leaderboard or hosted scorecard store — JSON is the contract; a report
  page is later.
- A cross-machine baseline database — `--baseline <path>` file is v1;
  artifact-chained baselines in CI.
- Parallel provider execution — providers run sequentially so per-provider cost
  attribution stays clean (parallel is a later iteration).
- New task-grading semantics — reuse benchmark pass/fail grading unchanged; no
  diff grading, no new task formats.
- Sandboxing the daily-driver `chat` path — ci-benchmark uses the benchmark
  runner's existing Docker isolation only.

## 4. Design

### 4.1 Command shape: `pi-run ci-benchmark` (not `eval --ci-scorecard`)

**Recommendation: a dedicated subcommand.** Justification:

1. **Distinct exit-code contract.** `eval` returns pytest's exit code and
   `--benchmark` returns its own set (0/1/2/3/4/7). A CI gate needs an
   unambiguous "quality gate failed" code (8) that CI maps straight to a red
   build, and it must distinguish *infra failure* (3/4/7) from *quality failure*
   (8) from *budget breach* (6). Encoding that inside eval's flag matrix muddies
   both surfaces.
2. **Flag surface.** eval already owns `--quick`, `--benchmark [name]`,
   `--benchmark-dry-run`, `--provider`, `--model` with cross-flag validation in
   `parseEvalArgs`. Adding `--providers`, `--models`, `--baseline`,
   `--fail-below`, `--max-budget-usd`, `--runs`, `--quick-profile` would create
   a confusing two-mode flag matrix (`--quick` vs `--benchmark` vs
   `--ci-scorecard`).
3. **Precedent.** `pi-run cost` is already a dedicated subcommand with its own
   usage text and exit-code semantics; ci-benchmark is the same kind of vertical
   ("run a gate") and should stand alone.
4. **Extensibility.** The gate will grow (baseline stores, regression alerts,
   chat notifications) without touching eval's parse loop.

### 4.2 CLI

```
pi-run ci-benchmark --providers openai,deepseek [--models openai/gpt-5.6-terra,deepseek/deepseek-v4-flash]
    [--fail-below <pass-rate>] [--max-budget-usd <n>] [--baseline <path>]
    [--baseline-tolerance <n>] [--runs <n>] [--quick-profile]

Run the benchmark suite against 2+ providers and gate on the scorecard.
  --providers <a,b>        Comma-separated providers (>= 2; order-significant).
  --models <m1,m2>         Optional per-provider model overrides (same order as --providers).
  --fail-below <rate>      Fail if any provider pass rate < rate (e.g. 0.8).
  --max-budget-usd <n>     Fail (exit 6) if total run cost >= n. PI_MAX_BUDGET_USD also applies.
  --baseline <path>        Previous scorecard/run JSON to diff pass rates against.
  --baseline-tolerance <n> Max allowed per-provider pass-rate drop vs baseline (default 0.05).
  --runs <n>               Repeat each provider suite n times; gate on median pass rate (default 1).
  --quick-profile          Cap per-task agent timeout at 60s (cheap, best-effort smoke run).
```

Validation (usage errors, exit 2): `--providers` required with ≥2 entries; each
provider must resolve via `ResolveProvider`; `--models` length must equal
`--providers` length; `--fail-below` must be in [0,1]; `--baseline` file must
parse; `--runs` ≥ 1.

### 4.3 Scorecard JSON schema (`eval/benchmark-results/scorecard-<run>.json`)

`<run>` = timestamp + joined provider names, e.g.
`20260811T150405-openai-deepseek`.

```json
{
  "schemaVersion": 1,
  "runId": "20260811T150405-openai-deepseek",
  "timestamp": "2026-08-11T15:04:05Z",
  "suite": "eval/benchmarks (5 tasks)",
  "quickProfile": false,
  "runs": 1,
  "gates": { "failBelow": 0.8, "maxBudgetUsd": 5.0, "baselineTolerance": 0.05 },
  "baselinePath": "eval/benchmark-results/scorecard-20260804T090000-openai-deepseek.json",
  "providers": [
    {
      "provider": "openai",
      "model": "openai/gpt-5.6-terra",
      "passed": 5,
      "total": 5,
      "errors": 0,
      "passRate": 1.0,
      "costUsd": 0.0412,
      "avgLatencyMs": 18734.5,
      "tokens": 128430
    },
    {
      "provider": "deepseek",
      "model": "deepseek/deepseek-v4-flash",
      "passed": 4,
      "total": 5,
      "errors": 1,
      "passRate": 0.8,
      "costUsd": 0.0021,
      "avgLatencyMs": 9430.2,
      "tokens": 45210
    }
  ],
  "baseline": {
    "path": "eval/benchmark-results/scorecard-20260804T090000-openai-deepseek.json",
    "regressions": [
      { "provider": "deepseek", "baseline": 1.0, "current": 0.8, "tolerance": 0.05 }
    ]
  },
  "passed": false
}
```

Field definitions (all computed from existing data):

- `passRate` = `passed / total`, reusing `aggregateBenchmarkResults` per provider;
  `errors` counts tasks that errored (agent crash, Docker failure) — an error
  makes the run incomplete and fails the gate.
- `costUsd` = per-provider spend delta for the run: snapshot `currentSpend(root)`
  before, call `recordRunSpend(root, start, "benchmark", provider, model, pre)`
  after (new ledger mode `"benchmark"`), `costUsd = post − pre`. Providers that
  report no `usage.cost` show 0 — consistent with the cost spec (no estimation).
- `avgLatencyMs` = mean of per-task `durationSecs` × 1000 — the end-to-end
  per-task wall time the runner already records (workspace prep + agent +
  Docker verification).
- `tokens` = `inputTokens + outputTokens` from the ledger entries written for
  this provider's run.
- `--runs > 1`: each provider's suite runs n times; the scorecard reports the
  **median** pass rate (and median cost/latency/tokens), and the gate reads the
  median row.

### 4.4 Gate logic (exit codes)

Failures are ordered so CI can distinguish causes:

1. Any task `error` in any provider run → run incomplete → **exit 8**
   ("scorecard gate failed: run incomplete").
2. Any provider `passRate < failBelow` → **exit 8**.
3. `--baseline` given and any provider present in both baseline and current has
   `current passRate < baseline passRate − tolerance` → **exit 8** (regression).
   Providers present in only one side are reported but not failed; a provider
   missing from the current run is a warning.
4. Total run cost (Σ per-provider `costUsd`) ≥ `maxBudgetUsd` (or
   `PI_MAX_BUDGET_USD`) → **exit 6** (reuse `exitBudgetExceeded` — CI sees
   "budget", not "quality").
5. Infra failures keep their existing codes: Docker missing → 7, missing key →
   3, node/pi missing → 4, usage → 2.

`scorecard.passed` is `false` on 1–4; the gate passes (exit 0) otherwise.

### 4.5 Baseline file format (v1)

`--baseline` accepts either:

- a prior **scorecard** JSON (schemaVersion 1 — `providers[].{provider,
  passRate}`), or
- a prior **per-provider run** JSON (`eval/benchmark-results/<run-id>.json` —
  the shipped `benchmarkRunResult` shape `{provider, model, summary.score}`).

`parseBaseline(path)` sniffs which shape and returns `map[provider]passRate`;
both are stdlib `encoding/json`, and `benchmarkRunResult` needs no changes. A
baseline is a point-in-time reference (e.g. the last green scheduled run);
re-baselining is a deliberate act (point `--baseline` at a newer artifact), not
automatic.

### 4.6 CI example (GitHub Actions)

```yaml
# .github/workflows/provider-scorecard.yml
name: provider-scorecard
on:
  workflow_dispatch:        # manual run
  schedule:
    - cron: "0 3 * * 1"     # weekly Monday 03:00 — not on every push (cost)

jobs:
  scorecard:
    runs-on: ubuntu-latest  # Docker is preinstalled on GitHub-hosted runners
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - name: Build pi-run
        run: |
          go build -o bin/pi-run ./cmd/pi-run
          # GitHub runner shells do not put the workspace bin/ on PATH.
          echo "$GITHUB_WORKSPACE/bin" >> "$GITHUB_PATH"
      - name: Set up Node via nvm + install the Pi CLI
        # pi-run resolves the pi binary from the HIGHEST nvm-managed Node
        # toolchain (~/.nvm/versions/node/<v>/bin/pi); it does NOT use a PATH
        # node from actions/setup-node. Install node + pi under nvm instead.
        # Set PI_NODE_VERSION to pin an exact nvm version.
        run: |
          export NVM_DIR="$HOME/.nvm"
          [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
          nvm install "${PI_NODE_VERSION:-latest}"
          # The REAL Pi coding agent: the plain `pi` npm package is an
          # unrelated legacy CLI that prints "3" for every invocation
          # (silent garbage in eval runs).
          npm install -g @earendil-works/pi-coding-agent
      - name: Pull last scorecard (baseline)
        uses: actions/download-artifact@v4
        with:
          name: benchmark-scorecard
          path: eval/benchmark-results/
        continue-on-error: true  # first run has no baseline yet
      - name: Resolve baseline (latest scorecard if any)
        # The CLI writes only timestamped scorecard-<run>.json; give the gate a
        # fixed scorecard-latest.json name when a prior artifact exists.
        run: |
          if ls eval/benchmark-results/scorecard-*.json >/dev/null 2>&1; then
            cp "$(ls -t eval/benchmark-results/scorecard-*.json | head -1)" \
              eval/benchmark-results/scorecard-latest.json
          fi
        continue-on-error: true
      - name: Run provider scorecard gate
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
        run: |
          BASELINE=""
          if [ -f eval/benchmark-results/scorecard-latest.json ]; then
            BASELINE="--baseline eval/benchmark-results/scorecard-latest.json"
          fi
          pi-run ci-benchmark --providers openai,deepseek \
            --fail-below 0.8 --max-budget-usd 5.0 $BASELINE
      - name: Upload scorecard (becomes next run's baseline)
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-scorecard
          path: eval/benchmark-results/scorecard-*.json
          retention-days: 90
```

Notes: providers run sequentially (clean cost attribution); keys come from repo
secrets and are never echoed (the CLI only logs key *names*, matching today's
behavior); the artifact-chained `scorecard-latest.json` is the v1 file-based
baseline — the resolve-baseline step copies the newest timestamped scorecard to
that fixed name, and the gate step only passes `--baseline` when the file
actually exists (a missing file is a usage error, so the flag must be
conditional); pi must be installed under an **nvm-managed** Node toolchain (the
CLI resolves `~/.nvm/versions/node/<highest>/bin/pi`, not the PATH node from
`actions/setup-node`) — pin `PI_NODE_VERSION` to match; constrained CI without
Docker can still run `pi-run eval --benchmark-dry-run` as a format-only gate.

### 4.7 Reuse of `runBenchmarkLive` + cost plumbing

- `runBenchmarkLive(tasks, opts, root)` is refactored into
  `runProviderBenchmark(tasks, opts, root) (benchmarkRunResult, error)` — the
  same per-task loop (workspace prep → `execPiDirTimeout` agent run → Docker
  build/run → grade), but returning the result data instead of printing/writing
  and returning an int. `runBenchmarkLive` becomes a thin wrapper so
  `eval --benchmark` behavior and exit codes are unchanged (protected by the
  existing benchmark_test.go).
- Per provider, `ci-benchmark` does: `pre := currentSpend(root)` → `start :=
  time.Now()` → `res := runProviderBenchmark(...)` → `post, _ :=
  recordRunSpend(root, start, "benchmark", provider, model, pre)` →
  `costUsd = post − pre`. Sequential providers keep attribution clean.
- Scorecard writing reuses `writeBenchmarkResults`'s `eval/benchmark-results/`
  dir + `json.MarshalIndent` pattern; human output reuses the `text/tabwriter`
  pattern from `printCostTable`.
- No new dependencies: stdlib-only (`encoding/json`, `os/exec`,
  `text/tabwriter`, `sort`).

## 5. Implementation Plan

1. **Refactor benchmark.go**: extract `runProviderBenchmark(tasks, opts, root)
   (benchmarkRunResult, error)` from `runBenchmarkLive`; keep the
   `eval --benchmark` wrapper behavior byte-identical; run the existing
   benchmark_test.go to confirm no regression.
2. **`internal/cli/scorecard.go`**: per-provider aggregation (passRate/errors via
   `aggregateBenchmarkResults`, cost delta via `recordRunSpend` mode
   `"benchmark"`, `avgLatencyMs`, tokens from ledger attribution), `--runs`
   median logic, scorecard JSON writer + human table.
3. **Gates**: `--fail-below`, budget (reuse `resolveBudgetCap`, exit 6),
   incomplete-run check; new `exitScorecardFailed = 8` and `--exit-codes` table
   update in app.go.
4. **Baseline**: `parseBaseline(path)` accepting scorecard or run JSON →
   per-provider pass-rate map; `--baseline-tolerance` regression check.
5. **CLI wiring**: `ci-benchmark` command in app.go dispatch + usage text; flag
   parsing/validation (≥2 providers, model-list length, rate range).
6. **CI wiring**: `.github/workflows/provider-scorecard.yml` (§4.6).
7. **Tests (hermetic, no Docker/keys)**: `scorecard_test.go` — fixture run
   results → aggregation math; gate-logic table tests (failBelow, budget,
   baseline regression, tolerance boundary, incomplete run); baseline parser
   (both shapes); flag parsing; JSON round-trip of the scorecard schema.
8. **Docs**: README "Provider Scorecard in CI" section + `docs/benchmarks.md`
   pointer; CHANGELOG v0.6.0 entry (below).

## 6. Changelog Entry (draft)

```markdown
## [0.6.0] - 2026-08-11

### Added
- `pi-run ci-benchmark`: provider scorecard in CI. Runs the benchmark suite
  against 2+ providers (`--providers openai,deepseek`, optional `--models`),
  writes a per-provider scorecard (`eval/benchmark-results/scorecard-<run>.json`:
  pass rate, cost, latency, tokens), and gates the build: exit 8 when any
  provider drops below `--fail-below <rate>` or regresses vs `--baseline <path>`
  (default tolerance 0.05), exit 6 when run cost hits `--max-budget-usd`.
  Includes `--runs <n>` median repeats for flaky runs and `--quick-profile` for
  cheap scheduled smoke runs.
```

## 7. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Live provider runs are expensive in CI | High | Budget cap (`--max-budget-usd`, exit 6) + `--quick-profile`; default schedule is weekly/manual, not every push; keep the suite at 5 seed tasks |
| Flaky agent runs flip the gate | Medium | `--runs <n>` gates on median pass rate; `--baseline-tolerance` absorbs small drift |
| Baseline drift over time | Medium | Baseline is an explicit file (`--baseline <path>`); re-baselining is a deliberate artifact pick; regressions list baseline and current in the scorecard for audit |
| Docker required in CI | Low-Med | GitHub-hosted ubuntu runners ship Docker; otherwise use a Docker-enabled self-hosted runner or fall back to `--benchmark-dry-run` (format-only gate) |
| Provider keys in CI | Low | Keys from repo secrets only; CLI never logs values, only key env names; `pi-run doctor` preflight |
| Per-provider cost attribution muddied by concurrent runs | Low | Sequential provider execution in v1; parallel is a later iteration with its own attribution |
| Gate semantics confuse CI users | Low | One exit code per failure class (2/3/4/6/7/8) documented in `--exit-codes`; the scorecard `passed` field is the single source of truth |

## 8. Decision

**Recommend proceeding.** The gap analysis ranked "Docker-isolated benchmark"
and "cost/budget" as the two P0s; both shipped in v0.5.0. This spec is the
**third leg that makes them matter**: it converts choose → measure → control
into a repeatable, budgeted, gated CI artifact — a provider scorecard that fails
a build on regression or overspend. It is uniquely ours: no other harness pairs
a Docker-isolated scored task runner with real-cost attribution and a budgeted
gate in one stdlib-only binary. Scope is v1-minimal (file-based baseline,
sequential runs, no leaderboard) with clean extension points (cross-machine
baseline store, parallel providers, dashboard). All plumbing
(`runBenchmarkLive`, `recordRunSpend`, `resolveBudgetCap`,
`writeBenchmarkResults`, exit codes 6/7) exists and is tested today.
