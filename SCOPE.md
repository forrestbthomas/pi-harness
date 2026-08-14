# Scope Contract
**Task:** OSS-2 — Contributor on-ramp v2 | **Plan:** BACKLOG rank 1 (OSS-2, RICE ~2.0, 0.5 pw), EPIC-6 sequence | **Date:** 2026-08-14 | **Status:** ACTIVE

## In Scope
- **CONTRIBUTING.md** — contributor on-ramp v2:
  - **First-issue path** — new "Your first contribution" section: find a `good first issue` (label convention), comment to claim it, ask questions on the issue, expect a maintainer ping; what counts as in-scope vs a new feature needing a backlog entry.
  - **Review SLA** — maintainers aim to review/respond to PRs within 7 days (mirrors SECURITY.md's "You should receive a response within 7 days").
  - **MIT-in/MIT-out line** — all contributions are accepted under the MIT License (LICENSE); the project ships MIT — contributions come in MIT and go out MIT.
- **.github/PULL_REQUEST_TEMPLATE.md** — bugfix carve-out under Roadmap traceability: in-scope bugfixes (fixing broken shipped behavior) need no ROADMAP/BACKLOG citation; the scope rule's intent is anti-creep, not blocking fixes.
- **README.md** — Contributing section: point at the new CONTRIBUTING content, no duplication.
- **LICENSE** — *only if approved*: fix the copyright holder name to the canonical identity `forrestbthomas` (matches remote + module path; OSS-1 aligned identity everywhere except this line, which still says `forrestbthomas1` from the 2026-08-02 OSS spec).
- **PM reconciliation** (cycle ritual): BACKLOG OSS-2 → SHIPPED; EPICS.md DoD note; STATUS.md regenerate; CHANGELOG.md dated entry.

## Out of Scope
- OWN-1 CODEOWNERS ownership matrix (separate item; `.github/CODEOWNERS` already exists minimal).
- SECURITY supported-versions ritual line (separate backlog item).
- GOV-3 wire-drift-guards-into-CI (separate item).
- New issue templates (bug/feature templates already exist; the first-issue path lives in CONTRIBUTING + the label convention).
- Bot/automation (no first-issue bot, no stale-bot).
- No Go/Python product code changes.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] Verify the PR template renders the bugfix carve-out checkbox.
- [ ] Confirm the review-SLA wording mirrors SECURITY.md's 7-day language.
