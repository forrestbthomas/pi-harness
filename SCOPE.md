# Scope Contract
**Task:** W5 — Upstream per-tool-call timeout | **Plan:** `docs/superpowers/specs/2026-08-13-per-tool-call-timeout-upstream.md` | **Date:** 2026-08-13 | **Status:** ACTIVE (approved 2026-08-13) — Part A in progress

> Supersedes the W1 self-healing contract (CLOSED 2026-08-13; record preserved in git history).

## In Scope
- **Files — harness (`pi-harness`):**
  - `.pi/settings.json` — pin `npm:pi-subagents@0.45.1` → `0.48.0` (Part A)
  - `.pi/npm/package.json` + `.pi/npm/package-lock.json` — sync to the pinned version (Part A)
  - `eval/tests/test_harness_config.py` — pin test must keep passing (Part A)
  - `eval/tests/` — new wedge-observation contract test (Part C)
  - `BACKLOG.md`, `ROADMAP.md`, `STATUS.md`, `CHANGELOG.md` — DoD closure / reconciliation
- **Files — upstream (`nicobailon/pi-subagents`, Part B):**
  - `src/extension/schemas.ts` — optional per-call `toolTimeoutMs` alongside `timeoutMs`
  - `src/agents/agents.ts`, `src/agents/agent-serializer.ts` — agent frontmatter `toolTimeoutMs`
  - `src/extension/config.ts`, `src/shared/types.ts` — `config.toolTimeoutMs` + `PI_SUBAGENT_TOOL_TIMEOUT_MS` env
  - `src/runs/background/subagent-runner.ts`, `src/runs/shared/async-execution.ts` — async per-tool timer (arm/clear, termination reuse)
  - `src/runs/foreground/execution.ts`, `src/runs/foreground/subagent-executor.ts` — foreground per-tool timer + validation
  - `test/unit/`, `test/integration/` — new + extended timeout tests
  - `docs/configuration.md`, `docs/agents.md`, `docs/tool-reference.md`, `CHANGELOG.md`
- **Features:**
  1. **Part A (harness PR, lands first):** upgrade pin to 0.48.0; sync lockfile; `config-check` + subagent smoke green.
  2. **Part B (upstream issue + PR):** opt-in `toolTimeoutMs` knob threaded through the run-timeout ladder (call > agent > config > env); per-tool timer armed on `tool_execution_start`, cleared on `tool_execution_end`; expiry reuses the existing `registerTimeout` SIGTERM→SIGKILL path with `timedOut: true` + tool-specific error; deadline collapses with remaining run budget (`min`); supervisor allowlist (`contact_supervisor`, `intercom`); validation mirrors `resolveConfigDefaultTimeoutMs` (rejects ≤0 and > `MAX_TIMER_DELAY_MS`); tests (unit resolver, async + foreground integration incl. "output flowing, no end event", run-budget-wins, no-spurious-kill, Windows skip convention); docs + CHANGELOG.
  3. **Part C (harness, after upstream release with the feature):** re-verify pin; wedge contract test proves `PI_SELF_HEAL` observation; close W5 DoD.
- **Boundaries:**
  - Kill-run semantics (`timedOut: true`); no soft-interrupt/retry in v1.
  - Opt-in only; no built-in default per-tool timeout.
  - `contact_supervisor`/`intercom` exempt from the per-tool timer.
  - No changes to the child's own `bash` tool timeout behavior.
  - Harness changes limited to pin upgrade + observation; no watchdog changes.
  - Upstream PR based on current `main`; composes with #979 machinery; honors `settled || timedOut || stopped` gates.

## Out of Scope
- Soft-interrupt-and-retry semantics.
- A built-in default per-tool timeout.
- Changing the child `bash` tool's own timeout behavior upstream.
- Harness watchdog changes (W1 shipped).
- Scorecard surfacing of `PI_SELF_HEAL` events (separate backlog item).
- Windows-specific fixes beyond honoring the existing test-skip convention.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] — (none yet)
