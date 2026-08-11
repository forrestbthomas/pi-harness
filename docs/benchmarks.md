# Adding Benchmark Tasks

This guide explains how to add a task to the `pi-run eval --benchmark` suite.
Tasks live under `eval/benchmarks/<name>/` and are validated hermetically with
`pi-run eval --benchmark-dry-run` (no Docker, no API keys — CI-safe).

## Layout

```
eval/benchmarks/<name>/
├── task.json             # required
├── environment/Dockerfile  # optional; default base is python:3.12-slim
├── src/                  # task workspace the agent edits (required for local tasks)
├── tests/run.sh          # verification script; exit 0 = pass (required)
└── solution/             # optional oracle for future diff grading
```

The **`src/` layout is a convention**: task prompts and `tests/run.sh` must
reference files as `src/...` (e.g. `PYTHONPATH=src python3 tests/test_x.py` or
`bash src/build.sh`). The runner preserves the `src/` subdirectory in the
workspace exactly as-is, so grading reaches the real code the agent edited.

## task.json fields

| Field | Required | Description |
|---|---|---|
| `id` | yes | Unique, safe name (`[a-zA-Z0-9][a-zA-Z0-9._-]*`); used in Docker image/container names and result paths. Duplicate ids across tasks are rejected. |
| `prompt` | one of | The task instruction given to the agent. |
| `instruction` | one of | Path to a markdown file with the task instruction (relative to the task dir). |
| `setupCmd` | no | Shell command run inside the container before `tests/run.sh`. |
| `repo` | no | Git URL cloned as the workspace, replacing `src/`. |
| `timeoutSecs` | no | Per-task wall-clock timeout for the agent run and container grading (default 300). |
| `testScript` | no | Relative path to the verification script (default `tests/run.sh`). |
| `dockerfile` | no | Relative path to a custom `environment/Dockerfile`. |
| `solution` | no | Relative path to an oracle solution directory (for future diff grading; not required for v1 pass/fail grading). |

Example:

```json
{
  "id": "fix-divide-by-zero",
  "prompt": "Fix the divide-by-zero crash in src/calc.py so safe_divide(1, 0) returns 0.0.",
  "timeoutSecs": 300,
  "testScript": "tests/run.sh"
}
```

## tests/run.sh contract

`tests/run.sh` is the grading script. **Exit 0 means the task is solved**;
any other exit code means failure. It runs inside the container with the
workspace mounted at `/workspace` and the current directory set to the
workspace root, so `src/` and `tests/` are directly addressable:

```sh
#!/usr/bin/env bash
# Benchmark verification: exit 0 = task solved, any other exit = fail.
set -euo pipefail
PYTHONPATH=src python3 tests/test_calc.py
```

The seed tasks (`fix-divide-by-zero`, `implement-factorial`, `add-verbose-flag`,
`fix-json-parsing`, `fix-shell-script`) are working examples — copy one and
adapt.

## Validating and running

```bash
# Validate every task's format (hermetic; use in CI)
pi-run eval --benchmark-dry-run

# Validate one task
pi-run eval --benchmark <name> --benchmark-dry-run

# Run one task against a provider (requires Docker + a key)
pi-run eval --benchmark <name> --provider deepseek --model deepseek/deepseek-v4-flash
```

## Quality bar

- The task must be **solved by its own `solution/`** (when shipped) and
  **failed by its buggy `src/`** (for no-solution tasks) — the hermetic
  smoke test `TestSeedBenchmarksGradeWithSolutions` checks exactly this for
  the seed suite, so new tasks should follow the same shape.
- Keep tasks small, dependency-light, and runnable in the pinned
  `python:3.12-slim` base (or your custom `environment/Dockerfile`).
- Ids must be unique across the whole `eval/benchmarks/` tree.
