# pi-harness — Backlog (ranked)

**Owner:** forrestthomas · **Last updated:** 2026-08-14
Ranked by RICE (Reach × Impact × Confidence / Effort; from
`productskills/feature-prioritization`). Higher = do next. Workstreams are
promoted to `ROADMAP.md` when they become active. See `STATUS.md` for the
one-screen snapshot, `EPICS.md` for the epic grouping layer, and
`docs/roadmap-workflow.md` for the cycle ritual.

## Ranked queue (2026-08-14 maturity-scan pass — EPIC-1 trimmed, EPIC-6 added; see `EPICS.md`)

| Rank | Item | RICE | Epic | Effort | DoD sketch |
|---|---|---|---|---|---|
| 1 | **GOV-3 — Wire drift guards into CI** (enforce GOV-1; today hermetic-only) | ~2.0 | EPIC-6 | 0.15 pw | add `test_docs_drift.py` + `test_pm_drift.py` to a CI job (python-quick or nightly deterministic); verify a deliberate drift fails CI. Found 2026-08-14 (GOV-2 verify): neither guard runs in any workflow despite the docs-audit "runs in CI (python-quick)" claim |

| 2 | **OWN-1 — CODEOWNERS ownership matrix + fallback-to-lead** | ~1.5 | EPIC-6 | 0.25 pw | real matrix (Go core, /eval, .github, scripts/) + explicit fallback; seeds the charter's second-owner split trigger |

| 3 | **EVAL-16 — Harness-change eval gate (pilot-first)** | ~1.2 | EPIC-1 | 0.5 pw | scorecard-delta on eval-touching PRs; pilot after W5 Part C; promote to enforced gate on first caught regression (zero-token silent-success class) |

| 4 | **HEAL-5 — `pi-run hooks` post-rebase hook** (auto-continue wedged rebase) | 1.20 | EPIC-2 | 0.5 pw | post-rebase hook invokes `pi-run self-heal` after agent timeout; hermetic test |

| 5 | **EVAL-12 — Live re-baseline after W10** (scheduled: tonight after the 03:00 UTC nightly) | 1.10 | EPIC-1 | 0.25 pw | pull live-results artifact, review the 54-case report, commit `--update-baseline`; **DoD: 0 unbaselined live cases** + standing re-baseline cadence (after every dataset/harness change; history in `eval/baselines/`) |

| 6 | **SECURITY — supported-versions bump in release ritual** | ~1.0 | EPIC-6 | 0.25 pw | same PR as CHANGELOG entry bumps the supported-versions table (SECURITY.md) |

| 7 | **COST-1 — Cost-aware router, within-provider tier choice only** | 0.80 | EPIC-3 | 2 pw | per-task-tier model choice **within the explicit provider**; cross-provider routing explicitly out of scope (charter NOT-6) + decision ticket |

| 8 | **TAX-1 — Opt-in usage evidence via cost ledger** | ~0.7 | EPIC-6 | 0.25 pw | per-provider/per-flag counts from `.pi/cost-ledger.jsonl` (local, opt-in, no keys) → RICE Reach becomes measured |

| 9 | **EVAL-11 — Auto-open GitHub issue on N self-heals** | 0.50 | EPIC-1 | 0.5 pw | unblocked by W6; opens an issue when self-heal count crosses a threshold |

| 10 | **COST-2 — Model-catalog auto-refresh in CI** | 0.50 | EPIC-3 | 0.5 pw | hermetic drift check so routing never targets a stale/removed model |

| 11 | **EVAL-9 — Known-flaky quarantine mechanism** | 0.40 | EPIC-1 | 0.5 pw | managed flake triage with re-entry review |

| 12 | **HEAL-2 — Tune watchdog silent-window/restart-budget** | 0.35 | EPIC-2 | 0.25 pw | **data-gated**: ≥1 week non-zero `PI_SELF_HEAL` wedge coverage + W5 Part C verified (EVAL-6 is the pump) |

| 13 | **HEAL-3 — Auto-resume decision** (decision ticket) | 0.30 | EPIC-2 | 0.25 pw | record ship-or-park rationale for auto-resume after clean git-state recovery |

| 14 | **PORT-0 — Quarterly park re-confirm ticket** (replaces EPIC-4 PORT-1/PORT-2) | ~0.1 | EPIC-6 | 0.1 pw/q | re-confirm "parked, local is enough today" OR unpark on a consumer issue; all EPIC-4's DoD actually demands |


| *(SHIPPED — see CHANGELOG)* | | | | | |
| 1 | **OSS-2 — Contributor on-ramp v2 — SHIPPED (#122)** | ~2.0 | EPIC-6 | 0.5 pw | CONTRIBUTING first-issue path (`good first issue` label) + 7-day review SLA mirroring SECURITY + MIT-in/MIT-out line; PR-template bugfix carve-out (no ROADMAP citation for in-scope fixes); LICENSE copyright name → canonical `forrestbthomas` |
| 1 | **GOV-2 — Governance relocation + spec archive + BACKLOG table repair — SHIPPED (#119)** | ~2.5 | EPIC-6 | 0.25–0.5 pw | 19 dated specs → `docs/governance/specs-archive/` + `decisions.md` index; living PM docs stay at root (GOV-2 scope decision — session-entry contract + GOV-1 path reads + coding-054 grader); ranked-table repair landed #118; SCOPE.md archived to `docs/governance/scope-history/` |
| 16 | **OSS-1 — Canonical module path + identity alignment + CI install check — SHIPPED (#102, v0.10.0)** | ~18 | EPIC-6 | 0.25 pw | `go.mod:1` (`forrestthomas1`) vs README/remote (`forrestbthomas`) vs CODEOWNERS (`forrestthomas`); `go install github.com/forrestthomas/pi-harness@latest` must work; CI check installs/builds from the documented URL |

| 17 | **GOV-1 — PM drift + charter-conformance + data-driven guard — SHIPPED (#112)** | ~3.0 | EPIC-6 | 0.5 pw | hermetic guard extends `test_docs_drift.py`: EPICS↔BACKLOG↔STATUS consistency, RICE-rank order, charter-NOT-clause checks, data-vs-prose (README dataset count vs `coding_samples.jsonl`); drop semantic layer if noisy after a month |

| 18 | **EVAL-15 — Split-seam verification (dry-run + contract doc) — SHIPPED (#107)** | ~1.5 | EPIC-1 | 0.5 pw | `docs/benchmark-seam.md` contract + hermetic tarball self-containment (no `pi-run`/Go dep) + `eval/tests/` classified harness-contract vs benchmark; makes pi-bench split cheap when triggered |

| 19 | **EVAL-6 — Agentic case family** (multi-turn/tool-using/subagent/stall-recovery) — slice 1 (tool-using) SHIPPED #111; slice 2 (multi-turn/subagent) open | 1.40 | EPIC-1 | 1 pw | new task surface exercises harness differentiators; **the data pump HEAL-2 is gated on** (print-mode cannot wedge → `self-heal events: 0`) |

| 20 | **EVAL-13 — Cost-variance tolerance in nightly gate — SHIPPED (#105)** | ~1.0 | EPIC-1 | 0.5 pw | `costFlakes` in scorecard, median-of-N per-case cost, fail only ≥2 runs over or median shift; kills the coding-010 false-cost-alarm class |

| 21 | **TAX-2 — Flag/env-var prune audit — SHIPPED (#110)** | ~0.8 | EPIC-6 | 0.25 pw | audit each flag/env/exit-code against `git log -S` + workflows; delete unused; trim README tables |

| 22 | **EVAL-14 — Benchmark provenance parity — SHIPPED (#106)** | ~0.7 | EPIC-1 | 0.5 pw | ci-benchmark scorecard carries live-schema provenance (datasetVersion, agentModel, judgeModel, piVersion); provider-scorecard gates on it |

| 23 | **EVAL-8 — Judge-case stabilization — SHIPPED (#108)** | 0.50 | EPIC-1 | 0.5 pw | majority-of-3, more deterministic graders; bound judge variance |

| 24 | **REL-1 — Version-truth stamp — SHIPPED (#104, v0.10.0)** (nightly/CI ldflags `piVersion`, never `dev`) | ~3.0 | EPIC-6 | 0.1 pw | `-ldflags "-X …/cli.Version=$(git describe --tags --always)"` in nightly/CI builds + hermetic piVersion-parses test; rides **v0.10.0** (Architect/TechLead finding: scorecards currently stamp `dev`) |

| 25 | **REL-2 — Changelog ledger repair — SHIPPED (#104, v0.10.0)** (v0.7.0 gap + 0.4.3→0.6.0 jump) | ~2.0 | EPIC-6 | 0.1 pw | backfill `[0.7.0]` (tag exists, no section) + audit full history; GOV-1 enforces "every tag has a changelog entry" from v0.11.0; rides **v0.10.0** (OSS: release-blocking for 1.0) |

| 26 | **REL-3 — Post-release brew verify CI job — SHIPPED (#109)** | ~1.5 | EPIC-6 | 0.25 pw | release.yml step installs the formula in a temp prefix and asserts `pi-run version` == tag (fixes fire-and-forget `TAP_PUSH_TOKEN` push); rides **v0.11.0** (TechLead: highest-failure-risk unverified step) |

| 27 | **REL-4 — Node-drift guard + `doctor` warning — SHIPPED (#109)** | ~1.0 | EPIC-6 | 0.25 pw | hermetic pin check (Node 22.19.0 / Go 1.26.5 / deepeval ~4.1 / pi-subagents 0.48.0) + `doctor` warns when `resolveNodeVersion` picks highest-installed; rides **v0.11.0** (TechLead: no roadmap item owned it) |

| 28 | **REL-5 — Baseline-provenance fix — SHIPPED (v0.10.0)** (`PI_MODEL_TIER=cheap` in nightly) | ~1.5 | EPIC-6 | 0.1 pw | nightly sets `PI_MODEL_TIER=cheap` when committing re-baselines so `agentModel` is never `unknown` (W2 note, Dogfooder); rides **v0.10.0** with EVAL-12 |

_Applied 2026-08-14 (maturity-scan convergence; see `.pi/debate/maturity-scan/synthesis.md` and `docs/knowledge-base/decision/2026-08-14-persona-debate-scope.md`): EPIC-1 trimmed (EVAL-10 REMOVED, EVAL-7 PARKED, EVAL-5 capped at 50, +EVAL-13/14/15/16, EVAL-6 → RICE 1.4 after EVAL-12, EVAL-12 gains 0-unbaselined DoD + standing cadence); EPIC-2 (HEAL-1 demoted to idea inbox, HEAL-2 data-gated); EPIC-3 (COST-1 within-provider only); EPIC-4 PARKED → PORT-0; EPIC-5 DISSOLVED (DX-1/DX-2 → idea inbox); EPIC-6 NEW (Repo maturity)._

## Idea inbox (unranked, capture-only)

- **Watchdog liveness heartbeat (HEAL-1)** — demoted from rank 1 (2026-08-14): a user-deferred pitch-guess (RICE 3.20* = Reach 2 · Impact 1 · Conf 0.8 / 0.5 pw) must not hold the queue's top slot. Re-promotion trigger: a watchdog-missed wedge class in `PI_SELF_HEAL` data, or explicit user re-prioritization; re-score from evidence. Design question kept here: where it lives (harness watchdog vs upstream pi-subagents vs parent-agent nudge) and whether it writes `.pi/heal/events.jsonl`.
- **Eval output gallery (EVAL-10)** — removed from EPIC-1 (2026-08-14): scorecard JSON + `score_run.py` step-summary already render human-readable output; unpark if a human/external consumer demonstrably reads the artifacts.
- **DX-1 — Harness skill compatibility audit** — returned from EPIC-5 (dissolved); data-driven drift-guard half folded into GOV-1. Re-shape on a real skill collision.
- **DX-2 — `pi-run context` session-stats** — user-parked ("treat context as a separate feature"); re-prioritize on user request.
- **Dataset growth 20 → 100 (EVAL-5 follow-up)** — capped at 50 (2026-08-14); each case is permanent rent (grader + reference + flake triage + nightly cost). Growth only on a consumer or 30 days of regression-catch evidence at ≤$2/night.

## How items get in / out

- **In:** one-paragraph pitch + DoD + rough RICE, added here, ranked. Epic items are tagged with their epic id and indexed in `EPICS.md`.
- **Promoted:** user approves moving to ROADMAP as an active workstream with a budget.
- **Out:** DoD closed in ROADMAP → CHANGELOG; or explicitly rejected (record why).
- **Scope rule:** any change that serves none of the above is out of scope unless it earns a backlog entry first.
- **Cadence:** re-rank at cycle start (see `docs/roadmap-workflow.md`); prune items with no evidence after 3 months; close epics like workstreams (DoD → CHANGELOG).
- **Charter conformance (2026-08-14):** every backlog item must serve a CHARTER.md in-scope clause or an earned exception; items that contradict a non-goal are parked or removed (GOV-1 will mechanize this).
