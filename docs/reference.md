# pi-harness Reference

The command, configuration, and internals reference for the
[Pi Coding Agent Harness](../README.md). This page is the manual; the README is
the front door. Content moved here 2026-08-15 during the README minimalization —
nothing was deleted, only relocated.

## Table of contents

- [Provider table](#provider-table)
- [Environment variables](#environment-variables)
- [`pi-run` command reference](#pi-run-command-reference)
- [Exit codes](#exit-codes)
- [Model routing & cost-aware tiers](#model-routing--cost-aware-tiers)
- [Permissions](#permissions)
- [Cost & budgets](#cost--budgets)
- [Hooks](#hooks)
- [Provider scorecard in CI](#provider-scorecard-in-ci)
- [Benchmarks](#benchmarks)
- [Self-healing agent runs](#self-healing-agent-runs)
- [Nightly live eval](#nightly-live-eval)
- [Skills](#skills)
- [Project layout](#project-layout)
- [Adding new evaluations](#adding-new-evaluations)
- [Pi project settings](#pi-project-settings)
- [Subagent-driven development](#subagent-driven-development)
- [Troubleshooting](#troubleshooting)

---

## Provider table

`providers.json` is the data-driven provider table (17 providers in total —
OpenAI, OpenRouter, DeepSeek, Anthropic, Gemini, Groq, local OpenAI-compatible
endpoints, Azure OpenAI, and 9 more OpenAI-/Anthropic-compatible cloud
providers). Keys are resolved **env-first**, then from an optional secret store
(`BW_GET` override; Bitwarden is a documented example).

> **Ollama needs the provider extension** (added 2026-08-16): `pi-run setup`
> installs it to Pi's global agent dir so `pi-run chat --provider ollama`
> works from **any** project. If you run a pre-v0.11.1 binary or skipped
> setup, run these commands from the pi-harness checkout:
> ```sh
> agent_dir="${PI_AGENT_DIR:-$HOME/.pi/agent}"
> mkdir -p "$agent_dir/extensions/lib"
> cp .pi/extensions/ollama.ts "$agent_dir/extensions/"
> cp .pi/extensions/lib/ollama-catalog.ts "$agent_dir/extensions/lib/"
> ```
> (or re-run `pi-run setup`). The model picker discovers the local daemon's
> installed tags via `/api/tags` (bounded timeout; last-known catalog as
> fallback).

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
| Ollama (local) | keyless (no key needed) | `ollama` | `ollama/llama3.1` | `http://localhost:11434/v1` |
| Mistral | `MISTRAL_API_KEY` | `mistral` | `mistral/mistral-large-latest` | `https://api.mistral.ai/v1` |
| Cohere | `COHERE_API_KEY` | `cohere` | `cohere/command-r-plus` | `https://api.cohere.com/compatibility/v1` |
| Together | `TOGETHER_API_KEY` | `together` | `together/llama-3.3-70b-instruct` | `https://api.together.xyz/v1` |
| Perplexity | `PERPLEXITY_API_KEY` | `perplexity` | `perplexity/sonar-pro` | `https://api.perplexity.ai` |
| Fireworks | `FIREWORKS_API_KEY` | `fireworks` | `fireworks/llama-3.3-70b-instruct` | `https://api.fireworks.ai/inference/v1` |
| Moonshot (Kimi) | `MOONSHOT_API_KEY` | `moonshot` | `moonshot/kimi-k2` | `https://api.moonshot.cn/v1` |
| xAI (Grok) | `XAI_API_KEY` | `xai` | `xai/grok-4` | `https://api.x.ai/v1` |
| AWS Bedrock | `BEDROCK_API_KEY` | `bedrock` | `bedrock/claude-sonnet-4` | `https://bedrock-runtime.<region>.amazonaws.com/anthropic/v1` |

**Add a provider without recompiling:** edit `providers.json` (add a row:
name, key env var, pi provider, default model, optional `baseURL`), then run
`pi-run providers` to verify it lists. Installed binaries use the embedded
17-provider table; set `PI_RUN_PROVIDERS_FILE` to load a provider table from
another path. Providers with a `baseURL` are routed through pi's
OpenAI-compatible (`openai`) or Anthropic-compatible (`anthropic`) provider.
There is **no automatic cross-provider fallback** — the provider is explicit
(`--provider` / `PI_PROVIDER`).

### API key resolution (secret manager)

API keys are resolved **env-first** (e.g. `export OPENAI_API_KEY=...`), then
from a configured secret manager. The backend is selected by
`PI_SECRET_BACKEND` (default `bitwarden`):

- `bitwarden` — via the `bw_get` helper (override its path with `BW_GET`).
  Requires an unlocked vault (`bw unlock`).
- `1password` — via the `op` CLI (`op read "op://<Vault>/<ITEM_NAME>/credential"`).
  Vault defaults to `Personal`, override with `OP_VAULT`.
- `env-only` — no fallback; env var only.

Every `pi-run` path resolves keys in the same order: env var first, then the
backend. **Exception:** `pi-run eval` checks only environment variables when
deciding whether to run live tests, so it never blocks on a locked vault. A
missing key is an error that tells you what to do. `pi-run doctor` reports the
configured backend's status (never values).

## Environment variables

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
| `PI_SELF_HEAL` | Set `1` to record watchdog stall/group-kill/recovery events to `.pi/heal/events.jsonl` (scorecard observability) |
| `PI_STALL_TIMEOUT_SECS` | Watchdog silent-window: terminate a non-interactive run with no stdout after N seconds (default 300; 0 disables) |
| `PI_WATCHDOG_GRACE_SECS` | Process-group-kill grace between SIGTERM and SIGKILL (default 10; 0 = immediate) |
| `EVAL_RUNS_PER_CASE` | Live-suite agent runs per case (nightly sets 5; the flake-aware gate, EVAL-2) |
| `EVAL_JUDGE_RUNS` | LLM-judge repeats per case, pass = majority (default 3; EVAL-8 judge stabilization) |
| `PI_EVAL_REPORT` | Path (relative to `eval/`) where the conftest hook writes the pytest report for `score_run.py` |
| `OPENAI_MODEL_NAME` | Judge model pin for the LLM-judged metrics (nightly sets `gpt-4.1-mini`) |

## `pi-run` command reference

| Command | Behavior |
|---|---|
| `pi-run chat [flags] [prompt...]` | Launch Pi interactively (default provider: openai) |
| `pi-run print [flags] "<prompt>"` | One-shot `pi -p --no-session` |
| `pi-run eval [--quick]` | Run the DeepEval pytest suite (`--quick` = smoke subset) |
| `pi-run eval --benchmark [name]` | Run Docker-isolated benchmark tasks (all by default; requires Docker) |
| `pi-run eval --benchmark-dry-run` | Validate benchmark task formats only (no Docker, no keys) |
| `pi-run eval -- <pytest selector...>` | Run a focused test or pass pytest arguments through |
| `pi-run eval --help` | Show eval-specific usage without running pytest |
| `pi-run resume [flags] [prompt...]` | Continue the most recent Pi session (`pi --continue`) |
| `pi-run cost [--json] [--since <date>] [--reset]` | Aggregate real spend from Pi session files, per provider/model, with total |
| `pi-run ci-benchmark --providers <a,b> [flags]` | Provider scorecard in CI: gate on pass rate / budget / baseline |
| `pi-run providers` | List configured providers, default models, and available model tiers |
| `pi-run project-understand [--out <dir>]` | Generate deterministic project-understanding docs from the checkout |
| `pi-run self-heal [--abort]` | Detect and recover a wedged git state |
| `pi-run hooks list` / `hooks run <event>` | List or run `.pi/hooks.json` hook commands |
| `pi-run config-check` | Deterministic harness checks (no keys, no network) |
| `pi-run doctor` | Health report: node, pi, vault, per-provider keys, models, venv |
| `pi-run setup` | Create `eval/.venv`, install deps, refresh model catalogs |
| `pi-run install` | Build `bin/pi-run` and symlink it onto your PATH |
| `pi-run clean` | Remove `eval/.venv` and pytest caches |
| `pi-run --exit-codes` | Print the stable exit-code table |
| `pi-run version` / `help` | Version / usage |

## Exit codes

`0` ok · `1` generic · `2` usage · `3` missing API key · `4` node/pi not found
· `5` eval venv missing · `6` budget exceeded · `7` docker unavailable
(benchmarks) · `8` scorecard gate failed (ci-benchmark) · `9` **watchdog
terminated** (stall/group-kill timeout).

## Model routing & cost-aware tiers

`pi-run` selects the provider (`--provider` / `PI_PROVIDER`; default `openai`),
resolves the key, and launches `pi` with the right `--provider` / `--model`
flags. `chat`/`print` launch pi with `--offline` by default so startup network
ops never hang on the flaky pi.dev endpoint — the stored model catalogs are
used instead; `pi-run setup` is the explicit online path.

`--model-tier fast|balanced|cheap` (env `PI_MODEL_TIER`) picks a model *within
the explicitly selected provider*. Design law: tier selection **never changes
the provider** and **never silently falls back** — an unknown or unmapped tier
is an exit-2 usage error that lists the valid or available tiers.

- `--model-tier balanced` (default when omitted) = the provider's
  `defaultModel`.
- `--model-tier` and `--model` as flags are mutually exclusive (exit 2).
- Env `PI_MODEL_TIER` + explicit `--model` → `--model` wins (flag beats env
  default).
- `resume` rejects the flag and ignores the env (a resumed session keeps its
  model).
- `pi-run providers` shows the available tiers per provider; `pi-run
  config-check` validates `modelTiers` in providers.json.
- `/model` picks a model interactively in-session; `Ctrl+P` cycles the enabled
  palette; `--model` overrides the provider default;
  `openrouter/auto` lets OpenRouter pick the best model.

## Permissions

`chat`/`print` accept a harness-level permission tier (`--permission-mode`,
env `PI_PERMISSION_MODE`) mapped to Pi's real tool-control surface:

```bash
pi-run chat --permission-mode plan           # read-only: --tools read,grep,find,ls
pi-run chat --read-only                      # alias for --permission-mode plan
pi-run chat --permission-mode acceptEdits    # Pi defaults (file edits allowed)
pi-run chat --permission-mode bypassPermissions  # --approve (trust project-local files)
```

Valid modes mirror the Claude Code set — `default`, `plan`, `acceptEdits`,
`bypassPermissions` (unknown modes are usage errors, exit 2). Policy: the
`worker` agent runs under `default`/`acceptEdits` for implementation; the
`reviewer` and `scout` agents run under `plan`/`--read-only` (see `.pi/agents/`).

## Cost & budgets

`pi-run` reports **real spend** — every Pi session file records per-message
`usage.cost` (USD), so no price tables are needed:

```bash
pi-run cost                    # per-provider/model table + total
pi-run cost --json             # machine-readable
pi-run cost --since 2026-08-01 # only sessions modified at/after <date>
pi-run cost --reset            # archive the spend ledger, start a fresh period
```

**Budget cap** — refuse to launch before spend crosses a limit:

```bash
pi-run chat  --max-budget-usd 5.00
PI_MAX_BUDGET_USD=5.00 pi-run chat   # or the env var
```

Before launching, `pi-run` computes cumulative spend (session files + the
append-only ledger `.pi/cost-ledger.jsonl`) and exits with **code 6** if it is
already at/above the cap. `--cost-mode <mode>` tags a run's ledger entry
(modes: `chat`, `print`, `resume`, `backfill`, `benchmark`, `live-eval`).
`pi-run cost --reset` archives the ledger and writes a reset marker.

## Hooks

`pi-run` can run shell commands around its own invocations via `.pi/hooks.json`
(harness-level hooks; CI-friendly).

| Event | Fires |
|---|---|
| `pre-eval` | before the DeepEval pytest suite runs |
| `post-eval` | after the suite finishes — **always**, even when pytest fails |
| `pre-chat` | before `pi` is launched (`chat`/`print`/`resume`) |

Schema: `{"hooks": {"pre-eval": [{"cmd": "...", "timeoutSecs": 60}], ...}}`.
Per hook: `cmd` (required; runs via `sh -c` from the harness root),
`timeoutSecs` (default 30; a hung hook is killed and counts as a failure with
exit 124), `continueOnError` (default false). A missing `.pi/hooks.json` is a
no-op. Inspect/trigger manually: `pi-run hooks list`, `pi-run hooks run
pre-eval`.

## Provider scorecard in CI

`pi-run ci-benchmark` runs the benchmark suite against **2+ providers** and
**gates the build** on the result:

```bash
pi-run ci-benchmark --providers openai,deepseek --fail-below 0.8 --max-budget-usd 5.0
pi-run ci-benchmark --providers openai,deepseek --fail-below 0.8 \
  --baseline eval/benchmark-results/scorecard-latest.json
```

| Flag | Meaning |
|---|---|
| `--providers <a,b>` | Comma-separated providers, **≥ 2**, order-significant |
| `--models <m1,m2>` | Optional per-provider model overrides (same order) |
| `--fail-below <rate>` | Fail (exit 8) if any provider pass rate < `<rate>` |
| `--max-budget-usd <n>` | Fail (exit 6) if total run cost ≥ `n` |
| `--baseline <path>` | Previous scorecard to diff pass rates against |
| `--baseline-tolerance <n>` | Max allowed per-provider pass-rate drop (default `0.05`) |
| `--runs <n>` | Repeat each provider suite n times; gate on the **median** pass rate |
| `--quick-profile` | Cap per-task agent timeout at 60 s — cheap smoke run |

Each run writes a machine-readable scorecard to
`eval/benchmark-results/scorecard-<run>.json` (gitignored) and prints a
human-readable table. Providers run sequentially so per-provider cost
attribution stays clean; any errored task makes the run incomplete and fails
the gate. Benchmarks require Docker (exit 7 when unavailable). Run it in CI via
`.github/workflows/provider-scorecard.yml` (weekly/manual).

## Benchmarks

```bash
pi-run eval --benchmark-dry-run       # validate formats (no Docker, no keys)
pi-run eval --benchmark               # full suite (requires Docker + a key)
pi-run eval --benchmark fix-divide-by-zero --provider deepseek --model deepseek/deepseek-v4-flash
```

Each task lives in `eval/benchmarks/<name>/` and ships a `task.json` plus a
`tests/run.sh` verification script (exit 0 = pass). The agent edits a local
workspace; only **verification** runs in the container. Results print per-task
pass/fail with timing and an aggregate score, plus a JSON report at
`eval/benchmark-results/<run-id>.json`. See [docs/benchmarks.md](benchmarks.md)
for how to add your own tasks.

## Self-healing agent runs

`pi-run` detects and recovers wedged agent runs (spec
`docs/governance/specs-archive/2026-08-13-self-healing-design.md`):

- **`pi-run self-heal`** — detect in-progress git state (a wedged rebase) and
  recover it: `GIT_EDITOR=true git rebase --continue` when conflicts are
  resolved, or report `needs-attention` with conflict paths. `--abort` is
  explicit-only.
- **Non-interactive child env** — every launch injects `GIT_EDITOR=true`,
  `GIT_SEQUENCE_EDITOR=true`, `GIT_TERMINAL_PROMPT=0`, `PAGER=cat` so child
  bash tools never block on an interactive editor/pager.
- **Process-group kill** — timed-out runs kill the whole process tree
  (SIGTERM → grace → SIGKILL), not just the direct `pi` child.
- **Output-stall watchdog** — non-interactive runs with no output for
  `PI_STALL_TIMEOUT_SECS` (default 300) are terminated; chat is never
  auto-killed.
- **Escalation packet** — killed runs write `.pi/heal/<timestamp>-report.json`
  (goal, side-effect ledger, pending state, trigger evidence, resume handle)
  and exit `9` (watchdog terminated).
- **Observability** — `PI_SELF_HEAL=1` logs stall/group-kill/recovery events to
  `.pi/heal/events.jsonl`, surfaced in scorecards.

## Nightly live eval

The nightly workflow (`.github/workflows/nightly-live-eval.yml`) evaluates the
agent against the live dataset with a two-job split:

- **Deterministic job** — the hermetic suite (config checks, contract tests,
  dataset schema lint, scorer unit tests); no provider key needed.
- **Live job** — each case 5× (`EVAL_RUNS_PER_CASE`) via `pi-run print
  --model-tier cheap` plus LLM-judged metrics (judge majority-of-3 via
  `EVAL_JUDGE_RUNS`), then gates per-case pass rates and cost-per-task against
  `eval/baselines/live-baseline.json` (0.05 tolerance; flake-aware cost gate).
  **Missing provider key is a hard failure, never a silent skip.**
  Budget-capped via `PI_MAX_BUDGET_USD` (default $2/night); results
  artifact-retained 90 days. The EVAL-16 delta report is emitted into
  `eval/live-results/` (report-only pilot).

Re-baselining is deliberate: run the suite green locally, then commit a new
baseline via `score_run.py --update-baseline --allow`.

## Skills

Pi auto-discovers skills from `~/.agents/skills/`, `.pi/skills/`, packages, and
project settings. Five curated collections are installed via
`bash scripts/install-skills.sh` (durable clones under `~/.pi/agent/skills/`):

- **Superpowers** — planning/execution skills
- **agent-skills** — engineering skills
- **scope-lock** — anti-scope-creep SCOPE.md boundary contracts
- **productskills** — PM skills
- **spec-coding-skills** — spec-plan, spec-crlp, spec-index

Re-run `bash scripts/install-skills.sh` any time to refresh. Invoke a skill
in-session with `/skill:<name>` or just describe the task.

## Project layout

```
.
├── AGENTS.md                  # Project instructions loaded by Pi
├── cmd/pi-run/                # CLI entry point
├── internal/cli/              # CLI implementation + unit tests
├── scripts/                   # bootstrap, install-skills, release scripts
├── .pi/                       # Pi project settings + packages (npm/, git/)
├── eval/
│   ├── datasets/              # tasks.json (count authority) + samples + graders + references
│   ├── baselines/             # committed live baseline (live-baseline.json)
│   ├── scripts/               # score_run.py baseline gate + scorer; score_delta.py (EVAL-16)
│   ├── benchmarks/            # Docker-isolated benchmark tasks
│   ├── live-results/          # git-ignored nightly report + seam-report.json
│   └── tests/                 # hermetic + live test files (contract, eval, drift guards)
├── docs/                      # architecture, reference (this page), knowledge-base, governance
├── CHARTER.md                 # the project's boundary contract
└── README.md                  # front door — this project's landing page
```

## Adding new evaluations

1. Add sample data to `eval/datasets/coding_samples.jsonl`.
2. Create a new test file in `eval/tests/`.
3. Use `run_pi_print()` from `conftest.py` to capture agent outputs.
4. Run `pi-run eval` to see the results.

To add a benchmark task instead, create `eval/benchmarks/<name>/task.json` +
`tests/run.sh` and validate with `pi-run eval --benchmark-dry-run`.

## Pi project settings

Key settings in `.pi/settings.json`: `defaultThinkingLevel` (`medium`),
`compaction` (16k reserve), `retry` (3 agent-level retries), `sessionDir`
(`.pi/sessions`), `packages` (the curated plugin list). Project-local packages
require approval on first use: `pi list -a`.

> **Memory is optional — default none.** The harness ships no memory engine;
> the docs are the source of truth; any memory engine is user-loaded via
> `.pi/settings.json` packages. (Decision: `docs/knowledge-base/decision/2026-08-15-memory-engine-spike.md`.)

## Subagent-driven development

This harness is subagent-capable via the pi-subagents extension (installed in
`.pi/settings.json` packages). Builtin agents: `scout` (recon), `researcher`
(web/docs research), `worker` (implementation), `reviewer` (code review),
`oracle` (second opinion), `delegate` (general delegate). Invoke in plain
language — "Use reviewer to review this diff", "Run worker to implement this
plan". See `examples/subagents.md`.

## Troubleshooting

- **Pi packages not visible?** Run `pi list -a` (project approval required).
- **Engine warnings during install?** Ensure a current Node version is
  installed via nvm and available to `pi-run` (check `pi-run doctor`).
- **DeepEval tests skipped?** Live DeepEval tests run only when a supported
  provider key is in the environment. Export `OPENAI_API_KEY` (or another
  supported provider key) and re-run `pi-run eval`. Keys held only in a secret
  manager are not read by the live-eval gate.
- **`pi update --models` times out?** The pi.dev model-catalog endpoint is
  intermittently unreachable. `pi-run` mitigates this: every pi process runs
  with `NODE_OPTIONS=--dns-result-order=ipv4first`, `chat`/`print` launch pi
  with `--offline`, and `pi-run setup` retries the refresh 3× then warns
  instead of failing.
