# Architecture

`pi-harness` is a provider-agnostic coding-agent harness: a single Go CLI
(`pi-run`) that launches the [Pi coding agent](https://pi.dev/) against any
supported AI provider and evaluates its outputs with a [DeepEval](https://github.com/confident-ai/deepeval)
pytest suite.

```
┌────────────────────────────────────────────────────────────┐
│                        pi-run (Go CLI)                     │
│  cmd/pi-run/main.go → internal/cli/app.go (dispatch)       │
│                                                            │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ providers   │  │ keys.go      │  │ pi.go             │  │
│  │ (JSON table)│  │ (env →       │  │ (nvm-aware spawn) │  │
│  │ providers.go│  │  secret store)│  │  --offline etc.   │  │
│  └─────────────┘  └──────────────┘  └───────────────────┘  │
│                                                            │
│  eval.go · setup.go · doctor.go · config_check.go          │
└────────────────────────────────────────────────────────────┘
        │                        │
        ▼                        ▼
   pi (agent runtime)      eval/ (DeepEval pytest suite)
   launched per provider   .venv + datasets + tests
```

## Components

- **`internal/cli/app.go`** — command dispatch (`chat`, `print`, `resume`, `eval`,
  `cost`, `ci-benchmark`, `config-check`, `project-understand`, `mcp-server`,
  `hooks`, `self-heal`, `doctor`, `setup`, `install`, `clean`, `providers`,
  `version`), flag parsing, pass-through of extra args to `pi`.
- **`internal/cli/providers.go`** — the provider routing table. Loaded from
  `providers.json` at init (fallback to built-in defaults). Each entry: name,
  key env var, `pi --provider` value, default model, optional `baseURL`.
- **`internal/cli/keys.go`** — key resolution: env var first, then an optional
  secret store via `BW_GET` (Bitwarden documented as an example). Never logs
  values.
- **`internal/cli/pi.go`** — spawns `pi` with the nvm node bin dir prepended to
  PATH, `--offline` for startup reliability, and provider/model flags.
- **`internal/cli/eval.go` / `setup.go`** — venv management + pytest wrapper.
- **`internal/cli/doctor.go` / `config_check.go`** — health and deterministic
  config checks. Personal-machine checks (symlink, dotfiles, skills) are gated
  behind `PI_RUN_PERSONAL=1` so the harness passes on a fresh clone.
- **`eval/`** — the DeepEval suite: datasets, conftest fixtures, and tests
  (deterministic + live-LLM, the latter skipped when no key is present).

## Portability

- No external Go dependencies (stdlib only).
- No hardcoded user paths in shipped code.
- `pi-run doctor` / `config-check` pass on a fresh machine; personal checks are
  opt-in via `PI_RUN_PERSONAL=1`.
