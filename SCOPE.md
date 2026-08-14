# Scope Contract
**Task:** REL-3 (post-release brew verify CI) + REL-4 (Node-drift guard) — release-machinery hardening | **Plan:** BACKLOG REL-3/REL-4 (EPIC-6, RICE ~1.5/~1.0); ROADMAP §Release Milestones v0.11.0 (both are v1.0.0 gates) | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged

## In Scope
- **Files — REL-3 (post-release brew verify):**
  - `.github/workflows/release.yml` — after the "Update Homebrew tap formula" step, add "Verify Homebrew install" step: `brew install --build-from-source` (or tap + install) in a temp prefix and assert `pi-run version` == `$GITHUB_REF_NAME`. On failure, mark the job failed (a tag shipped with a broken brew install is a silent distribution lie — TechLead's fire-and-forget `TAP_PUSH_TOKEN` finding).
  - `scripts/verify-homebrew-formula.sh` (NEW, small) — the verify logic: install into a temp prefix, run `pi-run version`, compare to the tag. Reusable locally.
  - `CHANGELOG.md`, `SCOPE.md`, decision record — updates.
- **Files — REL-4 (Node-drift guard):**
  - `internal/cli/doctor.go` — add an informational warning when `resolveNodeVersion` picks a major different from the documented reference (Node 22 LTS): e.g. `[info] node v24.x resolved — CI/tests pin Node 22 LTS; set PI_NODE_VERSION=v22.x for parity`. Not a failure (user machines legitimately vary), but the drift becomes visible.
  - `internal/cli/doctor_test.go` — hermetic test: with a fake nvm dir containing v24.x and PI_NODE_VERSION unset, doctor output contains the drift warning; with PI_NODE_VERSION=v22.x, no warning.
  - `eval/tests/test_docs_drift.py` — pin-drift CI guard: assert the nightly workflow's `PI_NODE_VERSION` (v22.19.0) is mentioned in doctor.go as the reference pin (so a silent CI pin bump without doctor parity fails).
  - `CHANGELOG.md`, `SCOPE.md`, decision record — updates.
- **Features:**
  1. REL-3: a shipped tag whose Homebrew install is broken fails the release job (closes the silent distribution break class).
  2. REL-4: user-machine Node drift is visible in `doctor`; the CI pin can't change without the doctor reference following.
- **Boundaries:**
  - REL-3 runs only on the release workflow (tag push), not CI per-PR.
  - REL-4 is informational in `doctor` (never a FAIL), matching the existing [info] convention.
  - No behavior change to pi launching; no change to resolveNodeVersion selection logic.

## Out of Scope
- Changing the Node pin itself (v22.19.0 stays).
- GOV-2/GOV-1, EVAL-6, TAX-2 (separate items).
- Making `doctor` FAIL on Node drift (user machines legitimately run other majors).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] After merge: next release tag exercises the new brew-verify step.
- [ ] Consider REL-4 extension: warn when deepeval/Go pins drift (COST-2 / GOV-1 toolchain extension later).
