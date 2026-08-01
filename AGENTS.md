# Agent Harness: Project Instructions

This repository is a **coding agent evaluation harness** built around the [Pi coding agent](https://pi.dev/) and the [DeepEval](https://github.com/confident-ai/deepeval) LLM evaluation framework.

## Project Goals

1. Provide a reproducible Pi environment for running coding agents.
2. Run automated evaluations against agent outputs using DeepEval.
3. Keep agent behavior safe, correct, and measurable.
4. Build and evaluate Golang Kubernetes controllers.

## Conventions

- All Pi configuration lives in `.pi/`.
- All evaluation code, datasets, and Python dependencies live in `eval/`.
- Go/Kubernetes controller projects live at the repository root or under `projects/`.
- Prefer deterministic, reproducible steps over ad-hoc commands.
- Never commit API keys, tokens, or kubeconfig contents.
- Keep changes minimal and focused; do not refactor unrelated code.

## Controller Project Conventions

- Use Kubebuilder or controller-runtime layout.
- CRDs live in `config/crd/bases/`.
- API types live in `api/<version>/`.
- Controllers live in `internal/controller/`.
- Samples live in `config/samples/`.
- Always run `make manifests` and `make generate` after type changes.
- Always run `go build ./...` and `go test ./...` before declaring success.
- Do not apply manifests to a live cluster without explicit confirmation.

## Workflow

1. Configure provider credentials via environment variables or `/login`. Keys
   are stored in **Bitwarden** (CLI `bw`, folder "Dev API Keys") and resolved
   on demand with `~/bin/bw_get <ENV_VAR>` — no static keys live in shell rc
   files. The harness defaults to **OpenAI** (`OPENAI_API_KEY`, `gpt-4o`
   through the OpenAI API directly); `OPENROUTER_API_KEY` is the fallback
   router. Legacy providers (e.g. `KIMI_API_KEY`) remain configured.
   Unlock the vault first: `bw unlock` (or `export BW_SESSION=...`).
2. Run Pi from the repository root so it loads `AGENTS.md`, `.pi/SYSTEM.md`, and project packages.
3. Capture agent outputs to `eval/outputs/` or pipe them into DeepEval tests.
4. Run `make pi-eval` to execute the evaluation suite.
5. When working on a controller project, run Pi from that project's directory or from the harness root with the project path in the prompt.

## Safety Rules

- Do not run destructive commands (`rm -rf`, `git reset --hard`, database drops, CRD deletion in shared clusters) without explicit confirmation.
- Do not install global system packages; use `eval/.venv` for Python dependencies and `.pi/npm/` for Pi packages.
- Do not push sessions or evaluation outputs containing sensitive data.

## Useful Commands

- `make pi` — launch Pi interactively.
- `make pi-print "<prompt>"` — run Pi in print mode.
- `make pi-eval` — run the full DeepEval test suite.
- `make pi-eval-quick` — run a small smoke subset.
- `make pi-config-check` — run deterministic harness-config checks (no API key needed).
- `bash scripts/verify-harness.sh` — one-shot environment/setup verification.

<!-- This section is maintained by the coding agent via lore (https://github.com/BYK/loreai) -->
## Long-term Knowledge

For long-term knowledge entries managed by [lore](https://github.com/BYK/loreai) (gotchas, patterns, decisions, architecture), see [`.lore.md`](.lore.md) in the project root.
<!-- End lore-managed section -->
