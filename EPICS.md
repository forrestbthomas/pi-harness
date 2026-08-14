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
- A user-deferred item keeps its RICE rank but is flagged and is not promoted without explicit re-prioritization.

## Epic index

| Epic | Outcome | Appetite |
|---|---|---|
| EPIC-1 — Eval suite | smoke gate → research-grade benchmark (0 flake false-fails, attributable, 50+ cases) | ≤ 2 cycles |
| EPIC-2 — Self-healing resilience | every hang/wedge class detected, recovered, observable; watchdog tuned on real data; auto-resume decided | ≤ 1.5 cycles |
| EPIC-3 — Cost intelligence & routing | know/control cost per provider+task; route to best model per task tier | ≤ 1 cycle |
| EPIC-4 — Portability & distribution | harness runs where users are (Windows, cloud) | ≤ 1 cycle (research-first) |
| EPIC-5 — Insight & DX | users and the harness understand the harness (self-audit, session stats) | ≤ 1 cycle |

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

---

## EPIC-2 — Self-healing: from recovery to resilience

**Outcome.** When this epic is done:

- Every hang/wedge class the watchdog targets is **observable** (events flow to the scorecard), **tuned on real data** (`PI_SELF_HEAL` accumulated), and the remaining gaps (liveness heartbeats, auto-resume) are **decided or shipped**.
- Post-rebase auto-continuation closes the git-state loop: a wedged rebase after an agent timeout **recovers without a human nudge**.
- The user has an explicit record of why loop/stuck detection stays out (or the evidence to turn it on).

**Appetite:** ≤ 1.5 cycles total; scope is cut, not extended.

**Why now:** W1 shipped the watchdog and W6 made events observable; the follow-ups (tuning with real data, heartbeat, auto-resume, post-rebase hook) are the difference between "recovers with a nudge" and "recovers on its own". The heartbeat idea is user-deferred but belongs here when re-prioritized.

**Sequence (dependency-ordered):**

1. HEAL-2 — tune watchdog params on real `PI_SELF_HEAL` data (data exists since W6)
2. HEAL-5 — post-rebase hook (auto-continue wedged rebase)
3. HEAL-1 — watchdog liveness heartbeat (user-deferred; re-confirm before promotion)
4. HEAL-3 — auto-resume decision (decision ticket; cheap)
5. HEAL-4 — loop/stuck detection (evidence-gated; stays deferred without thresholds)

**DoD:**

- [ ] Watchdog params (silent window, restart budget) tuned from ≥1 week of event data; values documented.
- [ ] Post-rebase hook ships with a hermetic test.
- [ ] Heartbeat shipped or explicitly re-deferred with the decision recorded.
- [ ] Auto-resume decision recorded (ship or park with rationale).
- [ ] EPIC-2 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| HEAL-1 | Watchdog liveness heartbeat ("still actively working") — **user-deferred 2026-08-13; promote only on explicit re-prioritization** | 3.20* | 0.5 pw |
| HEAL-2 | Tune watchdog silent-window/restart-budget from `PI_SELF_HEAL` data (W1 retro follow-up) | 0.35 | 0.25 pw |
| HEAL-3 | Auto-resume agent run after clean git-state recovery (decision ticket; W1 spec follow-up) | 0.30 | 0.25 pw |
| HEAL-4 | Loop/stuck-pattern detection (OpenHands StuckDetector class; deferred, needs thresholds) | 0.20 | 2 pw |
| HEAL-5 | Post-rebase hook: `pi-run hooks` auto-continues wedged rebase after agent timeout | 1.20 | 0.5 pw |

\* Rough RICE from the idea-inbox pitch (Reach 2 · Impact 1 · Conf 0.8 / 0.5 pw); deliberate user deferral keeps it out of promotion despite the rank.

---

## EPIC-3 — Cost intelligence & routing

**Outcome.** When this epic is done:

- Users see **cost per provider, per model, per task tier** in one view (dashboard exists for spend; tier-level routing is decided/shipped).
- The harness **auto-routes each task to the best model for its tier** (cheap/fast/strong) using a refreshed model catalog — no manual model picking.
- The model catalog **refreshes in CI** so routing never targets a stale/removed model (the gpt-5.x-mini catalog bug class is gone).

**Appetite:** ≤ 1 cycle.

**Why now:** the cost-aware-routing design is written (2026-08-12) and the catalog-drift failure mode burned us on 2026-08-12 (invalid model-tier IDs). The plumbing (cost ledger, budget caps, provider scorecard) exists; routing is the payoff.

**Sequence (dependency-ordered):**

1. COST-2 — model-catalog auto-refresh in CI (routing input must be current)
2. COST-1 — cost-aware router (per-task-tier model choice, P2/P3)

**DoD:**

- [ ] Catalog refresh in CI (hermetic drift check).
- [ ] Router ships or is parked with a recorded decision; per-tier routing documented.
- [ ] EPIC-3 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| COST-1 | Cost-aware router: per-task-tier model choice (2026-08-12 cost-aware-routing spec) | 0.80 | 2 pw |
| COST-2 | Model-catalog auto-refresh in CI | 0.50 | 0.5 pw |

---

## EPIC-4 — Portability & distribution

**Outcome.** When this epic is done:

- The harness runs where users are: **Windows** story documented/verified or explicitly parked, and a **cloud eval backend** design exists (local is still enough today — parked decision is re-confirmed).
- Releases continue to carry a working binary + Homebrew formula with verified assets (existing release ritual stays green).

**Appetite:** ≤ 1 cycle, research-first (both items are large; split into design + build).

**Why now:** portability is the anti-lock-in pitch ("run the same eval across any provider, on any machine"). Windows is the biggest addressable gap; cloud eval only becomes worth building when local baselines mature.

**Sequence (dependency-ordered):**

1. PORT-2 — cloud eval backend design doc (parked; re-confirm the "local is enough today" call)
2. PORT-1 — Windows support (Go stdlib is portable; the pi/nvm/brew story is the real work)

**DoD:**

- [ ] Cloud eval backend design doc written or the parked decision re-confirmed in ROADMAP.
- [ ] Windows support: documented status + at least a CI smoke or an explicit park with rationale.
- [ ] EPIC-4 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| PORT-1 | Windows support | 0.11 | 4 pw |
| PORT-2 | Cloud eval backend (parked; needs design doc) | 0.20 | 4 pw |

---

## EPIC-5 — Insight & DX

**Outcome.** When this epic is done:

- The harness **audits itself**: skill-collision/compatibility checks with user-chosen reconciliation (never auto-reconcile).
- Users get **session/context stats** (`pi-run context`) for cost + behavior insight.

**Appetite:** ≤ 1 cycle.

**Why now:** both are "know thyself" items — the harness evaluates agents, so it should evaluate its own skill stack, and users should understand their own session spend/context. Both are parked/small and benefit from being bet on as a unit.

**Sequence (dependency-ordered):**

1. DX-1 — harness continuous self-evaluation (skill audit, parked idea → shaped)
2. DX-2 — context-engine un-park (`pi-run context` session-stats)

**DoD:**

- [ ] Skill audit ships or is parked with a recorded decision.
- [ ] `pi-run context` session-stats ships or is re-parked with a recorded decision.
- [ ] EPIC-5 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| DX-1 | Harness continuous self-evaluation (skill compatibility audit) | 0.25 | 1 pw |
| DX-2 | context-engine un-park (`pi-run context` session-stats) | 0.17 | 1.5 pw |
