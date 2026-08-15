# Spec — EVAL-17: Agentic case family, slice 2 (multi-turn/subagent + chat-session runner)

Date: 2026-08-15 · Status: DRAFT (awaiting scope approval) · Workstream: EVAL-17 (BACKLOG rank 1, EPIC-1)
Plan location for scope-lock: `SCOPE.md` (root) · This spec persists at `docs/specs/eval17-agentic-slice2.md` until shipped, then archives per GOV-2.

## Goal

Add slice 2 of the agentic eval family: a **chat-session runner** plus a small
**multi-turn / subagent case surface**, so the eval measures the harness's
multi-turn and delegation behavior and generates the wedge data HEAL-2 is
gated on (print-mode cannot wedge; chat sessions can).

## Problem Statement

Every live case today runs `pi-run print` — a single-turn, no-session
invocation. The harness's real differentiators are unmeasured:
multi-turn chat sessions (`pi-run chat` / `resume`), subagent delegation
(`pi-subagents`, already pinned), and the watchdog acting on a *session* that
can stall on a tool call. EPIC-2's HEAL-2 (watchdog tuning) is data-gated on
**≥1 week of non-zero `PI_SELF_HEAL` wedge coverage** — print-mode cannot
produce wedges, so the pump must come from a session-running surface.
EVAL-17 is that pump, and it is also the second half of the agentic family
(EVAL-6 slice 1 shipped tool-using cases via `--permission-mode plan`).

## Context From Memory

- EVAL-6 slice 1 (#111): agentic (tool-using) cases run `pi-run print
  --permission-mode plan` → read-only tools (read,grep,find,ls). Same pattern
  extends to chat runs.
- MEM-1 (2026-08-15): eval agent runs are **memory-free** — a binding seam
  invariant (`docs/benchmark-seam.md` §contract). No memory/stateful package
  may enter the eval spawn path; adding one = re-baseline trigger. The
  already-pinned `pi-subagents` package is **tool delegation, not memory** —
  the subagent case relies on it; no NEW package is added.
- pi-run mechanics (`internal/cli/app.go`, `pi.go`): `print` = `pi -p
  [--no-session] <prompt>`; `resume` = `pi --continue <prompt>` (prompt passes
  through — the multi-turn mechanism); `--cost-mode` is **rejected for
  resume** (resumed turns still record spend deltas in the ledger, tagged
  `resume`); session persistence only when a budget cap is set
  (`--max-budget-usd`), so turn 1 must set a per-case cap.
- Count authority: `tasks.json`; README guard sums `surface == "live"` (55).
  Chat cases are `surface: "chat"` → the 55 holds.
- EVAL-12 cadence: dataset/harness-behavior changes → re-baseline as a data
  release (eval lane), 0 unbaselined, provenance recorded.
- EVAL-18 run-step gate: per-case flake/regression already handles any new
  cases with no schema change.
- Dogfood posture (2026-08-15): the eval is the loop's curriculum, not a
  product surface. EVAL-17 serves the loop (more honest measurement, the
  HEAL-2 pump).

## In Scope

1. **Chat-session runner** — `run_pi_session(turns, sample, ...)` in
   `eval/conftest.py`: turn 1 via `pi-run print --max-budget-usd <cap>
   --cost-mode live-eval [--permission-mode plan] "<turn1>"` (forces session
   persistence); turns 2..N via `pi-run resume "<turnN>"`. Returns the full
   transcript + per-run stats (pass/costUsd/judgeCostUsd=0/tokens/latencyMs).
   `PI_SELF_HEAL=1` set on chat runs (the HEAL-2 pump — watchdog events land
   in `.pi/heal/events.jsonl` and the scorecard surface).
2. **Hermetic tests** — `eval/tests/test_chat_runner.py` (keyless): command
   construction, turn feeding, transcript assembly, stats aggregation,
   timeout/empty-output failure honesty, session isolation strategy.
3. **Wire into the live suite** — `test_live_suite._run_agent_once` branches
   on `surface == "chat"` → `run_pi_session`. Report shape unchanged
   (score_run.py, gate, baseline untouched).
4. **Three chat cases** (category `agentic`, surface `chat`, next free
   `coding-0NN` ids) with deterministic graders + references:
   - **Multi-turn correction**: turn 1 asks for an approach; turn 2 challenges
     it with a constraint; the final answer must show the corrected reasoning.
   - **Subagent delegation**: the agent must delegate a review to a subagent
     and report the finding; the grader verifies the transcript names the
     delegation and reports the finding.
   - **Follow-up clarification**: turn 2 supplies missing information; the
     final answer must incorporate it.
5. **datasetVersion bump + re-baseline** (data release, eval lane): run the
   new chat cases a few times locally, record honest baseline entries
   (passRate/cost/tokens/latency), 0 unbaselined, provenance stamped.
6. **Docs**: `docs/benchmark-seam.md` surface note (chat surface + memory-free
   reaffirmation), CHANGELOG, BACKLOG EVAL-17 → SHIPPED, EPICS-1 note, STATUS
   row, spec archived per GOV-2 on ship.

## Out Of Scope

- **Runtime-agnostic scorer** (deferred new surface — dogfood posture).
- Subagent *support* in pi-run (pi-subagents is upstream; the eval only
  exercises it).
- Sandboxing (EVAL-7 — parked on a contamination incident).
- Memory engines / any new package in the eval spawn path (MEM-1 closed).
- EVAL-16 enforcement promotion.
- Dataset growth beyond the 3 chat cases (55 live held; count authority
  unchanged).
- Changing print-mode semantics or existing case grading.
- The standalone eval product surface (pi-bench split — consumer-triggered).

## Constraints

- **Memory-free invariant**: chat runs use the same pi-run spawn path; no
  memory/stateful packages added. pi-subagents (already pinned) is tool
  delegation; any NEW package = re-baseline trigger (recorded).
- **Count authority**: `tasks.json`; live count stays 55 (surface `chat`).
- **Nightly budget** $2; each chat case has a per-case `--max-budget-usd` cap
  (bounds a runaway session; also forces session persistence).
- **Report shape unchanged**: `pass/costUsd/judgeCostUsd/tokens/latencyMs` —
  score_run.py + gate + baseline schema untouched.
- **Deterministic graders only** for the new cases (no new LLM-judge cases).
- **Sequential runs**: the nightly runs cases sequentially; the runner must
  isolate sessions per case (most-recent-session ambiguity) — per-case
  session isolation is a stated implementation strategy (dedicated cwd or
  explicit session path if pi supports it; verified at implementation).

## Assumptions

- 3 chat cases × EVAL_RUNS_PER_CASE (5×) fits the $2 nightly (est +$0.10–0.30).
- Chat cases join the deterministic suite in the nightly (same test path).
- The subagent case works with the already-pinned pi-subagents package (no
  new package, no re-baseline trigger beyond the dataset change itself).
- Re-baseline rides the eval lane (data release), never a CLI patch.

## Blocking Questions

None. The session-isolation mechanism has a stated fallback; all other
ambiguities are low-risk assumptions.

## Acceptance Criteria

- **Chat case runs end-to-end.** Given a `surface == "chat"` case with N
  turns, when the suite runs it, then the runner drives N turns (print then
  resume), captures the transcript, grades the final output with the
  deterministic grader, and records pass/costUsd/tokens/latencyMs in the same
  shape as print cases.
- **Memory-free spawn path.** Given a chat case, when the runner launches,
  then the spawn commands are `pi-run print` / `pi-run resume` with no memory
  packages added (the seam invariant holds).
- **Watchdog pump.** Given a chat run that stalls, when the watchdog fires,
  then the run fails honestly (non-zero exit / per-turn timeout), `PI_SELF_HEAL=1`
  events are recorded, and the scorecard never shows a silent pass.
- **Subagent delegation is graded.** Given the delegation case, when the
  agent responds, then the grader verifies the transcript names the subagent
  handoff and reports the finding.
- **0 unbaselined.** Given the datasetVersion bump, when the baseline is
  checked, then every new chat case has a committed baseline entry with
  provenance.
- **Gate integration, no schema change.** Given the nightly report, when
  score_run gates it, then chat cases flow through the existing flake /
  regression / cost logic.
- **55 held.** Given the dataset, when the live count is computed
  (`surface == "live"`), then it stays 55 and the README guard passes.
- **Hermetic.** The runner tests pass without API keys.

## Edge Cases

- Empty/missing turn output (model failure mid-session) → run fails honestly,
  never a silent pass (same rule as print).
- Session isolation across cases (most-recent-session ambiguity) → per-case
  cwd/session strategy.
- `resume` with no prior session (turn 1 failed) → honest failure.
- Budget exceeded mid-session → exit 6 recorded, run fails honestly.
- Per-turn timeout (180s) + overall case cap → honest failure.

## Validation Plan

- **Hermetic (keyless)**: `test_chat_runner.py` + existing suite
  (`test_docs_drift`, `test_pm_drift`, `test_dataset_schema`, `test_score_run`,
  `test_score_delta`).
- **Guards**: 55-live-count guard, claims table, dataset schema (surface
  `chat` accepted), drift guards — all green.
- **Live**: the new chat cases run locally a few times → honest baseline
  entries; the next nightly exercises them end-to-end.
- **Go**: `go build ./... && go vet ./... && go test ./...` (no Go changes
  expected; if the session-persistence enabler needs a pi-run flag, that is
  flagged as a scope change with Go tests).

## Execution Plan

1. `run_pi_session` in `eval/conftest.py` + `eval/tests/test_chat_runner.py`
   (hermetic).
2. Wire `_run_agent_once` surface branch + `PI_SELF_HEAL=1` in
   `test_live_suite.py`.
3. Dataset: 3 chat cases (next free ids, category `agentic`, surface `chat`,
   `turns`, expected_output, grader, reference) + deterministic graders +
   references.
4. datasetVersion bump; local chat-case runs → re-baseline entries (data
   release).
5. Docs: benchmark-seam surface note, CHANGELOG, BACKLOG/EPICS/STATUS.
6. Full hermetic + guard suite green; implementation PR + data-release PR;
   nightly verifies; BACKLOG row → SHIPPED.

## Open Questions

- Chat cases at 5× vs a lower chat-specific run count (cost/latency
  tradeoff) — assumed 5×; revisit after the first nightly if budget/latency
  says so.
- ~~Session-isolation mechanism~~ **Resolved at implementation**: turn 1 pins
  `--session-id eval-<case>` (pi-run enabler: pinned session persists without
  a cumulative budget cap); turns 2+ continue `pi-run resume --session
  eval-<case>` (pinned resume skips `--continue`, which pi rejects; `--session`
  is non-interactive, unlike `--resume` which opens the TUI).

## Self-Check

- No blocking questions. Assumptions are low-risk and revisable. Scope
  boundaries explicit (in/out lists). Acceptance criteria observable.
  Validation plan concrete. Execution plan sequenced. Memory-free invariant
  and count authority are recorded constraints.
