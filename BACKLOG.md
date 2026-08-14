# pi-harness — Backlog (ranked)

**Owner:** forrestthomas · **Last updated:** 2026-08-13
Ranked by RICE (Reach × Impact × Confidence / Effort; from
`productskills/feature-prioritization`). Higher = do next. Workstreams are
promoted to `ROADMAP.md` when they become active. See `STATUS.md` for the
one-screen snapshot and `docs/roadmap-workflow.md` for the cycle ritual.

## Ranked queue (2026-08-14 pass — EPIC-1..5 consolidated; see `EPICS.md`)

| Rank | Item | RICE | Epic | Effort | DoD sketch |
|---|---|---|---|---|---|
| 1 | **HEAL-1 — Watchdog liveness heartbeat** ("still actively working") — **user-deferred 2026-08-13; promote only on explicit re-prioritization** | 3.20* | EPIC-2 | 0.5 pw | emits heartbeat for long runs (elapsed, current step); auditable via PI_SELF_HEAL; *rough RICE from idea-inbox pitch* |
| 2 | **EVAL-5 — Dataset growth 20 → 50–100 stratified cases** (regression pairs, diff-grading) | 1.60 | EPIC-1 | 2 pw | ≥50 cases across ≥6 categories; code tasks graded on diff not final text; regression twins per bug-fix |
| 3 | **EVAL-2 — Flake-aware gate** (1-of-N flake ≠ regression, `flakes` in scorecard, n=3→5) | 1.40 | EPIC-1 | 0.5 pw | gate fails only on ≥2 failed runs or real regression; scorecard reports flake rate; hermetic tests |
| 4 | **EVAL-1 — Always-upload eval artifacts** on any gate outcome (incl. `.pi/heal/events.jsonl`) | 1.30 | EPIC-1 | 0.25 pw | upload step `if: always()`; artifact = report + summary + heal events; folds former #2 |
| 5 | **HEAL-5 — `pi-run hooks` post-rebase hook** (auto-continue wedged rebase) | 1.20 | EPIC-2 | 0.5 pw | post-rebase hook invokes `pi-run self-heal` after agent timeout; hermetic test |
| 6 | **EVAL-6 — Agentic case family** (multi-turn/tool-using/subagent/stall-recovery) | 1.10 | EPIC-1 | 1 pw | new task surface exercises harness differentiators, not just print mode |
| 7 | **EVAL-3 — Dataset versioning + provenance** in scorecard (datasetVersion, pi version, judge model) | 0.90 | EPIC-1 | 0.25 pw | summary records dataset/pi/judge provenance; contamination/drift guard |
| 8 | **COST-1 — Cost-aware router** (per-task-tier model choice; 2026-08-12 cost-aware-routing spec) | 0.80 | EPIC-3 | 2 pw | route cheap/fast/strong per task tier; P2/P3 from research synthesis |
| 9 | **EVAL-7 — Sandbox live runs** (Docker isolation for live suite) | 0.70 | EPIC-1 | 2 pw | live suite runs in containers; reproducible, no tree pollution |
| 10 | **EVAL-4 — Wire `--heal-events` into `ci-benchmark`** | 0.60 | EPIC-1 | 0.25 pw | provider scorecard surfaces self-heal events like W6 |
| 11 | **COST-2 — Model-catalog auto-refresh in CI** | 0.50 | EPIC-3 | 0.5 pw | CI step refreshes catalogs when drift detected (gpt-5.x-mini bug class) |
| 12 | **EVAL-11 — Auto-open GitHub issue on N self-heals** (was #5; unblocked by W6) | 0.50 | EPIC-1 | 0.5 pw | opens issue with escalation packet when live-eval self-heals N times |
| 13 | **EVAL-8 — Judge-case stabilization** (majority-of-3, more deterministic graders) | 0.50 | EPIC-1 | 0.5 pw | bound LLM-judge variance on the 4 judge-graded cases |
| 14 | **EVAL-9 — Known-flaky quarantine mechanism** | 0.40 | EPIC-1 | 0.5 pw | flaky cases quarantined with re-entry review; tracked in scorecard |
| 15 | **HEAL-2 — Tune watchdog params** from `PI_SELF_HEAL` data (W1 retro follow-up) | 0.35 | EPIC-2 | 0.25 pw | silent-window/restart-budget tuned from ≥1 week of events; values documented |
| 16 | **HEAL-3 — Auto-resume after clean git recovery** (decision ticket; W1 spec follow-up) | 0.30 | EPIC-2 | 0.25 pw | ship or park with recorded rationale |
| 17 | **DX-1 — Harness continuous self-evaluation** (skill compatibility audit; parked idea) | 0.25 | EPIC-5 | 1 pw | audit skills for duplicates/conflicts; classify; never auto-reconcile |
| 18 | **HEAL-4 — Loop/stuck-pattern detection** (OpenHands StuckDetector class; deferred, needs thresholds) | 0.20 | EPIC-2 | 2 pw | evidence-gated; stays deferred without empirical thresholds |
| 19 | **PORT-2 — Cloud eval backend** (parked; needs design doc) | 0.20 | EPIC-4 | 4 pw | design doc first; local is enough today |
| 20 | **EVAL-10 — Eval output gallery / human-readable baseline diff** (was #8) | 0.20 | EPIC-1 | 1 pw | render baseline diff in `pi-run eval` output |
| 21 | **DX-2 — context-engine un-park** (`pi-run context` session-stats) | 0.17 | EPIC-5 | 1.5 pw | user parked it; un-park trigger: user re-prioritizes |
| 22 | **PORT-1 — Windows support** | 0.11 | EPIC-4 | 4 pw | Go stdlib portable but pi/nvm/brew story is Unix-first |

_Promoted 2026-08-13: per-tool-call timeout upstream → ROADMAP W5 (was rank #1). Promoted 2026-08-13: surface `PI_SELF_HEAL` events in scorecard + enable in CI → ROADMAP W6 (was rank #1). Bet on 2026-08-14: EVAL-1 + EVAL-2 → ROADMAP W7 (evidence artifacts + flake-aware gate). Bet on 2026-08-14: EVAL-3 → ROADMAP W8 (dataset versioning + provenance). Bet on 2026-08-14: EVAL-4 → ROADMAP W9 (self-heal in provider scorecard); EVAL-5 → ROADMAP W10 (dataset growth, contract approved). Closed 2026-08-14: W8 — merged (#89). Closed 2026-08-14: W7 — merged (#87); next live nightly verifies end-to-end. Closed 2026-08-14: W6 — merged (#83) and verified on the 2026-08-14 manual nightly (`selfHeal` block surfaced; 0 events on a healthy run; gate failed on unrelated coding-005 pass-rate dip and coding-010 cost spike). Consolidated 2026-08-14 into EPIC-1..5 (EPICS.md): EVAL-1..11 (EPIC-1); HEAL-1..5 incl. idea-inbox heartbeat + W1 follow-ups + post-rebase hook (EPIC-2); COST-1..2 (EPIC-3); PORT-1..2 (EPIC-4); DX-1..2 (EPIC-5). Folded former #2 nightly archives → EVAL-1, #5 auto-open issue → EVAL-11, #8 output gallery → EVAL-10. Cost-cap in CI scorecard verified already done (provider-scorecard.yml --max-budget-usd). Closed 2026-08-13: doctor non-interactive-env guard (was rank #1 before promotion) — landed in `main`, see CHANGELOG [Unreleased]. Also closed earlier the same day: pin pi-subagents and owner-only artifact perms (in v0.9.2)._

## Idea inbox (unranked, capture-only)

- **Watchdog liveness heartbeat** — moved to **HEAL-1 (EPIC-2)**; kept here only for the open design question: where it lives (harness watchdog vs upstream pi-subagents vs parent-agent nudge) and whether it writes `.pi/heal/events.jsonl`.

## How items get in / out

- **In:** one-paragraph pitch + DoD + rough RICE, added here, ranked. Epic items are tagged with their epic id and indexed in `EPICS.md`.
- **Promoted:** user approves moving to ROADMAP as an active workstream with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record why).
- **Scope rule:** any change that serves none of the above is out of scope unless it earns a backlog entry first.
- **Cadence:** re-rank at cycle start (see `docs/roadmap-workflow.md`); prune items with no evidence after 3 months; close epics like workstreams (DoD → CHANGELOG).
