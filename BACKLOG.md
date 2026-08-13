# pi-harness — Backlog (ranked)

**Owner:** forrestthomas · **Last updated:** 2026-08-13
Ranked by rough RICE (Reach × Impact × Confidence / Effort). Higher = do next. Workstreams are promoted to `ROADMAP.md` when they become active.

## Active-adjacent (next candidates)

| Rank | Item | RICE sketch | Why | DoD sketch |
|---|---|---|---|---|
| 1 | **Per-tool-call timeout upstream contribution** (pi-subagents #150) | Reach: ecosystem · Impact: high (kills the wedge class) · Conf: 0.7 · Effort: medium | The one self-healing piece that truly belongs upstream; our watchdog fixes 2–5 but not mid-tool wedges | PR merged upstream; our `--self-heal` observes it |
| 2 | **Pin pi-subagents version** | Effort: trivial | Currently `npm:pi-subagents` unpinned (0.45.1 installed) — reproducibility risk | Pinned to known-good release; config-check verifies |
| 3 | **`--self-heal` observability flag** | Impact: low-medium · Effort: low | Measure post-prevention incident rate before full auto-recovery | Flag logs stall/git-state events; metric surfaced in scorecard |

## Deferred (would be good, not now)

| Rank | Item | Why deferred | Unpark trigger |
|---|---|---|---|
| 4 | Cloud eval backend | Needs design doc; local is enough today | Second eval environment with keys |
| 5 | context-engine un-park (`pi-run context` session-stats) | User parked it | User re-prioritizes |
| 6 | Model-catalog auto-refresh in CI | Nice-to-have; setup already refreshes | Catalog drift observed again |
| 7 | Windows support | Go stdlib portable but pi/nvm/brew story is Unix-first | User need |

## Idea inbox (unranked, capture-only)

- `pi-run doctor` should verify non-interactive env is present (prevention regression guard)
- Eval output gallery / human-readable diff of scorecard vs baseline
- Auto-open GitHub issue with escalation packet when a live-eval run self-heals N times
- Nightly run should archive the watchdog/git-state events for postmortem queries
- `pi-run hooks` post-rebase hook to auto-continue a wedged rebase after agent timeout

- Harness continuous self-evaluation (sidenote 2026-08-13): audit skills present in a harness/project for compatibility (duplicate names, conflicting instructions, stale collections); classify compatible/incompatible; offer the user an option to reconcile (never automatic). Separate workstream from self-healing W1; natural home is pi-run doctor/config-check or a new self-eval command.
## How items get in / out

- **In:** one-paragraph pitch + DoD + rough RICE, added here, ranked.
- **Promoted:** user approves moving to ROADMAP as an active workstream with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record why).
- **Scope rule:** any change that serves none of the above is out of scope unless it earns a backlog entry first.
