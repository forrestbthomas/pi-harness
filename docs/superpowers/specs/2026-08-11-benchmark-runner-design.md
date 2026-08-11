# `pi-run eval --benchmark` — Docker-Isolated Task Runner — Design

**Date:** 2026-08-11
**Status:** Proposed (from competitive gap analysis, docs/competitive-gap-analysis-2026-08.md)
**Target release:** v0.5.0 (feature)

## 1. Context & Motivation

**Gap:** Our eval suite (`pi-run eval`) is a smoke/quality gate — 5 hand-written
samples in `eval/datasets/coding_samples.jsonl`, run in the local working tree
with no isolation. The community's eval harnesses (SWE-bench, terminal-bench,
aider, OpenHands) are **research-grade benchmarks**: Docker-isolated task
environments, scored pass/fail, reproducible, and comparable across agents and
providers.

**Opportunity:** our core differentiator is "provider-agnostic harness + built-in
eval." A benchmark mode that runs the **same tasks in isolated containers across
any provider** ("run the same eval against OpenAI, Anthropic, DeepSeek, or a
local model — see which actually solves your tasks") is a story no other project
can tell. OpenCode has providers but no eval; SWE-bench has eval but no harness
UX; we would have both.

## 2. Current State (verified)

- `eval/tests/` has 5 pytest files; `eval/datasets/coding_samples.jsonl` has 5 samples.
- `pi-run eval` runs pytest against `eval/.venv`; `--quick` = deterministic subset;
  live tests skip when no provider key is present.
- The CLI is stdlib-only Go 1.26; Python deps are `~=`-bounded (deepeval, pytest,
  pytest-timeout, pypdf, python-dotenv). No Docker dependency today.
- Existing optional-tool patterns to reuse: `pi_available` fixture skips tests
  when Pi is missing; cross-language contract tests skip when `python3` is absent.

## 3. Scope

### In scope

1. **Benchmark task format** (`eval/benchmarks/<name>/task.json`):
   - `id`, `prompt` (or `instruction.md` path), optional `setupCmd`,
     optional `repo` (git URL to clone as the task workspace),
     optional `timeoutSecs`, optional `solution` path.
   - `tests/run.sh` — verification script; **exit 0 = pass** (any other = fail).
   - Optional `environment/Dockerfile` — isolated test environment
     (default: a pinned Python/Ubuntu base image).
2. **`pi-run eval --benchmark [name]`** — run the benchmark suite:
   - For each task: build/run the task environment in Docker, apply the agent's
     edits, run `tests/run.sh`, grade pass/fail, record timing.
   - `--provider <name>` / `--model <id>` select the agent; run the same suite
     against multiple providers for comparison.
   - Requires Docker; if `docker` is unavailable, print a clear message and
     exit with a distinct code (reuse the optional-tool skip pattern).
   - `--benchmark-dry-run` validates task format without Docker (hermetic).
3. **Result output**: human-readable summary (per-task pass/fail + aggregate
   score) and JSON under `eval/benchmark-results/<run-id>.json` (gitignored).
4. **Seed benchmark suite**: 5-10 tasks mirroring common coding tasks (fix a
   failing test, implement a function, refactor, add a CLI flag), each with a
   Dockerfile + `tests/run.sh`. Keep it small and dependency-light for CI.
5. **Tests**: Go unit tests for task parsing/validation + result aggregation
   (hermetic, no Docker); eval suite gains a benchmark-format validation test.

### Explicitly NOT in scope

- Full SWE-bench compatibility (thousands of tasks, gold-patch diff grading) —
  start with pass/fail test grading; diff-based grading is a later iteration.
- Cloud eval (Modal/AWS) — local Docker only.
- Running the *agent* inside Docker — the agent edits a local workspace; only
  **verification** runs in the container (simpler, matches our local-run model).
- A leaderboard UI — JSON output is the contract; a report page is later.
- Sandboxing the daily-driver `chat` path — benchmark-only isolation.

## 4. Design

### Task format (`eval/benchmarks/<name>/task.json`)

```json
{
  "id": "fix-divide-by-zero",
  "prompt": "Fix the divide-by-zero crash in eval/benchmarks/fix-divide-by-zero/src/calc.py so all unit tests pass.",
  "setupCmd": "pip install -r requirements.txt",
  "timeoutSecs": 300,
  "testScript": "tests/run.sh",
  "dockerfile": "environment/Dockerfile",
  "solution": "solution/calc.py"
}
```

Layout:

```
eval/benchmarks/fix-divide-by-zero/
├── task.json
├── environment/Dockerfile      # optional; default pinned base
├── src/…                       # task workspace the agent edits
├── tests/run.sh                # exit 0 = pass
└── solution/…                  # optional oracle (for future diff grading)
```

### Flow (per task)

1. Parse `task.json`; if `repo` given, clone into a temp workspace; else copy
   `src/` into the workspace.
2. Run the agent: `pi -p "<prompt>" --provider <p> --model <m>` with cwd =
   workspace (reuse `execPi`/`launchEnv` plumbing).
3. Build the task image from `dockerfile` (or default base) with the workspace
   mounted/copied in; run `setupCmd` then `tests/run.sh`.
4. Exit 0 → pass; else fail. Record elapsed time and (for live runs) the
   provider/model used.

### CLI

```
pi-run eval --benchmark [name] [--provider <p>] [--model <m>] [--benchmark-dry-run]
```

`name` optional (all benchmarks if omitted). `--provider`/`--model` passed to
the agent launch. `--benchmark-dry-run` validates all task.json files and exits
without Docker/keys (hermetic, CI-safe).

### Go structure (stdlib-only)

```
internal/cli/benchmark.go        // task parsing, validation, orchestration
internal/cli/benchmark_test.go   // hermetic unit tests (no Docker)
```

Docker invocation is `exec.Command("docker", ...)` — stdlib-only, no new deps.

## 5. Implementation Plan

1. `eval/benchmarks/` seed suite (5-10 tasks) + `task.json` schema + validation.
2. `internal/cli/benchmark.go`: parse/validate tasks, dry-run mode, aggregation.
3. Docker orchestration: build/run image, mount workspace, run tests/run.sh.
4. `runEval` flag wiring (`--benchmark`, `--benchmark-dry-run`, `--provider`,
   `--model`).
5. Result JSON writer + human summary; gitignore `eval/benchmark-results/`.
6. Tests: hermetic Go unit tests + a benchmark-format eval test (no Docker).
7. Docs: README "Benchmarks" section + `docs/benchmarks.md` task-contribution
   guide; CHANGELOG v0.5.0 entry.

## 6. Changelog Entry (draft)

```markdown
## [0.5.0] - 2026-08-11

### Added
- `pi-run eval --benchmark`: Docker-isolated, scored benchmark runner. Run the
  same task suite against any provider and compare results. Includes a seed
  benchmark suite, hermetic dry-run validation, and JSON result output.
```

## 7. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Docker not installed → benchmark unusable | Medium | Clear skip message + distinct exit code; dry-run works without Docker; tests skip |
| Containers are heavy for CI | Medium | Keep seed suite small; document `--benchmark-dry-run` for CI |
| Agent edits don't map to container | Medium | Tests run against the same workspace files the agent edited (mounted volume), so edits are reflected |
| Benchmark task quality varies | Medium | `--benchmark-dry-run` validates format; task-contribution guide; start small (5-10) |

## 8. Decision

**Recommend proceeding.** This is the single highest-leverage feature from the
gap analysis: it converts our "built-in eval" from a smoke gate into a
**provider-comparison benchmark**, directly serving the anti-lock-in pitch.
Scope is deliberately v1-minimal (local Docker, test-pass grading, seed suite)
with clear extension points (diff grading, cloud eval) for later.
