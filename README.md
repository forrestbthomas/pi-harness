> **pi-harness** — a provider-agnostic coding-agent harness + evaluation suite that keeps you out of AI-vendor lock-in.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org/dl/)
[![CI](https://github.com/forrestbthomas/pi-harness/actions/workflows/ci.yml/badge.svg)](https://github.com/forrestbthomas/pi-harness/actions/workflows/ci.yml)

# Pi Coding Agent Harness + DeepEval Evaluation Suite

A ready-to-use coding agent harness built around the [Pi coding agent](https://pi.dev/) and the open-source [DeepEval](https://github.com/confident-ai/deepeval) LLM evaluation framework. The harness runtime is driven by a single Go CLI, **`pi-run`** — the one source of truth for provider routing, API key resolution, Pi launching, evaluation, setup, and health checks.

## What You Get

- **`pi-run` CLI** — a compiled Go binary (repo `bin/pi-run`) that owns the harness runtime: `chat`, `print`, `eval`, `config-check`, `doctor`, `setup`, `install`, `clean`, `version`.
- **Pi CLI** installed globally (via nvm) and configured for this project.
- **Curated Pi packages** installed project-locally:
  - `pi-mcp-adapter` — token-efficient MCP adapter.
  - `pi-web-access` — web fetch/search for agents.
  - `@demigodmode/pi-web-agent` — reliable web search/fetch with explicit boundaries.
  - `@loreai/pi` — Lore memory engine.
  - `pi-spark` — daily-experience polish.
  - `dot-pi` (from GitHub) — curated extensions, skills, prompts, and rules.

> `@zigai/pi-ui-tweaks` was removed because its bundled settings schema is currently incompatible with this Pi version.
- **Project context files**: `AGENTS.md`, `.pi/SYSTEM.md`, `.pi/APPEND_SYSTEM.md`.
- **DeepEval environment** in `eval/.venv` with sample tests and datasets. Python deps live in `eval/requirements.txt` (DeepEval + pytest stack, `~=`-bounded). Add new dependencies deliberately; `pi-run setup` installs them.
- **Automation** via the `pi-run` CLI (no Makefile, no shell functions).

## Prerequisites

- Node.js via `nvm` (`pi-run` selects the highest nvm-installed semantic version; override with `PI_NODE_VERSION`).
- Python 3.11+ (for the DeepEval suite).
- Go 1.26+ (only to build/update `pi-run`).
- An API key. The harness is **provider-agnostic**: it ships with a data-driven
  provider table (`providers.json`) covering OpenAI (default), OpenRouter,
  DeepSeek, Anthropic, Gemini, Groq, and a local OpenAI-compatible endpoint
  (Ollama/vLLM). Keys are resolved **env-first**, then from an optional secret
  store (`BW_GET` override; Bitwarden is a documented example). See
  [API Key Resolution](#api-key-resolution) and `pi-run providers`.

| provider | key (env var / secret-manager item) | `pi-run --provider` | default model |
|---|---|---|---|
| OpenAI (default) | `OPENAI_API_KEY` | `openai` | `openai/gpt-5.6-terra` |
| OpenRouter | `OPENROUTER_API_KEY` | `openrouter` | `openai/gpt-5.6-terra` |
| DeepSeek (direct) | `DEEPSEEK_API_KEY` | `deepseek` | `deepseek/deepseek-v4-flash` |
| Anthropic | `ANTHROPIC_API_KEY` | `anthropic` | `anthropic/claude-sonnet-4` |
| Gemini | `GEMINI_API_KEY` | `gemini` | `gemini/gemini-2.5-pro` |
| Groq | `GROQ_API_KEY` | `groq` | `groq/llama-3.3-70b-versatile` |
| Local (Ollama/vLLM) | `LOCAL_API_KEY` | `local` | `local/model` (baseURL `http://localhost:11434/v1`) |

> **Add a provider without recompiling:** edit `providers.json` (add a row:
> name, key env var, pi provider, default model, optional `baseURL`), then run
> `pi-run providers` to verify it lists. Installed binaries use the embedded
> seven-provider table; set `PI_RUN_PROVIDERS_FILE` to load a provider table
> from another path. There is **no automatic cross-provider fallback** — the
> provider is explicit (`--provider` / `PI_PROVIDER`).

## Quick Start

**macOS with Homebrew (fastest path):**

```bash
brew install forrestbthomas/tap/pi-run
pi-run config-check        # no API key needed for this
```

**From source (all platforms):**

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

Tag a release (`git tag v0.3.0 && git push --tags`) to trigger the GitHub
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

## Model Routing

`pi-run` selects the provider (`--provider` / `PI_PROVIDER`; default `openai`),
resolves the key, and launches `pi` with the right `--provider` / `--model`
flags. `chat`/`print` launch pi with `--offline` by default so startup network
ops (version check, changelog, catalog refresh) never hang on the flaky pi.dev
endpoint — the stored model catalogs are used instead; `pi-run setup` is the
explicit online path. Everything else on the command line is passed through to
`pi` unchanged.

- `/model` — pick a model interactively in-session (OpenAI GPT models are listed
  first via the `enabledModels` order in `.pi/settings.json`).
- `Ctrl+P` — cycle through the enabled model palette.
- `--model` — override the provider default, e.g.
  `pi-run print --provider deepseek --model deepseek/deepseek-v4-pro "..."`,
  `pi-run print --provider openrouter --model anthropic/claude-sonnet-4 "..."`.
- `openrouter/auto` — let OpenRouter pick the best model for the task
  (`pi-run chat --provider openrouter --model openrouter/auto`).

The default model (`openai/gpt-5.6-terra`) is pinned in `.pi/settings.json`;
change it there or with `/model` and the session persists the choice. To
refresh model catalogs (new models / pricing, including the deepseek catalog),
run `pi-run setup` once with network access.

## `pi-run` Command Reference

| Command | Behavior |
|---|---|
| `pi-run chat [flags] [prompt...]` | Launch Pi interactively (default provider: openai) |
| `pi-run print [flags] "<prompt>"` | One-shot `pi -p --no-session` |
| `pi-run eval [--quick]` | Run the DeepEval pytest suite (`--quick` = smoke subset) |
| `pi-run eval -- <pytest selector...>` | Run a focused test or pass pytest arguments through (for example `tests/test_x.py::test_y`) |
| `pi-run eval --help` | Show eval-specific usage without running pytest |
| `pi-run resume [flags] [prompt...]` | Continue the most recent Pi session (`pi --continue`) |
| `pi-run providers` | List configured providers and default models |
| `pi-run config-check` | Deterministic harness checks (no keys, no network) |
| `pi-run doctor` | Health report: node, pi, vault, per-provider keys, models, venv |
| `pi-run setup` | Create `eval/.venv`, install deps, refresh model catalogs |
| `pi-run install` | Build `bin/pi-run` and symlink it onto your PATH |
| `pi-run clean` | Remove `eval/.venv` and pytest caches |
| `pi-run --exit-codes` | Print the stable exit-code table |
| `pi-run version` / `help` | Version / usage |

Exit codes: `0` ok · `1` generic · `2` usage · `3` missing API key · `4` node/pi not found · `5` eval venv missing.

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
    └── tests/
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
