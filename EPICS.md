# Epics — themed initiatives that group backlog items

## Why epics exist

The backlog is a ranked queue of individual bets. An **epic** groups related
items that together move one measurable outcome, so a theme can be shaped,
sequenced, budgeted, and reviewed as a unit instead of as disconnected PRs.
Epics are the PM layer between the ROADMAP (active workstreams) and the ranked
BACKLOG.

## Epic template

- **Outcome** — what is measurably true when the epic is done (numbers, not features).
- **Appetite** — the time/effort budget (Shape Up): fixed time, variable scope.
- **Why now** — one paragraph.
- **Sequence** — the dependency order items should be tackled in (evidence → analysis → growth).
- **DoD** — the exit checklist (closes like a workstream: DoD → CHANGELOG).
- **Items** — child backlog items (ids, RICE) that roll up to this epic.

Rules:

- An item belongs to at most one epic; items without an epic are `—`.
- The BACKLOG remains the single ranked source of truth; the epic item table is an index.
- Epics are bet on (promoted to ROADMAP) and closed like workstreams; status is tracked in STATUS.md.

---

## EPIC-1 — Eval suite: from smoke gate to research-grade benchmark

**Outcome.** When this epic is done:

- Every nightly/weekly eval leaves a **complete evidence artifact** (report, summary, heal events) on **every** gate outcome — zero lost runs.
- **Gate false-fails from single-run flake → 0/week**; the scorecard distinguishes flake from regression and reports flake rate.
- Every scorecard is **attributable**: dataset version, pi version, and judge model recorded.
- Dataset grows **20 → 50+ cases** across ≥6 categories with regression pairs and diff-graded code tasks.
- Live runs exercise **agentic/tool-using behavior** (not just print mode) and are **sandboxed**; judge variance is bounded; known-flaky cases are **quarantined** with re-entry review.

**Appetite:** ≤ 2 cycles (2 weeks) total across items; each item keeps its own pw estimate; scope is cut, not extended (scope-lock applies per item).

**Why now:** the eval is a smoke gate with n=3 runs. Friday's gate failed on a single-run flake (`coding-005`) and the evidence was lost because artifacts only upload on success. Growing the dataset before honest flake handling and attributable runs would just amplify noise.

**Sequence (dependency-ordered):**

1. EVAL-1 — always-upload artifacts → evidence on every outcome
2. EVAL-2 — flake-aware gate → stop false-fail noise, report flakes
3. EVAL-3 — dataset versioning + provenance → make runs attributable
4. EVAL-4 — `--heal-events` in ci-benchmark → consistent observability
5. EVAL-5 — dataset growth + stratification + diff-grading → scale on honest ground
6. EVAL-6 — agentic case family → exercise harness differentiators
7. EVAL-7 — sandbox live runs → reproducibility
8. EVAL-8 — judge stabilization → bound variance
9. EVAL-9 — quarantine mechanism → managed flake triage
10. EVAL-10 — output gallery + EVAL-11 — auto-open issue → visualization/automation

**DoD:**

- [ ] Each item closed via its own PR with hermetic tests (BACKLOG traceability).
- [ ] Scorecard reports flakes vs. regressions; no false-fail from 1-of-N flake.
- [ ] Dataset version + provenance in every scorecard; ≥50 cases.
- [ ] Every eval outcome leaves a full artifact (report, summary, heal events).
- [ ] EPIC-1 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| EVAL-1 | Always-upload eval artifacts (incl. `.pi/heal/events.jsonl`) on any gate outcome | 1.30 | 0.25 pw |
| EVAL-2 | Flake-aware gate: 1-of-N flake ≠ regression, `flakes` in scorecard, n=3→5 | 1.40 | 0.5 pw |
| EVAL-3 | Dataset versioning + provenance in scorecard | 0.90 | 0.25 pw |
| EVAL-4 | Wire `--heal-events` into `ci-benchmark` scorecard | 0.60 | 0.25 pw |
| EVAL-5 | Dataset growth 20 → 50–100 stratified cases, regression pairs, diff-grading | 1.60 | 2 pw |
| EVAL-6 | Agentic case family: multi-turn/tool-using/subagent/stall-recovery | 1.10 | 1 pw |
| EVAL-7 | Sandbox live runs (Docker isolation for live suite) | 0.70 | 2 pw |
| EVAL-8 | Judge-case stabilization (majority-of-3, more deterministic graders) | 0.50 | 0.5 pw |
| EVAL-9 | Known-flaky quarantine mechanism | 0.40 | 0.5 pw |
| EVAL-10 | Eval output gallery / human-readable baseline diff (was BACKLOG #8) | 0.20 | 1 pw |
| EVAL-11 | Auto-open GitHub issue on N self-heals (was BACKLOG #5; unblocked by W6) | 0.50 | 0.5 pw |
