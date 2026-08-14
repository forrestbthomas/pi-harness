# Scope Contract
**Task:** GOV-3 — Wire drift guards into CI (enforce GOV-1) | **Plan:** BACKLOG rank 1 (GOV-3, RICE ~2.0, 0.15 pw), EPIC-6 sequence | **Date:** 2026-08-14 | **Status:** ACTIVE

## In Scope
- **.github/workflows/ci.yml — python-quick job** (runs on every push):
  - Add `tests/test_docs_drift.py` and `tests/test_pm_drift.py` to the pytest invocation (the docs-audit-claimed home; today neither guard runs in any workflow).
  - Add a `git fetch --tags --force` step before the pytest run so `_git_tags()` finds the release tags — otherwise the tag↔changelog invariants (v0.7.0-gap class, REL-2) skip silently on the shallow checkout.
- **Verify a deliberate drift fails CI** (DoD): demonstrate locally that the guards catch a planted drift (same failure class GOV-1 is meant to enforce), then run a post-merge throwaway-branch check: push a branch with a planted drift, confirm python-quick fails, delete the branch.
- **PM reconciliation** (cycle ritual): BACKLOG GOV-3 → SHIPPED; EPICS.md EPIC-6 DoD note; STATUS.md regenerate; CHANGELOG.md dated entry.

## Out of Scope
- GOV-1 guard logic changes (extending `test_pm_drift.py`/`test_docs_drift.py` themselves — the guards are shipped; this item only wires them into CI).
- Nightly deterministic job changes (python-quick is the enforcement point; the nightly already runs a fixed deterministic list).
- OWN-1 CODEOWNERS matrix (separate item).
- EVAL-12 re-baseline (separate item, time-blocked on the nightly).
- No product code changes.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] Post-merge: throwaway-branch drift check proves python-quick fails on a planted drift.
