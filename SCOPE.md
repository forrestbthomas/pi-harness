# Scope Contract
**Task:** Execute the documentation-audit change-set (5 Technical Writer subagents' converged 4-wave plan) | **Plan:** `.pi/debate/docs-audit/synthesis.md` | **Date:** 2026-08-14 | **Status:** ACTIVE

## In Scope (the 4 waves, as 3 PRs to avoid a git mess)
- **PR 1 — Wave 1 (P0): Truth restoration.** Dataset-count sweep to `tasks.json` authority (README/seam-doc/PM docs/test docstrings/nightly header → 62/54/49/54, datasetVersion .7) + restore the data-vs-prose drift guard in `test_docs_drift.py`; purge mcp-server ghost (plugins.md:54, gap-analysis banner, hardening spec); spec-status SHIPPED convention on 10 eval specs; PM reconciliation (13 backlog rows SHIPPED, STATUS regen, extend `test_pm_drift.py` "CHANGELOG entry ⇒ not open row").
- **PR 2 — Wave 2 (P1): Status & sequencing truth.** Canonical release procedure (tag-release.sh as CONTRIBUTING step 2; commit prefixes from git log); extend drift guards to Surface E (CONTRIBUTING prefixes, AGENTS.md step-3 target, SYSTEM.md commands ⊆ help); "Superseded" banners repo-wide; manifest↔jsonl parity lint + fix 13 records; GOV-2/GOV-1 sequencing + KB index row.
- **PR 3 — Wave 3+4 (P2/P3): Hygiene.** Template-residue purge (kind/CRD), AGENTS.md phantom eval/outputs → live-results, README Nightly/Skills/Layout refresh, spec citation hygiene, stamps (EPICS/anti-lockin/understand banner), CODEOWNERS/CoC defer notes.
- **Files (PR 1):** README.md, docs/benchmark-seam.md, docs/plugins.md, docs/competitive-gap-analysis-2026-08.md, ROADMAP.md, EPICS.md, STATUS.md, BACKLOG.md, eval/tests/test_live_suite.py, eval/tests/test_benchmark_seam.py, eval/tests/test_docs_drift.py, eval/tests/test_pm_drift.py, docs/superpowers/specs/2026-08-1[1-4]-*.md (10 specs), CHANGELOG.md, SCOPE.md, decision record.
- **Boundaries:** No product code changes (except test guards which are hermetic test files — those are the "guards" the audit restores). No docs deleted. CHARTER/SECURITY/benchmarks.md/anti-lockin.md/score_run docstrings protected from churn. Each PR verified (go build/test, hermetic pytest, docs-drift + pm-drift guards green).

## Out of Scope
- The coupling-count "11 vs 6 vs 13" dispute (resolved by mechanizing the count into seam-report — done in EVAL-15; PR 1 only cites the artifact).
- Content rewrites of dated research docs (banners only).
- Actual relocation GOV-2 (deferred; only sequencing text fixed).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | Wave 1 | P0 truth-restoration sweep (counts, guards, ghost, spec statuses, PM reconciliation) | docs-audit synthesis | Permit | PR #113 merged |
| 2 | Wave 2 | P1 status/sequencing truth (release procedure, Surface-E guards, banners, parity lint, GOV-2/GOV-1) | docs-audit synthesis | Permit | PR landing this change |

# Follow-up Tasks
- [ ] After PR 1: verify the data-vs-prose guard catches a deliberate count regression.
- [ ] After PR 3: confirm no Surface E drift guard failures.
