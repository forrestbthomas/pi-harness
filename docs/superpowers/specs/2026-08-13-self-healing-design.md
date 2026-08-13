# Self-Healing Agent Runs (watchdog, process-group kill, git-state recovery, escalation) — Design

**Date:** 2026-08-13
**Status:** Proposed (from self-healing research brief `docs/self-healing-research-2026-08.md`, lanes 1–4)
**Target release:** v0.9.1 (after #50–#58 fixes ship)
**Depends on:** PR #59 (non-interactive env — the prevention layer, merged into this plan's baseline); ROADMAP W1 / BACKLOG items 1–3.
**Skill workflow:** spec-plan (this doc), then scope-lock before implementation, spec-crlp on drift, spec-index for durable memory.

## Goal

After this work, a hung, wedged, or silently-stalled `pi-run` agent run (chat/print/benchmark/subagent) is detected automatically, terminated with its whole process tree, and either recovered (git state healed, run resumed) or escalated with an evidence packet — without requiring the user to notice, nudge, or manually unstick it.

## Problem Statement

The 2026-08-12 incident: a subagent's `bash` tool ran `git rebase --continue` with no `GIT_EDITOR` set; git spawned `vi`, inherited the terminal, and blocked with zero output for 10 minutes until the user manually nudged. Research (`docs/self-healing-research-2026-08.md`) confirms this is a known industry-wide class: Claude Code #69804/#28482/#44783, Codex #4337, Gemini CLI #13590 all document indefinite silent hangs with **no first-party self-healing** anywhere — detection/recovery exists only in aftermarket wrappers (amux, cc-resilient, subcodex-mcp).

Our stack has four concrete gaps the prevention env (PR #59) does **not** close:
1. **Direct-child-only kill** — `execPiDirTimeout` uses `exec.CommandContext` → kills only the pi process; grandchildren (bash tools, git, detached pi-subagents runners) survive a timeout (the Codex #4337 class).
2. **No stall detector** — silence is indistinguishable from progress until the fixed wall-clock expires; chat/print/resume have no timeout at all.
3. **No git-state recovery** — a wedged rebase stays wedged after a kill; the next run starts on a dirty repo.
4. **No escalation packet** — a timed-out run produces no evidence/handoff for the operator.

## Context From Memory

- **No relevant memory found** in spec-index (fresh knowledge base, `docs/knowledge-base/` empty). Durable context below comes from the research synthesis and context-engine notes (wd_harness):
- **Research lanes (2026-08-13):** platforms (no first-party self-healing anywhere), frameworks (opencode #36869 per-tool timeout + synthesized result; OpenHands StuckDetector 5 patterns with false-positive risk #5355/#12892; AutoGPT circuit breaker), reliability (three budgets in three currencies; no-progress = harness-owned progress predicate, never agent self-report; graduated response ladder inform→constrain→escalate; backoff+jitter+restart budget; escalate with handoff packet), stack (pi-subagents #150 tool-call timeout open; pi #5778 in-loop bounded, #5944 post-exit hang open; our `launchEnv` now ships prevention env).
- **Installed pi-subagents:** 0.45.1, **unpinned** in `.pi/settings.json` (`npm:pi-subagents`).
- **Local reality:** `/tmp/pi-run` on PATH was stale (pre-#53) and passed the config probe while serving old model-tier values — local staleness is a recurring trap; CI builds fresh.
- **Remotes:** `github` is real (PRs #30–#60); `origin` (gitlab) is a dormant mirror whose default branch auto-set to a pushed branch — harmless, undeletable via push.

## In Scope

- **Watchdog command `pi-run self-heal`** (new CLI surface): detect in-progress git state and recover it; report state machine (ok / recovered / needs-attention / aborted).
- **Process-group kill** for timed-out runs: replace `exec.CommandContext` direct-child kill in `execPiDirTimeout` with `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pgid, SIGTERM)` → SIGKILL escalation (default grace 10s), so grandchildren die with the parent. `exec.CommandContext` must be **fully dropped**, not kept alongside — its deadline watcher would race the group-kill and truncate the grace window. Guard `ESRCH`/pgid-reuse when the leader exits between deadline and `kill(-pgid)`.
- **Output-activity stall detector** for non-interactive spawns (print/benchmark): goroutine consuming child stdout resets a timer on any byte; after N silent minutes → SIGTERM → SIGKILL the group; gated OFF for interactive chat (human thinking is not a hang). Requires teeing child stdout (`io.MultiWriter(os.Stdout, pipe)`) since spawns currently bind `cmd.Stdout = os.Stdout`.
- **Watchdog-terminated exit code:** new exit code `9` = "watchdog terminated" (SIGTERM/SIGKILL by watchdog, stall, or group-kill timeout), distinct from generic `1`; documented in the usage exit-code table.
- **Escalation packet:** when a run is killed by watchdog/timeout, write `.pi/heal/<timestamp>-report.json` (original goal, side-effect ledger / git status+diff summary, pending state, trigger evidence (last-output timestamp, bytes), resume handle) **and** print a short human-readable stderr summary. For `--no-session` runs the resume handle is the explicit string `none (session not persisted)` — never a fabricated session reference.
- **Git-state auto-recovery:** scan for `.git/rebase-merge`/`rebase-apply` via `git rev-parse --git-path`; if in-progress rebase and `git ls-files -u` empty → `GIT_EDITOR=true git rebase --continue`; if conflicts → record, do not guess; recovery recorded in the run report.
- **`--self-heal` observability flag:** log stall/git-state events; post-prevention incident rate metric surfaced in scorecard (BACKLOG item 3).
- **Hermetic tests** for all of the above (fake pi that hangs; fixture repo with rebase state; fake collector for events).

## Out Of Scope

- **Per-tool-call timeout inside pi** (tintinweb/pi-subagents #150) — an upstream contribution (BACKLOG item 1), separately tracked; our watchdog bounds the run, not the in-loop tool.
- **Loop/stuck-pattern detection** (OpenHands StuckDetector) — high false-positive risk (#5355/#12892), requires empirical thresholds from our own success distribution; deferred.
- **Upstream pi fixes** (#5944 post-exit hang, #3020 provider idle) — observed by the watchdog, not fixed here.
- **Auto-abort of conflicted rebases** — `--abort` is destructive; only reported + explicit flag, never automatic.
- **Windows support** (Go portable but process-group semantics differ; Unix-first).
- **Auto-resume of the agent run itself** — the watchdog recovers *state* and gives a resume handle; re-driving the agent automatically is a follow-up decision.

## Constraints

- **Go 1.26 stdlib-only** — no external deps; `os/exec` + `syscall` on Unix; build-tag guard for non-Unix.
- **Never log or echo secret values** — env-var names only; keys env-first then `PI_SECRET_BACKEND`.
- **Hermetic tests** — no hardcoded user paths, no ambient credentials, no `test-key`-shaped literals (use `testvalue`).
- **Project `.pi/settings.json` stays hermetic** — no machine-specific paths (the skills script registers only in global settings).
- **Non-interactive spawns only** — stall detector and auto-kill must never fire during interactive chat; chat keeps manual control.
- **Escalation is bounded** — watchdog auto-recovers at most N times per run (configurable), then reports; it must not self-DoS downstream APIs.
- **Backwards compatible** — existing flags/commands unchanged; new behavior opt-in where it could surprise (chat).

## Assumptions

- The vi-hang class is now prevented by PR #59's env; the watchdog is for the residual classes (streaming stall, wedged tool, post-exit hang, dirty git state from an interrupted run).
- A process that produces **no stdout bytes for the configured window** is stuck (not thinking) in print/benchmark modes — matches research guidance (progress = harness-owned observable, not agent self-report).
- `git ls-files -u` empty is a safe precondition for `rebase --continue`; we never guess conflict resolutions.
- Default silent-window: 300s (matches existing `defaultBenchmarkTimeout`); configurable via env `PI_STALL_TIMEOUT_SECS` and flag. Note: at the default, the stall window equals the benchmark run bound, so the stall detector is decisive for `print` (no wall clock) and tasks with `timeoutSecs > 300`; for default 300s benchmark tasks the wall clock fires first.
- Default SIGTERM→SIGKILL grace window: 10s.
- Default restart budget: 1 auto-recovery attempt per run; escalation after that.
- Escalation packet destination: `.pi/heal/<timestamp>-report.json` (relative to run cwd) + stderr summary; watchdog-kill exit code is `9`.
- Resume handle: real session id when persisted (budget cap active); explicit `none (session not persisted)` otherwise.

## Blocking Questions

None — the design lands entirely inside pi-run's own spawn surface with no upstream dependency and no user-visible behavior change outside watchdog paths.

## Acceptance Criteria

```gherkin
Scenario: Timed-out run kills the whole process tree
  Given a benchmark task whose pi process spawns a long-lived grandchild
  When the agent timeout expires
  Then the process group is SIGTERM'd
  And SIGKILL escalation follows after the grace window
  And no grandchild process survives (verified via pgid in test)

Scenario: Silent run is detected by output stall
  Given a print-mode run that emits no stdout for the configured window
  When the stall timer expires
  Then the process group is terminated
  And the run report records stall trigger evidence (last-output timestamp, silent seconds)

Scenario: Interactive chat is never auto-killed
  Given a chat session with no output for longer than the window
  When the watchdog runs
  Then no kill occurs (chat is excluded from stall detection)

Scenario: Wedged rebase is auto-recovered
  Given a repo with an in-progress rebase and no unmerged paths
  When `pi-run self-heal` runs
  Then it runs `GIT_EDITOR=true git rebase --continue`
  And reports status "recovered"

Scenario: Conflicted rebase is reported, not guessed
  Given a repo with an in-progress rebase and unmerged paths
  When `pi-run self-heal` runs
  Then it does not modify git state
  And reports status "needs-attention" with conflict paths

Scenario: Killed run produces an escalation packet
  Given a run terminated by the watchdog or timeout
  When the run exits
  Then `.pi/heal/<timestamp>-report.json` exists with original goal, side-effect ledger, pending state, trigger evidence, and a resume handle
  And the process exit code is 9 (watchdog terminated)
  And stderr contains a short human-readable summary

Scenario: Killed non-session run reports an honest resume handle
  Given a print-mode run launched with --no-session that is killed by the watchdog
  When the escalation packet is written
  Then the resume handle is the literal string "none (session not persisted)"
```

Non-behavioral criteria:
- `go build ./...`, `go vet ./...`, `go test ./...` green; hermetic pytest contract tests green.
- No new external Go deps.
- `pi-run self-heal --help` documented in usage; exit-code table gains `9` = watchdog terminated.
- Watchdog events (stall, group-kill, git-recovery, escalation) emitted under `--self-heal` observability flag and surfaced in scorecard.
- Merge-order: `docs/self-healing-research-2026-08.md` lives on `feat/pm-layer` (PR #60); the spec branch must not block on it, but the citation must resolve before the W1 implementation PR merges (pm-layer merges first, or the file is vendored into the implementation branch).

## Edge Cases

- **Grandchild already detached** (`setsid` by pi-subagents runners): pgid kill may not catch; report partial-kill evidence, do not claim full success.
- **Stall timer vs slow-but-progressing run:** timer resets on ANY byte; a run emitting occasional output is never killed (graduated, not fixed).
- **Concurrent runs in same repo:** git-state scan is repo-global; recovery must be idempotent and refuse if another run is active (lock file or `git rebase` refusal).
- **`--abort` path:** only via explicit `pi-run self-heal --abort` flag; never automatic; logs the destructive choice.
- **Non-Unix build:** process-group kill code guarded by build tag; command still builds with a clear "not supported on this platform" error.
- **Stale binary on PATH** (the `/tmp/pi-run` trap): watchdog tests use `PI_RUN_BIN` pointing at a fresh build; CI builds fresh (already enforced by python-contract job).
- **Missing git repo:** `self-heal` with no `.git` reports "no git state" cleanly (exit 0).

## Validation Plan

- **Go unit tests:** pgid kill kills grandchildren (spawn `sh -c 'sleep 100 & wait'` in test, assert group reaped); stall detector fires on silence and resets on bytes (fake io pipe); self-heal continues/aborts on fixture repos; escalation packet shape.
- **Hermetic pytest contract tests:** `pi-run self-heal` argv/exit codes against fake node/pi (existing `conftest` pattern); `--self-heal` events emitted to fake collector (existing OTel fake-collector pattern).
- **Manual smoke:** create a real conflicted rebase in a throwaway repo; run `pi-run self-heal`; observe report; `--abort` path.
- **CI:** deterministic job runs the new tests; nightly live job gets `--self-heal` observability flag on (no behavior change, metrics only).

## Execution Plan

1. **CLI surface** — add `self-heal` command (usage, exit codes, `--abort` flag) in `internal/cli/`; wire into app command list.
2. **Process-group kill** — add `SysProcAttr{Setpgid:true}` + group SIGTERM→SIGKILL in `execPiDirTimeout`; build-tag guard; unit test with grandchild.
3. **Stall detector** — stdout-consuming goroutine with byte-reset timer; only for print/benchmark non-interactive spawns; env/flag `PI_STALL_TIMEOUT_SECS`; unit test (silence vs bytes).
4. **Git-state recovery** — `pi-run self-heal` git-state scan (`rev-parse --git-path rebase-merge`/`rebase-apply`, `git ls-files -u`); continue when clean, report when conflicted; idempotent; unit test on fixture repos.
5. **Escalation packet** — structured report writer (goal, side-effect ledger, pending state, trigger evidence, resume handle) shared by watchdog/timeout paths; JSON + human-readable.
6. **`--self-heal` observability** — event logging (stall/group-kill/git-recovery/escalation) + scorecard surfacing.
7. **Hermetic tests + CI** — Go tests, pytest contract tests, CI wiring; document in `--help`/usage.
8. **Docs + release** — update README/usage/CHANGELOG; land in v0.9.1.

## Open Questions

- Should the watchdog auto-resume the agent run (re-drive `pi-run resume`) after a clean git-state recovery? Deferred — recovery of state now, resume handle provided; auto-resume is a follow-up decision (BACKLOG).
- Restart-budget values (silent window 300s, 1 auto-recovery) need empirical tuning from our own success distribution once `--self-heal` observability accumulates data.

## Self-Check

- Status: Ready for implementation (pending PR #59/#60 review)
- Blocking Questions: None
- Safe To Implement: Yes
- Notes: Follows spec-plan template; scope-lock should generate SCOPE.md from this spec before implementation starts; drift → spec-crlp; durable decisions → spec-index (`docs/knowledge-base/`).
