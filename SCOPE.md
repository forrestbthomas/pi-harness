# Scope Contract
**Task:** EVAL-14 — Benchmark provenance parity (ci-benchmark scorecard carries live-schema provenance) | **Plan:** BACKLOG EVAL-14 (EPIC-1, RICE ~0.7, 0.5 pw); ROADMAP §Release Milestones v0.11.0 | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged

## In Scope
- **Files:**
  - `internal/cli/scorecard.go`:
    - New `scorecardProvenance` struct: `{ datasetVersion, agentModel, judgeModel, piVersion }` (all `omitempty`-safe; informational, never gated).
    - `buildScorecard`: populate `Provenance` from:
      - `datasetVersion` — read `eval/datasets/tasks.json` (best-effort, "unknown" on error — same class as Python `load_dataset_version()`)
      - `agentModel` — `PI_MODEL_TIER` env or "unknown"
      - `judgeModel` — `OPENAI_MODEL_NAME` env or "unknown"
      - `piVersion` — the `Version` ldflags var or "unknown"
    - `scorecard` struct gains `Provenance *scorecardProvenance \`json:"provenance,omitempty"\``
  - `internal/cli/testdata/scorecard-golden.json` — regenerate golden with provenance block.
  - `internal/cli/scorecard_test.go` — hermetic tests: golden round-trip still passes; provenance populated from env (PI_MODEL_TIER/OPENAI_MODEL_NAME) + tasks.json datasetVersion; missing env/file → "unknown" (best-effort, no crash).
  - `eval/tests/test_docs_drift.py` (or a new hermetic check) — assert the provider-scorecard workflow captures `pi-run version` for provenance (align with nightly's `PI_VERSION` handling).
  - `.github/workflows/provider-scorecard.yml` — record `pi-run version` into `PI_VERSION` env for the scorecard run (mirrors nightly `PI_VERSION=$(pi-run version)` step).
  - `CHANGELOG.md`, `SCOPE.md`, decision record — updates.
- **Features:**
  1. ci-benchmark scorecard JSON carries a `provenance` block matching the live schema — EPIC-1 DoD "provenance in every scorecard" now holds on **both** surfaces (nightly live + weekly ci-benchmark).
  2. Best-effort: missing datasetVersion/env never crashes the scorecard.
- **Boundaries:**
  - Informational only; no gate change, no baseline schema change, no failBelow/budget change.
  - No change to the Python live surface (already has provenance, EVAL-3).
  - No change to scorecard JSON consumers (adding an `omitempty` field is backward-compatible).

## Out of Scope
- EVAL-13 (cost-variance) — done (#105).
- EVAL-15 (split-seam), EVAL-8 (judge), EVAL-6 (agentic) — separate v0.11.0 items.
- GOV-2 (relocation), GOV-1 (drift guard) — separate EPIC-6 items.
- Changing the Python `build_provenance()` shape.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] After merge: confirm the next weekly provider-scorecard run surfaces provenance on the ci-benchmark scorecard.
