# Scope Contract
**Task:** EVAL-17 — Agentic slice 2: chat-session runner + multi-turn/subagent cases | **Plan:** docs/specs/eval17-agentic-slice2.md | **Date:** 2026-08-15 | **Status:** CLOSED — shipped 2026-08-15 (EVAL-17 PR)

## In Scope
- **Files:**
  - `eval/conftest.py` — add `run_pi_session` (chat-session runner)
  - `eval/tests/test_chat_runner.py` (new) — hermetic runner tests
  - `eval/tests/test_live_suite.py` — `_run_agent_once` branches on `surface == "chat"`; `PI_SELF_HEAL=1` on chat runs
  - `eval/datasets/coding_samples.jsonl` — 3 chat cases (next free `coding-0NN` ids, category `agentic`, `surface: "chat"`, `turns`)
  - `eval/datasets/tasks.json` — `datasetVersion` bump
  - `eval/datasets/graders/coding-0NN/grade.py` (×3, new) — deterministic graders
  - `eval/datasets/references/coding-0NN/answer.txt` (×3, new)
  - `eval/baselines/live-baseline.json` — re-baseline entries (data release)
  - `docs/benchmark-seam.md` — surface note (chat + memory-free reaffirmation)
  - `docs/specs/eval17-agentic-slice2.md` — this spec (archives per GOV-2 on ship)
  - `CHANGELOG.md`, `BACKLOG.md`, `EPICS.md`, `STATUS.md` — record the ship
- **Features:**
  - Chat-session runner: turn 1 via `pi-run print --max-budget-usd <cap> --cost-mode live-eval [--permission-mode plan]`, turns 2+ via `pi-run resume`; transcript + stats (pass/costUsd/judgeCostUsd/tokens/latencyMs)
  - 3 chat cases: multi-turn correction, subagent delegation, follow-up clarification — deterministic graders on the final answer/transcript
  - Watchdog pump: `PI_SELF_HEAL=1` on chat runs (HEAL-2 data source)
  - Re-baseline: new cases recorded honestly, 0 unbaselined, provenance stamped
- **Boundaries:**
  - Memory-free eval invariant: same pi-run spawn path, no new packages; pi-subagents (already pinned) is tool delegation, not memory
  - Live count stays 55 (`surface: "chat"` ≠ `"live"`); count authority = tasks.json
  - Report shape unchanged — score_run.py / gate / baseline schema untouched
  - Deterministic graders only (no new LLM-judge cases)

## Out of Scope
- Runtime-agnostic scorer (deferred new surface — dogfood posture)
- Subagent support in pi-run (upstream pi-subagents)
- Sandboxing (EVAL-7 — parked)
- Memory engines / any new package in the eval spawn path (MEM-1 closed)
- EVAL-16 enforcement promotion
- Dataset growth beyond the 3 chat cases
- Changing print-mode semantics or existing case grading
- The standalone eval product surface (pi-bench split — consumer-triggered)

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | emergent | pi-run enabler: `--session-id`/`--session` pins a session without a cumulative budget cap; pinned resume skips `--continue` | `--max-budget-usd` is cumulative across the ledger (rejected the per-case cap approach); pi rejects `--continue` + a session pin; `--resume` opens the TUI (non-interactive capture fails) | Permit (anticipated in spec: "if the session-persistence enabler needs a pi-run flag") | `internal/cli/pi.go` + `pi_test.go`; runner uses print `--session-id` + resume `--session` |
| 2 | emergent | Chat runs get a **per-run unique** `--session-id` (`eval-<case>-<hex>`), not per-case | A shared id made EVAL_RUNS_PER_CASE samples continue run 1's session — not independent samples (contaminated measurement) | Permit (correctness of the measurement) | `_run_agent_once` session id includes `secrets.token_hex(4)` |
| 3 | emergent | Chat cost/tokens attributed from the **pinned session file** (`_session_usage`), not the ledger diff | The ledger diff double-counts multi-turn sessions (cumulative per-launch entries + interleaving) — reported $0.86/549k-tokens for a ~$0.05 session | Permit (honesty of the scorecard) | `_session_usage` reads `message.usage` blocks of `*_<session-id>.jsonl` |
| 4 | user-expansion (scope cut) | coding-057 (subagent delegation) **parked** — ships 056+058 only | Turn-2 resume timed out after a subagent-spawning turn 1 in the 5× re-baseline: subagent-in-scripted-session is not reliable yet. The eval caught the limitation; 057 stays the graded case for the future fix | Permit (user chose ship-056-058-park-057) | Follow-up task: re-add coding-057 when subagent-in-session works (runner/timeout investigation) |

# Follow-up Tasks
- [ ] Archive spec to `docs/governance/specs-archive/` on ship (GOV-2)
- [ ] Record the chat cases' re-baseline entries with provenance (done in this PR)
- [x] Confirm session-isolation mechanism at implementation (pi `--session` vs per-case cwd) — resolved: `--session-id` (turn 1) + `--session` (resume)
- [ ] **coding-057 (subagent delegation)**: re-add when subagent-in-scripted-session works (scope change #4 — the measured limitation; investigate the turn-2 resume timeout after a subagent-spawning turn 1)
