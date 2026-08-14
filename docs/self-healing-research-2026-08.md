# Self-Healing Coding Agents — Research Synthesis

> **Superseded (2026-08-14):** research input; the self-healing design shipped
> as W1 (v0.9.1, PRs #59–#63) and per-tool timeout is W5 (Part B merged
> upstream `a660ea3`). Figures quoted here (pi-subagents 0.45.1 unpinned,
> direct-child-only kill) are pre-implementation. See
> `docs/governance/specs-archive/2026-08-13-self-healing-design.md` and ROADMAP W1/W5.

**Date:** 2026-08-13
**Status:** Research input for the self-healing design spec
**Sources:** 4 parallel research lanes (platforms, frameworks, reliability engineering, our stack); ~50 primary sources (vendor issues, PRs, docs, npm packages, engineering guides).

## TL;DR

**No shipping coding-agent platform — CLI or managed — has true built-in self-healing for hangs.** The three major CLIs (Claude Code, Codex, Gemini CLI) all have open issues documenting indefinite silent hangs with no client-side timeout or heartbeat; recovery is manual (Ctrl+C, kill, `--resume`). Managed platforms (Devin, Factory, Cursor, Copilot, Windsurf, Kiro) ship per-tool timeouts and manual cancel/retry, and respond to stuck-run complaints with bugfixes or refunds — not auto-recovery. Detection-and-restart self-healing exists **only in aftermarket wrappers** (amux, cc-resilient, subcodex-mcp, pi-non-interactive), and there is **no first-party equivalent anywhere**. This is a genuine, industry-wide gap — and it is one pi-harness can own.

## 1. The failure classes (what actually hangs)

Research converged on four distinct failure classes, each needing a different defense:

1. **Interactive-process deadlock** — a child command opens an editor/pager/credential prompt that inherits the terminal and blocks with zero output. Our exact incident (`git rebase --continue` → `vi` → 10-min silent hang). Also: non-TTY stdin hangs (pi #2078), `/dev/stdin` reads (Claude Code #16306), `tail -f | claude -p` EOF waits (#34455), `git` via `less` pager (Gemini CLI #13590). **Prevention at the environment boundary is the right defense** (`GIT_EDITOR=true`, `GIT_SEQUENCE_EDITOR=true`, `GIT_TERMINAL_PROMPT=0`, `PAGER=cat`, `isatty` guards) — detection arrives after the process is already unrecoverable.
2. **Streaming/API stall** — the provider stream or tool execute promise never resolves. Fixed upstream in pi-agent-core (#5778: `streamTimeoutMs`/`toolTimeoutMs` Promise-races), but post-completion non-exit (#5944) and provider idle (#3020) remain open.
3. **Wedged tool call** — a tool (bash, MCP, browser) executes forever; the agent loop waits on `status=running` indefinitely. The canonical fix (opencode PR #36869): **per-tool execution timeout that aborts the call and synthesizes a tool-result** so the agent *continues* instead of wedging. Blocked on the orphaned-`tool_use` session-corruption bug (#21326/#22001) unless the synthesized result is injected correctly. In pi-subagents this is open issue #150 (tool_call_timeout_ms).
4. **Stuck loop / no progress** — the agent repeats the same action with no progress. OpenHands StuckDetector (5 patterns: same-action→same-observation 4×, same-action→same-error 3×, monologue 3×, alternating pairs 6×, context-window errors); LangSmith LoopDetectionMiddleware (N edits to same file → factual nudge); AutoGPT two-layer circuit breaker (3 identical tool failures → stop; 6 empty calls → abort stream). **False positives are the recurring killer**: OpenHands #5355 and AutoGPT #12892 both killed legitimate long waits / focused iteration and required tuning. Graduated response (inform → constrain → escalate) beats hard abort.

## 2. What platforms actually ship

| Platform | Hang detection | Timeout | Recovery |
|---|---|---|---|
| Claude Code | ❌ none (open #25979, #33949) | hooks only (600s) | manual `--resume`; `CLAUDE_CODE_PROCESS_WRAPPER` escape hatch |
| Codex CLI | ❌ none (#16649, #14048) | ~366s shell timeout but **kills wrong process** (#4337 — no process-group kill, orphans hold pipes open) | manual; `codex resume` |
| Gemini CLI | ❌ none (#24707) | hardcoded 5-min shell timeout | manual |
| Devin | ❌ none | per-tool | refunds, bugfixes (Dec-24 post) |
| Factory | ❌ none | per-tool | "Mission Control" = *humans* monitor/unblock (#889: "Retrying does not recover") |
| Cursor bg agents | ❌ none | 900+s tool timeout | manual shorter prompts |
| Copilot agent | ❌ none (#293987, #274094) | MCP >5min waits | reload VS Code |
| Windsurf | ❌ none | per-tool | bugfixes |
| Kiro | ❌ none (#8279) | broken `api.timeout` config (#7487) | manual `kirocrew doctor` |

**Wrappers that DO self-heal** (all aftermarket): amux (tmux multiplexer, ANSI-parses output → classify working/stuck → auto-restart), cc-resilient (5s API ping + no-output stall detection → kill+restart), subcodex-mcp (stall detection + auto-recovery for Codex subagents), claude-code-resilient-wrapper (expect-based, auto-"continue"), pi-non-interactive (env injection — prevention only).

## 3. Reliability-engineering principles (the "how to build it right")

From the Agent Reliability Engineering Design Guide (hidekazu-konishi.com) + AWS Builders' Library + OTel GenAI semconv:

- **Three termination paths, in three currencies**: goal satisfied, budget exhausted (steps / tokens / wall-clock each bound a different failure mode), guard fired. A missing wall-clock budget = unbounded hang by definition.
- **Three distinct no-progress detectors**: exact repetition (fingerprint tool+canonicalized args), stagnation (harness-owned transcript-length growth, never the agent's self-report), cycling (repeated state hash). Fingerprint the result too: "same call, same error" is the strong signal.
- **Graduated response ladder, not abort**: inform (inject factual observation) → constrain (remove the action) → escalate. Most repetitions resolve at rung 1.
- **Timeouts compose**: propagate an absolute deadline, clamp every inner layer, reserve wind-down time. `steps × retries × timeout` multiplies to hours of worst case.
- **Enforced ceiling above advertised budget**: tell the model the budget so it winds down gracefully.
- **Restart with backoff + budget**: exponential backoff with jitter; per-task restart cap or a failing agent self-DOSes. Retries must decrement the shared budget.
- **Escalate with a handoff packet**: after bounded retries, hand the human the original goal, side-effect ledger, pending state, trigger evidence, and a resume handle — not a transcript.
- **Observability that enables healing**: OTel `gen_ai.operation.name=execute_tool/invoke_agent` spans, per-step guard-state record, append-only side-effect ledger written *before* the effect. "Which guard should have caught this?" is the postmortem question.

## 4. What pi-harness already controls (the build surface)

**Upstream pi/pi-subagents state:**
- pi-subagents **0.45.1 installed, unpinned** (`npm:pi-subagents` in `.pi/settings.json`). Run-level timeouts exist (foreground/async `timeoutMs`, SIGTERM→SIGKILL escalation for direct children, SIGUSR2 interrupt of detached runners, recursive descendant interrupt). **No per-tool-call timeout** (tintinweb/pi-subagents #150 open; T50-supervisor confirms the gap). Unreleased #1030 terminates async writers as POSIX process groups — the strongest upstream signal.
- pi-agent-core: in-loop stream/tool waits bounded (#5778 closed). Non-TTY stdin guard shipped (#2078 closed). Post-completion non-exit (#5944) and provider idle (#3020) still open.

**Our Go CLI (`pi-run`):**
- `execPi`/`execPiDir` — no timeout (chat/print/resume).
- `execPiDirTimeout` — `context.WithTimeout` + `exec.CommandContext` → **kills only the direct pi child**, not the process group; grandchildren survive. This is the #4337-class bug in our own stack.
- `launchEnv` — **already ships the non-interactive env** (GIT_EDITOR=true etc., merged PR #59): the *prevention* layer is done.
- `benchmark.go` `agentTimeout` (default 300s) — the only bound in the CLI.
- All spawn paths use `cmd.Stdin = os.Stdin` — pi and its bash tools share the harness TTY (the editor-inheritance root cause, now mitigated by env).

**Git-state recovery primitives (easy to compose, no turnkey library):**
- Detect: `git rev-parse --git-path rebase-merge` (worktree-safe), `REBASE_HEAD`, `git ls-files -u` (conflict state).
- Continue without editor: `GIT_EDITOR=true git rebase --continue` / `git -c core.editor=true rebase --continue`. Roo-Code PR #7819 sets GIT_EDITOR=true for all rebase commands (direct prior art).
- Safety rule: auto-continue only when `git ls-files -u` is empty; else `--abort` (destructive) or report — never guess conflict resolutions.

## 5. The gap, precisely

Everything that hangs in our stack that isn't already fixed:
1. **Mid-tool wedges are only caught by the run-level deadline** (kills the whole child, forfeits recovery) — no per-tool timeout in pi-subagents (#150).
2. **Our own CLI kill is direct-child-only** — a timed-out benchmark leaves orphaned bash/git children holding pipes (Codex #4337 class).
3. **No no-progress/stall detector anywhere** — silence is indistinguishable from progress until the wall-clock expires.
4. **No git-state recovery** — a wedged rebase stays wedged after the kill; the next run starts on a dirty repo.
5. **No escalation packet** — a timed-out run produces no evidence/handoff for the operator.

A self-healing layer in `pi-run` (process-group kill + output-stall detection + git-state scan + recovery report) fixes 2–5 with **zero upstream changes**; 1 is the one piece that really belongs upstream (#150).

## Sources

Lane 1 — platforms: Claude Code #25979/#33949/#2183/#26729 + troubleshooting/hooks docs; Codex #4337/#16649/#14048 + subagents docs; Gemini CLI #24707/#13590/#4322/#3375; Cognition Dec-24 blog; Factory docs + #889; Cursor forum; Copilot/vscode #293987/#274094/#259677; Windsurf #233/#261; Kiro docs + #8279/#7487; amux, cc-resilient, subcodex-mcp, pi-non-interactive, claude-code-resilient-wrapper.

Lane 2 — frameworks: OpenHands StuckDetector docs + stuck_detector.py + #5355/#10350/#11799/#5500; opencode #36869/#21326/#22001; pydantic-ai retries/messages docs; Notifly tool-timeout/error/cost-runaway/agent-stuck; LangGraph recursionLimit + checkpointers/time-travel + #7361; agentpatterns.ai loop detection + LangChain harness blog; CrewAI docs + #3847/#4126; AutoGPT #12499/#12636/#12892/#11547/#12700; smolagents reference + #2560/#2566.

Lane 3 — reliability: Agent Reliability Engineering Design Guide (hidekazu-konishi.com); AWS Builders' Library (timeouts/retries/backoff+jitter); pi #2078/#2645/#5571 + multica #3118; Claude Code #69804/#16306/#34455; OTel semantic-conventions-genai gen-ai-spans; langchain #36139; Geodocs health-check spec; Zylos process supervision; Multigrid stopping-conditions/on-call runbooks; Solana Garden retry/backoff.

Lane 4 — stack: nicobailon/pi-subagents CHANGELOG + v0.32.0 release + #978/#979; tintinweb #150; earendil-works/pi #5778/#5944/#2078/#3020; T50-Systems/pi-subagent-supervisor; local `internal/cli/pi.go`, `benchmark.go`, `app.go`; pre-commit check_merge_conflict.py; is-rebase-or-merge npm; git-for-windows rebase-branch.sh; GitKraken merge-mate-cli; Roo-Code #7819; git-rebase docs.

## Gaps

- Factory/Cursor/Copilot internal stuck-run detection mechanics are closed-source (inferred from docs/issues/forums).
- opencode #36869 runtime behavior verified only from PR text.
- pi-subagents pinned version unpinned in settings.json — should pin to a known-good ≥0.45.1 with the #1030-era fixes when released.
- No public hang-rate/SLA data from any vendor; restart-budget values must be chosen empirically from our own success distribution.
- Post-prevention incident rate of the vi-hang class is unmeasured — suggests a `--self-heal` observability flag before full auto-recovery.
