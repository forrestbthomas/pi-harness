# Scope Contract
**Task:** TAX-2 — Flag/env-var prune audit | **Plan:** BACKLOG TAX-2 (EPIC-6, RICE ~0.8, 0.25 pw); ROADMAP §Release Milestones v0.11.0 | **Date:** 2026-08-14 | **Status:** CLOSED — 0 changes logged

## In Scope
- Audit every README-documented flag and env var against actual usage (`git grep`, Go `os.Getenv`/flag reads, workflow references).
- Fix the gap found: the watchdog env vars (`PI_SELF_HEAL`, `PI_STALL_TIMEOUT_SECS`, `PI_WATCHDOG_GRACE_SECS`) are real, tested Go knobs but were only in prose, not the README env table — add the three rows.
- Add a docs-drift guard so future env vars can't silently miss the table.

## Findings (honest outcome)
- **No dead flags.** All ~33 README flags are used (harness-owned or documented pi pass-through).
- **No dead env vars.** All Go `os.Getenv` reads correspond to real, tested knobs.
- **One documentation gap** (the "README lies" class the audit exists to catch): 3 watchdog env vars were in prose but missing from the env table. Fixed + guarded.
- `PI_KEY` in an initial grep was a false match (`OPENAI_API_KEY` suffix).

## Out of Scope
- Changing/deleting any flag or env var (none were dead; deletion is only justified for dead surfaces).
- GOV-2/GOV-1, EVAL-6, REL-3/4 (done), COST-2.
