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

## Project Management (added 2026-08-13)

This project is managed against `ROADMAP.md` and `BACKLOG.md` (ranked by rough
RICE). Rituals every session must honor:

1. **Scope discipline** — before implementing any new workstream, invoke the
   `scope-lock` skill to generate a `SCOPE.md` boundary contract from the
   approved plan; flag deviations instead of silently absorbing them. If a
   requested change serves no roadmap/backlog item, ask before building.
2. **Prioritization** — new ideas get a one-paragraph pitch + DoD + rough RICE
   in `BACKLOG.md` before becoming active work; promotion to `ROADMAP.md` is a
   user decision.
3. **Definition of Done** — every workstream in `ROADMAP.md` has an explicit
   DoD checklist; a change is not complete until its DoD boxes are closed and
   `verification-before-completion` has run.
4. **Skills** — PM/SDLC skills are installed via `scripts/install-skills.sh`
   (durable clones under `~/.pi/agent/skills/`): `scope-lock` (anti-scope-creep),
   `productskills` (feature-prioritization, roadmap-planning, scope-cutting,
   prd-writing, …), `spec-coding-skills` (spec-plan, spec-crlp, spec-index).
   Re-run the script to refresh; `pi-run config-check` verifies registration.

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

## Subagent Support

This harness is subagent-capable via the `pi-subagents` package (in
`.pi/settings.json`). Use natural language to delegate: "Use reviewer to
review this diff", "Run worker to implement this plan". See
`examples/subagents.md` for a worked scout → worker → reviewer loop.

**Subagent timeouts (required):** always pass an explicit `timeoutMs` on
subagent launches (e.g. `timeoutMs: 600000`), because async subagent runs
have no default timeout and a child `bash` tool call can block forever when a
command starts a background process that inherits the terminal. Project agent
wrappers in `.pi/agents/` already set a 10-minute default, but explicit launch
values are still preferred. Children are instructed (in the wrappers) to pass
`timeout:` on every `bash` call and to avoid long-running/background commands.

<!-- This section is maintained by the coding agent via lore (https://github.com/BYK/loreai) -->
## Long-term Knowledge

For long-term knowledge entries managed by [lore](https://github.com/BYK/loreai) (gotchas, patterns, decisions, architecture), see [`.lore.md`](.lore.md) in the project root.
<!-- End lore-managed section -->
