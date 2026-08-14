# The Harness ↔ Eval Seam (versioned contract)

> **EVAL-15 (2026-08-14).** This document pins the boundary between the
> harness product (Go CLI, this repo) and the eval/benchmark measurement
> layer, so the charter's promise — *"the versioned contract keeps the
> pi-bench split cheap either way"* — is a testable claim, not a hope. It is
> the manifest for the hermetic dry-run in `eval/tests/test_benchmark_seam.py`,
> which inventories current coupling honestly (it does not pretend the seam is
> clean today).
>
> **Split trigger (charter):** the eval layer becomes its own repo (pi-bench)
> only when **EPIC-1's DoD closes AND an external consumer appears** (or a
> release is actually blocked by cross-layer coupling). This doc is the
> handoff kit for that moment.

## What lives on the eval side of the seam

| Path | Role | Versioning |
|---|---|---|
| `eval/datasets/tasks.json` | Manifest: `datasetVersion` + task table (58 tasks: 50 live + 8 benchmark) | `datasetVersion` `YYYY-MM-DD.N`, schema-lint guarded |
| `eval/datasets/coding_samples.jsonl` | 50 live cases (JSONL), each with `grader` + `graderRef` + `reference` | under the manifest `datasetVersion` |
| `eval/datasets/graders/` | 42 deterministic grader scripts (exit 0 = pass) | content moves with the dataset |
| `eval/datasets/references/` | 50 reference answers/solutions (schema lint proves each passes) | content moves with the dataset |
| `eval/benchmarks/` | 8 edit-based Docker benchmark tasks (`task.json` + hidden `tests/run.sh`) | under the manifest `datasetVersion` |
| `eval/scripts/score_run.py` | Baseline gate + scorecard builder for the live suite | `SCHEMA_VERSION = 1` |
| `eval/grader.py` | Shared grading harness (`run_grader(task_dir, output)`) | stdlib-only |
| `eval/secret_backend.py` | Secret resolution mirror for the Python side | stdlib-only; contract-tested against Go |
| `eval/baselines/live-baseline.json` | Committed live baseline (schemaVersion 1, per-case passRate/cost) | `runsPerCase` + generated stamp |

## The scorecard contract (what the harness and the eval both write/read)

- **Live nightly** (`score_run.py` summary JSON): `schemaVersion`, `run`,
  `totals`, `gate`, `unbaselined`, `flakes`, `costFlakes`, `provenance`
  (`datasetVersion`/`agentModel`/`judgeModel`/`piVersion`), `cases`, `selfHeal`.
- **ci-benchmark** (Go `scorecard` struct): `schemaVersion`, `runId`,
  `timestamp`, `suite`, `gates`, `providers`, `baseline`, `selfHeal`,
  `provenance` (same shape, EVAL-14), `passed`.
- **Contract rule:** both surfaces carry the same `provenance` fields and a
  resolvable `piVersion` (never `dev`/`unknown` on a released build).

## Known coupling today (recorded honestly, 2026-08-14)

These are the things a pi-bench split must decouple. The dry-run
(`test_benchmark_seam.py`) inventories them mechanically into
`eval/live-results/seam-report.json`:

1. **`score_run.py` computes `repo_root()` as `parents[2]`** and reads
   `eval/datasets/tasks.json`, `eval/baselines/live-baseline.json`, and
   `eval/live-results/` from it — the eval layer assumes the harness repo
   layout.
2. **`score_run.py` reads `.pi/heal/events.jsonl`** (self-heal observability,
   W6/W9) — a harness-runtime path the eval surface consumes.
3. **11 `eval/tests/*.py` spawn the `pi-run` binary** (contract + live tests)
   via `pi_run_bin`/`subprocess` — the eval test surface currently depends on
   the Go binary (the nightly `go build`s it for exactly this reason).
4. **`conftest.py` fixture** `pi_run_bin` builds the harness binary and probes
   `--exit-codes`/usage — the eval harness needs a built `pi-run`.
5. **`eval/secret_backend.py` is contract-tested against Go**
   (`internal/cli/secret_contract_test.go`) — cross-language coupling.

**What this means:** the seam is *real* (schema-versioned, provenance-carrying)
but *not yet self-contained*. The split is cheap *only after* the coupling
above is decoupled or explicitly owned by the split. That is the honest
finding the dry-run exists to record.

## What the split (when triggered) takes with it

`eval/datasets/`, `eval/benchmarks/`, `eval/scripts/score_run.py`,
`eval/grader.py`, `eval/secret_backend.py`, `eval/baselines/` — and the
Python test surface that does **not** require the Go binary. The harness keeps
`internal/cli/` (including the Go scorecard), the workflows that orchestrate
the nightly/weekly runs, and the seam contract itself (this doc moves with the
eval side).
