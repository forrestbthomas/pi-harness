> **pi-harness** — a self-healing, measurable, provider-agnostic coding-agent harness that keeps you out of AI-vendor lock-in.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org/dl/)
[![CI](https://github.com/forrestbthomas/pi-harness/actions/workflows/ci.yml/badge.svg)](https://github.com/forrestbthomas/pi-harness/actions/workflows/ci.yml)

# Pi Coding Agent Harness

A self-healing, measurable coding-agent harness built around the [Pi coding agent](https://pi.dev/) and the open-source [DeepEval](https://github.com/confident-ai/deepeval) LLM evaluation framework. The harness runtime is driven by a single Go CLI, **`pi-run`** — the one source of truth for provider routing, API key resolution, Pi launching, evaluation, setup, and health checks.

> **What this project is, and what it is not** — see [`CHARTER.md`](CHARTER.md) for the boundary contract. One product: the harness. The eval suite is its measurement layer (the "measurable" in the star), not a separate product today; a standalone benchmark repo (pi-bench) is a triggered future split, not current scope. We do not build a PM system, a spec library, an observability platform, or a general MCP platform as product surface.

## What You Get

- **`pi-run` CLI** — a compiled Go binary (repo `bin/pi-run`) that owns the harness runtime: `chat`, `print`, `resume`, `cost`, `ci-benchmark`, `eval`, `config-check`, `doctor`, `setup`, `install`, `clean`, `project-understand`, `self-heal`, `providers`, `hooks`, `version`.
- **Pi CLI** installed globally (via nvm) and configured for this project.
- **Curated Pi packages** installed project-locally:
  - `pi-web-access` — web fetch/search for agents.
  - `@demigodmode/pi-web-agent` — reliable web search/fetch with explicit boundaries.
  - `@loreai/pi` — Lore memory engine.
  - `pi-spark` — daily-experience polish.
  - `dot-pi` (from GitHub) — curated extensions, skills, prompts, and rules.

> `@zigai/pi-ui-tweaks` was removed because its bundled settings schema is currently incompatible with this Pi version.
- **Project context files**: `AGENTS.md`, `.pi/SYSTEM.md`, `.pi/APPEND_SYSTEM.md`.
- **DeepEval environment** in `eval/.venv` with sample tests and datasets. Python deps live in `eval/requirements.txt` (DeepEval + pytest stack, `~=`-bounded; `pytest-json-report` is retained, but the nightly live-eval report is written by a conftest hook via `PI_EVAL_REPORT`, not by pytest-json-report). Add new dependencies deliberately; `pi-run setup` installs them.
- **Automation** via the `pi-run` CLI (no Makefile, no shell functions).

## Prerequisites

- Node.js via `nvm` (`pi-run` selects the highest nvm-installed semantic version; override with `PI_NODE_VERSION`).
- Python 3.11+ (for the DeepEval suite).
- Go 1.26+ (only to build/update `pi-run`).
- An API key. The harness is **provider-agnostic**: it ships with a data-driven
  provider table (`providers.json`) covering OpenAI (default), OpenRouter,
  DeepSeek, Anthropic, Gemini, Groq, local OpenAI-compatible endpoints
  (Ollama/vLLM), Azure OpenAI, and 9 more OpenAI-/Anthropic-compatible cloud
  providers (Mistral, Cohere, Together, Perplexity, Fireworks, Moonshot, xAI,
  AWS Bedrock, Ollama) — 17 providers in total. Keys are resolved **env-first**, then
  from an optional secret store (`BW_GET` override; Bitwarden is a documented
  example). See [API Key Resolution](#api-key-resolution) and `pi-run
  providers`.

| provider | key (env var / secret-manager item) | `pi-run --provider` | default model | baseURL (OpenAI- or Anthropic-compatible) |
|---|---|---|---|---|
| OpenAI (default) | `OPENAI_API_KEY` | `openai` | `openai/gpt-5.6-terra` | — |
| OpenRouter | `OPENROUTER_API_KEY` | `openrouter` | `openai/gpt-5.6-terra` | — |
| DeepSeek (direct) | `DEEPSEEK_API_KEY` | `deepseek` | `deepseek/deepseek-v4-flash` | — |
| Anthropic | `ANTHROPIC_API_KEY` | `anthropic` | `anthropic/claude-sonnet-4` | — |
| Gemini | `GEMINI_API_KEY` | `gemini` | `gemini/gemini-2.5-pro` | — |
| Groq | `GROQ_API_KEY` | `groq` | `groq/llama-3.3-70b-versatile` | — |
| Local (Ollama/vLLM) | `LOCAL_API_KEY` | `local` | `local/model` | `http://localhost:11434/v1` |
| Azure OpenAI | `AZURE_OPENAI_API_KEY` | `azure` | `azure/gpt-5.6-terra` | `https://<your-resource>.openai.azure.com/openai/v1` |
| Ollama (local) | `OLLAMA_API_KEY` | `ollama` | `ollama/llama3.1` | `http://localhost:11434/v1` |
| Mistral | `MISTRAL_API_KEY` | `mistral` | `mistral/mistral-large-latest` | `https://api.mistral.ai/v1` |
| Cohere | `COHERE_API_KEY` | `cohere` | `cohere/command-r-plus` | `https://api.cohere.com/compatibility/v1` |
| Together | `TOGETHER_API_KEY` | `together` | `together/llama-3.3-70b-instruct` | `https://api.together.xyz/v1` |
| Perplexity | `PERPLEXITY_API_KEY` | `perplexity` | `perplexity/sonar-pro` | `https://api.perplexity.ai` |
| Fireworks | `FIREWORKS_API_KEY` | `fireworks` | `fireworks/llama-3.3-70b-instruct` | `https://api.fireworks.ai/inference/v1` |
| Moonshot (Kimi) | `MOONSHOT_API_KEY` | `moonshot` | `moonshot/kimi-k2` | `https://api.moonshot.cn/v1` |
| xAI (Grok) | `XAI_API_KEY` | `xai` | `xai/grok-4` | `https://api.x.ai/v1` |
| AWS Bedrock | `BEDROCK_API_KEY` | `bedrock` | `bedrock/claude-sonnet-4` | `https://bedrock-runtime.<region>.amazonaws.com/anthropic/v1` |

> **Add a provider without recompiling:** edit `providers.json` (add a row:
> name, key env var, pi provider, default model, optional `baseURL`), then run
> `pi-run providers` to verify it lists. Installed binaries use the embedded
> 17-provider table; set `PI_RUN_PROVIDERS_FILE` to load a provider table
> from another path. Providers with a `baseURL` are routed through pi's
> OpenAI-compatible (`openai`) or Anthropic-compatible (`anthropic`) provider;
> entries like `azure` and `bedrock` need your resource/region filled in. There
> is **no automatic cross-provider fallback** — the provider is explicit
> (`--provider` / `PI_PROVIDER`).

## Quick Start

**macOS with Homebrew (fastest path):**

```bash
brew install forrestbthomas/tap/pi-run
pi-run config-check        # no API key needed for this
```

**From source (macOS/Linux/WSL):**

```bash
# 1. Clone the repo
git clone https://github.com/forrestbthomas/pi-harness.git
cd pi-harness

# 2. One-command bootstrap (Node + pi + bin/pi-run + eval/.venv)
bash scripts/bootstrap.sh

# 3. Provide an API key (plain env var is the primary path)
export OPENAI_API_KEY=sk-...        # or OPENROUTER_API_KEY / DEEPSEEK_API_KEY

# 4. Use the binary built by bootstrap
bin/pi-run config-check
bin/pi-run doctor

# 5. Launch Pi interactively (OpenAI -> gpt-5.6-terra by default)
bin/pi-run chat

# 6. Or run a quick print-mode query
bin/pi-run print "List all Python files in this repo"

# 7. Route to another provider
bin/pi-run chat --provider deepseek
```

> `pi-run` is built by the bootstrap script into `bin/pi-run`. To use the
> shorter `pi-run` command, add the repository's `bin/` directory to your PATH:
>
> ```bash
> export PATH="$PWD/bin:$PATH"
> ```
>
> Or run `bin/pi-run install` to symlink it into a directory on your PATH.

## Releases

Tag a release (`git tag v0.9.1 && git push --tags`) to trigger the GitHub
Actions release workflow, which cross-compiles `pi-run` for linux/darwin/windows
× amd64/arm64 and attaches the binaries to the release. To build locally:
`bash scripts/build-release.sh <tag>`.

### Install via Homebrew (recommended on macOS)

Installing through Homebrew avoids the macOS Gatekeeper warning
("Apple could not verify ... is free of malware") that appears when you
download an unsigned binary directly from a GitHub release. Homebrew handles
the quarantine attribute, so the binary runs without the security prompt.

```bash
brew install forrestbthomas/tap/pi-run
```

Or tap first, then install:

```bash
brew tap forrestbthomas/tap
brew install pi-run
```

> **For AI agents:** when a developer asks you to install pi-harness on macOS,
> prefer the Homebrew tap (`brew install forrestbthomas/tap/pi-run`) over
> downloading the raw release binary. Direct downloads of unsigned binaries
> trigger Gatekeeper; prefer Homebrew. If a direct download is necessary,
> verify the release checksum and accept the OS warning (or right-click → Open)
> rather than removing the quarantine attribute.

### API Key Resolution (Secret Manager)

API keys are resolved **env-first** (e.g. `export OPENAI_API_KEY=...`), then
from a configured secret manager. The backend is selected by `PI_SECRET_BACKEND`
(default `bitwarden`):

- `bitwarden` — via the `bw_get` helper (override its path with `BW_GET`).
  Requires an unlocked vault (`bw unlock`).
- `1password` — via the `op` CLI (`op read "op://<Vault>/<ITEM_NAME>/credential"`).
  Requires `op` CLI installed and signed in. Vault defaults to `Personal`,
  override with `OP_VAULT`.
- `env-only` — no fallback; env var only.

Every `pi-run` path resolves keys in the same order: env var first, then the
backend. **Exception:** `pi-run eval` checks only environment variables when
deciding whether to run live tests, so it never blocks on a locked vault; keys
held only in a secret manager require exporting them to the environment (or
setting `PI_SECRET_BACKEND=env-only` plus the key in env) before running the
full suite. There is **no automatic cross-provider fallback** — the provider
is explicit (`--provider`, or `PI_PROVIDER` env). A missing key is an error
that tells you what to do:

```
no DEEPSEEK_API_KEY available: export it, or check your secret manager
```

`pi-run doctor` reports the configured backend's status (never values).

### Environment variables

| Variable | Purpose |
|---|---|
| `PI_PROVIDER` | Default provider when `--provider` is omitted (default `openai`) |
| `OPENAI_API_KEY` etc. | Provider API keys, resolved env-first |
| `PI_SECRET_BACKEND` | `bitwarden` (default), `1password`/`op`, or `env-only`/`env` |
| `BW_GET` | Path to the Bitwarden `bw_get` helper |
| `OP_VAULT` | 1Password vault name (default `Personal`) |
| `PI_NODE_VERSION` | Override Node-version selection |
| `PI_RUN_PROVIDERS_FILE` | Load the provider table from a custom path |
| `PI_RUN_PERSONAL` | Opt into personal-machine checks such as the `~/bin/pi-run` symlink |
| `HARNESS_ROOT` | Override repository-root detection |
| `PI_MAX_BUDGET_USD` | Default spend cap for `chat`/`print` when `--max-budget-usd` is omitted |
| `PI_PERMISSION_MODE` | Default permission tier for `chat`/`print` when `--permission-mode` is omitted |
| `PI_MODEL_TIER` | Default model tier (`fast`/`balanced`/`cheap`) for `chat`/`print` when `--model-tier` is omitted; ignored by `resume` |
| `DEEPEVAL_MODEL` | Select a non-OpenAI DeepEval judge model |

## Running Evaluations

```bash
# Smoke tests that do not require an API key
pi-run eval --quick

# Harness config checks (provider defaults, skills, dotfiles) - no API key
pi-run config-check

# Full DeepEval suite (requires a provider key)
pi-run eval
```

The DeepEval **judge** (the LLM-as-a-judge used by metrics like
AnswerRelevancy/Faithfulness/Hallucination) defaults to OpenAI. To evaluate
without depending on OpenAI, set a non-OpenAI provider key and `DEEPEVAL_MODEL`:

```bash
export OPENROUTER_API_KEY=<YOUR_OPENROUTER_KEY> # or another provider key
export DEEPEVAL_MODEL=openrouter/anthropic/claude-sonnet-4
pi-run eval
```

`pi-run eval --quick` and the config tests run without any key.

### Self-healing agent runs

`pi-run` can detect and recover wedged agent runs so a hang never needs a human
nudge (spec `docs/superpowers/specs/2026-08-13-self-healing-design.md`):

- **`pi-run self-heal`** — detect in-progress git state (a wedged rebase) and
  recover it: `GIT_EDITOR=true git rebase --continue` when conflicts are
  resolved, or report `needs-attention` with conflict paths (never guesses
  resolutions). `--abort` is explicit-only.
- **Non-interactive child env** — every launch injects `GIT_EDITOR=true`,
  `GIT_SEQUENCE_EDITOR=true`, `GIT_TERMINAL_PROMPT=0`, `PAGER=cat` so child
  bash tools can never block on an interactive editor/pager (#59).
- **Process-group kill** — timed-out runs kill the whole process tree
  (SIGTERM → 10s grace → SIGKILL), not just the direct `pi` child.
- **Output-stall watchdog** — non-interactive runs (print/benchmark) that emit
  no output for `PI_STALL_TIMEOUT_SECS` (default 300) are terminated; chat is
  never auto-killed.
- **Escalation packet** — killed runs write `.pi/heal/<timestamp>-report.json`
  (goal, side-effect ledger, pending state, trigger evidence, resume handle)
  and exit `9` (watchdog terminated).
- **Observability** — set `PI_SELF_HEAL=1` to log stall/group-kill/recovery
  events to `.pi/heal/events.jsonl` (set `PI_SELF_HEAL=1` to enable;
  deterministic tests leave it off).

### Nightly live eval

The nightly workflow (`.github/workflows/nightly-live-eval.yml`) evaluates the
agent against a **20-task dataset** (`eval/datasets/coding_samples.jsonl`)
with a two-job split:

- **Deterministic job** — runs the hermetic suite (config checks, contract
  tests against a fresh `pi-run` binary, dataset schema lint, scorer unit
  tests); no provider key needed.
- **Live job** — runs each deterministic task 3× via `pi-run print
  --model-tier cheap` plus LLM-judged metrics (`TaskCompletionMetric`,
  G-Eval rubric), then gates per-case pass rates and cost-per-task against
  `eval/baselines/live-baseline.json` (0.05 tolerance; cost regression >2×
  baseline fails) with `eval/scripts/score_run.py`. **Missing provider key is
  a hard failure, never a silent skip.** Budget-capped via
  `PI_MAX_BUDGET_USD` (default $2/night); results artifact-retained 90 days.

Judge model is pinned via `OPENAI_MODEL_NAME` (deepeval reads that knob, not
`DEEPEVAL_MODEL`). Re-baselining is deliberate: run the suite green locally,
then commit a new baseline via `score_run.py --update-baseline --allow`.

## Benchmarks

`pi-run eval --benchmark` runs the same coding tasks in Docker-isolated
containers against any provider — see which model actually solves your tasks.

```bash
# Validate all task formats (hermetic: no Docker, no API keys) — CI-safe
pi-run eval --benchmark-dry-run

# Run the full benchmark suite against the default provider (requires Docker + a key)
pi-run eval --benchmark

# Run one task, routed to another provider/model
pi-run eval --benchmark fix-divide-by-zero --provider deepseek --model deepseek/deepseek-v4-flash
```

Each task lives in `eval/benchmarks/<name>/` and ships a `task.json` plus a
`tests/run.sh` verification script (exit 0 = pass):

```
eval/benchmarks/fix-divide-by-zero/
├── task.json             # id, prompt (or instruction.md), optional setupCmd/repo/timeoutSecs
├── environment/Dockerfile  # optional; default base is python:3.12-slim
├── src/                  # task workspace the agent edits
├── tests/run.sh          # exit 0 = pass, anything else = fail
└── solution/             # optional oracle (for future diff grading)
```

The agent edits a local workspace (copied from `src/`, or cloned from `repo`);
only **verification** runs in the container, against the same files the agent
edited. Results print per-task pass/fail with timing and an aggregate score,
and a JSON report is written to `eval/benchmark-results/<run-id>.json`
(gitignored). Benchmarks require Docker — `--benchmark-dry-run` is the
hermetic format-validation path for CI. See
[docs/benchmarks.md](docs/benchmarks.md) for how to add your own tasks.

## Model Routing

`pi-run` selects the provider (`--provider` / `PI_PROVIDER`; default `openai`),
resolves the key, and launches `pi` with the right `--provider` / `--model`
flags. `chat`/`print` launch pi with `--offline` by default so startup network
ops (version check, changelog, catalog refresh) never hang on the flaky pi.dev
endpoint — the stored model catalogs are used instead; `pi-run setup` is the
explicit online path. Everything else on the command line is passed through to
`pi` unchanged.

### Cost-aware model tiers (`--model-tier`)

`pi-run chat|print --model-tier fast|balanced|cheap` (env `PI_MODEL_TIER`)
picks a model *within the explicitly selected provider*. Design law: tier
selection **never changes the provider** and **never silently falls back** —
an unknown or unmapped tier is an exit-2 usage error that lists the valid or
available tiers.

- `--model-tier balanced` (default when omitted) = the provider's
  `defaultModel`.
- `--model-tier` and `--model` as flags are mutually exclusive (exit 2).
- Env `PI_MODEL_TIER` + explicit `--model` → `--model` wins (flag beats env
  default; an exported env never breaks existing `--model` invocations).
- `resume` rejects the flag and ignores the env (a resumed session keeps its
  model).
- `pi-run providers` shows the available tiers per provider, and
  `pi-run config-check` validates `modelTiers` in providers.json.

Example:
```
pi-run print --provider openai --model-tier cheap "summarize this repo"
PI_MODEL_TIER=fast pi-run print "quick pass"
```

- `/model` — pick a model interactively in-session (OpenAI GPT models are listed
  first via the `enabledModels` order in `.pi/settings.json`).
- `Ctrl+P` — cycle through the enabled model palette.
- `--model` — override the provider default, e.g.
  `pi-run print --provider deepseek --model deepseek/deepseek-v4-pro "..."`,
  `pi-run print --provider openrouter --model anthropic/claude-sonnet-4 "..."`.
- `openrouter/auto` — let OpenRouter pick the best model for the task
  (`pi-run chat --provider openrouter --model openrouter/auto`).

### Permissions

`chat`/`print` accept a harness-level permission tier (`--permission-mode`,
env `PI_PERMISSION_MODE`). Pi has no native permission-mode flag, so each tier
maps to Pi's real tool-control surface:

```bash
pi-run chat --permission-mode plan           # read-only: --tools read,grep,find,ls
pi-run chat --read-only                      # alias for --permission-mode plan
pi-run chat --permission-mode acceptEdits    # Pi defaults (file edits allowed)
pi-run chat --permission-mode bypassPermissions  # --approve (trust project-local files)
PI_PERMISSION_MODE=plan pi-run chat          # or the env var
```

Valid modes mirror the Claude Code set — `default`, `plan`, `acceptEdits`,
`bypassPermissions` (unknown modes are usage errors, exit 2). Policy: the
`worker` agent runs under `default`/`acceptEdits` for implementation; the
`reviewer` and `scout` agents run under `plan`/`--read-only` (see
`.pi/agents/`).

The default model (`openai/gpt-5.6-terra`) is pinned in `.pi/settings.json`;
change it there or with `/model` and the session persists the choice. To
refresh model catalogs (new models / pricing, including the deepseek catalog),
run `pi-run setup` once with network access.

## Cost & Budgets

`pi-run` reports **real spend** — every Pi session file records per-message
`usage.cost` (USD) for each provider/model used, so no price tables are needed:

```bash
pi-run cost                    # per-provider/model table + total
pi-run cost --json             # machine-readable
pi-run cost --since 2026-08-01 # only sessions modified at/after <date>
pi-run cost --reset            # archive the spend ledger, start a fresh period
```

`cost` scans `.pi/sessions/*.jsonl` (including subagent child sessions) and
sums `usage.cost.total`, grouped by provider/model, counting how many session
files each group appears in. Messages without `usage.cost` are skipped; if a
provider reports no cost, it is simply not counted.

**Budget cap** — refuse to launch before spend crosses a limit:

```bash
pi-run chat  --max-budget-usd 5.00
pi-run print --max-budget-usd 5.00 "expensive task"
PI_MAX_BUDGET_USD=5.00 pi-run chat   # or the env var
```

Before launching, `pi-run` computes cumulative spend (session files + the
append-only ledger `.pi/cost-ledger.jsonl`) and exits with **code 6** if it is
already at/above the cap:

**Cost attribution** — `chat`/`print` accept `--cost-mode <mode>` to tag a
run's ledger entry explicitly (modes: `chat`, `print`, `resume`, `backfill`,
`benchmark`, `live-eval`; default is the command name). CI-tagged runs (the
nightly live eval uses `--cost-mode live-eval`) make per-surface spend
attribution unambiguous in `.pi/cost-ledger.jsonl`.

```
pi-run: print: budget exceeded: $5.001234 already spent (cap $5.000000) — raise --max-budget-usd, or start a fresh period with `pi-run cost --reset`
```

After each run the run's spend is appended to the ledger
(`{ts, provider, model, inputTokens, outputTokens, costUsd, mode}`); the ledger
preserves spend even after sessions are cleaned up, and a warning is printed if
the cap is exceeded mid-run. `pi-run cost --reset` archives the ledger to
`.pi/cost-ledger-<ts>.archive.jsonl` and writes a reset marker — budget checks
then count only sessions since the marker (session files are never deleted).
The ledger and marker are gitignored.

Notes (v1 contract): the pre-flight check is best-effort — spend recorded by
*parallel* subagent sessions is attributed to whichever `pi-run` run finishes
last, and runs launched outside `pi-run` are counted from their session files.
Plain `pi-run print` runs stay one-shot (`--no-session`, no session file); when
`--max-budget-usd` is set, the print session is persisted so its spend can be
recorded in the ledger.

## Hooks

`pi-run` can run shell commands around its own invocations via `.pi/hooks.json`
— the harness-level counterpart to agent-internal hooks (Claude Code pre/post-
tool hooks, Copilot's `errorOccurred`). Useful for CI: notify a chat room when
an eval starts or finishes, upload artifacts, or gate commands on external
checks.

Supported events:

| Event | Fires |
|---|---|
| `pre-eval` | before the DeepEval pytest suite runs |
| `post-eval` | after the suite finishes — **always**, even when pytest fails |
| `pre-chat` | before `pi` is launched (`chat`/`print`/`resume`) |

Schema (`.pi/hooks.json`):

```json
{
  "hooks": {
    "pre-eval":  [{"cmd": "./scripts/ci/notify.sh start", "timeoutSecs": 60}],
    "post-eval": [{"cmd": "./scripts/ci/notify.sh done", "continueOnError": true}],
    "pre-chat":  [{"cmd": "git status --porcelain"}]
  }
}
```

Per hook: `cmd` (required; runs via `sh -c` from the harness root),
`timeoutSecs` (default 30; a hung hook is killed and counts as a failure with
exit code 124), and `continueOnError` (default false — a failed hook aborts the
`pi-run` invocation with the command's exit code unless this is true).

Hooks are entirely optional: a missing `.pi/hooks.json` is a no-op. Inspect or
trigger them manually:

```bash
pi-run hooks list          # show configured hooks
pi-run hooks run pre-eval  # run an event's hooks now
```

## Provider Scorecard in CI

`pi-run ci-benchmark` runs the benchmark suite against **2+ providers** and
**gates the build** on the result — the choose → measure → control loop as a
repeatable CI artifact: choose providers, measure them with the
[Benchmarks](#benchmarks) suite, and control spend and quality with explicit
gates.

```bash
# Run the suite against two providers and gate on pass rate + budget
pi-run ci-benchmark --providers openai,deepseek --fail-below 0.8 --max-budget-usd 5.0

# Compare against a previous scorecard/run JSON (regression gate, default tolerance 0.05)
pi-run ci-benchmark --providers openai,deepseek --fail-below 0.8 \
  --baseline eval/benchmark-results/scorecard-latest.json
```

| Flag | Meaning |
|---|---|
| `--providers <a,b>` | Comma-separated providers, **≥ 2**, order-significant (e.g. `openai,deepseek`) |
| `--models <m1,m2>` | Optional per-provider model overrides (same order as `--providers`); defaults to each provider's `defaultModel` |
| `--fail-below <rate>` | Fail (exit 8) if any provider pass rate < `<rate>` (e.g. `0.8`) |
| `--max-budget-usd <n>` | Fail (exit 6) if total run cost ≥ `n` (`PI_MAX_BUDGET_USD` also applies) |
| `--baseline <path>` | Previous scorecard or per-provider run JSON to diff pass rates against (file-based baseline) |
| `--baseline-tolerance <n>` | Max allowed per-provider pass-rate drop vs baseline (default `0.05`) |
| `--runs <n>` | Repeat each provider suite n times; gate on the **median** pass rate (default 1) |
| `--quick-profile` | Cap per-task agent timeout at 60 s — cheap, best-effort smoke run |

Each run writes a machine-readable scorecard to
`eval/benchmark-results/scorecard-<run>.json` (gitignored; per-provider pass
rate, cost, latency, tokens) and prints a human-readable table. Providers run
sequentially so per-provider cost attribution stays clean; any errored task
makes the run incomplete and fails the gate. Benchmarks require Docker (exit 7
when unavailable).

**Exit codes:** `6` budget exceeded · `7` docker unavailable · `8` scorecard
gate failed (incomplete run, pass rate below `--fail-below`, or regression vs
`--baseline`).

**Run it in CI:** see
[`.github/workflows/provider-scorecard.yml`](.github/workflows/provider-scorecard.yml)
— a weekly-scheduled / manual GitHub Actions job that runs the scorecard across
two providers, gates on `--fail-below` + `--max-budget-usd`, and uploads the
scorecard JSON as an artifact that becomes the next run's baseline.

See [docs/benchmarks.md](docs/benchmarks.md) for how to add your own benchmark
tasks.

## `pi-run` Command Reference

| Command | Behavior |
|---|---|
| `pi-run chat [flags] [prompt...]` | Launch Pi interactively (default provider: openai) |
| `pi-run print [flags] "<prompt>"` | One-shot `pi -p --no-session` |
| `pi-run eval [--quick]` | Run the DeepEval pytest suite (`--quick` = smoke subset) |
| `pi-run eval --benchmark [name]` | Run Docker-isolated benchmark tasks (all by default; requires Docker) |
| `pi-run eval --benchmark-dry-run` | Validate benchmark task formats only (no Docker, no keys) |
| `pi-run eval -- <pytest selector...>` | Run a focused test or pass pytest arguments through (for example `tests/test_x.py::test_y`) |
| `pi-run eval --help` | Show eval-specific usage without running pytest |
| `pi-run resume [flags] [prompt...]` | Continue the most recent Pi session (`pi --continue`) |
| `pi-run cost [--json] [--since <date>] [--reset]` | Aggregate real spend from Pi session files (`usage.cost`), per provider/model, with total |
| `pi-run ci-benchmark --providers <a,b> [flags]` | Provider scorecard in CI: run the benchmark suite against 2+ providers, gate on pass rate / budget / baseline |
| `pi-run providers` | List configured providers, default models, and available model tiers |
| `pi-run project-understand [--out <dir>]` | Generate deterministic project-understanding docs (product.md / tech.md / structure.md) from the checkout |
| `pi-run self-heal [--abort]` | Detect and recover a wedged git state (`rebase --continue` when clean, `needs-attention` with conflict paths; `--abort` explicit-only) |
| `pi-run hooks list` / `hooks run <event>` | List or run `.pi/hooks.json` hook commands |
| `pi-run config-check` | Deterministic harness checks (no keys, no network) |
| `pi-run doctor` | Health report: node, pi, vault, per-provider keys, models, venv |
| `pi-run setup` | Create `eval/.venv`, install deps, refresh model catalogs |
| `pi-run install` | Build `bin/pi-run` and symlink it onto your PATH |
| `pi-run clean` | Remove `eval/.venv` and pytest caches |
| `pi-run --exit-codes` | Print the stable exit-code table |
| `pi-run version` / `help` | Version / usage |

Exit codes: `0` ok · `1` generic · `2` usage · `3` missing API key · `4` node/pi not found · `5` eval venv missing · `6` budget exceeded · `7` docker unavailable (benchmarks) · `8` scorecard gate failed (ci-benchmark) · `9` watchdog terminated (stall/group-kill timeout).

## Skills

Pi auto-discovers skills from `~/.agents/skills/`, `.pi/skills/`, packages, and
the `skills` array in settings. Two curated collections are pre-installed:

- **Superpowers** (`obra/superpowers`): a skills collection (brainstorming,
  writing-plans, executing-plans, systematic-debugging,
  test-driven-development, ...). Refresh from
  https://github.com/obra/superpowers.
- **Addy Osmani's agent-skills**: cloned into a local skills directory and
  wired via the `skills` array in `.pi/settings.json`. Includes
  spec-driven-development, code-review-and-quality, test-driven-development,
  and more.

For a durable copy, run `bash scripts/install-skills.sh` once (with network).
It clones both collections into a Pi auto-discovered location, points the
settings `skills` arrays at the durable clone, and is idempotent — re-run it
any time to `git pull` both collections.

Invoke a skill in-session with `/skill:<name>` or just describe the task — the
agent loads the matching skill automatically. `enableSkillCommands` is on in
`.pi/settings.json`.

## Project Layout

```
.
├── AGENTS.md                  # Project instructions loaded by Pi
├── .gitignore
├── go.mod                     # Go module github.com/forrestthomas1/pi-harness (pi-run CLI)
├── cmd/pi-run/                # CLI entry point
├── internal/cli/              # CLI implementation + unit tests
├── bin/                       # Git-ignored build output (bin/pi-run)
├── README.md
├── scripts/
│   └── install-skills.sh      # Durable skill install into ~/.pi/agent/skills/
├── .pi/
│   ├── settings.json          # Pi project settings + package list (incl. pi-subagents)
│   ├── SYSTEM.md              # Replaces Pi's default system prompt
│   ├── APPEND_SYSTEM.md       # Appends harness-specific guardrails
│   ├── npm/                   # Project-local npm packages
│   └── git/                   # Project-local git packages
└── eval/
    ├── .venv/                 # Python virtual environment
    ├── .env.example
    ├── requirements.txt
    ├── pytest.ini
    ├── conftest.py            # Shared fixtures and Pi runner helper (uses pi-run)
    ├── datasets/
    │   └── coding_samples.jsonl
    ├── benchmarks/            # Docker-isolated benchmark tasks (task.json + tests/run.sh)
    │   └── benchmark-results/  # Git-ignored JSON run reports
    └── tests/
        ├── test_benchmark_format.py  # Hermetic benchmark task-format checks
        ├── test_coding_correctness.py
        ├── test_code_quality.py
        ├── test_agent_task_completion.py
        ├── test_secret_resolution.py
        └── test_harness_config.py   # Deterministic config checks (no API key)
```

## Adding New Evaluations

1. Add sample data to `eval/datasets/coding_samples.jsonl`.
2. Create a new test file in `eval/tests/`.
3. Use `run_pi_print()` from `conftest.py` to capture agent outputs (it runs `pi-run print`).
4. Run `pi-run eval` to see the results.

To add a benchmark task instead, create `eval/benchmarks/<name>/task.json` +
`tests/run.sh` (see [Benchmarks](#benchmarks)) and validate it with
`pi-run eval --benchmark-dry-run`.

## Pi Project Settings

Key settings in `.pi/settings.json`:

- `defaultThinkingLevel`: `medium`
- `compaction`: enabled with 16k reserve tokens
- `retry`: 3 agent-level retries
- `sessionDir`: `.pi/sessions`
- `packages`: the curated plugin list

Project-local packages require approval on first use. Run:

```bash
pi list -a
```

to see them, or `pi config -a` to enable/disable individual resources.

## Subagent-Driven Development

This harness is subagent-capable via the [pi-subagents](https://github.com/nicobailon/pi-subagents)
extension (installed in `.pi/settings.json` packages). Pi can delegate focused
work to child sessions with their own tools.

Builtin agents:

| Agent | Use it when you want... |
|-------|--------------------------|
| `scout` | Fast local codebase recon |
| `researcher` | Web/docs research with sources |
| `worker` | Implementation work (edits files, validates) |
| `reviewer` | Code review against a task/plan |
| `oracle` | A second opinion before acting |
| `delegate` | A lightweight general delegate |

Invoke in plain language: "Use reviewer to review this diff", "Ask oracle for a
second opinion", "Run worker to implement this plan". See
`examples/subagents.md` for a worked example.

## Safety Notes

- Pi packages run with full system access; only well-known, public packages were installed.
- Do not commit API keys; `eval/.env` and `.pi/sessions/` are ignored.
- Run destructive commands only after explicit confirmation.

## Troubleshooting

- **Pi packages not visible?** Run `pi list -a` (project approval required).
- **Engine warnings during install?** Ensure a current Node version is installed via nvm and available to `pi-run` (check `pi-run doctor`).
- **DeepEval tests skipped?** Live DeepEval tests run only when a supported
  provider key is in the environment. Export `OPENAI_API_KEY` / another
  supported provider key and re-run `pi-run eval`. Keys held only in a secret
  manager are not read by the live-eval gate.
- **`pi update --models` times out?** The pi.dev model-catalog endpoint is
  intermittently unreachable from some networks (TLS connects but HTTP never
  responds — on both IPv6 and IPv4). `pi-run` mitigates this three ways: every
  pi process runs with `NODE_OPTIONS=--dns-result-order=ipv4first` (the IPv6
  route is deterministically dead), `chat`/`print` launch pi with `--offline`
  so startup never touches the endpoint, and `pi-run setup` retries the refresh
  3× then **warns instead of failing** — the stored catalogs already resolve
  every default model (`pi-run doctor` reports model resolvability as
  informational).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, testing, and how to add a
provider. Report security issues privately per [SECURITY.md](SECURITY.md); all
participants agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## Avoiding Vendor Lock-In

See [docs/anti-lockin.md](docs/anti-lockin.md) — BYO-key, BYO-model, and local
model support keep your agent workflow portable across providers.
