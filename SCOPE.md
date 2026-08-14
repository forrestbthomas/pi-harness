# Scope Contract
**Task:** EVAL-16 — Harness-change eval gate, pilot-first | **Plan:** BACKLOG rank 1 (EVAL-16, RICE ~1.6, 0.3–0.5 pw), EPIC-1, `.pi/debate/whats-next/synthesis.md` | **Date:** 2026-08-14 | **Status:** ACTIVE

## Context (from the debate)
The loop must measure changes to itself: a change to the eval/harness surface can silently shift every scorecard (the zero-token silent-success class) and nothing measures the delta today. Pilot-first: build the delta mechanics hermetically now, learn delta-vs-noise, promote to an enforced gate on the first caught regression. **Pilot is independent of W5 Part C** (orthogonal surfaces — agent-output honesty vs tool-timeout telemetry).

## In Scope (the pilot slice)
1. **Eval-surface classifier** (hermetic) — a utility that classifies a PR's changed paths into eval-surface categories, reusing EVAL-15's seam inventory (`eval/tests/test_benchmark_seam.py` + `docs/benchmark-seam.md` §5 surface: `eval/datasets/`, `eval/benchmarks/`, `eval/scripts/score_run.py`, `eval/grader.py`, `eval/secret_backend.py`, `eval/baselines/`, `eval/tests/`, `tasks.json`, Go `internal/cli/scorecard*`, nightly/provider-scorecard workflows). Unit-tested with fixtures.
2. **Scorecard-delta renderer** (hermetic) — a small module (new `eval/scripts/score_delta.py`) that compares two scorecards (candidate report vs committed baseline, or two nightlies) per case: pass-rate delta + cost delta, reusing the EVAL-2 flake + EVAL-13 cost tolerance model; emits a compact delta report (JSON + markdown). Unit tests with hermetic fixtures — **no live runs, no API cost**.
3. **CI wiring (report-only)** — a cheap step in `ci.yml` (python-quick or a new step) on eval-touching PRs: runs the classifier, runs the hermetic eval checks, produces a PR delta-report artifact (changed surfaces, invariants checked, per-case delta if a candidate scorecard is supplied, and a `needs-nightly-verification` flag for live-surface changes). **Report-only: never fails the PR in the pilot.**
4. **Promotion rule recorded** — in the DoD + a short design note: promote to an enforced gate (fail CI on the delta class) on the first caught regression, or once the delta-vs-noise mechanics are validated against real nightlies.
5. **Nightly-artifact delta** (user-approved addition): the nightly workflow also runs `score_delta.py --diff` comparing each nightly's scorecard against the committed baseline and writes the delta report into `eval/live-results/` (already uploaded as the `live-eval-results` artifact) — report-only.

## Out of Scope
- **Enforcing the gate** (promotion is a later, evidence-gated decision — the pilot is report-only by design).
- W5 Part C (`toolTimeoutMs` observation — orthogonal surface; rides v0.12.0).
- New dataset cases or graders (EVAL-5/EVAL-17 territory).
- Changing the Go scorecard struct or the `score_run.py` gate semantics (read/reuse only).
- New live-run spend (the pilot is hermetic).
- EVAL-12 re-baseline (tonight's scheduled data release — separate item).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | user-expansion | Also wire the scorecard-delta report into the nightly artifact (nightly-vs-baseline diff in `live-eval-results`) | User chose the wider pilot when approving the contract | Permit | PR (this change) |

# Follow-up Tasks
- [ ] After the pilot ships: record the first scorecard-delta observed on a real PR/nightly (delta-vs-noise baseline).
- [ ] Promotion decision ticket: enforced-gate criteria (first caught regression OR validated delta mechanics).
