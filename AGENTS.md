# Agent Harness: Project Instructions

This repository is a **coding agent evaluation harness** built around the [Pi coding agent](https://pi.dev/) and the [DeepEval](https://github.com/confident-ai/deepeval) LLM evaluation framework.

## Project Goals

1. Provide a reproducible Pi environment for running coding agents.
2. Run automated evaluations against agent outputs using DeepEval.
3. Keep agent behavior safe, correct, and measurable.

## Conventions

- All Pi configuration lives in `.pi/`.
- All evaluation code, datasets, and Python dependencies live in `eval/`.
- The harness runtime is a single Go CLI: `pi-run` (module `github.com/forrestthomas1/pi-harness`, source under `cmd/pi-run` and `internal/cli/`).
- Prefer deterministic, reproducible steps over ad-hoc commands.
- Never commit API keys, tokens, or kubeconfig contents.
- Keep changes minimal and focused; do not refactor unrelated code.

## Workflow

1. Provider credentials come from **Bitwarden** (CLI `bw`, folder "Dev API
   Keys"), resolved on demand by `pi-run` via `~/bin/bw_get <ENV_VAR>` — no
   static keys live in shell rc files. The default provider is **OpenAI**
   (`OPENAI_API_KEY`, `gpt-5.6-terra` via the OpenAI API directly);
   `OPENROUTER_API_KEY` (OpenRouter) and `DEEPSEEK_API_KEY` (DeepSeek direct)
   are explicit alternatives via `--provider` / `PI_PROVIDER`. Unlock the
   vault first: `bw unlock` (or `export BW_SESSION=...`).
2. Run Pi from the repository root so it loads `AGENTS.md`, `.pi/SYSTEM.md`, and project packages.
3. Capture agent outputs to `eval/outputs/` or pipe them into DeepEval tests.
4. Run `pi-run eval` to execute the evaluation suite.

## Safety Rules

- Do not run destructive commands (`rm -rf`, `git reset --hard`, database drops, CRD deletion in shared clusters) without explicit confirmation.
- Do not install global system packages; use `eval/.venv` for Python dependencies and `.pi/npm/` for Pi packages.
- Do not push sessions or evaluation outputs containing sensitive data.

## Useful Commands

- `pi-run chat` — launch Pi interactively (default OpenAI).
- `pi-run print --provider deepseek "<prompt>"` — run Pi in print mode with an explicit provider.
- `pi-run eval` — run the full DeepEval test suite.
- `pi-run eval --quick` — run a small smoke subset.
- `pi-run config-check` — run deterministic harness-config checks (no API key needed).
- `pi-run doctor` — one-shot environment/setup verification.

<!-- This section is maintained by the coding agent via lore (https://github.com/BYK/loreai) -->
## Long-term Knowledge

For long-term knowledge entries managed by [lore](https://github.com/BYK/loreai) (gotchas, patterns, decisions, architecture), see [`.lore.md`](.lore.md) in the project root.
<!-- End lore-managed section -->
