# Pi Coding Agent Harness + DeepEval Evaluation Suite

A ready-to-use coding agent harness built around the [Pi coding agent](https://pi.dev/) and the open-source [DeepEval](https://github.com/confident-ai/deepeval) LLM evaluation framework.

## What You Get

- **Pi CLI** installed globally and configured for this project.
- **Curated Pi packages** installed project-locally:
  - `pi-mcp-adapter` — token-efficient MCP adapter.
  - `pi-web-access` — web fetch/search for agents.
  - `@demigodmode/pi-web-agent` — reliable web search/fetch with explicit boundaries.
  - `@loreai/pi` — Lore memory engine.
  - `pi-spark` — daily-experience polish.
  - `dot-pi` (from GitHub) — curated extensions, skills, prompts, and rules.

> `@zigai/pi-ui-tweaks` was removed because its bundled settings schema is currently incompatible with this Pi version.
- **Project context files**: `AGENTS.md`, `.pi/SYSTEM.md`, `.pi/APPEND_SYSTEM.md`.
- **DeepEval environment** in `eval/.venv` with sample tests and datasets.
- **Golden dataset** for labeling K8s controller tasks: `eval/datasets/golden_k8s_controllers.jsonl`.
- **Example Go project**: `github-repo-controller/` — a Kubebuilder-generated Kubernetes controller for managing GitHub repositories.
- **Automation** via `Makefile`.

> **Note on "DeepEva"**: the request mentioned "DeepEva"; no project by that name was found, so this harness installs **DeepEval** (`confident-ai/deepeval`), the dominant open-source LLM evaluation framework. If you meant a different tool, let me know.

## Prerequisites

- Node.js 22.19+ (managed via `nvm`; the Makefile selects `v22.19.0`).
- Python 3.11+.
- An API key. The harness is wired **OpenAI-first with OpenAI GPT models
  preferred by default** — OpenAI models route through the OpenAI API directly,
  with OpenRouter as the fallback. Keys are stored in **Bitwarden** (CLI `bw`,
  folder "Dev API Keys") and resolved on demand via `~/bin/bw_get` (item name
  == env var name) — no static keys live in shell rc files. See
  [API Key Resolution (Bitwarden)](#api-key-resolution-bitwarden). Relevant keys:
  - `OPENAI_API_KEY` (preferred) — Pi talks to the OpenAI provider directly with
    `gpt-4o` via api.openai.com.
  - `OPENROUTER_API_KEY` — fallback router (https://openrouter.ai) when no
    OpenAI key is set; also used for `openrouter/*` models selected via `/model`.
    Create a key at https://openrouter.ai/keys (starts with `sk-or-v1-`).
  - Legacy providers keep working: `KIMI_API_KEY` is exported from
    `~/.kimi-code/config.toml` into Bitwarden, and other Pi providers
    (`ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, Groq, DeepSeek, ...) work via `/login`.

## Quick Start

Shell aliases were added to `~/.bashrc` and `~/.zshrc`:

```bash
pi-harness        # cd to harness root and launch Pi interactively
pi-harness-print  # cd to harness root and launch Pi in print mode
```

Reload your shell config (`source ~/.zshrc` or `source ~/.bashrc`) then:

```bash
# 1. Install Python dependencies (creates eval/.venv)
make install

# 2. Make an API key available. Keys live in Bitwarden (folder "Dev API Keys"):
bw unlock          # unlock the vault (or export BW_SESSION); make pi fetches keys via bw_get
# or: export OPENAI_API_KEY=sk-...             # env var overrides Bitwarden (preferred)
# or: export OPENROUTER_API_KEY=sk-or-v1-...   # fallback router

# 3. Sanity-check the setup (no API key needed)
make pi-config-check
bash scripts/verify-harness.sh

# 4. Launch Pi interactively (OpenAI -> gpt-4o by default)
pi-harness

# 5. Or run a quick print-mode query
pi-harness-print "List all Python files in this repo"
```

### API Key Resolution (Bitwarden)

All API keys are stored in Bitwarden (CLI `bw`, folder **"Dev API Keys"**,
item name == env var name) instead of static values in `~/.zshrc` /
`~/.bashrc`. `~/bin/bw_get <ITEM_NAME>` fetches a secret on demand
(`~/bin/bw_get --status` reports `unlocked | locked | unauthenticated`). The
shell functions `pi-harness` / `pi-harness-print` (in `~/.zshrc` and
`~/.bashrc`), the `Makefile` targets (`make pi`, `make pi-print`), and
`eval/conftest.py` (`get_secret()`) all resolve keys in this order:

1. `OPENAI_API_KEY` env var (explicit override)
2. Bitwarden via `bw_get` (requires an unlocked vault: `bw unlock`)
3. `OPENROUTER_API_KEY` env var / Bitwarden (fallback router)

With an OpenAI key they launch `pi --provider openai --model gpt-4o` (direct
OpenAI API); without one, an OpenRouter key falls back to
`pi --provider openrouter --model openai/gpt-4o`. `openrouter/*` models selected
via `/model` always route through OpenRouter.

If the vault is locked, run `bw unlock` in your terminal first (or
`export BW_SESSION=...`). `BW_AUTO_UNLOCK=1` makes the interactive shell
functions prompt for the master password.

## Running Evaluations

```bash
# Smoke tests that do not require an API key
make pi-eval-quick

# Harness config checks (OpenRouter defaults, skills, dotfiles) - no API key
make pi-config-check

# Full DeepEval suite (requires a provider key: OPENAI_API_KEY via Bitwarden, or OPENROUTER_API_KEY)
make pi-eval
```

## Model Routing

Pi 0.80.10 ships built-in `openai` and `openrouter` providers. The harness
defaults to **OpenAI** (`openai` / `gpt-4o`), so requests go to OpenAI GPT-4o
through api.openai.com directly. When no OpenAI key is available, Pi falls
back to the `openrouter` provider (`https://openrouter.ai/api/v1`,
OpenAI-compatible). Because OpenRouter is a router, you can switch to any
model or provider it hosts:

- `/model` — pick a model interactively (OpenAI GPT models are listed first via
  the `enabledModels` order in `.pi/settings.json`).
- `Ctrl+P` — cycle through the enabled model palette.
- `--model` / `--provider` flags — e.g.
  `pi --provider openrouter --model openai/gpt-4.1 "refactor this"`,
  `pi --provider openrouter --model anthropic/claude-sonnet-4 "review this"`,
  `pi --provider openrouter --model google/gemini-2.5-pro "..."`.
- `openrouter/auto` — let OpenRouter pick the best model for the task
  (`pi --provider openrouter --model openrouter/auto`).

To refresh the OpenRouter catalog (new models / pricing), run
`pi update --models` once with network access. The default model
(`openai/gpt-4o`) is pinned in `.pi/settings.json`; change it there or with
`/model` and the session persists the choice.

## Skills

Pi auto-discovers skills from `~/.agents/skills/`, `.pi/skills/`, packages, and
the `skills` array in settings. Two curated collections are pre-installed:

- **Superpowers** (`obra/superpowers`): already installed at
  `~/.agents/skills/` (brainstorming, writing-plans, executing-plans,
  systematic-debugging, test-driven-development, ...). To update, refresh the
  copy from https://github.com/obra/superpowers.
- **Addy Osmani's agent-skills**: cloned at
  `~/Projects/tmp/agent-skills/` and wired via the `skills` array in both
  `.pi/settings.json` and `~/.pi/agent/settings.json`
  (`.../agent-skills/skills`). Includes spec-driven-development,
  code-review-and-quality, test-driven-development, and more. Update with
  `git -C ~/Projects/tmp/agent-skills pull`.

For a durable copy that survives outside `~/Projects/tmp`, run
`bash scripts/install-skills.sh` once (with network). It clones both
collections into `~/.pi/agent/skills/` (a Pi auto-discovered location), points
the settings `skills` arrays at the durable clone, and is idempotent — re-run
it any time to `git pull` both collections.

Invoke a skill in-session with `/skill:<name>` or just describe the task — the
agent loads the matching skill automatically. `enableSkillCommands` is on in
`.pi/settings.json`.

## Project Layout

```
.
├── AGENTS.md                  # Project instructions loaded by Pi
├── .gitignore
├── Makefile
├── README.md
├── scripts/
│   ├── verify-harness.sh      # One-shot setup verifier (no API key needed)
│   └── install-skills.sh      # Durable skill install into ~/.pi/agent/skills/
├── .pi/
│   ├── settings.json          # Pi project settings + package list
│   ├── SYSTEM.md              # Replaces Pi's default system prompt
│   ├── APPEND_SYSTEM.md       # Appends harness-specific guardrails
│   ├── npm/                   # Project-local npm packages
│   └── git/                   # Project-local git packages
└── eval/
    ├── .venv/                 # Python virtual environment
    ├── .env.example
    ├── requirements.txt
    ├── pytest.ini
    ├── conftest.py            # Shared fixtures and Pi runner helper
    ├── datasets/
    │   └── coding_samples.jsonl
    └── tests/
        ├── test_coding_correctness.py
        ├── test_code_quality.py
        ├── test_agent_task_completion.py
        └── test_harness_config.py   # Deterministic config checks (no API key)
├── github-repo-controller/      # Example K8s controller (Kubebuilder)
│   ├── api/v1/                  # CRD Go types
│   ├── cmd/main.go              # Manager entrypoint
│   ├── internal/controller/     # Reconcile logic + tests
│   ├── internal/github/         # GitHub client interface + fake
│   ├── config/crd/bases/        # Generated CRD YAML
│   └── README.md
```

## GitHub Repository Controller

A Kubebuilder-generated example lives in `github-repo-controller/`. It compiles and passes unit tests with a fake client:

```bash
cd github-repo-controller
make generate
make manifests
go build ./...
go test ./...
```

The controller is intentionally stubbed at the GitHub API layer. The next step is to implement `internal/github/client.go` with a real GitHub REST client and wire it into `cmd/main.go`.

## Golden Dataset for Labeling

`eval/datasets/golden_k8s_controllers.jsonl` contains prompts and reference answers covering CRD design, reconcile loops, RBAC, testing, and finalizers. Each record has a `label` field initially set to `"pending"` for human review. Use it to benchmark or fine-tune the agent.

## Adding New Evaluations

1. Add sample data to `eval/datasets/coding_samples.jsonl` or `eval/datasets/golden_k8s_controllers.jsonl`.
2. Create a new test file in `eval/tests/`.
3. Use `run_pi_print()` from `conftest.py` to capture agent outputs.
4. Run `make pi-eval` to see the results.

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

## Safety Notes

- Pi packages run with full system access; only well-known, public packages were installed.
- Do not commit API keys; `eval/.env` and `.pi/sessions/` are ignored.
- Run destructive commands only after explicit confirmation.

## Troubleshooting

- **Pi packages not visible?** Run `pi list -a` (project approval required).
- **Engine warnings during install?** Upgrade Node to 22.19.0 or newer.
- **DeepEval tests skipped?** Unlock Bitwarden (`bw unlock`) so `get_secret()`
  can fetch a provider key via `bw_get`, or set `OPENAI_API_KEY` / another
  supported provider key.
