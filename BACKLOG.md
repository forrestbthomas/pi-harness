# pi-harness — Backlog (ranked)

**Owner:** forrestthomas · **Last updated:** 2026-08-13
Ranked by RICE (Reach × Impact × Confidence / Effort; from
`productskills/feature-prioritization`). Higher = do next. Workstreams are
promoted to `ROADMAP.md` when they become active. See `STATUS.md` for the
one-screen snapshot and `docs/roadmap-workflow.md` for the cycle ritual.

## Ranked queue (2026-08-13 pass)

| Rank | Item | RICE | Tag | Effort | DoD sketch |
|---|---|---|---|---|---|
| 1 | **`pi-run doctor` verifies non-interactive env** (regression guard for #59) | 3.60 | Blocker | 0.25 pw | `doctor` checks `GIT_EDITOR`/`PAGER`/`GIT_TERMINAL_PROMPT` in launchEnv; fails loudly if the hang-prevention env is missing |
| 2 | **Per-tool-call timeout upstream contribution** (pi-subagents #150) | 2.80 | Blocker | 1.5 pw | PR merged upstream; our `PI_SELF_HEAL=1` events observe it |
| 3 | **Surface `PI_SELF_HEAL` events in scorecard + enable in CI** | 1.40 | Enabler | 0.5 pw | `nightly-live-eval.yml` sets `PI_SELF_HEAL=1`; scorecard gains a self-heal-event count; unblocks #7/#8 |
| 4 | **`pi-run hooks` post-rebase hook** (auto-continue wedged rebase) | 1.20 | Enabler | 0.5 pw | post-rebase hook invokes `pi-run self-heal` after agent timeout; hermetic test |
| 5 | **Nightly archives watchdog/git-state events** for postmortems | 1.00 | Enabler | 0.25 pw | nightly uploads `.pi/heal/events.jsonl` as an artifact |
| 6 | Model-catalog auto-refresh in CI | 0.50 | Enabler | 0.5 pw | CI step refreshes catalogs when drift detected |
| 7 | Auto-open GitHub issue with escalation packet when live-eval self-heals N times | 0.50 | Enabler | 0.5 pw | Depends on #3 (events must exist first) |
| 8 | Harness continuous self-evaluation (skill compatibility audit) | 0.25 | Enabler | 1 pw | audit skills for duplicates/conflicts; classify; never auto-reconcile |
| 9 | Cloud eval backend | 0.20 | Enabler | 4 pw | needs design doc; local is enough today |
| 10 | Eval output gallery / human-readable diff of scorecard vs baseline | 0.20 | Enabler | 1 pw | render baseline diff in `pi-run eval` output |
| 11 | context-engine un-park (`pi-run context` session-stats) | 0.17 | Enabler | 1.5 pw | user parked it; un-park trigger: user re-prioritizes |
| 12 | Windows support | 0.11 | Enabler | 4 pw | Go stdlib portable but pi/nvm/brew story is Unix-first |

_Closed 2026-08-13: pin pi-subagents and owner-only artifact perms (backlog #2/#3) — landed in `main`, see CHANGELOG [0.9.2]._

## How items get in / out

- **In:** one-paragraph pitch + DoD + rough RICE, added here, ranked.
- **Promoted:** user approves moving to ROADMAP as an active workstream with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record why).
- **Scope rule:** any change that serves none of the above is out of scope unless it earns a backlog entry first.
- **Cadence:** re-rank at cycle start (see `docs/roadmap-workflow.md`); prune items with no evidence after 3 months.
