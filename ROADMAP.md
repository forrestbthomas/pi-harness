# pi-harness — Product Roadmap

**Owner:** forrestthomas · **Last updated:** 2026-08-13
**How this doc is used:** every workstream decision is ranked against this roadmap. If a proposed change does not serve a roadmap item (or a new item that earns its way in via the backlog), it is out of scope. See `BACKLOG.md` for the ranked queue.

## North star

A **self-healing, measurable, distributable coding-agent harness**: you point `pi-run` at a provider, get a real agent, get an honest pass-rate and cost, and trust it not to hang your machine or silently lie about results.

## Guiding principles

1. **Honesty over optimism** — an eval that skips itself is indistinguishable from one that passed; a run that hung and recovered is reported, not hidden.
2. **Self-healing by default** — hangs, wedges, and dirty git states should resolve automatically or escalate with evidence; the harness must never need a human nudge to unstick.
3. **Deterministic where possible, live where it matters** — hermetic tests for contract; live eval for signal.
4. **Provider-agnostic, stdlib-only core** — the Go CLI stays dependency-free; everything else is optional, env-gated, or additive.
5. **Bounded scope** — every workstream gets an explicit Definition of Done and a budget; scope creep is flagged via `scope-lock` before it lands.

## Active workstreams (ranked)

| # | Workstream | Status | DoD (Definition of Done) | Notes |
|---|---|---|---|---|
| W1 | **Self-healing layer** | **SHIPPED — v0.9.1 (2026-08-13)** | DONE — watchdog detects no-output stalls (`PI_STALL_TIMEOUT_SECS`, default 300s; chat excluded); DONE — process-group kill reaps grandchildren (SIGTERM → 10s grace → SIGKILL); DONE — git-state auto-recovery (`pi-run self-heal`: `rebase --continue` when clean, `needs-attention` with conflict paths, explicit `--abort` only); DONE — escalation packet on bounded retries (`.pi/heal/<timestamp>-report.json` + stderr summary, exit code 9); DONE — hermetic tests (Go unit tests + exit-code-9 contract pin); DONE — `--self-heal` observability events (shipped as env `PI_SELF_HEAL=1`; CLI flag + scorecard surfacing deferred to backlog) | Retrospective: shipped #59–#63; the #59 non-interactive env killed the interactive-editor/pager hang class outright, and the watchdog covers the residual stall/wedge/git-state classes. Follow-ups in BACKLOG: per-tool-call timeout upstream (BACKLOG #1); tune silent-window/restart-budget once `PI_SELF_HEAL=1` data accumulates (SCOPE.md follow-up) |
| W2 | **Real live-eval baseline** | Unblocked — final steps | DONE — clean nightly green run with the real pi package (2026-08-13 05:11 UTC; note: gate was vacuous — empty baseline); OPEN — `score_run.py --update-baseline --allow` committed as `eval/baselines/live-baseline.json` (file is still an empty placeholder, `cases: {}`); OPEN — gate catches a deliberate regression | Next: pull the 05:11 UTC run's report (`eval/live-results/report.json` artifact), re-run the gate with `--update-baseline --allow`, review the diff, commit; then prove the gate with a deliberate-regression smoke check. The nightly never self-updates its baseline by design — committing is a human-reviewed act (`nightly-live-eval.yml` gate step) |
| W3 | **v0.9.1 patch release** | Tagged + released — Homebrew verify remains | DONE — tag `v0.9.1` at `3a4a6fc` + GitHub Release with binaries (2026-08-13), carrying #50–#63 (impostor-pi fix, grader calibration, report writer, W1 self-healing); OPEN — Homebrew tap formula updated in `forrestbthomas/homebrew-tap` (`release.yml` auto-runs `scripts/update-homebrew-formula.sh` on tag push — confirm, or re-run manually); OPEN — `brew install pi-run` + version verify | External steps (tap repo + local brew); the old "validate against W1/W2 before tagging" item is dropped — validation happened pre-tag |
| W4 | **Project-management layer** | Active (ongoing ritual) | DONE — ROADMAP/BACKLOG maintained (this update); DONE — Tier-1 PM skills installed (`scope-lock`, `productskills`, `spec-coding-skills` under `~/.pi/agent/skills/`); DONE — prioritization ritual used for every new workstream (RICE ranking in BACKLOG) | One-time setup (#60) complete; stays open as the standing governance layer — the Recurring rituals section below is its living DoD |

## Parked / deferred (deliberately not active)

| Item | Why parked | Unpark trigger |
|---|---|---|
| Cloud eval backend (design doc) | Deferred to backlog per user; local eval is the current target | A second machine/CI-with-keys pattern emerges; local baseline matures |
| context-engine workstream | User parked pre-implementation ("treat context as a separate feature") | User re-prioritizes; separate `pi-run context` session-stats feature also deferred |
| Upstream pi-subagents release watch | Waiting on upstream async-timeout fix release (our #978/#979 merged) | New upstream release; re-verify pinned version in `.pi/settings.json` (BACKLOG #2) |
| Docker weekly eval | Kept weekly (not nightly) by design decision | If nightly signal degrades without container isolation |

## Recurring rituals (how this doc stays true)

1. **New workstream →** write a one-paragraph pitch with a DoD; add to `BACKLOG.md`; rank with RICE (from `productskills/feature-prioritization`).
2. **Before implementation →** `scope-lock` generates a `SCOPE.md` contract from the approved plan; deviations are flagged, not silently absorbed.
3. **Before merge →** the change must close its DoD checkboxes; `verification-before-completion` applies.
4. **End of milestone →** update ROADMAP statuses; move done items to CHANGELOG; re-rank the backlog.
