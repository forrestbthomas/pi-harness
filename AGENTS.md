# Agent Harness: Project Instructions

This repository is a **self-healing, measurable coding-agent harness** built around the [Pi coding agent](https://pi.dev/) and the [DeepEval](https://github.com/confident-ai/deepeval) LLM evaluation framework.

## Charter & scope (read first)

[`CHARTER.md`](CHARTER.md) is the project's boundary contract — read it before proposing or building anything. In one line: **one product, the harness** (`pi-run` + its measurement layer). We are explicitly **not** a benchmark product, a PM system, a spec library, an observability platform, or a general MCP platform. The eval suite under `eval/` is the harness's measurement layer (the "measurable" in the north star), not a separate product today; a standalone benchmark repo (pi-bench) is a *triggered* future split (EPIC-1 DoD closed **and** an external consumer), not current scope. "Distributable" is earned: macOS/Homebrew is the shipped leg; Windows/cloud are non-goals until a consumer asks.

## Project Goals

1. Provide a reproducible Pi environment for running coding agents.
2. Run automated evaluations against agent outputs using DeepEval.
3. Keep agent behavior safe, correct, and measurable.

## Conventions

- All Pi configuration lives in `.pi/`.
- All evaluation code, datasets, and Python dependencies live in `eval/`.
- The harness runtime is a single Go CLI: `pi-run` (module `github.com/forrestbthomas/pi-harness`, source under `cmd/pi-run` and `internal/cli/`).
- Prefer deterministic, reproducible steps over ad-hoc commands.
- Never commit API keys, tokens, or kubeconfig contents.
- Keep changes minimal and focused; do not refactor unrelated code.

## Repository navigation (how to find things)

| Path | What lives there |
|---|---|
| `cmd/pi-run/`, `internal/cli/` | The Go CLI (single module `github.com/forrestbthomas/pi-harness`): command dispatch + all logic (watchdog, self-heal, cost, provider routing, doctor, config-check) |
| `eval/` | DeepEval suite: datasets (`datasets/`), tests (`tests/`), scripts (`scripts/`, incl. `score_run.py`), baselines (`baselines/`) |
| `docs/` | Architecture, specs (`superpowers/specs/`), knowledge base (`knowledge-base/`), workflow (`roadmap-workflow.md`), research (`*.md`) |
| `.pi/` | Pi runtime config: `settings.json` (project packages incl. pinned `pi-subagents`), `SYSTEM.md`, agent wrappers (`agents/`), npm workspace (`npm/`) |
| `.github/workflows/` | CI: `ci.yml`, `nightly-live-eval.yml`, `release.yml`, `provider-scorecard.yml`, `codeql.yml` |
| `scripts/` | Bootstrap, install-skills, build-release, update-homebrew-formula, tag-release |
| `examples/` | Worked examples (subagents, plugins) |
| Root docs | `ROADMAP.md` (workstreams + DoD), `BACKLOG.md` (ranked RICE queue), `STATUS.md` (one-screen snapshot), `SCOPE.md` (active scope-lock contract), `CHANGELOG.md` (shipped = versions), `AGENTS.md` (this file) |

**Session entry:** start at `STATUS.md` → `ROADMAP.md` → `BACKLOG.md`, then
follow the ritual in `docs/roadmap-workflow.md` (cycle cadence, ownership,
traceability, release + git hygiene rules).

## Project Management (added 2026-08-13)

This project is managed against `ROADMAP.md` and `BACKLOG.md` (ranked by rough
RICE). **Every session starts by reading `STATUS.md` (one-screen) →
`ROADMAP.md` → `BACKLOG.md`**, then follows the cycle ritual in
`docs/roadmap-workflow.md`. Rituals every session must honor:

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
3. Capture agent outputs to `eval/live-results/` (the score_run consumer) or pipe them into DeepEval tests.
4. Run `pi-run eval` to execute the evaluation suite.

## Safety Rules

- Do not run destructive commands (`rm -rf`, `git reset --hard`, git history rewrite) without explicit confirmation.
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

## Long-term Knowledge

Historical project knowledge is frozen in [`.lore.md`](.lore.md) (no longer
maintained by lore; kept as a dated record). For current state, see
`ROADMAP.md`, `BACKLOG.md`, `CHANGELOG.md`, and `docs/`.
