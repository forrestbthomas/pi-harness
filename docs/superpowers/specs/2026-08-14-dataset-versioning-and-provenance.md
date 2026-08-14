# Spec — EVAL-3: Dataset Versioning + Scorecard Provenance — W8

**Date:** 2026-08-14 · **Status:** DRAFT (awaiting user approval at scope-lock gate)
**Source:** BACKLOG EVAL-3 (0.90, EPIC-1) — bet on by user (2026-08-14). Sequence position: must land before EVAL-5 (dataset growth) so growth is attributable.

## Goal

Make every scorecard **attributable**: it records which dataset version, pi version, agent tier, and judge model produced the numbers — and the dataset itself carries an explicit version that a hermetic guard forces to be bumped on every content change.

## Problem Statement

The scorecard records `report`/`baseline` paths but not **which dataset content** or **which pi binary** produced a run. As EVAL-5 grows the dataset (and as pi releases land), an unattributable scorecard is noise: you can't tell whether a pass-rate change came from the agent, the dataset, the judge, or the toolchain. `write_baseline` already records `agentModel`/`judgeModel` from env; the summary does not, and the dataset has no content version at all.

## Context From Code

- `eval/datasets/tasks.json`: has `schemaVersion: 1` (the schema, not the content) + `surfaces` + `tasks`. No content version.
- `eval/tests/test_dataset_schema.py`: hermetic dataset lint (20 records, enums, graderRef, reference-provable, etc.) — the natural home for a version guard.
- `eval/scripts/score_run.py`:
  - `repo_root()` (line ~51) resolves `<root>` from `__file__` — score_run can read `eval/datasets/tasks.json` hermetically.
  - `build_summary` includes `run` (report/baseline/runs/tolerance/budgetUsd/exitcode); `build_compact_summary` copies `run`.
  - `write_baseline` reads `PI_MODEL_TIER` / `OPENAI_MODEL_NAME` from env (provenance precedent).
- Version: `internal/cli/app.go:11` — Version injected via ldflags; `pi-run version` prints it (app.go:110). The nightly can capture it into `$GITHUB_ENV`.

## In Scope

- **`eval/datasets/tasks.json`**: add `datasetVersion` (e.g. `"2026-08-14.1"`; format `YYYY-MM-DD.N`). `schemaVersion` stays as the schema marker.
- **`eval/scripts/score_run.py`**: add a `provenance` block to the summary and compact summary:
  - `datasetVersion` — read from `eval/datasets/tasks.json` via `repo_root()` (missing → `"unknown"`).
  - `agentModel` — env `PI_MODEL_TIER` (default `"unknown"`).
  - `judgeModel` — env `OPENAI_MODEL_NAME` (default `"unknown"`).
  - `piVersion` — env `PI_VERSION` (default `"unknown"`).
- **Guard (contamination/drift):** `eval/tests/test_dataset_schema.py` asserts `tasks.json` carries `datasetVersion` matching `^\d{4}-\d{2}-\d{2}\.\d+$` — every dataset content change must bump it.
- **Tests (hermetic):** `test_score_run.py` — summary + compact carry `provenance.datasetVersion` equal to the real tasks.json value; env-driven agent/judge/pi recorded; defaults `"unknown"` when unset.
- **Nightly:** capture `pi-run version` into `PI_VERSION` (`$GITHUB_ENV`) before the score_run step so piVersion is real.
- **Docs/records:** CHANGELOG `[Unreleased]`; ROADMAP W8 row; STATUS; BACKLOG (EVAL-3 → active); SCOPE.md; spec.

## Out Of Scope

- EVAL-5 dataset growth (this only adds the version field + provenance).
- Per-case provenance (dataset/pi are run-level).
- Re-baselining or any baseline-math change (provenance is informational).
- Provider-scorecard (`ci-benchmark`) provenance — Go side, separate ticket if wanted.

## Constraints

- Stdlib-only Python in `eval/`; hermetic tests (no keys/network).
- Provenance must never change gate math.
- `datasetVersion` bump rule is enforced by the schema lint, not by convention.

## Assumptions

- A date-based version (`YYYY-MM-DD.N`) is the right granularity for a single-owner dataset; switch to a stricter scheme if the dataset becomes multi-contributor.
- `PI_VERSION` env is the cleanest hermetic channel for pi version; the nightly populates it.

## Blocking Questions

None — bet approved; decisions above are the first pass.

## Acceptance Criteria

Gherkin:

```gherkin
Scenario: Scorecard records provenance
  Given tasks.json has datasetVersion "2026-08-14.1" and env has PI_MODEL_TIER/PI_VERSION
  When score_run.py runs
  Then the summary and compact summary contain provenance.datasetVersion
  And provenance.agentModel/judgeModel/piVersion match the env (or "unknown" when unset)

Scenario: Dataset version is guarded
  Given tasks.json lacks a well-formed datasetVersion
  When the dataset schema lint runs
  Then it fails with a clear message
```

Non-behavioral:

- Hermetic tests cover both; the schema lint runs in the deterministic CI job.
- `go build/vet/test`, the deterministic pytest set, and docs-drift stay green.

## Edge Cases

- tasks.json missing/unreadable → `datasetVersion: "unknown"` (best-effort, never fatal).
- Env vars unset → `"unknown"` for that field (same pattern as `write_baseline`).
- datasetVersion format drift → schema lint fails (the guard).

## Validation Plan

- `eval/.venv/bin/python -m pytest eval/tests/test_score_run.py eval/tests/test_dataset_schema.py eval/tests/test_docs_drift.py -v`.
- Full deterministic CI set locally.
- `go build ./... && go vet ./... && go test ./...`.

## Execution Plan (ordered)

1. RED: failing tests — schema-lint `datasetVersion` guard; score_run provenance assertions (missing field fails first).
2. GREEN: add `datasetVersion` to tasks.json; implement `provenance` in score_run; add the nightly PI_VERSION step.
3. Docs: CHANGELOG, ROADMAP W8, STATUS, BACKLOG.
4. Verification-before-completion.
5. Land via PR (`BACKLOG EVAL-3 — dataset versioning + provenance`), ff-only sync.
6. Post-merge docs reconciliation + W8 DoD close.

## Open Questions

- None blocking.

## Self-Check

Goal user-visible ✅ · problem stated ✅ · context from code present ✅ · in/out scope non-conflicting ✅ · constraints actionable ✅ · assumptions visible ✅ · blocking questions none ✅ · acceptance criteria observable/testable ✅ · edge cases covered ✅ · validation plan concrete ✅ · execution order matches EPIC-1 sequence (EVAL-3 before EVAL-5) ✅
