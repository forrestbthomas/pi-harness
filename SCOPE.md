# Scope Contract
**Task:** Self-healing agent runs — W1 | **Plan:** docs/superpowers/specs/2026-08-13-self-healing-design.md | **Date:** 2026-08-13 | **Status:** CLOSED — 2 changes logged

## In Scope
- **Files:**
  - `internal/cli/pi.go` — process-group kill (drop `exec.CommandContext`, add `SysProcAttr{Setpgid:true}` + group SIGTERM→SIGKILL, ESRCH guard); stall-detector tee of child stdout; escalation packet write; `--self-heal` event logging
  - `internal/cli/app.go` — register `self-heal` command; usage + exit-code table gains `9` = watchdog terminated
  - `internal/cli/benchmark.go` — wire stall detector into the agent run path (only where it does not change existing behavior)
  - `internal/cli/` (new) — `self_heal.go` (git-state scan/recover), `watchdog.go` (stall timer), `escalation.go` (`.pi/heal/<timestamp>-report.json` + stderr summary), `exitcodes`/usage constants
  - `eval/tests/` (new) — contract tests: `test_self_heal.py` (argv/exit codes against fake pi), `test_watchdog.py` (stall fires on silence, resets on bytes), `test_escalation.py` (packet shape, exit 9, resume-handle literal)
  - `docs/` — README/usage/CHANGELOG entries; spec stays source of truth
- **Features:**
  1. `pi-run self-heal` command: detect `.git/rebase-merge`/`rebase-apply` via `git rev-parse --git-path`; if in-progress rebase and `git ls-files -u` empty → `GIT_EDITOR=true git rebase --continue`; else report `needs-attention` with conflict paths; `--abort` flag only, never automatic; idempotent (refuse if another run active)
  2. Process-group kill on timeout: SIGTERM → 10s grace → SIGKILL; reaps grandchildren
  3. Output-activity stall detector: non-interactive spawns only (print/benchmark); reset on any byte; `PI_STALL_TIMEOUT_SECS` env/flag (default 300); chat excluded
  4. Exit code `9` on watchdog termination (stall, group-kill timeout)
  5. Escalation packet: `.pi/heal/<timestamp>-report.json` + stderr summary; resume handle = session id when persisted, literal `none (session not persisted)` otherwise
  6. `--self-heal` observability flag: log stall/group-kill/git-recovery/escalation events; scorecard surfacing
- **Boundaries:**
  - Only watchdog paths change behavior (non-interactive spawns, timeout paths); interactive chat unchanged
  - Hermetic: no new external Go deps; no hardcoded user paths; project `.pi/settings.json` untouched
  - Tests use `PI_RUN_BIN` fresh build; fixture repos under test tmp dirs

## Out of Scope
- Per-tool-call timeout inside pi (upstream pi-subagents #150) — BACKLOG #1, separate PR
- Loop/stuck-pattern detection (OpenHands StuckDetector) — deferred (false-positive risk)
- Upstream pi fixes (#5944 post-exit hang, #3020 provider idle) — observed only
- Auto-abort of conflicted rebases — `--abort` is explicit-flag only
- Auto-resume of the agent run after recovery — resume handle provided, re-drive deferred
- Windows support — process-group kill build-tagged for Unix
- The `docs/knowledge-base/` entries and research doc (already landed on other branches)
- The stale checked-in `bin/pi-run` binary (pre-existing staleness; not a W1 defect to fix here)

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | ambiguity | Feature 6 `--self-heal` observability flag implemented as env `PI_SELF_HEAL=1`, not a CLI flag; scorecard surfacing deferred | Env gating lets CI set it for nightly without touching deterministic behavior; a CLI flag would thread through spawn for no functional gain; scorecard surfacing needs `--self-heal` incident data first (BACKLOG #3) | Defer (documented in watchdog.go:53-58) | Env-gated events + follow-up for flag/scorecard |
| 2 | emergent | Hermetic contract tests promised in SCOPE.md (`test_self_heal.py`, `test_watchdog.py`, `test_escalation.py`) delivered as Go unit tests + `test_contract_exit_codes.py` pin instead | Go tests cover the same behavior (stall/reset, group-kill reap, packet shape, exit 9) with stronger process-level assertions; a duplicate pytest layer adds maintenance without new coverage | Permit (Go tests supersede) | Add thin pytest contract smoke for `self-heal` argv/exit codes if CI needs it; README/CHANGELOG entries added |

# Follow-up Tasks
- [ ] After W1 merges: upstream per-tool timeout contribution (BACKLOG #1) — scope change #0
- [ ] After W1 ships + `--self-heal` data accumulates: tune silent-window/restart-budget empirically — scope change #0
