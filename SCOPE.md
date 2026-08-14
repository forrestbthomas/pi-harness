# Scope Contract
**Task:** EVAL-8 — Judge stabilization (deterministic conversion + majority-of-3 knob) | **Plan:** BACKLOG EVAL-8 (EPIC-1, RICE 0.50, 0.5 pw); ROADMAP §Release Milestones v0.11.0 | **Date:** 2026-08-14 | **Status:** CLOSED — 1 change logged

## In Scope
- **Files:**
  - `eval/datasets/graders/coding-002/grade.py` (NEW) — deterministic grader for `is` vs `==` (checks value-equality vs identity concepts).
  - `eval/datasets/graders/coding-004/grade.py` (NEW) — deterministic grader for binary-search complexity (accepts `O(log n)` / `O(log2 n)` / big-O log variants).
  - `eval/datasets/graders/coding-013/grade.py` (NEW) — deterministic grader for list-vs-tuple (checks mutable/immutable concepts).
  - `eval/datasets/coding_samples.jsonl` — flip coding-002/004/013 `grader` → `deterministic` + set `graderRef`; keep `reference` (oracle rule: references must pass their new graders).
  - `eval/datasets/tasks.json` — bump `datasetVersion` (EVAL-3 rule).
  - `eval/tests/test_live_metrics.py` — **majority-of-3 judge knob**: `EVAL_JUDGE_RUNS` env (default 3): run the judge stack N times per case, pass = majority; report `judgeRuns` in properties; docstring updated.
  - `eval/scripts/score_run.py` — **BUGFIX (scope change #1): judge-case passes never reach the gate.** `extract_case_id` only accepts `test_live_suite.py` nodeids, so `test_live_metrics.py` judge passes are dropped — judge-graded cases (5) contribute cost but no pass signal. Fix: accept both `test_live_suite.py` and `test_live_metrics.py` nodeids (and `test_agent_task_completion.py`? check) so judge-case `pass`/`judgeCostUsd`/`judgeRuns` are collected like deterministic ones. Hermetic test: a report containing a metrics-layer case yields a pass in the gate.
  - `.github/workflows/nightly-live-eval.yml` — set `EVAL_JUDGE_RUNS: '3'` (explicit; default already 3).
  - `CHANGELOG.md`, `SCOPE.md`, decision record — updates.
- **Features:**
  1. 3 easy concept cases (002/004/013) leave the judge surface → deterministic, cheaper, zero-variance. Judge-graded cases: 8 → 5.
  2. Remaining judge cases run majority-of-3 (`EVAL_JUDGE_RUNS=3`) **and their pass actually reaches the gate** (the pre-existing dropped-nodeid bugfix).
- **Boundaries:**
  - Oracle rule holds: new deterministic references MUST pass their graders (schema test enforces).
  - No change to judge criteria/rubrics (only repeat-count + determinism surface).
  - No change to `score_run.py` gate math, Go side, or workflows beyond the env var.
  - `EVAL_JUDGE_RUNS` default 3 is the stabilized behavior; env-gated for cost control.

## Out of Scope
- Converting more judge cases (036-040) — deferred; the 3 easy ones are the deterministic-able set today.
- Changing judge models/rubrics (EVAL-8 bounds variance; it does not redesign judging).
- EVAL-6 (agentic), EVAL-13/14/15 (done), GOV-2/GOV-1, REL-3/REL-4.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | emergent | `score_run.py` drops judge-case passes (`test_live_metrics.py` nodeids never match `extract_case_id`'s `LIVE_SUITE_FILE` gate) — judge cases contribute cost but NO pass signal to the gate | Found while verifying EVAL-8: without the fix, majority-of-3 judge work would be dead code and the 5 judge cases would remain ungated | **Permit (within EVAL-8)** — judge cases must actually count for EVAL-8 to mean anything; fix `extract_case_id` to accept the metrics/agent-task files + hermetic test | Done — `extract_case_id` accepts `test_live_metrics.py`; metrics layer now parametrizes only the 5 judge cases (was all 50, which would double-count deterministic runs); +hermetic test |

# Follow-up Tasks
- [ ] After merge: next live nightly judges the 5 remaining cases with majority-of-3; confirm judge variance visibly drops in the scorecard.
