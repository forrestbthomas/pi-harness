# Scope Contract
**Task:** GOV-1 (mechanical core) — PM-layer drift guard: EPICS↔BACKLOG↔STATUS consistency + changelog-never-skips-tag | **Plan:** BACKLOG GOV-1 (EPIC-6, RICE ~3.0, 0.5 pw); ROADMAP §Release Milestones v0.11.0 | **Date:** 2026-08-14 | **Status:** CLOSED — 1 change logged

## In Scope
- **Files:**
  - `eval/tests/test_pm_drift.py` (NEW) — hermetic PM-layer drift guards (extends the `test_docs_drift.py` pattern; runs keyless in the deterministic suite):
    1. **EPICS↔BACKLOG consistency**: every BACKLOG ranked-row id with an Epic column exists in the EPICS.md item tables, and every EPICS.md item-table id appears in BACKLOG. (The "BACKLOG is the single ranked source of truth; the epic item table is an index" invariant.)
    2. **Changelog-never-skips-tag**: every `[Unreleased]` + released version section in CHANGELOG.md has a matching `vX.Y.Z` git tag, and (the class that burned us) every tag has a changelog section — checked via `git tag --list` (the repo is a git checkout in CI, so this is available; the test asserts the v0.7.0 gap class is closed).
    3. **STATUS↔ROADMAP**: STATUS "Now" rows reference shipped workstreams that exist in ROADMAP's active table or CHANGELOG (light sanity; the exact "In design after shipping" class).
  - `CHANGELOG.md`, `SCOPE.md`, decision record — updates.
- **Features:**
  1. The planning layer self-audits mechanically — the drift classes that burned us (EPIC-4/5 surviving the charter, the v0.7.0 changelog gap, STATUS stale after shipping) become CI failures instead of ritual reminders.
  2. Deterministic, hermetic (except `git tag` which is available in CI), runs in the deterministic job.
- **Boundaries:**
  - GOV-1 mechanical core only — the charter-citation / RICE-floor semantic layer (Purist's half) is deferred to GOV-1 slice 2 (avoid noise until the mechanical core is proven).
  - **Sequencing deviation (scope change #1):** GOV-1 ships BEFORE GOV-2 (relocation). The debate said GOV-2 first ("you can't guard a location that doesn't exist"), but the mechanical guard checks file *contents* (ids, sections, tags) — path-independent. GOV-2 (relocation) is deferred; when it lands, only the guard's path constants change.
  - No change to ROADMAP/BACKLOG/EPICS/STATUS content (the guard *checks* them; fixing any drift it finds is a follow-up if the test fails).

## Out of Scope
- GOV-2 (relocation + spec archive) — deferred (15-file churn for zero-consumer presentation value; the Skeptic's tax rule).
- GOV-1 semantic layer (charter-citation / RICE floor) — GOV-1 slice 2, after the mechanical core is proven non-noisy.
- EVAL-6 slice 2 (multi-turn/subagent — needs a chat-session runner), EVAL-12 (pending nightly), COST-2.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | user-expansion | GOV-1 ships before GOV-2 (debate sequenced GOV-2 first) | The mechanical guard checks file contents (ids/sections/tags), not locations — path-independent; GOV-2's relocation is presentation-only churn with zero consumers today | **Permit** — GOV-1 mechanical core first; GOV-2 deferred with its own backlog item; when GOV-2 lands, only path constants change | Done — test_pm_drift.py ships; GOV-2 stays deferred in BACKLOG |

# Follow-up Tasks
- [ ] GOV-2 (relocation) — deferred; update guard path constants when it lands.
- [ ] GOV-1 slice 2 (charter-citation / RICE-floor semantic layer) — after one month of mechanical-core CI proving non-noisy.
