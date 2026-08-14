# Governance Home — decisions, archived specs, scope contracts

Internal governance lives here, off the contributor-facing surface
(`CHARTER.md` clause 3). The **living** ritual docs — `ROADMAP.md`,
`BACKLOG.md`, `STATUS.md`, `EPICS.md`, and the active `SCOPE.md` — stay at the
repo root because they are the session-entry contract, are path-read by the
GOV-1 drift guard (`eval/tests/test_pm_drift.py`, `eval/tests/test_docs_drift.py`),
and a live eval grader (`eval/datasets/graders/coding-054/grade.py`) tests for
`ROADMAP.md` (GOV-2 scope decision, 2026-08-14).

## Decisions

Durable decision records (the "why") live in
[`docs/knowledge-base/decision/`](../knowledge-base/decision/). This file
indexes the dated planning specs archived here and the decisions they record.

### Archived dated specs (`specs-archive/`)

Moved 2026-08-14 (GOV-2): 19 dated planning specs from
`docs/superpowers/specs/` → `docs/governance/specs-archive/` via `git mv`
(full text preserved, git history intact; no content rewritten). Status is as
archived; specs without an in-file status banner are marked from the ROADMAP
workstream that shipped (or still runs) them.

| Date | Spec | Status (as archived) | Decision / superseded by |
|---|---|---|---|
| 2026-08-01 | `2026-08-01-terminal-chatbot-eval-design.md` | Approved for specification review | Eval harness terminal design; implementation plan in `docs/superpowers/plans/2026-08-01-terminal-chatbot-eval-implementation.md` |
| 2026-08-02 | `2026-08-02-open-source-ready-design.md` | (no in-file banner) | Open-source readiness; largely superseded by OSS-1 canonical identity (#102) + OSS-2 on-ramp (backlog) |
| 2026-08-02 | `2026-08-02-pi-run-cli-design.md` | Approved (pending written-spec review) | The Go CLI shape (`cmd/pi-run` → `internal/cli/`); shipped across v0.x |
| 2026-08-09 | `2026-08-09-secret-manager-and-personalization-design.md` | Approved (both parts) | Env-first → `bw_get` secret resolution + `PI_RUN_PERSONAL`; shipped (secret resolution) |
| 2026-08-09 | `2026-08-09-v0.3.0-robustness-design.md` | Approved for spec (user-reviewed) | v0.3.0 robustness; shipped |
| 2026-08-10 | `2026-08-10-go-1.26-toolchain-design.md` | SHIPPED — Go 1.26 toolchain adopted (v0.4.2 era; `go.mod` go 1.26.5) | Toolchain pins are contract (REL-4 Node-drift guard) |
| 2026-08-11 | `2026-08-11-benchmark-runner-design.md` | SHIPPED — benchmark runner landed in v0.5.0 (PR #25) | `pi-run ci-benchmark` family |
| 2026-08-11 | `2026-08-11-ci-benchmark-scorecard-design.md` | SHIPPED — `pi-run ci-benchmark` + provider scorecard (v0.6.0 era; scorecard.go) | Still cited by `provider-scorecard.yml` §4.6 |
| 2026-08-11 | `2026-08-11-cost-tracking-design.md` | SHIPPED — cost ledger + budget caps (v0.5.0, PR #25; cost.go) | `.pi/cost-ledger.jsonl` + `PI_MAX_BUDGET_USD`; COST-1/2 follow in backlog |
| 2026-08-12 | `2026-08-12-cost-aware-routing-design.md` | SHIPPED — `--model-tier` + `PI_MODEL_TIER` (v0.8.0, PR #38) | Cross-provider routing explicitly out of scope (charter NOT-6) |
| 2026-08-12 | `2026-08-12-deterministic-eval-hardening-design.md` | SHIPPED — deterministic eval hardening (v0.9.0, PRs #40/#42–#44) | mcp-server/OTel contract-test sections superseded by the v0.10.0 cut-list removal (CHANGELOG) |
| 2026-08-12 | `2026-08-12-live-agent-eval-v2-design.md` | SHIPPED — live eval v2 (v0.9.0, PRs #45–#48) | Still cited by `nightly-live-eval.yml` §4.3 (two-speed pipeline) |
| 2026-08-13 | `2026-08-13-per-tool-call-timeout-upstream.md` | (no in-file banner) | **ACTIVE** — W5 Part C pending upstream release carrying `toolTimeoutMs`; SCOPE.md ACTIVE (file-surface expansion 2026-08-13) |
| 2026-08-13 | `2026-08-13-self-healing-design.md` | SHIPPED — self-healing W1 (v0.9.1, PRs #59–#63) | Watchdog / group-kill / git-state recovery / escalation packet / exit 9 |
| 2026-08-13 | `2026-08-13-surface-self-heal-events-scorecard.md` | (no in-file banner) | SHIPPED per ROADMAP W6 (#83) — `selfHeal {nEvents, byKind}` in scorecard |
| 2026-08-14 | `2026-08-14-dataset-growth-to-50.md` | (no in-file banner) | SHIPPED per ROADMAP W10 (EVAL-5) — 54 live cases, `tasks.json` count authority |
| 2026-08-14 | `2026-08-14-dataset-versioning-and-provenance.md` | (no in-file banner) | SHIPPED per ROADMAP W8 (EVAL-3, #89) — `datasetVersion` + provenance |
| 2026-08-14 | `2026-08-14-flake-aware-gate-and-evidence-artifacts.md` | (no in-file banner) | SHIPPED per ROADMAP W7 (EVAL-1/EVAL-2, #87) — flake-aware gate + always-upload |
| 2026-08-14 | `2026-08-14-self-heal-in-provider-scorecard.md` | (no in-file banner) | SHIPPED per ROADMAP W9 (EVAL-4, #91) |

### Scope contracts (`scope-history/`)

Superseded `SCOPE.md` boundary contracts archive here (the
superseded-contracts-stay-as-history convention from `docs/roadmap-workflow.md`).

| Date closed | Contract | Outcome |
|---|---|---|
| 2026-08-14 | `2026-08-14-docs-audit.md` | Docs-audit change-set — 3 waves/3 PRs (#113–#116) merged, 3 scope changes logged, CLOSED |

## Convention going forward

- New **dated planning specs** land under `docs/governance/specs-archive/`
  (not `docs/superpowers/specs/`).
- **Durable decisions** (the "why") go to `docs/knowledge-base/decision/`.
- **Live workstream boundary contracts** stay in `SCOPE.md` at the repo root;
  superseded contracts archive to `docs/governance/scope-history/`.
- The living ritual docs (ROADMAP/BACKLOG/STATUS/EPICS) stay at root — moving
  them is out of scope unless a consumer asks (GOV-2 scope decision).
