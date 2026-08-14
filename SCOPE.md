# Scope Contract
**Task:** EVAL-6 (slice 1) — Agentic case family: tool-using cases that exercise the harness's read-only tool surface | **Plan:** BACKLOG EVAL-6 (EPIC-1, RICE 1.4, 1 pw total — this is the first ~0.5 pw slice); ROADMAP §Release Milestones v0.11.0 | **Date:** 2026-08-14 | **Status:** CLOSED — 1 change logged

## In Scope
- **New dataset category `agentic`** (tool-using):
  - `eval/tests/test_dataset_schema.py` — add `agentic` to `CATEGORIES` + `CATEGORY_BUDGET` (min 3, max 8); the category counts must stay within budget for all 6 existing categories too.
  - `eval/datasets/coding_samples.jsonl` — add **4 new tool-using cases** (coding-051..054, `category: agentic`, `grader: deterministic`), each requiring a read-only tool fact the agent must gather from the repo:
    - coding-051: read `go.mod` → report the Go version (tool: read)
    - coding-052: list `eval/datasets/graders/` → count the graders (tool: ls/find)
    - coding-053: read `providers.json` → count the providers (tool: read)
    - coding-054: grep the README for the "Release Milestones" section heading → name it (tool: grep)
  - `eval/datasets/graders/coding-051..054/grade.py` — deterministic graders on the tool-grounded fact (accept the true value; a hallucinated/wrong value fails). References contain the true values (oracle rule: references provably pass).
  - `eval/datasets/references/coding-051..054/` — reference answers.
  - `eval/datasets/tasks.json` — add the 4 tasks + bump `datasetVersion` (EVAL-3 rule).
- **Live-suite tool enabling:**
  - `eval/tests/test_live_suite.py` — `_run_agent_once` passes `--permission-mode plan` (read-only tools `read,grep,find,ls`) for `agentic`-category cases (only those — the other 50 keep the current no-tools invocation so their grading semantics don't change).
  - `eval/conftest.py` — no change needed (run_pi_print already forwards extra_args); verify.
- **Docs/records:** `CHANGELOG.md`, `SCOPE.md`, decision record.
- **Features:**
  1. A new task surface exercises the harness's tool-using behavior (not just print mode) — the EVAL-6 "exercise harness differentiators" goal, slice 1.
  2. Agentic cases are the first surface that can produce real `PI_SELF_HEAL` wedge observability (tool calls can stall — the data pump HEAL-2 is gated on).
  3. Deterministic grading on tool-grounded facts — a model that hallucinates the answer fails, so the eval genuinely measures tool use.
- **Boundaries:**
  - Oracle rule holds: new deterministic references MUST pass their graders.
  - Only `agentic` cases get `--permission-mode plan`; existing 50 cases keep their invocation (no semantic change to existing grading).
  - No change to judge/score_run gate math, Go side, or workflows.
  - This is slice 1 of EVAL-6 (tool-using); multi-turn/subagent/stall-recovery slices are follow-ups within the same backlog item.

## Out of Scope
- Multi-turn / subagent / interactive-session cases (EVAL-6 follow-up slices — need a different runner than `pi-run print`).
- GOV-2/GOV-1, EVAL-12 (pending nightly), COST-2.
- Changing existing case invocations or graders.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | ambiguity | The 50-cap lint (`test_dataset_has_exactly_50_records`) conflicts with adding the `agentic` category | EVAL-5's "hold at 50" was about the six existing categories growing to 100; EVAL-6 is a NEW category axis (tool-using) with its own budget, explicitly sequenced as the next EPIC-1 bet | **Permit** — the lint becomes "≥ 50 + per-category budgets" (new `agentic` (3,8) budget); the six original categories stay within their existing budgets | Done — lint relaxed to ≥50; `agentic` (3,8) added; taxonomy updated in Python + Go benchmark parser + manifest |

# Follow-up Tasks
- [ ] EVAL-6 slice 2: multi-turn/subagent cases (needs a chat-session runner) — same backlog item.
- [ ] After merge: next live nightly runs the 4 agentic cases with tools; confirm PI_SELF_HEAL observability potential on the new surface.
