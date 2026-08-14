# pi-harness — Backlog (ranked)

**Owner:** forrestthomas · **Last updated:** 2026-08-13
Ranked by RICE (Reach × Impact × Confidence / Effort; from
`productskills/feature-prioritization`). Higher = do next. Workstreams are
promoted to `ROADMAP.md` when they become active. See `STATUS.md` for the
one-screen snapshot and `docs/roadmap-workflow.md` for the cycle ritual.

## Ranked queue (2026-08-14 pass — EPIC-1 items added; see `EPICS.md`)

| Rank | Item | RICE | Epic | Effort | DoD sketch |
|---|---|---|---|---|---|
| 1 | **EVAL-5 — Dataset growth 20 → 50–100 stratified cases** (regression pairs, diff-grading) | 1.60 | EPIC-1 | 2 pw | ≥50 cases across ≥6 categories; code tasks graded on diff not final text; regression twins per bug-fix |
| 2 | **EVAL-2 — Flake-aware gate** (1-of-N flake ≠ regression, `flakes` in scorecard, n=3→5) | 1.40 | EPIC-1 | 0.5 pw | gate fails only on ≥2 failed runs or real regression; scorecard reports flake rate; hermetic tests |
| 3 | **EVAL-1 — Always-upload eval artifacts** on any gate outcome (incl. `.pi/heal/events.jsonl`) | 1.30 | EPIC-1 | 0.25 pw | upload step `if: always()`; artifact = report + summary + heal events; folds former #2 |
| 4 | **`pi-run hooks` post-rebase hook** (auto-continue wedged rebase) | 1.20 | — | 0.5 pw | post-rebase hook invokes `pi-run self-heal` after agent timeout; hermetic test |
| 5 | **EVAL-6 — Agentic case family** (multi-turn/tool-using/subagent/stall-recovery) | 1.10 | EPIC-1 | 1 pw | new task surface exercises harness differentiators, not just print mode |
| 6 | **EVAL-3 — Dataset versioning + provenance** in scorecard (datasetVersion, pi version, judge model) | 0.90 | EPIC-1 | 0.25 pw | summary records dataset/pi/judge provenance; contamination/drift guard |
| 7 | **EVAL-7 — Sandbox live runs** (Docker isolation for live suite) | 0.70 | EPIC-1 | 2 pw | live suite runs in containers; reproducible, no tree pollution |
| 8 | **EVAL-4 — Wire `--heal-events` into `ci-benchmark`** | 0.60 | EPIC-1 | 0.25 pw | provider scorecard surfaces self-heal events like W6 |
| 9 | Model-catalog auto-refresh in CI | 0.50 | — | 0.5 pw | CI step refreshes catalogs when drift detected |
| 10 | **EVAL-11 — Auto-open GitHub issue on N self-heals** (was #5; unblocked by W6) | 0.50 | EPIC-1 | 0.5 pw | opens issue with escalation packet when live-eval self-heals N times |
| 11 | **EVAL-8 — Judge-case stabilization** (majority-of-3, more deterministic graders) | 0.50 | EPIC-1 | 0.5 pw | bound LLM-judge variance on the 4 judge-graded cases |
| 12 | **EVAL-9 — Known-flaky quarantine mechanism** | 0.40 | EPIC-1 | 0.5 pw | flaky cases quarantined with re-entry review; tracked in scorecard |
| 13 | Harness continuous self-evaluation (skill compatibility audit) | 0.25 | — | 1 pw | audit skills for duplicates/conflicts; classify; never auto-reconcile |
| 14 | Cloud eval backend | 0.20 | — | 4 pw | needs design doc; local is enough today |
| 15 | **EVAL-10 — Eval output gallery / human-readable baseline diff** (was #8) | 0.20 | EPIC-1 | 1 pw | render baseline diff in `pi-run eval` output |
| 16 | context-engine un-park (`pi-run context` session-stats) | 0.17 | — | 1.5 pw | user parked it; un-park trigger: user re-prioritizes |
| 17 | Windows support | 0.11 | — | 4 pw | Go stdlib portable but pi/nvm/brew story is Unix-first |

_Promoted 2026-08-13: per-tool-call timeout upstream → ROADMAP W5 (was rank #1). Promoted 2026-08-13: surface `PI_SELF_HEAL` events in scorecard + enable in CI → ROADMAP W6 (was rank #1). Closed 2026-08-14: W6 — merged (#83) and verified on the 2026-08-14 manual nightly (`selfHeal` block surfaced; 0 events on a healthy run; gate failed on unrelated coding-005 pass-rate dip and coding-010 cost spike). Folded 2026-08-14 into EPIC-1 items: #2 nightly archives → EVAL-1, #5 auto-open issue → EVAL-11, #8 output gallery → EVAL-10. Closed 2026-08-13: doctor non-interactive-env guard (was rank #1 before promotion) — landed in `main`, see CHANGELOG [Unreleased]. Also closed earlier the same day: pin pi-subagents and owner-only artifact perms (in v0.9.2)._

## Idea inbox (unranked, capture-only)

- **Watchdog liveness heartbeat ("still actively working")** — after N minutes
  of a long-running agent/subagent with no update, emit a heartbeat ("still
  actively working — <elapsed>, <current step>") to the parent agent / terminal
  viewer so a legitimately long run isn't mistaken for a hang. Pitch: honest
  observability — a real 2026-08-13 case (deepseek thinking-high ran 19+ min
  with no output and was suspected hung). Rough RICE: Reach 2 · Impact 1 ·
  Conf 0.8 · Effort 0.5 pw → ~3.2, but deliberately deferred by user to a later
  cycle. Open design question (discuss when prioritized): where it lives —
  harness watchdog (run-level) vs upstream pi-subagents (per-child) vs
  parent-agent nudge; and whether it also writes `.pi/heal/events.jsonl`
  (`PI_SELF_HEAL`) so long runs are auditable.

## How items get in / out

- **In:** one-paragraph pitch + DoD + rough RICE, added here, ranked. Epic items are tagged with their epic id and indexed in `EPICS.md`.
- **Promoted:** user approves moving to ROADMAP as an active workstream with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record why).
- **Scope rule:** any change that serves none of the above is out of scope unless it earns a backlog entry first.
- **Cadence:** re-rank at cycle start (see `docs/roadmap-workflow.md`); prune items with no evidence after 3 months; close epics like workstreams (DoD → CHANGELOG).
