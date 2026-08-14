# Epics — themed initiatives that group backlog items

**Owner:** forrestthomas · **Last updated:** 2026-08-14

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
| EPIC-1 — Eval suite | smoke gate → research-grade **measurement** (0 flake false-fails, attributable, 54 live cases held) | ≈ 2.5–3 pw (one author, honest core) |
| EPIC-2 — Self-healing resilience | every hang/wedge class detected, recovered, observable; watchdog tuned on real data; auto-resume decided | ≤ 1.5 cycles |
| EPIC-3 — Cost intelligence & routing | know/control cost per provider+task; **within-provider** tier routing (never cross-provider) | ≤ 1 cycle |
| EPIC-4 — Portability & distribution | **PARKED** — non-goals per CHARTER.md until a consumer asks; PORT-0 quarterly re-confirm | 0.1 pw/quarter |
| EPIC-6 — Repo maturity | backlog shrinks; a stranger can install and fix in <10 min; the planning layer self-audits | ≈ 1.5 pw, then closes |
| ~~EPIC-5 — Insight & DX~~ | **DISSOLVED 2026-08-14** — DX-1/DX-2 returned to the idea inbox with recorded decisions | — |

---

## EPIC-1 — Eval suite: from smoke gate to research-grade measurement

**Outcome.** When this epic is done:

- Every nightly/weekly eval leaves a **complete evidence artifact** (report, summary, heal events) on **every** gate outcome — zero lost runs.
- **Gate false-fails from single-run flake → 0/week** — for **pass rate AND cost** (EVAL-2 + EVAL-13); the scorecard distinguishes flake from regression and reports flake/cost-flake rate.
- Every scorecard is **attributable**: dataset version, pi version, and judge model recorded — **on both surfaces** (live nightly gate **and** ci-benchmark provider scorecard) (EVAL-3 + EVAL-14).
- Dataset holds **54 live cases** across 7 categories (incl. the agentic tool-using family) with regression pairs and diff-graded code tasks (EVAL-5 capped the six original categories; growth only on a consumer or a 30-day regression-catch signal; `tasks.json` is the count authority).
- The **split seam is verified, not assumed**: the eval suite's self-containment is proven by dry-run so the triggered pi-bench split stays cheap (EVAL-15).
- The **loop measures itself**: changes to the harness report their own scorecard delta (EVAL-16, pilot-first).
- Live runs exercise **agentic/tool-using behavior** (EVAL-6); sandboxing is **parked** behind a real contamination incident (EVAL-7); judge variance is bounded (EVAL-8); known-flaky cases are **quarantined** (EVAL-9).

**Appetite:** ≈ 2.5–3 pw honest core for one author (restated from "≤ 2 cycles" — the 12-item plan was ~4.5× over budget at 9.0 pw; scope is cut, not extended).

**Why now:** the eval is a smoke gate with n=5 runs and a committed 17-of-54 baseline (31.5%). Friday's gate failed on a single-run flake (`coding-005`) and the 2026-08-14 nightly failed on a **cost spike** (`coding-010`) — the flake/cost false-fail classes are real, the baseline covers only 31.5% of the benchmark, and the loop does not yet measure changes to itself. Making the gate honest and the seam provable comes before any growth.

**Sequence (dependency-ordered):**

1. EVAL-12 — live re-baseline after W10 → 0 unbaselined cases; standing cadence (scheduled, tonight)
2. EVAL-6 — agentic case family → the data pump EPIC-2's HEAL-2 is gated on (print-mode cannot wedge)
3. EVAL-8 — judge stabilization → bound variance
4. EVAL-13 — cost-variance tolerance in the gate → kill the false-cost-alarm class
5. EVAL-14 — benchmark provenance parity → "every scorecard" DoD enforced on both surfaces
6. EVAL-15 — split-seam verification → prove "split cheap either way" physically
7. EVAL-16 — harness-change eval gate (pilot-first; promote on first caught regression)
8. EVAL-11 — auto-open issue on N self-heals → automation
9. EVAL-9 — quarantine mechanism → managed flake triage
10. EVAL-7 — sandbox live runs (PARKED: trigger = a real contamination incident or external consumer)

**DoD:**

- [ ] Each item closed via its own PR with hermetic tests (BACKLOG traceability).
- [ ] Scorecard reports flakes vs. regressions AND cost-flakes; no false-fail from 1-of-N flake or a single-run cost spike.
- [ ] Dataset version + provenance in **every** scorecard (live **and** ci-benchmark); 54 live cases held.
- [ ] 0 unbaselined live cases after EVAL-12; re-baseline is a standing cadence (after every dataset/harness-behavior change; history in `eval/baselines/`).
- [ ] Split-seam dry-run passes (or the gap is documented as the reason the split needs the dry-run).
- [ ] EPIC-1 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| EVAL-12 | Live re-baseline after W10: 0 unbaselined cases + standing cadence | 1.10 | 0.25 pw |
| EVAL-6 | Agentic case family: multi-turn/tool-using/subagent/stall-recovery | 1.40 | 1 pw |
| EVAL-13 | Cost-variance tolerance in nightly gate (`costFlakes`, median-of-N, ≥2-run fail) | 1.00 | 0.5 pw |
| EVAL-15 | Split-seam verification: contract doc + hermetic dry-run + test classification | 1.50 | 0.5 pw |
| EVAL-16 | Harness-change eval gate (scorecard delta on eval-touching PRs; pilot-first) | 1.20 | 0.5 pw |
| EVAL-14 | Benchmark provenance parity (ci-benchmark scorecard carries live-schema provenance) | 0.70 | 0.5 pw |
| EVAL-11 | Auto-open GitHub issue on N self-heals | 0.50 | 0.5 pw |
| EVAL-8 | Judge-case stabilization (majority-of-3, more deterministic graders) | 0.50 | 0.5 pw |
| EVAL-9 | Known-flaky quarantine mechanism | 0.40 | 0.5 pw |
| EVAL-7 | Sandbox live runs (Docker isolation) — **PARKED**: trigger = real contamination incident or external consumer | 0.70 | 2 pw |
| ~~EVAL-5~~ | Dataset growth 20 → 50 — **SHIPPED (W10); capped at 50**; growth only on consumer/regression-catch signal | 1.60 | 2 pw |
| ~~EVAL-10~~ | Eval output gallery — **REMOVED 2026-08-14** (scorecard JSON is the gallery; idea-inbox line only) | 0.20 | 1 pw |

---

## EPIC-2 — Self-healing: from recovery to resilience

**Outcome.** When this epic is done:

- Every hang/wedge class the watchdog targets is **observable** (events flow to the scorecard), **tuned on real data** (`PI_SELF_HEAL` accumulated), and the remaining gaps (liveness heartbeats, auto-resume) are **decided or shipped**.
- Post-rebase auto-continuation closes the git-state loop: a wedged rebase after an agent timeout **recovers without a human nudge**.
- The user has an explicit record of why loop/stuck detection stays out (or the evidence to turn it on).

**Appetite:** ≤ 1.5 cycles total; scope is cut, not extended.

**Why now:** W1 shipped the watchdog and W6 made events observable; the follow-ups (tuning with real data, heartbeat, auto-resume, post-rebase hook) are the difference between "recovers with a nudge" and "recovers on its own". The heartbeat idea is user-deferred but belongs here when re-prioritized — and **HEAL-2 is data-gated**: the 2026-08-14 nightly verified `self-heal events: 0` on a healthy run, so tuning must wait for non-zero wedge-class coverage, which only EVAL-6 (agentic cases) can produce.

**Sequence (dependency-ordered):**

1. HEAL-2 — tune watchdog params on real `PI_SELF_HEAL` data (**data-gated**: ≥1 week non-zero wedge coverage + W5 Part C verified; EVAL-6 is the data pump)
2. HEAL-5 — post-rebase hook (auto-continue wedged rebase)
3. HEAL-1 — watchdog liveness heartbeat (user-deferred; re-confirm before promotion; re-score from `PI_SELF_HEAL` data, not the 3.20* pitch)
4. HEAL-3 — auto-resume decision (decision ticket; cheap)
5. HEAL-4 — loop/stuck detection (evidence-gated; stays deferred without thresholds)

**DoD:**

- [ ] Watchdog params (silent window, restart budget) tuned from ≥1 week of event data; values documented (data-gate precondition in the item).
- [ ] Post-rebase hook ships with a hermetic test.
- [ ] Heartbeat shipped or explicitly re-deferred with the decision recorded (re-promotion only on watchdog-missed wedge evidence or user re-prioritization).
- [ ] Auto-resume decision recorded (ship or park with rationale).
- [ ] EPIC-2 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| HEAL-5 | Post-rebase hook: `pi-run hooks` auto-continues wedged rebase after agent timeout | 1.20 | 0.5 pw |
| HEAL-2 | Tune watchdog silent-window/restart-budget from `PI_SELF_HEAL` data (**data-gated**; EVAL-6 pump + W5 Part C) | 0.35 | 0.25 pw |
| HEAL-3 | Auto-resume agent run after clean git-state recovery (decision ticket; W1 spec follow-up) | 0.30 | 0.25 pw |
| HEAL-1 | Watchdog liveness heartbeat — **user-deferred; demoted to idea inbox 2026-08-14**; re-score from data, not pitch | 3.20* | 0.5 pw |
| HEAL-4 | Loop/stuck-pattern detection (OpenHands StuckDetector class; deferred, needs thresholds) | 0.20 | 2 pw |

\* Rough RICE from the idea-inbox pitch (Reach 2 · Impact 1 · Conf 0.8 / 0.5 pw); demoted 2026-08-14 — a deferred pitch-guess must not hold rank 1 of the queue.

---

## EPIC-3 — Cost intelligence & routing

**Outcome.** When this epic is done:

- Users see **cost per provider, per model, per task tier** in one view (dashboard exists for spend; tier-level routing is decided/shipped).
- The harness **auto-routes each task to the best model for its tier** (cheap/fast/strong) **within the user's explicit provider** — never cross-provider (charter: "No automatic cross-provider fallback"; COST-1's DoD records this boundary + a decision ticket for the rejected cross-provider reading).
- The model catalog **refreshes in CI** so routing never targets a stale/removed model (the gpt-5.x-mini catalog bug class is gone).

**Appetite:** ≤ 1 cycle.

**Why now:** the cost-aware-routing design is written (2026-08-12) and the catalog-drift failure mode burned us on 2026-08-12 (invalid model-tier IDs). The plumbing (cost ledger, budget caps, provider scorecard) exists; routing is the payoff — bounded to within-provider tiering per the charter.

**Sequence (dependency-ordered):**

1. COST-2 — model-catalog auto-refresh in CI (routing input must be current)
2. COST-1 — cost-aware router, **within-provider tier choice only** (per-task-tier model choice; cross-provider routing explicitly out of scope)

**DoD:**

- [ ] Catalog refresh in CI (hermetic drift check).
- [ ] Router ships or is parked with a recorded decision; per-tier routing documented; charter boundary (no cross-provider fallback) recorded in the DoD + decision ticket.
- [ ] EPIC-3 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| COST-1 | Cost-aware router: **within-provider** per-task-tier model choice (cross-provider routing explicitly out of scope per charter) | 0.80 | 2 pw |
| COST-2 | Model-catalog auto-refresh in CI | 0.50 | 0.5 pw |

---

## EPIC-4 — Portability & distribution (PARKED)

**Status:** **PARKED 2026-08-14** — CHARTER.md out-of-scope #7: "Windows/cloud are non-goals until a consumer asks (a non-goal is an invitation to open an issue, not a refusal)." PORT-1 (RICE 0.11) and PORT-2 (RICE 0.20) are the two lowest items in the queue and directly contradict the charter; no consumer issue exists.

**Kept:** the release machinery (Homebrew formula + release.yml + tag/build/bootstrap scripts) — macOS/Homebrew is the shipped leg and stays.

**Replaced by:**
- **PORT-0** (EPIC-6) — quarterly park re-confirm decision ticket (0.1 pw/quarter): re-confirm "parked, local is enough today" OR unpark on a real consumer issue (Windows/cloud request from a stranger).

**Unpark trigger:** a consumer issue asking for Windows/cloud; the item re-enters the queue at measured RICE with that evidence attached.

---

## EPIC-5 — Insight & DX (DISSOLVED)

**Status:** **DISSOLVED 2026-08-14** — two parked ideas (DX-1 skill-audit 0.25, DX-2 context stats 0.17) glued by a theme do not make a stream of change. Both returned to the idea inbox with recorded decisions:
- `docs/knowledge-base/decision/continuous-self-evaluation-of-harness-parked-idea.md` (DX-1; its data-driven drift-guard half folds into GOV-1 under EPIC-6)
- DX-2 stays user-parked ("treat context as a separate feature")

**Re-shape trigger:** a real consumer pain — an actual skill collision, or a user asking for session stats.

---

## EPIC-6 — Repo maturity (NEW 2026-08-14)

**Outcome.** When this epic is done:

- The **backlog shrinks** (≤ ~12 active rows; TAX/GOV/OSS items shipped; the row closes).
- A **stranger can install and fix in <10 min**: the canonical Go install path works (OSS-1), CONTRIBUTING has a first-issue path + review SLA (OSS-2), CODEOWNERS is an ownership matrix (OWN-1).
- The **planning layer self-audits**: PM docs relocated under `docs/governance/`, dated specs archived (GOV-2), and a hermetic drift/conformance guard (GOV-1) makes "scope discipline by default" the same kind of invariant as exit-code-9.

**Appetite:** ≈ 1.5 pw, then **closes** (this is a cleanup epic with a closing DoD, not a sixth permanent stream).

**Why now:** the charter promises these follow-ups ("PM artifacts are slated to move under `docs/governance/` (relocation is a follow-up)", "dated planning specs are archived to one decisions file") but **no backlog item owns them**; meanwhile the queue still contradicts the charter (EPIC-4/EVAL-10 survived it), the documented install path is broken for strangers (`go.mod:1` vs README/remote), and the drift class has burned us three times (EPIC-4/5 survival, README "20-task dataset" vs 50 cases, 19 dated specs). Four personas independently proposed this epic; it absorbs governance, community, and debt streams.

**Sequence (dependency-ordered):**

1. GOV-2 — governance relocation + spec archive (GOV-1 shipped first per SCOPE.md scope change #1 — the guard is path-independent; GOV-2 is next)
2. OSS-1 — canonical install & identity (the single highest-value item in the scan: RICE ~18, 0.25 pw)
3. GOV-1 — drift + charter-conformance + data-driven guard (hermetic; extends `test_docs_drift.py`)
4. OSS-2 — contributor on-ramp v2 + SECURITY supported-versions ritual line
5. OWN-1 — CODEOWNERS ownership matrix
6. TAX-1 — opt-in usage evidence via cost ledger (RICE Reach becomes measured)
7. TAX-2 — flag/env-var prune audit
8. PORT-0 — quarterly park re-confirm ticket (ongoing, 0.1 pw/quarter)

**DoD:**

- [x] GOV-2 relocation + spec archive merged (19 dated specs → `docs/governance/specs-archive/` + `decisions.md` index; living PM docs stay at root per GOV-2 scope decision — session-entry contract + GOV-1 path reads + coding-054 grader; BACKLOG ranked-table repair landed #118).
- [ ] OSS-1 shipped: canonical module path + identity + CI install check (documented install path works).
- [x] GOV-1 shipped (#112); GOV-2 is the next EPIC-6 item.
- [ ] OSS-2/OWN-1/SECURITY-line shipped; TAX-1/TAX-2 shipped or parked with recorded decisions.
- [ ] Backlog ≤ ~12 active rows; EPIC-6 row closed in ROADMAP; CHANGELOG entries per shipped item.

**Items (index — source of truth is BACKLOG.md):**

| ID | Item | RICE | Effort |
|---|---|---|---|
| OSS-1 | Canonical module path + identity alignment + CI install check | ~18 | 0.25 pw |
| GOV-1 | PM drift + charter-conformance + data-driven guard (hermetic) | ~3.0 | 0.5 pw |
| GOV-2 | Governance relocation + spec archive + BACKLOG table repair (GOV-1 shipped first #112; GOV-2 next) | ~2.5 | 0.25–0.5 pw |
| OSS-2 | Contributor on-ramp v2: first-issue path, bugfix carve-out, SLA, MIT-in/out | ~2.0 | 0.5 pw |
| OWN-1 | CODEOWNERS ownership matrix + fallback-to-lead | ~1.5 | 0.25 pw |
| TAX-1 | Opt-in usage evidence via cost ledger (measured RICE Reach) | ~0.7 | 0.25 pw |
| TAX-2 | Flag/env-var prune audit | ~0.8 | 0.25 pw |
| SECURITY | Supported-versions bump in release ritual (same PR as CHANGELOG) | ~1.0 | 0.25 pw |
| PORT-0 | Quarterly park re-confirm ticket (replaces EPIC-4 PORT-1/PORT-2) | ~0.1 | 0.1 pw/quarter |
| REL-1 | Version-truth stamp: nightly/CI ldflags `piVersion` (never `dev`) + hermetic test | ~3.0 | 0.1 pw |
| REL-2 | Changelog ledger repair: v0.7.0 gap + 0.4.3→0.6.0 jump; GOV-1 guard from v0.11.0 | ~2.0 | 0.1 pw |
| REL-5 | Baseline-provenance fix: `PI_MODEL_TIER=cheap` in nightly when committing re-baselines | ~1.5 | 0.1 pw |
| REL-3 | Post-release brew verify CI job (formula installs in temp prefix, `pi-run version` == tag) | ~1.5 | 0.25 pw |
| REL-4 | Node-drift guard + `doctor` warning (Node/Go/deepeval/pi-subagents pins) | ~1.0 | 0.25 pw |

---

## Parked / deferred (deliberately not active)

| Item | Why parked | Unpark trigger |
|---|---|---|
| EPIC-4 (PORT-1 Windows, PORT-2 cloud) | Charter non-goals until a consumer asks | Consumer issue; PORT-0 quarterly re-confirm |
| EVAL-10 (output gallery) | Removed — scorecard JSON is the gallery | A human/external consumer demonstrably reading artifacts |
| EVAL-7 (sandbox live runs) | Nightly already runs on ephemeral GitHub VMs; theoretical contamination class | Real contamination incident in the nightly, or external consumer |
| HEAL-1 (watchdog heartbeat) | User-deferred; 3.20* is a pitch-guess | Watchdog-missed wedge class in `PI_SELF_HEAL` data, or user re-prioritization |
| HEAL-4 (loop/stuck detection) | Deferred; needs thresholds | Evidence of loop/stuck class with usable thresholds |
| DX-1 (skill audit) | Dissolved with EPIC-5; data-guard half → GOV-1 | Real skill collision; user re-prioritization |
| DX-2 (context stats) | User-parked ("treat context as a separate feature") | User re-prioritization |
| Cloud eval backend | Deferred to backlog per user | EPIC-4 (PORT-2): second machine/CI-with-keys pattern emerges |
| Upstream pi-subagents release watch | Waiting on release carrying `toolTimeoutMs` | New upstream release (W5 Part C) |
| Docker weekly eval | Kept weekly by design | If nightly signal degrades without container isolation |
