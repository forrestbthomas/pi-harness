# Scope Contract
**Task:** W8 — EVAL-3: dataset versioning + scorecard provenance | **Plan:** `docs/superpowers/specs/2026-08-14-dataset-versioning-and-provenance.md` | **Date:** 2026-08-14 | **Status:** ACTIVE (approved 2026-08-14)

> Supersedes the W7 flake-aware gate contract (CLOSED 2026-08-14; record preserved in git history). W5 Part C remains tracked in ROADMAP (upstream release gate).

## In Scope
- **Files — harness:**
  - `eval/datasets/tasks.json` — add `datasetVersion` (`YYYY-MM-DD.N`); `schemaVersion` unchanged
  - `eval/scripts/score_run.py` — `provenance { datasetVersion, agentModel, judgeModel, piVersion }` in summary + compact (env-driven for models/pi; tasks.json for dataset; defaults `"unknown"`)
  - `eval/tests/test_dataset_schema.py` — guard: tasks.json must carry a well-formed `datasetVersion`
  - `eval/tests/test_score_run.py` — provenance tests (real datasetVersion, env models/pi, `"unknown"` defaults, summary + compact)
  - `.github/workflows/nightly-live-eval.yml` — capture `pi-run version` into `PI_VERSION` (`$GITHUB_ENV`) before the score_run step
  - `CHANGELOG.md`, `ROADMAP.md` (W8 row), `STATUS.md`, `BACKLOG.md` (EVAL-3 → active), `SCOPE.md`, spec
- **Features:**
  1. Dataset carries an explicit, guarded content version.
  2. Every scorecard (summary + compact) records dataset/agent/judge/pi provenance.
  3. Hermetic tests + the schema lint enforce both.
- **Boundaries:**
  - Provenance is informational; gate math unchanged.
  - Dataset content is NOT changed (only the version field added).
  - Missing file/env → `"unknown"`, never fatal.

## Out of Scope
- EVAL-5 dataset growth.
- Per-case provenance.
- Re-baselining / baseline-math changes.
- `ci-benchmark` (Go) provenance.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] — (none yet)
