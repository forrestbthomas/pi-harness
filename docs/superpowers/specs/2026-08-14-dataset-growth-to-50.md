# Spec — EVAL-5: Dataset Growth 20 → 50 (stratified, regression twins, edit-graded benchmarks) — W10

**Date:** 2026-08-14 · **Status:** DRAFT (awaiting user approval at scope-lock gate)
**Source:** BACKLOG EVAL-5 (1.60, EPIC-1) — bet on by user (2026-08-14). Sequence position: after EVAL-3 (versioning), so growth is attributable.

## Goal

Grow the live eval dataset from **20 → 50 cases** (stratified ≥8 per category, difficulty spread, regression twins for bug-fix), add **edit-based benchmark tasks** (patch-application grading via Docker hidden tests) so the suite stops being a smoke gate and starts being a benchmark.

## Problem Statement

20 cases (7 easy / 8 medium / 5 hard; code-gen 4, concept 3, shell/ops 3, bug-fix 4, negative-edge 3, harness-routing 3) is too small to separate agent quality from per-case noise — a single case is ~5% of the signal. Growth needs guardrails that already exist: EVAL-3 versioning (attributable), the schema lint (references must provably pass graders), flake-aware gating (W7), and the benchmark surface for edit-grading.

## Context From Code

- `eval/datasets/tasks.json`: `datasetVersion: "2026-08-14.1"`, 20 live tasks + 5 benchmark tasks; `test_dataset_schema.py` enforces exactly-20 + category budgets `(min,max)` + reference-provable deterministic graders.
- `eval/datasets/coding_samples.jsonl` + `eval/datasets/graders/coding-NNN/grade.py` + `eval/datasets/references/`.
- Benchmark surface: `eval/benchmarks/<id>/` with `src/`, `tests/run.sh`, `task.json` — agents edit `src/`, hidden tests grade the edited tree (patch-application grading).
- Baseline: new cases are **unbaselined** (recorded, never failed) until a live re-baseline run.

## In Scope

- **Live suite growth (20 → 50):**
  - +30 new cases; target ≥8 per category (code-gen, concept, shell/ops, bug-fix, negative-edge, harness-routing); difficulty spread (easy/medium/hard).
  - **Regression twins:** for each bug-fix case, a twin that proves the fix did not break a sibling behavior (e.g., fix-divide-by-zero has "still handles floats").
  - Each new deterministic grader + reference must provably pass (schema lint enforces via `eval/grader.py`); judge cases get non-empty references.
- **Benchmark surface (+3 edit-based tasks):** new `eval/benchmarks/*` tasks where the agent edits `src/` and hidden `tests/run.sh` grades the edited tree (SWE-bench-style patch application); each task has `task.json`, `src/`, `tests/`.
- **`test_dataset_schema.py`:** exactly-20 → exactly-50; category budgets updated; keep all existing invariants.
- **`eval/datasets/tasks.json`:** `datasetVersion` bump (`2026-08-14.2`); add the new tasks to the manifest.
- **Docs/records:** CHANGELOG, ROADMAP W10 row, STATUS, BACKLOG, SCOPE.md, spec; note that a live re-baseline (follow-up act) records the new cases.

## Out Of Scope

- Gold-patch diff *comparison* grading (byte/structural patch diff vs a gold patch) — edit-based hidden-test grading is the v1; true gold-patch diffing is a follow-on ticket if the maintainers want it.
- Multi-turn / agentic case family (EVAL-6 — separate ticket; print-mode cases only here).
- Sandboxing the live suite (EVAL-7 — separate ticket; benchmark tasks are already Docker-isolated).
- Re-baselining in this PR (live-run act; new cases stay unbaselined until then).

## Constraints

- Schema lint must stay green with 50 cases (references provably pass graders; enums respected).
- Stdlib-only, hermetic tests; no keys/network in the deterministic suite.
- Every new grader follows the existing `grade.py` contract (stdin = candidate text, exit 0 = pass).

## Assumptions

- A +30-case first increment (20 → 50) is the right batch size; further growth (→ 100) is a follow-on under the same EPIC-1 item.
- Print-mode cases remain the live surface; agentic/edit behavior is exercised by the benchmark tasks.
- Content quality is gated by the schema lint + reviewer passes, not by volume.

## Blocking Questions

None.

## Acceptance Criteria

Gherkin:

```gherkin
Scenario: Dataset grows to 50 with stratification
  Given the live dataset
  Then it has exactly 50 cases with unique ids
  And each category has >= 8 cases
  And difficulty is spread across easy/medium/hard
  And every bug-fix case has a regression twin (sibling behavior still works)

Scenario: All graders are provable
  Given the 50-case dataset
  When the dataset schema lint runs
  Then every deterministic reference provably passes its grader
  And every judge case has a non-empty reference

Scenario: Edit-based benchmark tasks grade the edited tree
  Given a new benchmark task
  When the agent edits src/ and tests/run.sh runs
  Then the hidden tests pass only when the edit is correct

Scenario: Growth is attributable
  Given tasks.json
  Then datasetVersion is bumped and well-formed
```

Non-behavioral: schema lint green; new benchmark tasks pass the benchmark-format lint (`test_benchmark_format.py`); full CI green.

## Edge Cases

- New deterministic grader flaky under hidden tests → the lint's reference-provable check catches it before merge.
- Category budget conflict (adding >4 code-gen violates the old budget) → budgets updated in the same PR as the cases.
- Judge cases without graders → reference-only grading like existing judge cases.

## Validation Plan

- `eval/.venv/bin/python -m pytest eval/tests/test_dataset_schema.py eval/tests/test_benchmark_format.py eval/tests/test_score_run.py eval/tests/test_docs_drift.py -v`.
- For each new deterministic grader: run `grade.py` against its reference locally (exit 0) and against a wrong answer (exit 1) — the lint does the former; spot-check the latter.
- `go build/vet/test` (no Go change expected, but full suite).

## Execution Plan (ordered — content batches land as separate PRs under this contract)

1. Content batch A: +15 live cases (code-gen + bug-fix + twins), dataset version bump, schema-lint budget update; verify references.
2. Content batch B: +15 live cases (concept, shell/ops, negative-edge, harness-routing); verify references.
3. Benchmark batch: +3 edit-based tasks; verify hidden tests pass on the reference solution and fail on a wrong edit.
4. Docs: CHANGELOG, ROADMAP W10, STATUS, BACKLOG.
5. Verification-before-completion per batch.
6. Land batches via PRs (`BACKLOG EVAL-5 — dataset growth`), ff-only sync; live re-baseline noted as follow-up.

## Open Questions

- None blocking. (True gold-patch diffing is a follow-on decision.)

## Self-Check

Goal user-visible ✅ · problem stated ✅ · context from code present ✅ · in/out scope non-conflicting ✅ · constraints actionable ✅ · assumptions visible ✅ · blocking questions none ✅ · acceptance criteria observable/testable ✅ · edge cases covered ✅ · validation plan concrete ✅ · execution order matches EPIC-1 sequence ✅
