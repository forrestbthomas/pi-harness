# Spec — Per-Tool-Call Timeout (Upstream pi-subagents) — W5

**Date:** 2026-08-13 · **Status:** PART SHIPPED — run-level timeout pinned (Part A, 0.48.0); per-tool timeout merged upstream (Part B, a660ea3); Part C (observe via PI_SELF_HEAL) pending upstream release
**Locked Phase-1 decisions:** kill-run semantics (`timedOut: true`) · opt-in default (no built-in per-tool timeout) · supervisor-tool allowlist · file upstream issue first · Part A (pin upgrade) lands as a separate PR before Part B.

## Goal

Bound a single subagent tool call with an **opt-in per-tool timeout** in upstream pi-subagents; **adopt** the already-contributed run-level timeout machinery in the harness (pin upgrade); and **verify** the harness observes tool timeouts — closing the mid-tool wedge class that the harness's run-level stall watchdog cannot see.

## Problem Statement

A subagent tool call (e.g. `bash` running a command that spawns a background process inheriting pipes) can wedge **with output flowing**. The harness's output-stall watchdog (`PI_STALL_TIMEOUT_SECS`, exit 9) never fires because bytes keep arriving; the run continues indefinitely. Upstream pi-subagents already parses and tracks `tool_execution_start`/`tool_execution_end` (`currentToolStartedAt`, `pendingToolResults[*].startedAt`) but **never bounds a single tool call** — only the run level is bounded (our own #978/#979 shipped default wall-clock timeouts in ≥0.47.0). The harness pins `npm:pi-subagents@0.45.1`, which predates that machinery.

## Context From Memory

- **Research brief** (subagent 26e61370, HEAD `55fe0333`): full tool-call-loop map, config threading, implementation surface, test plan — all with permalinks. Key anchors: `runPiStreaming` (subagent-runner.ts#L520, tool events #L677–L680, #L2826, #L2866), `runSync` (execution.ts#L891, #L916), run-level termination `registerTimeout` (#L781–L790), deadline-collapse `min(step.timeoutMs, parentRemainingMs)` (#L1987), config ladder (`schemas.ts`#L328, `agents.ts`#L132, `config.ts`#L91/#L121, `resolveConfigDefaultTimeoutMs` validation), env-var convention (`PI_SUBAGENT_TOOL_BUDGET`, `PI_INTERCOM_ASK_TIMEOUT_MS`), MockPi test fixtures.
- **Upstream history:** our issue #978 → merged PR #979 (run-level default wall-clock timeout, shipped 0.47.0, maintainer-rebased from our `635c1bd`); #1018 added global `config.timeoutMs` (0.48.0). Issue #150 is an **unrelated** closed nvm-path bug — not a timeout request.
- **Harness side:** W1 shipped watchdog/process-group-kill/escalation/exit 9; `PI_SELF_HEAL=1` writes `.pi/heal/events.jsonl`; doctor now guards the non-interactive env. Pinned pi-subagents is 0.45.1 (predates the run-level timeout machinery).
- No `spec-index` store exists in this repo; memory layer is the context engine + this research brief.

## In Scope

- **Part A — adopt upstream (harness, separate PR):** bump `.pi/settings.json` pin `npm:pi-subagents@0.45.1` → latest ≥0.47.0 (0.48.0 at research time); sync `.pi/npm/package-lock.json`; `config-check` passes; subagent smoke (delegate/worker run) works.
- **Part B — contribute upstream (issue + PR):**
  - Per-tool-call timeout knob `toolTimeoutMs`: optional per-call param on the subagent tool schema (alongside `timeoutMs`), optional agent frontmatter, global `config.toolTimeoutMs` (via `resolveConfigDefaultTimeoutMs`-style validation), and `PI_SUBAGENT_TOOL_TIMEOUT_MS` env override. **Opt-in: no built-in default** (off unless configured anywhere).
  - Arm a per-tool timer on `tool_execution_start`; clear on `tool_execution_end`; on expiry reuse the existing process-tree termination path (SIGTERM → SIGKILL escalation), set `timedOut: true` with a **tool-specific error message**, and do **not** extend past the run budget (`min` with remaining time, #L1987 pattern).
  - **Allowlist:** no per-tool timer for `contact_supervisor` / `intercom` (legit blocking waits).
  - Tests: unit resolver/validation (precedence call > agent > config; invalid values; `MAX_TIMER_DELAY_MS` boundary); async integration "tool starts, output keeps flowing, `tool_execution_end` never arrives" → terminated at ~`toolTimeoutMs` with `timedOut: true` and tool-specific error while the run-level timeout has NOT yet fired; foreground analog; run-budget-wins interaction; completion path (no spurious kill). Follow the Windows test-skip convention.
  - Docs: `docs/configuration.md`, `docs/agents.md`, `docs/tool-reference.md`; `CHANGELOG.md` entry.
- **Part C — observe + verify (harness):** after an upstream release carrying the feature, re-verify the pin; verify a tool timeout surfaces in harness observation (`PI_SELF_HEAL` events / escalation) with a wedge contract test; close W5 DoD.

## Out Of Scope

- Soft-interrupt-and-retry semantics (defer; kill-run is the v1 contract).
- A built-in default per-tool timeout (opt-in only — blanket defaults would break legit long tools).
- Changing the child's own `bash` tool timeout behavior upstream (child's domain; #978 noted the `timeout` arg is model-passed, not a default).
- Harness watchdog changes (shipped in W1; Part C is observation only).
- Scorecard surfacing of `PI_SELF_HEAL` events (separate backlog item).
- Windows-specific fixes beyond honoring the existing test-skip convention.

## Constraints

- **Upstream repo:** TypeScript-only, ESM, `node --experimental-strip-types`, no build; `npm run typecheck` (tsc --noEmit); `npm test` (unit); `npm run test:integration`; no lint script; no CONTRIBUTING.md — contribution surface is package.json scripts + `CHANGELOG.md` entry + docs pages per release convention.
- **Composition with #979:** reuse `runSingleStepWithTimeout`/`registerTimeout` termination; honor `settled || timedOut || stopped` gates; collapse deadline with remaining run budget.
- **Event pairing may be non-1:1** (crash mid-tool, nested/parallel flows): timer keys on `pendingToolResults[flatIndex].startedAt` / `currentTool`; a missing `end` event means the timer fires (the desired wedge behavior).
- **Harness:** Go stdlib-only; exact pin format `npm:pi-subagents@<version>` (config-check enforces); `config-check` and `test_pi_subagents_pinned_exact_version` must keep passing.

## Assumptions

- Latest pi-subagents carrying the #979 machinery is 0.48.0 (verified at research HEAD).
- `toolTimeoutMs` is an acceptable name upstream; open to maintainer rename (open question).
- Kill-run on tool timeout with `timedOut: true` is acceptable v1 behavior; docs will state an elapsed timeout is not a mutation-safe boundary (per tool-reference.md:63 precedent).
- An upstream issue filed by us will be received in the same spirit as #978 (evidence-backed, verified repro).
- `contact_supervisor`/`intercom` are the only tools needing the allowlist; `subagent_wait` and other blocking tools remain bounded by run-level machinery.

## Blocking Questions

None — the Phase-1 gate decisions (semantics, default, allowlist, issue-first, Part A ordering) are locked by the user.

## Acceptance Criteria

Behavioral (Gherkin):

```gherkin
Scenario: Opt-in per-tool timeout kills a wedged tool (Part B)
  Given toolTimeoutMs is configured (call, agent, config, or env)
  When a tool starts and output keeps flowing and tool_execution_end never arrives
  Then the run terminates at ~toolTimeoutMs
  And the result/status carries timedOut: true
  And the error message names the tool and the budget
  And the run-level timeout has NOT yet fired

Scenario: Completion clears the timer (Part B)
  Given toolTimeoutMs is configured
  When tool_execution_end arrives before the budget
  Then the run is not killed and proceeds normally

Scenario: Supervisor tools are exempt (Part B)
  Given toolTimeoutMs is configured
  When contact_supervisor or intercom runs
  Then no per-tool timer is armed for it

Scenario: Run budget wins (Part B)
  Given toolTimeoutMs exceeds the remaining run-level budget
  When a tool starts
  Then the run-level deadline fires first

Scenario: Off by default (Part B)
  Given no toolTimeoutMs is configured anywhere
  When any tool runs
  Then no per-tool timer is armed

Scenario: Precedence ladder (Part B)
  Given toolTimeoutMs is set at multiple levels
  Then the per-call value wins over agent frontmatter, which wins over config
```

Non-behavioral:

- Part A: `.pi/settings.json` pin is ≥0.47.0 and exact; `package-lock.json` consistent; `pi-run config-check` passes; a subagent smoke run works.
- Part C: a wedge contract test demonstrates a tool timeout surfaces via `PI_SELF_HEAL` observation; W5 DoD closed → CHANGELOG `[Unreleased]`; ROADMAP/STATUS updated.

## Edge Cases

- Event-pair 1:1 violation (missing `end`, crash mid-tool) → timer fires (desired).
- Output flowing while stuck (the harness wedge class) → timer still fires.
- Legit long tools (`npm test`, installs, migrations) → unaffected unless configured; per-call override available.
- Legit blocking tools (`contact_supervisor`, `intercom`) → allowlisted.
- Invalid config (`≤ 0`, `> MAX_TIMER_DELAY_MS`) → rejected with the same error shape as run-level validation.
- Parallel/fanout children, interrupts/stops → composed via the existing gates; per-tool deadline collapses with remaining run budget.
- Windows CI → timeout tests follow the existing skip convention.

## Validation Plan

- **Upstream (Part B):** `npm run typecheck`; `npm test`; `npm run test:integration` — new + existing suites green.
- **Harness (Part A/C):** `go build`/`go vet`/`go test ./...`; `pi-run config-check`; `eval/.venv/bin/python -m pytest eval/tests/test_harness_config.py` (pin test); subagent smoke; wedge contract test (mock pi: `tool_execution_start` + flowing output + no `end` → killed at budget, observed via `PI_SELF_HEAL`).
- **Meta:** close W5 DoD rows; CHANGELOG `[Unreleased]` entry; ROADMAP/STATUS reconciliation per `docs/roadmap-workflow.md`.

## Execution Plan (ordered)

1. **Part A PR (harness):** bump pin → 0.48.0; sync lockfile; config-check + smoke; land via PR (ff-only sync).
2. **File upstream issue** (per the #978 pattern): describe the per-tool wedge gap with evidence and a repro sketch; reference this spec.
3. **Part B PR (upstream repo, based on current `main`):** implement the `toolTimeoutMs` knob + timer arm/clear + termination reuse + allowlist + tests + docs + CHANGELOG; iterate with maintainer feedback.
4. **Part C (harness, after upstream release with the feature):** re-verify pin; wedge contract test proving observation; close DoD → CHANGELOG; ROADMAP/STATUS update.

## Open Questions

- Upstream naming preference: `toolTimeoutMs` vs `toolCallTimeoutMs` (maintainer call).
- Whether the maintainer wants a default-on subset (e.g. mutating/`bash` tools only) — research recommends no; defer to maintainer.
- Whether to add a dedicated event-file signal (`subagent.step.tool_timed_out`) beyond `timedOut: true` — optional; research says existing result shape is sufficient for the harness; defer.

## Self-Check

Goal user-visible ✅ · problem stated ✅ · context from memory present ✅ · in/out scope non-conflicting ✅ · constraints actionable ✅ · assumptions visible, none secretly blocking ✅ · blocking questions none (Phase-1 decisions locked) ✅ · acceptance criteria observable/testable (Gherkin + bullets) ✅ · edge cases cover invalid config, missing events, state conflicts, legit blockers ✅ · validation plan concrete ✅ · execution order matches user-locked Part A-before-B ✅
