# Live Agent Evaluation v2 (dataset growth, nightly baseline gate, cost-per-task metrics) — Design

**Date:** 2026-08-12
**Status:** Proposed (from live-agent-eval research brief `38697b76`, landscape §1–§2)
**Target release:** v0.9.0 (infra / eval)
**Depends on:** the deterministic-eval-hardening design spec (sibling lane
`docs/spec-eval-hardening`, lands first) for exactly two surfaces: the CI build
step that puts `bin/pi-run` on `PATH` in eval CI jobs, and the hermetic Python
contract tests for `mcp-server` / OTel / `--model-tier` / `project-understand`.
See §4.1 for the precise dependency boundary; nothing else in this spec assumes
that lane.

## 1. Context & Motivation

**Current state (the gap):** the live agent-eval surface of pi-harness is a
5-sample JSONL dataset that is ~80% trivia/concept Q&A answerable without any
agent tool use (`eval/datasets/coding_samples.jsonl`), a nightly workflow that
**silently skips live tests when the provider key is absent**
(`.github/workflows/nightly-live-eval.yml:14` comment "skips gracefully if
absent") — the exact anti-pattern Autonoma warns about ("an eval job that skips
itself is indistinguishable from one that passed") — a live E2E test with **no
timeout** and single-run heuristic assertions
(`eval/tests/test_agent_task_completion.py`, `run_pi_print` in
`eval/conftest.py:63-90` spawns `subprocess.run` with no `timeout`), an
LLM-judged metric layer that is **scaffolded but dead** (`sample_cases` fixture
in `eval/conftest.py:38-54` is consumed by no test file), zero negative/edge
cases, and no cost-per-task metric despite full cost plumbing
(`internal/cli/cost.go`).

**Why v2 now:** the 2026 landscape is unambiguous that agent eval suites should
be small, curated, and deterministic-first — Anthropic recommends starting with
**20–50 tasks drawn from real failures** with reference solutions and graders;
DeepEval targets ~100 goldens max with coverage over volume; SWE-bench,
Terminal-Bench, and aider all grade with **deterministic tests, not LLM judges**,
wherever the task is executable. Nightly evals follow a **two-speed pipeline**:
deterministic checks on every commit; token-costly evals on a nightly cron with
**N-run averaging (3–5 runs) against per-case baselines with a tolerance band**
(0.05–0.1) instead of single-run assertions. Cost-per-task is a first-class
leaderboard axis everywhere (`$/instance` in SWE-bench, `total_cost` in aider,
`cost per successful execution` in tier-bench / Arize). This spec converts
those patterns into pi-harness's own nightly gate: a 20-task balanced dataset,
a baseline-gated scoring script, and cost-per-task metrics feeding the
existing scorecard + ledger.

**Goal:** make the nightly live eval an *honest, gated, budgeted* signal:

- grow the live dataset 5 → 20 with a schema v2 (category, difficulty, grader,
  regression tags) and balanced category coverage including negative/edge and
  harness-routing tasks;
- upgrade the nightly to a **two-job split** with `timeout-minutes: 30`, a
  parametrized 3-run suite, a `score_run.py` baseline gate (per-case
  pass-rate tolerance 0.05, cost-per-task >2× baseline fails), artifacts with
  90-day retention, and a **hard fail on missing key**;
- add cost-per-task metrics to the scorecard + nightly JSON schemas and
  `live-eval` mode to the spend ledger;
- unify governance via a single task manifest (`eval/datasets/tasks.json`)
  without forcing Docker seeds into the nightly.

## 2. Current State (verified)

| Area | Verified state |
|---|---|
| Live dataset | `eval/datasets/coding_samples.jsonl` — 5 samples: `coding-001` factorial (code-gen), `coding-002` `is` vs `==` (concept), `coding-003` bash `find` (shell), `coding-004` binary-search complexity (concept), `coding-005` JS debounce (code-gen). Fields: `id/input/expected_output/context/tags`. 4 of 5 answerable without agent tool use; **zero negative/edge cases** |
| Nightly workflow | `.github/workflows/nightly-live-eval.yml` (36 lines) — cron `0 3 * * *`, single `live-eval` job, `OPENAI_API_KEY` from secrets, `eval/.venv/bin/python -m pytest tests/ -v`; **no `timeout-minutes`, no budget cap, no artifacts, no baseline, no build of pi-run** (relies on a preinstalled `pi-run`) |
| Live E2E test | `eval/tests/test_agent_task_completion.py` — exactly 1 live task (`test_agent_produces_expected_factorial`, skipif-no-key, string heuristics "factorial"/"def "/"for\|recursion\|range"); `run_pi_print` (`eval/conftest.py:55`) spawns `pi-run print` with **no timeout** (`pytest-timeout` is in `eval/requirements.txt` but no `@pytest.mark.timeout` is set) |
| Dead metric seam | `sample_cases` fixture (`eval/conftest.py:38-54`) converts the dataset into `LLMTestCase`s; **no test file consumes it** (no `test_dataset_metrics.py` / `test_live_metrics.py` exist); today's scored metrics are deterministic `CodeQualityMetric` (`eval/tests/test_code_quality.py:11-69`) AND live LLM-judged `AnswerRelevancyMetric`/`FaithfulnessMetric`/`HallucinationMetric` (`eval/tests/test_coding_correctness.py`, skipif-no-key); the `sample_cases` fixture is consumed by NO test file |
| Hermetic Python tests (pattern to extend) | `eval/tests/test_harness_config.py` (subprocess `pi-run config-check`), `test_benchmark_format.py` (task.json schema + dry-run), `test_secret_resolution.py` (fake `bw_get`/`op` binaries in `tmp_path`) |
| CI python-quick | `.github/workflows/ci.yml:30-47` — runs `test_code_quality.py` + one dataset sanity test **without building pi-run** (latent: any binary-dependent test only works if `pi-run` is preinstalled); the build precedent exists in `provider-scorecard.yml:29-32` (`go build -o bin/pi-run ./cmd/pi-run` + `echo "$GITHUB_WORKSPACE/bin" >> "$GITHUB_PATH"`) |
| Provider scorecard | `.github/workflows/provider-scorecard.yml` — weekly Monday 03:00 + manual; `ci-benchmark --providers openai,deepseek --fail-below 0.8 --max-budget-usd 5.0`; baseline chained via `scorecard-latest.json` (shell `cp` glue at `provider-scorecard.yml:62-64`, untested); artifacts 90-day retention |
| Scorecard schema | `internal/cli/scorecard.go:52-62` `scorecardProvider{provider, model, passed, total, errors, passRate, costUsd, avgLatencyMs, tokens}`; `buildScorecard` at `scorecard.go:644` calls `time.Now().UTC()` for `Timestamp` (`scorecard.go:648`) and `scorecardRunID` (`scorecard.go:673`) — the determinism blocker, **owned by the hardening lane**, not this spec; `writeScorecard` (`scorecard.go:678`) writes `eval/benchmark-results/scorecard-<run>.json` |
| Scorecard tests | `internal/cli/scorecard_test.go` — `TestEvaluateScorecardGates` (15 cases), `TestScorecardJSONRoundTrip` (`reflect.DeepEqual`), `TestScorecardJSONOmitEmptyGates`, `TestWriteScorecard` (`t.TempDir`), `TestParseBaselineScorecardShape`, `nearlyEqual` epsilon helper, `captureRunStdout`/`captureRunStderr` helpers (in `app_test.go`, not `scorecard_test.go`) |
| Cost plumbing | `internal/cli/cost.go` — `recordRunSpend(root, start, mode, provider, model, pre)` (`cost.go:500`) appends to `.pi/cost-ledger.jsonl` (gitignored); mode comment at `cost.go:84-85` lists `chat|print|resume|backfill`; **`"benchmark"` mode is already written** by `scorecard.go:413` (spec §4.7 of the scorecard design) but not yet documented in the mode comment; `--max-budget-usd` / `PI_MAX_BUDGET_USD` cap with exit 6 (`cost.go:20`, `internal/cli/app.go:48,68`) |
| Feature surfaces (behavioral content for harness-routing tasks) | `internal/cli/mcp.go` (LOCAL-ONLY READ-ONLY JSON-RPC 2.0 stdio server, protocol `2025-03-26`, tools `providers|cost|benchmark_dry_run`); `internal/cli/otel.go` (env-gated `PI_OTLP_ENDPOINT`, one "invoke_agent" span per launch, best-effort — single warning channel `otelExportWarning`); `internal/cli/providers.go` + `app.go` (`--model-tier fast|balanced|cheap`, strict no-fallback, conflict with `--model` is a usage error exit 2); `internal/cli/project_understand.go` (deterministic `product.md`/`tech.md`/`structure.md`, no network/LLM) |
| Deps policy | `eval/requirements.txt` — policy comment requires deliberate deps and co-updating the README dependency-policy line; `deepeval~=4.1`, `pytest~=9.1`, `python-dotenv~=1.2`, `pypdf~=6.15`, `pytest-timeout~=2.4`. No `pytest-json-report` today |
| Gitignore | `eval/benchmark-results/` ignored (`eval/.pytest_cache/` too); `eval/baselines/`, `eval/live-results/`, `eval/results/` not yet present — `eval/baselines/` must be committed, `eval/live-results/` must be ignored |

## 3. Scope

### In scope

1. **Dataset v2** — `eval/datasets/coding_samples.jsonl` 5 → 20 with schema v2
   (`category`, `difficulty`, `grader`, `graderRef`, `reference`, `tags` incl.
   `regression-<thing>`); balanced category budget (§4.2); every task ships a
   reference solution that provably passes its grader; a new dataset-schema
   lint test (§6).
2. **Nightly gate** — `nightly-live-eval.yml` rewritten as a two-job split
   (deterministic + live): `timeout-minutes: 30`; `@pytest.mark.timeout(120)`
   on live tests; parametrized live suite over the dataset with
   `EVAL_RUNS_PER_CASE=3`; `eval/scripts/score_run.py` + `pytest-json-report`;
   per-case baseline gate (`eval/baselines/live-baseline.json`, tolerance
   0.05); cost-per-task regression >2× baseline fails; artifacts 90-day
   retention; `PI_MAX_BUDGET_USD` cap; key-missing is a **hard fail** with an
   explicit `::error::` (deterministic subset still runs).
3. **Cost-per-task** — extend the scorecard + nightly JSON schemas with
   `costPerTaskUsd`, `costPerSuccessfulTaskUsd` (guard div-by-zero),
   `tokensPerTask`, `judgeCostUsd`, `agentCostUsd`; add `live-eval` ledger mode
   (+ document the existing `benchmark` mode) and a `--cost-mode` flag for
   CI-tagged runs; pick the judge-cost single source of truth (§4.5).
4. **Unified task governance** — `eval/datasets/tasks.json` manifest
   referencing JSONL live tasks + `eval/benchmarks/*` seeds with shared
   categories; extend `test_benchmark_format.py` to validate the manifest; add
   `category`/`difficulty`/`grader` to the benchmark `task.json` schema.
5. **Live metrics layer** — `eval/tests/test_live_metrics.py` consuming the
   existing (dead) `sample_cases` fixture: `TaskCompletionMetric` + a custom
   G-Eval rubric for code tasks + a deterministic fast lane reusing
   `CodeQualityMetric`. Nightly-only.

### Explicitly OUT of scope

- **Dashboard / trend store** — artifact-chained history (90-day retention) is
  the trend signal for v1; a durable trend store or dashboard is future work
  (flagged in the hardening-lane research as out of scope).
- **Parallel providers** — live jobs run one provider sequentially; parallel
  execution and its cost-attribution implications are a later iteration.
- **Docker seeds in the nightly** — Docker stays weekly (`provider-scorecard.yml`);
  an optional future `docker-smoke` tier is documented, not built (§4.6).
- **Deterministic-run fixes in Go** — the `time.Now().UTC()` determinism
  blocker in `buildScorecard`/`scorecardRunID` (`scorecard.go:648,673`) and the
  `scorecard-latest.json` shell-glue test gap are owned by the
  deterministic-eval-hardening lane.
- **Hermetic contract tests for mcp/otel/model-tier/project-understand** — those
  live in the deterministic-eval-hardening spec (§4.1). This spec only consumes
  them in the deterministic job's test selection and adds *behavioral* content
  via the harness-routing dataset category.
- **Cross-machine baseline store / auto re-baselining** — baselines are
  deliberate, reviewed files (§4.4); the nightly never updates its own
  baseline.
- **README / CHANGELOG edits and the `providers.json` / seed edits** — owned by
  the parent (docs pass, §9).

## 4. Design

### 4.1 Dependency surface — what this spec takes from the hardening lane

**This spec must be independently implementable once the
deterministic-eval-hardening spec lands.** It consumes exactly two things from
that lane:

1. **The CI build step for eval jobs** — the `go build -o bin/pi-run
   ./cmd/pi-run` + `echo "$GITHUB_WORKSPACE/bin" >> "$GITHUB_PATH"` precedent
   (`provider-scorecard.yml:29-32`), applied by the hardening spec to the
   `python-quick` CI job so binary-dependent pytest tests are not latent
   (`ci.yml:30-47` runs pytest without building pi-run today). The rewritten
   nightly reuses the same step.
2. **The hermetic Python contract tests** — `test_mcp_server.py`,
   `test_otel.py`, `test_model_tier.py`, `test_project_understand.py` per the
   hardening spec. The nightly **deterministic job** adds them to its test
   selection; this spec does not define them.

Nothing else is imported: dataset work, `score_run.py`, the cost schema
additions, the manifest, and `test_live_metrics.py` are all defined here. If
the hardening lane changes shape, only §4.4's deterministic-job test list and
the build-step reference in this spec need adjusting.

### 4.2 Dataset schema v2 + category budget (decision 1)

**Decision:** grow `eval/datasets/coding_samples.jsonl` 5 → 20 with schema v2,
balanced across six categories, every task shipping a reference solution that
provably passes its grader.

**Schema v2** (additive over v1's `id/input/expected_output/context/tags`):

| Field | v1 | v2 | Type | Required | Semantics |
|---|---|---|---|---|---|
| `id` | ✓ | ✓ | string | yes | unique task id (`coding-001` … `coding-020`) |
| `input` | ✓ | ✓ | string | yes | the prompt handed to `pi-run print` |
| `expected_output` | ✓ | ✓ | string | yes | reference answer; kept for backward compat and used by judge graders / simple matchers |
| `context` | ✓ | ✓ | string | no | extra context string |
| `tags` | ✓ | ✓ | array\<string\> | yes (was no) | now required; must include at least one `regression-<thing>` tag naming the past failure this task guards (Autonoma convention) |
| `category` | — | ✓ | enum | yes | `code-gen` \| `bug-fix` \| `shell/ops` \| `concept` \| `negative-edge` \| `harness-routing` |
| `difficulty` | — | ✓ | enum | yes | `easy` \| `medium` \| `hard` |
| `grader` | — | ✓ | enum | yes | `deterministic` \| `judge` |
| `graderRef` | — | ✓ | path | deterministic tasks | relative path to an executable grader under `eval/datasets/graders/<id>/grade.py` (exit 0 = pass); omitted for `judge` tasks |
| `reference` | — | ✓ | path | yes | relative path to the reference solution under `eval/datasets/references/<id>/` that provably passes the grader (validated by the lint test, §6) |

Backward compatibility: `load_dataset` / `sample_cases` (`conftest.py:18-54`)
read only `id/input/expected_output/context`, so existing consumers keep
working; the lint test enforces that every record carries the full v2 field set
so partial-migration states fail in CI.

**Category budget** (target 20 total):

| Category | Budget | Kept from v1 | New | Notes |
|---|---|---|---|---|
| code-gen | 3–4 | `coding-001` (factorial), `coding-005` (JS debounce) | 1–2 | new ones get **executable** deterministic graders (run the emitted code against hidden tests), not string heuristics |
| bug-fix | 3–4 | — | 3–4 | print-mode-able siblings of benchmark seeds (`fix-divide-by-zero`, `fix-json-parsing` style, from `eval/benchmarks/*/solution/`) |
| shell/ops | 2–3 | `coding-003` (bash `find`) | 1–2 | agent must emit a shell one-liner / small script; grader runs it |
| concept/Q&A | 3 | `coding-002` (`is` vs `==`), `coding-004` (binary search) | 1 | judge-graded; deterministic graders are the exception here |
| negative/edge | 3–4 | — (currently **zero**) | 3–4 | refuse ambiguous requests, hallucination guards, input-validation tasks — the biggest coverage hole today |
| harness-routing | 2–3 | — | 2–3 | tasks whose correct answer exercises `--model-tier` / MCP / `PI_OTLP_ENDPOINT` **behaviorally** (e.g. "start a stub OTLP receiver and confirm pi-run print emits exactly one `invoke_agent` span and still exits 0 when export fails"; "verify `--model-tier cheap` resolves to the cheap model in the ledger") — turns the uncovered features into eval content, not just unit tests |

Ranges sum to 16–21; the target is 20 (Anthropic's 20–50 floor, DeepEval's
"~100 max, start small", and the practical 20-task harnesses).

**Sources for new tasks** (in priority order):

1. **Past real failures** — tier-resolution bugs, MCP protocol edge cases,
   OTel best-effort contract (each becomes a `regression-<thing>` tag).
2. **Benchmark seed solutions** — harvest `eval/benchmarks/*/solution/` dirs
   for print-mode-able siblings.
3. **Exercism micro-problems** — aider's source; small, unambiguous, easily
   graded.
4. **Adversarial / negative cases** — hallucination guards, input validation,
   ambiguous-request refusal.

**Reference-solution rule:** no task lands without a `reference` whose grader
passes. The lint test (and the grader harness, §6) enforces this hermetically —
mirroring Terminal-Bench's "oracle solutions run 5× to validate every task".

### 4.3 Nightly workflow architecture (decisions 2, 3, 6)

**Decision 2 (nightly mechanics):** rewrite `nightly-live-eval.yml` as a
two-job split with an explicit gate and artifacts.

**Decision 3 (key-missing):** a missing provider key is a **hard fail** with an
explicit `::error::` message, never a silent skip; the deterministic subset
always runs (that is why the jobs are split).

**Decision 6 (Docker):** Docker stays weekly (`provider-scorecard.yml`
unchanged). An optional future `docker-smoke` tier (2 cheapest seeds nightly,
`--runs 3`, `--quick-profile`, infra failures non-blocking) is documented in
§4.6 but **not in v1**.

Job split:

```yaml
name: nightly-live-eval
on:
  schedule: [{ cron: "0 3 * * *" }]
  workflow_dispatch: {}
concurrency: { group: nightly-live-eval, cancel-in-progress: true }
permissions: { contents: read }

jobs:
  deterministic:        # always runs; no provider key needed
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - checkout, setup-go, setup-python (as today)
      - Build pi-run            # hardening-lane step: go build -o bin/pi-run ./cmd/pi-run + $GITHUB_PATH
      - Install deps            # eval/.venv + pip install -r eval/requirements.txt
      - Run deterministic suite:
          eval/.venv/bin/python -m pytest tests/test_harness_config.py tests/test_secret_resolution.py \
            tests/test_dataset_schema.py tests/test_benchmark_format.py tests/test_code_quality.py \
            tests/test_mcp_server.py tests/test_otel.py tests/test_model_tier.py \
            tests/test_project_understand.py -v   # contract tests from the hardening lane

  live:                 # key-missing => HARD FAIL (explicit ::error::), never silent
    runs-on: ubuntu-latest
    timeout-minutes: 30
    needs: deterministic
    env:
      EVAL_RUNS_PER_CASE: "3"
      PI_MAX_BUDGET_USD: "2"      # nightly spend cap (cheap tier; see below)
      OPENAI_MODEL_NAME: "gpt-5.1-mini"   # cheap judge — deepeval 4.1.7 reads OPENAI_MODEL_NAME, NOT DEEPEVAL_MODEL (which this repo only prints as info)
      OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    steps:
      - checkout, setup-go, setup-python, Build pi-run (as deterministic)
      - Set up Node via nvm + install the Pi CLI   # provider-scorecard.yml:42-44 precedent
      - name: Key presence gate          # decision 3 — hard fail, not skip
        run: |
          if [ -z "$OPENAI_API_KEY" ]; then
            echo "::error::nightly-live-eval: OPENAI_API_KEY is missing — live eval cannot run." \
                 "A skipped eval is indistinguishable from a passed one; refusing to report green."
            exit 1
          fi
      - Run live suite (N runs per case):
          eval/.venv/bin/python -m pytest tests/test_live_suite.py tests/test_live_metrics.py \
            tests/test_agent_task_completion.py -v --json-report-file=eval/live-results/report.json (create `eval/live-results/` first — `mkdir -p` in the job, gitignored)
        env:
          EVAL_RUNS_PER_CASE: "3"
      - Run baseline gate:
          eval/.venv/bin/python eval/scripts/score_run.py \
            --report eval/live-results/report.json \
            --baseline eval/baselines/live-baseline.json \
            --tolerance 0.05 --runs 3 --out eval/live-results/live-$(date +%Y%m%dT%H%M%S).json
      - Upload live results:
          uses: actions/upload-artifact@v4
          with: { name: live-eval-results, path: eval/live-results/, retention-days: 90 }
```

**Gate math.** Per case, the suite runs each task `EVAL_RUNS_PER_CASE=3`
times (5 max). The gate reads the pytest-json-report JSON and rebuilds:

- `passRate(case) = mean over runs` (a run passes/fails binary, so the mean is
  the proportion — the correct statistic for a pass-rate gate; a median over 3
  binary runs collapses to 0/1 and is rejected for the rate gate);
- `costPerTaskUsd`, `latencyMs`, `tokensPerTask` = **median over runs**
  (robust to outliers, matching the scorecard's `--runs` median semantics,
  scorecard spec §4.3).

Regression (gate failure, exit 1): any case with
`currentPassRate < baselinePassRate − 0.05` (per-case tolerance), OR any case
with `costPerTaskUsd > 2 × baselineCostPerTaskUsd` (§4.5), OR total run spend
≥ `PI_MAX_BUDGET_USD` (budget, exit 6 semantics surfaced as a gate failure in
`score_run.py`), OR the run is incomplete (a case errored — infra, not a
quality signal). New cases absent from the baseline are reported as
`unbaselined` and do not fail the gate (they cannot regress against nothing).

**Baseline file** — `eval/baselines/live-baseline.json` (committed):

```json
{
  "schemaVersion": 1,
  "generated": "2026-08-12T00:00:00Z",
  "runsPerCase": 3,
  "agentModel": "openai/gpt-5.6-mini",
  "judgeModel": "openai/gpt-5.1-mini",
  "cases": {
    "coding-001": { "passRate": 1.0, "costPerTaskUsd": 0.0112, "tokensPerTask": 3980, "latencyMs": 14100 },
    "coding-002": { "passRate": 1.0, "costPerTaskUsd": 0.0008, "tokensPerTask": 412,  "latencyMs": 2100 }
  }
}
```

Re-baselining is a **deliberate, reviewed act** (the scorecard principle,
scorecard spec §4.5): `score_run.py --update-baseline` from a green manual run,
reviewed in a PR. The nightly never updates its own baseline, so regressions
cannot self-heal. A new dataset (or a model switch) triggers an explicit
re-baseline PR.

**Budgeting for cost.** The live job pins the judge via `OPENAI_MODEL_NAME` (the knob deepeval 4.1.7 actually reads: `model = model or settings.OPENAI_MODEL_NAME`; a bare `gpt-5.1-mini`, no `openai/` prefix) and
passes `--model-tier cheap` to the agent runs (`run_pi_print`'s existing
`extra_args`, `conftest.py:63-90`), with `PI_MAX_BUDGET_USD=2` as the nightly
cap. 20 tasks × 3 runs × (cheap agent + cheap judge) lands in low single-digit
USD/night (cf. ttxs69's 20×3 ≈ $50–100 at frontier prices — the cheap tier is
what makes nightly affordable).

**Artifacts.** `eval/live-results/` (gitignored, new ignore rule) holds the
pytest-json-report, the `score_run.py` summary JSON, and the
`$GITHUB_STEP_SUMMARY` markdown; uploaded with 90-day retention (mirrors
`provider-scorecard.yml`'s chained-artifact pattern) so trends can be derived
without a dashboard.

### 4.4 score_run.py behavior

`eval/scripts/score_run.py` — a stdlib-only Python script (the Autonoma
`run_evals.py` port):

```
usage: score_run.py --report <pytest-json-report.json> --baseline <live-baseline.json>
                    [--tolerance 0.05] [--runs 3] [--budget-usd 2.0] [--out <path>]
                    [--update-baseline] [--allow] [--json-summary <path>]
```

Behavior, in order:

1. **Parse** the pytest-json-report JSON (`report.tests[]`); recover the case id
   from each parametrized nodeid (`tests/test_live_suite.py::test_case[coding-001]`
   → `coding-001`) and read per-run `properties` (`pass`, `costUsd`,
   `judgeCostUsd`, `tokens`, `latencyMs`) recorded by the suite via
   `record_property`.
2. **Rebuild per-case aggregates**: `passRate` = mean over runs; cost /
   latency / tokens = median over runs (gate-math §4.3).
3. **Compute totals**: `overallPassRate`, `totalCostUsd = Σ(agent + judge)`,
   `costPerTaskUsd = totalCostUsd(agent + judge) / nCases` — **all-in** (differs from the Go scorecard's agent-only `costPerTaskUsd`; the two surfaces must not be compared directly; report agent-only as a separate `agentCostPerTaskUsd` field if cross-surface comparison is ever wanted), `costPerSuccessfulTaskUsd =
   totalCostUsd / nPassed` (**guarded: 0.0 when `nPassed == 0`**, never a
   division error), `tokensPerTask`.
4. **Compare vs baseline**: per-case regression if `passRate < baseline −
   tolerance`; cost regression if `costPerTaskUsd > 2 × baseline`; unbaselined
   cases recorded, not failed; incomplete runs (fewer than `--runs` completed
   per case, or a case errored) fail the run as incomplete.
5. **Emit**: write the summary JSON to `--out` (default
   `eval/live-results/live-<run>.json`), and — when `GITHUB_STEP_SUMMARY` is
   set — a markdown table of per-case current / baseline / delta with
   regressions flagged.
6. **Exit**: `0` pass · `1` any gate failure (regression, cost regression,
   incomplete, budget) · `2` usage error.

`--update-baseline` rewrites `eval/baselines/live-baseline.json` from the
current report and **requires `--allow`** (guards the deliberate-rebaseline
rule). The script is fully hermetic (fixture JSONs in tests; no keys, no
network).

### 4.5 Cost-per-task metrics (decision 4)

**Decision:** add cost-per-task metrics to both the scorecard and the nightly
run JSON; add the `live-eval` ledger mode (and document the already-shipped
`benchmark` mode); **the judge-cost single source of truth is DeepEval's
`metric.evaluation_cost`** (deepeval 4.1.7: `metrics/base_metric.py:67,109-112`, accrued from the judge model's `GenerationCost`; reset per case), summed per case by the metric layer — not a
separate ledger.

**Schema additions.** `scorecardProvider` (`internal/cli/scorecard.go:52-62`)
gains:

| Field | JSON | Type | Semantics |
|---|---|---|---|
| `costPerTaskUsd` | `costPerTaskUsd` | float64 | `costUsd / total` — **agent-only** (Docker grading has no judge; do NOT compare this directly with the nightly's all-in figure) |
| `costPerSuccessfulTaskUsd` | `costPerSuccessfulTaskUsd` | float64 | `costUsd / passed` — **`0` when `passed == 0` (div-by-zero guard)** |
| `tokensPerTask` | `tokensPerTask` | int | `tokens / total` |
| `agentCostUsd` | `agentCostUsd` | float64 | the existing per-provider `costUsd` (ledger real spend; keeps a stable name for the split) |
| `judgeCostUsd` | `judgeCostUsd,omitempty` | float64 | 0/omitted in the Go scorecard (Docker grading is deterministic — no judge); real in the nightly run JSON |

The nightly run JSON (`eval/live-results/live-<run>.json`, written by
`score_run.py`) carries the same fields per case plus `judgeCostUsd` and
`agentCostUsd` split, per §4.4.

**Ledger modes.** `recordRunSpend(root, start, mode, provider, model, pre)`
(`cost.go:500`) already accepts an arbitrary mode string and `"benchmark"` is
already written by `scorecard.go:413`. This spec:

- documents `benchmark` in the mode comment (`cost.go:84-85`), and
- adds `live-eval` as a documented mode, with a **`--cost-mode <mode>` flag on
  `chat`/`print`** (default: the command name, i.e. today's behavior unchanged)
  so the nightly can tag its agent runs `--cost-mode live-eval`. Validation:
  mode ∈ {`chat`, `print`, `resume`, `backfill`, `benchmark`, `live-eval`}
  (usage error exit 2 otherwise); the flag is inert except for the mode string
  passed to `recordRunSpend`. This keeps attribution explicit without a second
  cost-ingestion path.

**Judge-cost single source of truth — decision and justification.** Judge
calls happen **inside the pytest process** (DeepEval) and are invisible to the
session-file ledger that `cost.go` reads (`.pi/sessions/` usage records) — a
separate ledger entry for judge cost would require new cross-language plumbing
(Go CLI invocation from pytest or a shared file format) and would create a
second source of truth to keep in sync with DeepEval's own accounting. DeepEval
already reports real per-metric cost via `metric.evaluation_cost` on each metric instance,
priced from the same USD-per-token config the suite is
pinned to via `OPENAI_MODEL_NAME`. **Therefore `metric.evaluation_cost` is the single source of truth.** Caveat: cost accrues only when the judge model has known pricing or `OPENAI_COST_PER_INPUT_TOKEN`/`OPENAI_COST_PER_OUTPUT_TOKEN` are set (`openai_model.py:80-83`) — otherwise judge cost is silently 0 and must not be trusted; the nightly must assert judge cost > 0 for a judge-graded case or record the config gap.
(summed per case by `test_live_metrics.py` and attached to the case's JSON via
`record_property`) is the single source of truth for `judgeCostUsd`.**
Invariant preserved from the cost spec: **agent spend is real ledger data
(`usage.cost.total`), judge spend is real DeepEval-reported data — nothing is
ever estimated.**

**Gates.** (1) per-run budget — exists today (`--max-budget-usd`, exit 6,
`app.go:48,68`); the nightly sets `PI_MAX_BUDGET_USD` (already honored by
`resolveBudgetCap`). (2) **cost-per-task regression: `costPerTaskUsd > 2×
baseline` fails** (`score_run.py` for the nightly; the same rule slots into the
scorecard gate logic when a baseline exists). (3) `tokensPerTask` is recorded
as a provider-agnostic trend proxy (Anthropic: track cost/token on a static
bank of tasks); not gated in v1.

### 4.6 Unified task governance — the manifest (decisions 5, 6)

**Decision 5:** one registry, two execution surfaces — governance is unified,
execution is not.

`eval/datasets/tasks.json` (committed, schema-versioned):

```json
{
  "schemaVersion": 1,
  "categories": ["code-gen", "bug-fix", "shell/ops", "concept", "negative-edge", "harness-routing"],
  "surfaces": {
    "live":      { "kind": "jsonl",     "path": "eval/datasets/coding_samples.jsonl" },
    "benchmark": { "kind": "directory", "path": "eval/benchmarks" }
  },
  "tasks": [
    { "id": "coding-001",        "surface": "live",      "category": "code-gen",    "difficulty": "easy",   "grader": "deterministic", "tags": ["python", "math", "regression-factorial"] },
    { "id": "fix-divide-by-zero", "surface": "benchmark", "category": "bug-fix",     "difficulty": "easy",   "grader": "deterministic", "tags": ["python"] },
    { "id": "coding-012",        "surface": "live",      "category": "harness-routing", "difficulty": "medium", "grader": "deterministic", "tags": ["regression-otel-best-effort"] }
  ]
}
```

Rules enforced by the extended `test_benchmark_format.py` (§6):

- every record in `coding_samples.jsonl` appears in `tasks[]` with
  `surface: "live"`, and vice versa (bijective);
- every `eval/benchmarks/*/task.json` id appears with `surface: "benchmark"`
  (bijective);
- `category` ⊆ the shared taxonomy, `difficulty` ∈ {easy, medium, hard},
  `grader` ∈ {deterministic, judge};
- no duplicate ids; `tags` non-empty on live tasks.

**Benchmark task.json schema extension:** each `eval/benchmarks/*/task.json`
(which today has only `id/prompt/timeoutSecs/testScript/solution`) gains
`category`, `difficulty`, and `grader: "deterministic"` — the shared taxonomy
makes live vs benchmark scores comparable and is what makes the single manifest
meaningful.

**Why not merge execution (decision 6):** benchmark seeds need file mutation +
`tests/run.sh` verification inside Docker — `pi-run print` cannot do that, and
Docker-on-GH-runners is a documented flakiness source (runner-images #13746,
#11786) with dominant inter-runner variance. The weekly scorecard already
absorbs that noise with artifact-chained baselines + tolerance; nightly would
inherit it without cost justification. Hence: nightly = print-mode live suite +
`--benchmark-dry-run` (hermetic format gate, already in `test_benchmark_format.py`);
Docker = weekly, unchanged.

**Future `docker-smoke` tier (documented, NOT built in v1):** 2 cheapest seeds
nightly with `--runs 3` median gate + `--quick-profile` (60 s task timeout,
scorecard spec §4.2) + a Docker-setup retry step; infra failures (exit 7)
classified as "infra", not "quality". Revisit after the weekly scorecard has
accumulated real variance data.

### 4.7 Live metrics layer (decision 7)

`eval/tests/test_live_metrics.py` — consumes the existing **dead** `sample_cases`
fixture (`conftest.py:38-54`), parametrized over the dataset, three metrics:

1. **`TaskCompletionMetric`** (DeepEval) — "works almost everywhere" for agent
   evals; the generic completion signal.
2. **A custom G-Eval rubric for code tasks** — criteria per `category`
   (code-gen: correctness + idiomatic; bug-fix: root-cause addressed + no
   regression; shell/ops: behaves per prompt), `@pytest.mark.timeout(120)`.
3. **Deterministic fast lane** — reuse `CodeQualityMetric`
   (`test_code_quality.py:11-69`) as the cheap deterministic check on code
   outputs.

Nightly-only (the whole live job is where key-missing is a hard fail §4.3;
within-suite `skipif` guards only genuinely optional bits such as a
non-default provider key). This is the *judged* layer: `judgeCostUsd` is
collected here from `metric.evaluation_cost` (§4.5).

## 5. Implementation Plan

1. **Dataset v2** — rewrite `eval/datasets/coding_samples.jsonl` to 20 records
   (schema §4.2, budget §4.2, sources §4.2) with
   `eval/datasets/graders/<id>/grade.py` + `eval/datasets/references/<id>/` for
   every task; add the dataset-schema lint test.
2. **Grading harness** — shared `eval/grader.py` invoked by the lint test and
   the live suite: runs `grade.py` (or judge path) against printed output; the
   reference-solution-provable-pass rule is enforced hermetically.
3. **Live suite** — `eval/tests/test_live_suite.py` (parametrized over the
   dataset, `@pytest.mark.timeout(120)`, `EVAL_RUNS_PER_CASE` repeats,
   `record_property` per-case stats) + `test_live_metrics.py` (§4.7).
4. **Baseline + scorer** — `eval/baselines/live-baseline.json` bootstrap via a
   manual green run; `eval/scripts/score_run.py` (§4.4) + its unit tests;
   `pytest-json-report~=1.5` added to `eval/requirements.txt` with the policy
   comment updated (and the README dependency line — parent-owned, §9).
5. **Cost schema** — `internal/cli/scorecard.go` `scorecardProvider` gains the
   §4.5 fields (div-by-zero guard); `cost.go` mode comment + `live-eval` mode
   + `--cost-mode` flag on `chat`/`print`; extend `scorecard_test.go` and
   `cost_test.go`.
6. **Manifest** — `eval/datasets/tasks.json`; extend `test_benchmark_format.py`
   to validate it (bijections, taxonomy, schema extension fields on benchmark
   task.json).
7. **Nightly workflow** — rewrite `nightly-live-eval.yml` (§4.3); add
   `eval/live-results/` to `.gitignore`; commit `eval/baselines/`.
8. **Docs** — parent-owned README/CHANGELOG (see §9).

## 6. Tests (exact list)

All hermetic unless marked *live* (nightly-only). The `captureRunStdout` /
`captureRunStderr` / `nearlyEqual` / `t.TempDir` / fake-`bw_get` patterns are
already established in `scorecard_test.go` and `test_secret_resolution.py`.

**Python (eval/tests/):**

| Test file | Kind | What it asserts |
|---|---|---|
| `test_dataset_schema.py` | NEW, hermetic | every JSONL record: required v2 fields, `category` ∈ taxonomy, `difficulty` ∈ set, `grader` ∈ {deterministic, judge}, `tags` non-empty with a `regression-<thing>` tag, unique ids; `reference` file exists and **provably passes** the deterministic grader (runs the grader against the reference output); `graderRef` present iff `grader == deterministic` |
| `test_score_run.py` | NEW, hermetic | fixture pytest-json-report + baseline JSONs: aggregation math (mean pass rate, median cost/latency/tokens); tolerance boundary (`baseline − 0.05` exact edge); cost >2× baseline regression; `costPerSuccessfulTaskUsd` div-by-zero guard (`passed == 0` → 0.0); incomplete-run detection; `--update-baseline --allow` output shape; exit codes 0/1/2 |
| `test_benchmark_format.py` | UPDATED | manifest validation: JSONL ↔ `tasks[]` bijection, benchmark ids ↔ `tasks[]` bijection, shared taxonomy, no dupes; benchmark `task.json` now requires `category`/`difficulty`/`grader` |
| `test_live_suite.py` | NEW, *live* | parametrized over `grader == deterministic` cases; `@pytest.mark.timeout(120)`; runs each case `EVAL_RUNS_PER_CASE` times; records per-case `pass/costUsd/judgeCostUsd/tokens/latencyMs` via `record_property` |
| `test_live_metrics.py` | NEW, *live* | consumes `sample_cases`; `TaskCompletionMetric` + custom G-Eval rubric (per-category criteria) + deterministic fast lane (`CodeQualityMetric`); sums `metric.evaluation_cost` per case (§4.7) |

**Go (internal/cli/):**

| Test file | Kind | What it asserts |
|---|---|---|
| `scorecard_test.go` | UPDATED, hermetic | `costPerTaskUsd`/`costPerSuccessfulTaskUsd`/`tokensPerTask` aggregation math (fixtures via the existing `scorecardProviderFromRunAggregation` pattern); div-by-zero guard; `TestScorecardJSONRoundTrip` (reflect.DeepEqual) and `TestScorecardJSONOmitEmptyGates` extended to the new fields (`judgeCostUsd` omitted where N/A) |
| `cost_test.go` | UPDATED, hermetic | `recordRunSpend` mode `"live-eval"` writes a ledger entry with that mode; `--cost-mode` parsing on `chat`/`print`; unknown mode → exit 2; default mode unchanged when flag absent |

**CI:** `nightly-live-eval.yml` (rewritten per §4.3) — the deterministic job
must go green without any provider key; the live job must emit
`::error::` + exit 1 when `OPENAI_API_KEY` is absent (tested once manually).

## 7. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Live agent runs are expensive on a nightly schedule | High | Cheap tier pinned (`OPENAI_MODEL_NAME` judge + `--model-tier cheap` agent), `PI_MAX_BUDGET_USD=2` cap, `timeout-minutes: 30`; 20×3 at cheap prices is low single-digit USD/night |
| Single-run gating makes the nightly flaky → "disable it" | Medium | `EVAL_RUNS_PER_CASE=3` mean pass-rate + median cost, per-case baseline tolerance 0.05; incomplete runs fail loudly rather than silently skew |
| Missing key silently skips the eval (current anti-pattern) | High (today) | Job-split + explicit `::error::` hard fail (§4.3); deterministic job independent of key presence |
| New tasks with broken graders (pass@100 = 0) poison the gate | Medium | Reference-solution rule + `test_dataset_schema.py` proving the reference passes its grader before merge (Terminal-Bench oracle-validation practice) |
| Dataset drift / category imbalance sneaks in | Medium | Manifest bijection + taxonomy lint in `test_benchmark_format.py`; dataset is a versioned, PR-reviewed asset |
| Cost attribution muddied (agent vs judge, concurrent runs) | Medium | Agent = ledger real spend (`--cost-mode live-eval`); judge = DeepEval `metric.evaluation_cost` (single source of truth, §4.5); sequential runs keep attribution clean |
| Baseline staleness after model/task changes | Medium | Re-baselining is deliberate (`--update-baseline --allow`, reviewed PR); nightly never self-heals |
| `pytest-json-report` adds a dep | Low | Small, widely used; goes through the requirements policy comment + README dependency line per repo convention |
| Hard-fail key gate bricks the nightly on secret rotation | Low-Med | Error message names the secret and the fix; the deterministic job still runs, so CI is never fully red on secret problems |
| Docker seeds creep into nightly scope | Low | Explicitly out of scope + documented future `docker-smoke` tier (§4.6) |

## 8. Decision

**Recommend proceeding.** Every pattern in this spec is the 2026 consensus for
agent-eval harnesses: a small (20-task) balanced, deterministic-first dataset
with reference solutions (Anthropic 20–50; DeepEval coverage-over-volume);
a two-speed nightly with N-run averaging against per-case baselines and a
tolerance gate (Autonoma); cost-per-task as a first-class metric with a real
spend ledger and a deliberate judge-cost source of truth; and governance
unification (one manifest, shared categories) without forcing Docker seeds into
the nightly. The dependency surface on the deterministic-eval-hardening lane is
narrow and explicit (CI build step + contract tests, §4.1), and everything else
is implementable independently. The single new CLI surface is the small
`--cost-mode` flag; the rest is dataset content, one scoring script, two new
pytest files, schema extensions to existing structures, and one workflow
rewrite — all testable hermetically.

## 9. Docs Pass (parent-owned — NOT edited here)

- **README:** dependency-policy line for `pytest-json-report`; a short
  "Nightly live eval" note (dataset, gate, budget) — the existing
  "Evaluations" section pointer. Do NOT edit README in this lane.
- **CHANGELOG:** v0.9.0 entry (dataset v2, nightly gate, cost-per-task).
- **`.gitignore`:** add `eval/live-results/` (this lane may add it since it is
  a build artifact of the new nightly; coordinate with the parent).
- **This spec's review gate:** §Review checklist below.

## Review checklist

A reviewer can verify the implementation against this spec by checking:

- [ ] **Dataset v2:** `coding_samples.jsonl` has exactly 20 records, all
      carrying `category`/`difficulty`/`grader`/`reference` and a
      `regression-<thing>` tag; the category budget in §4.2 is respected
      (negative/edge and harness-routing present — no longer zero); every
      `reference` provably passes its grader (`test_dataset_schema.py`).
- [ ] **Nightly job split:** `.github/workflows/nightly-live-eval.yml` has a
      `deterministic` job that runs with no provider key and a `live` job with
      `timeout-minutes: 30`; the live job hard-fails with an explicit
      `::error::` when the key is missing — no `skipif` path on the key gate.
- [ ] **Gate math:** `score_run.py` rebuilds mean pass-rate and median
      cost/latency/tokens from the pytest-json-report; per-case regression =
      `passRate < baseline − 0.05`; cost regression = `>2× baseline`;
      `costPerSuccessfulTaskUsd` guards division by zero; exits 0/1/2.
- [ ] **Cost schema:** `scorecardProvider` gains `costPerTaskUsd`,
      `costPerSuccessfulTaskUsd`, `tokensPerTask`, `agentCostUsd`,
      `judgeCostUsd` (omitempty); `recordRunSpend` documents `benchmark` and
      adds `live-eval`; `--cost-mode` validated (exit 2 on unknown); judge
      cost's single source of truth is `metric.evaluation_cost` — no
      second ledger exists for judge spend.
- [ ] **Manifest:** `eval/datasets/tasks.json` passes the extended
      `test_benchmark_format.py` (JSONL ↔ tasks[] ↔ benchmark dir bijections,
      shared taxonomy, no dupes); every benchmark `task.json` carries
      `category`/`difficulty`/`grader`.
- [ ] **Live metrics:** `test_live_metrics.py` consumes `sample_cases`
      (the dead fixture is now alive) with TaskCompletion + G-Eval rubric +
      deterministic fast lane; nightly-only.
- [ ] **Scope discipline:** Docker seeds are NOT in the nightly; no dashboard,
      no parallel providers, no cross-machine baseline store; the hardening
      lane's determinism fixes are not duplicated here; README/CHANGELOG
      untouched by this lane.
- [ ] **Hermetic tests pass with no keys and no network** (`go test
      ./internal/cli/`, `pytest eval/tests/test_dataset_schema.py
      eval/tests/test_score_run.py eval/tests/test_benchmark_format.py`). `test_score_run.py` is hermetic (fixture JSON only) and MUST be in the deterministic job's pytest list so its unit tests execute in CI, not just in the live job.
