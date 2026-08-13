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
| W1 | **Self-healing layer** | In design (spec in progress) | Watchdog detects no-output stalls; process-group kill reaps grandchildren; git-state auto-recovery (`rebase --continue`/`--abort`); escalation packet on bounded retries; hermetic tests; `--self-heal` observability flag | Research sweep complete (`docs/self-healing-research-2026-08.md`); prevention env already merged (#59) |
| W2 | **Real live-eval baseline** | In progress | A clean nightly green run with the real pi package; `score_run.py --update-baseline --allow` committed as `eval/baselines/live-baseline.json`; gate catches a deliberate regression | Blocked on: one clean nightly run post #57/#58 |
| W3 | **v0.9.1 patch release** | Ready to ship | Tag + Homebrew tap update + brew verify carrying fixes #50–#58 | Includes impostor-pi fix, calibration, report writer; validate against W1/W2 before tagging |
| W4 | **Project-management layer** | In progress (this doc) | ROADMAP/BACKLOG maintained; Tier-1 PM skills installed (scope-lock, productskills, spec-coding-skills); prioritization ritual used for every new workstream | This PR |

## Parked / deferred (deliberately not active)

| Item | Why parked | Unpark trigger |
|---|---|---|
| Cloud eval backend (design doc) | Deferred to backlog per user; local eval is the current target | A second machine/CI-with-keys pattern emerges; local baseline matures |
| context-engine workstream | User parked pre-implementation ("treat context as a separate feature") | User re-prioritizes; separate `pi-run context` session-stats feature also deferred |
| Upstream pi-subagents release watch | Waiting on upstream async-timeout fix release (our #978/#979 merged) | New upstream release; re-verify pinned version in `.pi/settings.json` |
| Docker weekly eval | Kept weekly (not nightly) by design decision | If nightly signal degrades without container isolation |

## Recurring rituals (how this doc stays true)

1. **New workstream →** write a one-paragraph pitch with a DoD; add to `BACKLOG.md`; rank with RICE (from `productskills/feature-prioritization`).
2. **Before implementation →** `scope-lock` generates a `SCOPE.md` contract from the approved plan; deviations are flagged, not silently absorbed.
3. **Before merge →** the change must close its DoD checkboxes; `verification-before-completion` applies.
4. **End of milestone →** update ROADMAP statuses; move done items to CHANGELOG; re-rank the backlog.
