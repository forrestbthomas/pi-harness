# pi-run CLI — Harness Runtime Single Source of Truth

**Date:** 2026-08-02
**Status:** Approved (pending written-spec review)

## Goal

Give the Pi coding harness a first-class, native CLI (`pi-run`) that is the single
source of truth for the harness runtime: provider routing (OpenAI / OpenRouter /
DeepSeek), Bitwarden key resolution, Pi launching, evaluation, setup, and health
checks. The CLI replaces every shell function and Makefile target today, and this
repository becomes a true monorepo containing only the harness + eval code.

## Motivation / Current State

- Key resolution + provider selection logic is duplicated in 4 places:
  `pi-harness` / `pi-harness-print` in `~/.zshrc` and `~/.bashrc`, `make pi` /
  `make pi-print`, and `eval/conftest.py` (`get_secret()`). Every one hardcodes
  the same `OPENAI_API_KEY` → `OPENROUTER_API_KEY` fallback chain.
- Pi 0.80.10 ships a built-in `deepseek` provider (`DEEPSEEK_API_KEY` is a
  documented env var; the model resolver maps `deepseek → deepseek-v4-pro` as its
  default). The `deepseek` model catalog (base URL `https://api.deepseek.com/v1`)
  is obtained via `pi update --models`.
- DeepSeek models are also reachable today through OpenRouter
  (`--provider openrouter --model deepseek/deepseek-v4-flash`); 11 deepseek
  entries exist in the stored openrouter catalog.
- `pi` is installed as a global npm package under nvm
  (`~/.nvm/versions/node/v22.19.0/bin/pi`) and is NOT on PATH outside nvm.
- `DEEPSEEK_API_KEY` is already exported from Bitwarden in the rc files, and
  `eval/conftest.py` already lists it as a supported key.

## Decisions (all user-approved)

| # | Decision |
|---|---|
| D1 | CLI is written in **Go**, compiled to a native binary. |
| D2 | CLI **replaces** the `pi-harness`/`pi-harness-print` shell functions and all Makefile targets. |
| D3 | **Full takeover**: chat, print, eval, eval-quick, config-check, doctor, setup, install, clean, version. `make` disappears entirely. |
| D4 | Name **`pi-run`**; binary built to repo `bin/pi-run` (gitignored) and symlinked into `~/bin/` (already on PATH; `bw_get` lives there). |
| D5 | Provider/model selection via **`--provider` flag or `PI_PROVIDER` env**, default **`openai`** (no prompting, scriptable). `--model` overrides per-provider default. |
| D6 | Defaults: `openai` → `openai/gpt-5.6-terra`; `openrouter` → `openai/gpt-5.6-terra`; `deepseek` → `deepseek/deepseek-v4-flash`. |
| D7 | **No cross-provider auto-fallback.** Explicit provider routing only; a missing key is a clear error. (Deliberate behavior change vs. today.) |
| D8 | `github-repo-controller/` moves out to `~/Projects/github-repo-controller` as its own repo (git history preserved via subtree split). |
| D9 | `chatbot-project/` + `projects/chatbot/` move out to `~/Projects/chatbot` as their own repo (subtree split); `projects/` deleted. |
| D10 | `scripts/verify-harness.sh` removed (superseded by `pi-run doctor`); `scripts/install-skills.sh` kept (maintenance tool, not runtime). |
| D11 | Startup key exports in `~/.zshrc` / `~/.bashrc` stay (shared with Binance/Kraken/etc.; CLI prefers env anyway). Only the `pi-harness` functions are removed. |
| D12 | `eval/datasets/golden_k8s_controllers.jsonl` removed with the controller (not referenced by any test; only README). |

## Architecture

### Monorepo layout

```
harness/                          # monorepo root — only harness + eval remain
├── go.mod                        # module github.com/forrestthomas/harness (go 1.21)
├── cmd/pi-run/main.go            # thin main → internal/cli
├── internal/cli/
│   ├── app.go                    # subcommand dispatch, flags, help
│   ├── keys.go                   # resolveSecret(): env → bw_get
│   ├── providers.go              # routing table + defaults
│   ├── pi.go                     # node/pi resolution + spawn
│   ├── eval.go                   # pytest wrappers (full / quick)
│   ├── setup.go                  # venv, pip deps, pi update --models
│   ├── doctor.go                 # health checks (replaces verify-harness.sh)
│   └── *_test.go
├── bin/                          # gitignored (bin/pi-run)
├── eval/                         # DeepEval suite (unchanged, exec'd by CLI)
├── scripts/                      # install-skills.sh only
├── Makefile                      # REMOVED
└── README.md / AGENTS.md         # updated
```

Dependency policy: **stdlib only** — no external Go modules.

### Command surface

| Command | Behavior |
|---|---|
| `pi-run chat [flags] [prompt...]` | Interactive `pi` with resolved provider/model |
| `pi-run print [flags] "<prompt>"` | One-shot `pi -p --no-session "<prompt>"` |
| `pi-run eval [--quick]` | Run DeepEval pytest suite; `--quick` = smoke subset: `eval/tests/test_code_quality.py` + `eval/tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty` (mirrors today's `make pi-eval-quick`) |
| `pi-run config-check` | Deterministic harness checks (Go, no keys/network) |
| `pi-run doctor` | Health report: node, pi, vault status, per-provider key presence, model resolvable, venv |
| `pi-run setup` | Idempotent: create `eval/.venv`, `pip install -r eval/requirements.txt`, `pi update --models` |
| `pi-run install` | `go build ./cmd/pi-run -o bin/pi-run` + symlink `~/bin/pi-run` → repo `bin/pi-run` |
| `pi-run clean` | Remove `eval/.venv` + pytest caches |
| `pi-run version` | Print version |
| `pi-run help` | Usage |

Common flags on `chat`/`print`: `--provider <openai|openrouter|deepseek>`
(env `PI_PROVIDER` fallback; default `openai`), `--model <id>` (overrides
per-provider default). **Pass-through:** any other flag is forwarded to `pi`
unchanged (`--session-id`, `--tools`, `--thinking`, `--mode json`, …).

### Provider routing table

| provider | key env var | `pi --provider` | default model |
|---|---|---|---|
| `openai` (default) | `OPENAI_API_KEY` | `openai` | `openai/gpt-5.6-terra` |
| `openrouter` | `OPENROUTER_API_KEY` | `openrouter` | `openai/gpt-5.6-terra` |
| `deepseek` | `DEEPSEEK_API_KEY` | `deepseek` | `deepseek/deepseek-v4-flash` |

### Key resolution

`resolveSecret(name)`:
1. env var `name` → use it
2. else exec `~/bin/bw_get <name>` (respect `BW_GET` env override) → use stdout
   trimmed; non-zero exit / missing binary → unavailable
3. else return "missing" → caller errors with actionable message

Never log, echo, or persist key material. `doctor` / `config-check` report
presence only.

### Pi spawn

- Resolve node bin dir: `$HOME/.nvm/versions/node/<PI_NODE_VERSION>/bin`
  (default `v22.19.0`; `PI_NODE_VERSION` env override). Verify `node` exists.
- Child env PATH = `<node bin dir>:<existing PATH>`.
- Exec `pi --provider <p> --model <m> [--print --no-session] [pass-through...] [prompt]`.
- stdin/stdout/stderr inherited (interactive TTY works); `pi` exit code passes
  through.

### Error handling

| Exit code | Meaning |
|---|---|
| 0 | success |
| 1 | generic error |
| 2 | usage error |
| 3 | missing API key |
| 4 | node/pi not found |
| 5 | `eval/.venv` missing |

Actionable messages: missing key → "no OPENAI_API_KEY: export it, or run `bw
unlock` then `pi-run doctor`". Node missing → print the `PI_NODE_VERSION` it
looked for and how to override. Venv missing → "run `pi-run setup`".

### Testing

Go unit tests (hermetic — no network, no real `pi`):
- `keys_test.go` — env wins; `bw_get` output used when env unset; non-zero exit →
  error; fake `bw_get` injected via `BW_GET`.
- `providers_test.go` — defaults, `PI_PROVIDER` env, unknown provider → usage error.
- `pi_test.go` — arg construction per provider × chat/print; PATH prepend with
  fake node dir; exit-code passthrough.

Updated `eval/tests/test_harness_config.py`:
- `~/bin/pi-run` exists and symlinks into the repo `bin/pi-run`.
- dotfiles no longer define `pi-harness` functions.
- `Makefile` absent.
- `.pi/settings.json` defaults unchanged (`openai` / `openai/gpt-5.6-terra`,
  `enabledModels` order) and no literal keys.

Updated `eval/conftest.py`: `run_pi_print()` delegates to `pi-run print`
(instead of calling `pi` directly), preserving provider control for eval runs.

Manual verification: `pi-run doctor`, `pi-run config-check`, `pi-run print` with
each provider, one live `pi-run chat --provider deepseek`.

## Migration Plan (execution order)

1. Move `github-repo-controller/` → `~/Projects/github-repo-controller`
   (`git subtree split --prefix=github-repo-controller`, init own repo).
2. Move `chatbot-project/` + `projects/chatbot/` → `~/Projects/chatbot`
   (subtree split, init own repo); delete now-empty `projects/`.
3. Build the CLI + Go tests (TDD).
4. `pi-run install` → binary in `bin/`, symlink in `~/bin/`.
5. Remove `Makefile`; remove `pi-harness`/`pi-harness-print` from `~/.zshrc` and
   `~/.bashrc`; remove `scripts/verify-harness.sh`; keep `scripts/install-skills.sh`.
6. Rewrite `eval/tests/test_harness_config.py`; point `eval/conftest.py`
   `run_pi_print` at `pi-run print`.
7. Update `README.md` / `AGENTS.md` (commands table, provider docs; remove
   controller, chatbot, and golden-dataset sections); update `.lore.md` with the
   new architecture decision.

## Out of Scope

- No interactive provider picker.
- No config file (`viper`/YAML) — flags + `PI_PROVIDER` only.
- No changes to the DeepEval test logic itself (suite stays pytest).
- No changes to `scripts/install-skills.sh` behavior.
- No removal of the rc-file startup key exports (shared with other tooling).
- `github-repo-controller` and chatbot work continue in their new standalone
  repos — out of scope here.
