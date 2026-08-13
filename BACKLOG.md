# pi-harness — Backlog (ranked)

**Owner:** forrestthomas · **Last updated:** 2026-08-13
Ranked by rough RICE (Reach × Impact × Confidence / Effort). Higher = do next. Workstreams are promoted to `ROADMAP.md` when they become active.

## Active-adjacent (next candidates)

| Rank | Item | RICE sketch | Why | DoD sketch |
|---|---|---|---|---|
| 1 | **Per-tool-call timeout upstream contribution** (pi-subagents #150) | Reach: ecosystem · Impact: high (kills the wedge class) · Conf: 0.7 · Effort: medium | The one self-healing piece that truly belongs upstream; our v0.9.1 watchdog bounds the run (stall, group-kill, git state) but not mid-tool wedges | PR merged upstream; our `PI_SELF_HEAL=1` events observe it |
| 2 | **Pin pi-subagents version** (incl. lockfile) | Effort: trivial | `.pi/settings.json` still uses unpinned `npm:pi-subagents` (0.45.1 installed) — a settings-triggered `npm install` refresh would drift; a committed lockfile alone doesn't pin what settings.json requests | Pinned to a known-good release in `.pi/settings.json`; `package-lock.json` committed/verified; `config-check` verifies |
| 3 | **Session/artifact writer perms 0600/0700** | Reach: low-medium (local artifacts) · Impact: medium-high (sensitive files world-readable by default) · Conf: 1.0 (code evidence) · Effort: low | Manual chmod cleanup (700/600) done 2026-08-13, but runtime writers still default: `.pi/heal/` dirs 0755 and `-report.json`/`events.jsonl` 0644 (`internal/cli/escalation.go:56,70,229,241`) — packets contain goal text, diff summaries, resume handles; `.pi/sessions` writers have the same default-umask problem | Escalation packet + events + pi-run-owned session dirs created 0600/0700; contract test asserts perms |

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
- Surface `PI_SELF_HEAL=1` events in the scorecard (fragment deferred from SCOPE change #1; W1 shipped the env events)

- Harness continuous self-evaluation (sidenote 2026-08-13): audit skills present in a harness/project for compatibility (duplicate names, conflicting instructions, stale collections); classify compatible/incompatible; offer the user an option to reconcile (never automatic). Separate workstream from self-healing W1; natural home is pi-run doctor/config-check or a new self-eval command.

## How items get in / out

- **In:** one-paragraph pitch + DoD + rough RICE, added here, ranked.
- **Promoted:** user approves moving to ROADMAP as an active workstream with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record why).
- **Scope rule:** any change that serves none of the above is out of scope unless it earns a backlog entry first.
