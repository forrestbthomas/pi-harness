# Competitive Gap Analysis: AI Coding-Agent Harnesses (2026 Q2/3)

**Date:** 2026-08-11
**Revised:** 2026-08-11 — after v0.5.0 shipped (`pi-run eval --benchmark` + `pi-run cost`) and after discovering Pi-native competitors (Pi ModelBench, pi-bench, Agent Plugins 1.0.0) — see §1.5.
**Purpose:** High-signal comparison of public AI coding-agent harnesses vs. pi-harness — what they have that we don't — to feed both **roadmap** (next features) and **positioning** (README/marketing/launch emphasis).
**Method:** Web research across the Q2 2026 CLI-agent landscape (wal.sh comparison verified against official docs), eval-harness projects (SWE-bench, OpenHands, terminal-bench, aider), and platform agents (Claude Code, Codex CLI, OpenCode, Goose, Gemini CLI, Copilot CLI).

> **Superseded (2026-08-13):** written 2026-08-11; several present-tense "gaps"
> shipped in v0.8.0/v0.9.x (`mcp-server`, `--permission-mode`, hooks,
> `project-understand`, `--model-tier`). `pi-run mcp-server` was subsequently
> **removed in v0.10.0** (cut list #100; see CHANGELOG), and the provider
> count is now 17 with 8 benchmark tasks. Retained as a dated research record;
> see `ROADMAP.md` for current workstreams.

---

## 1. Where pi-harness sits

**What we are:** a provider-agnostic **harness + evaluation suite** around the Pi coding agent — a single Go CLI (`pi-run`) that owns provider routing (7 providers), secret resolution, launching Pi, and a DeepEval pytest suite.

**What we uniquely have (our differentiators):**
- **Provider-agnostic at the harness layer**: 7 providers (openai, openrouter, deepseek, anthropic, gemini, groq, local) with a data-driven `providers.json`, no cross-provider fallback — explicit, reproducible routing. OpenCode is the only comparable with broader breadth (75+ via Models.dev).
- **Built-in, no-key-required evaluation**: `pi-run eval` with DeepEval + pytest, deterministic smoke subset that runs with **zero keys** — most agents have no bundled eval at all.
- **Secret-manager pluggability**: Bitwarden/1Password/env-only via `PI_SECRET_BACKEND` — most CLI agents only read env vars.
- **Zero-external-deps Go stdlib-only CLI**: single compiled binary, Homebrew tap, cross-platform release — unusual for this category (most are Node/Python apps).
- **Docker-isolated benchmark runner (shipped v0.5.0)**: `pi-run eval --benchmark` runs the same task suite in isolated containers against any provider — scored pass/fail per task, JSON reports, 5 seed tasks, hermetic `--benchmark-dry-run` (no Docker/keys), exit 7 when Docker is unavailable.
- **Cost tracking + budget cap (shipped v0.5.0)**: `pi-run cost` aggregates real spend from Pi session files (`usage.cost`, no price tables) per provider/model, and `pi-run chat|print --max-budget-usd` refuses to launch past the cap with a dedicated exit code 6.

---

## 1.5 New development: Pi-native competitors arrived

Since the Q2 2026 research above, the benchmark/cost "space" gained **Pi-native competitors** — but they are **interactive extensions, not CLI/CI tools**:

| Project | What it is | What it does | How it differs from us |
|---|---|---|---|
| **Pi ModelBench** (github.com/Deca/Pi-ModelBench) | Pi extension, `/benchmark` slash command | Deterministic model comparison with profiles (coding, coding-agent, simplebench, reasoning, structured-data, customer-support, writing); per-model quality/latency/tokens/cost; `--runs` for variability; HTML/JSON reports | The **coding-agent profile is nearly identical to our seed benchmark tasks** — but it runs **inside a live Pi TUI session** (interactive extension, not a CLI) |
| **pi-bench** (github.com/fornace/pi-bench) | `/bench` slash command + standalone script | Probes every model with real streaming calls, ranks by latency/cost/quality, feeds model selection into pi-recap | Interactive/extension-oriented; no CI contract, no budget gate |
| **Agent Plugins 1.0.0** (agent-plugins.org) | Vendor-neutral packaging spec | Standard for packaging skills + MCP servers (Amazon/Cursor/Microsoft/OpenAI/Vercel/Google) | Packaging/interop standard, not a benchmark tool — relevant to our plugin story, not to eval |

**Implication:** model comparison is now a crowded *interactive* space. Our edge is making it a **repeatable, budgeted, gated CI artifact** — the CLI + Docker + cost + budget pieces we already ship are exactly what the extensions lack (see §5 new P0).

---

## 2. What others have that we don't (grouped by theme)

### A. Eval / benchmarking depth (the biggest gap)

| Project | Feature | What we lack |
|---|---|---|
| **SWE-bench** | Docker-isolated per-repo environments (layered base/env/instance images), patch application + test-run grading, `pass@1`/resolve metrics, cloud eval (Modal/AWS) | Our eval runs in the **local working tree** with no container isolation; dataset is 5 hand-written samples vs. 500+ SWE-bench tasks; no gold-patch validation |
| **terminal-bench (Harbor)** | **Continuous benchmark** with tagged releases, task-proposal rubric + automated review, Docker sandboxes, `--agent`/`--model` matrix runs, anti-cheat checks | We have no benchmark task corpus, no leaderboard, no agent-vs-agent comparison harness, no sandboxed task env |
| **aider** | 133-exercise Exercism benchmark (edit-format stress test), deterministic harness (logs request SHA hashes, strips timing), 2-attempt edit+fix loop | Our dataset is 5 samples; no edit-format benchmark; no determinism/randomness tracking |
| **OpenHands eval harness** | `user_response_fn` simulation (automated replies when agent asks), max-iteration cap, `EvalOutput` collection, parallel `run_evaluation` | Our live eval needs a real provider key; no simulated-user interaction loop; no per-instance parallel eval runner |

**Net:** our eval was a *smoke/quality gate* (runs in-tree, no keys for the deterministic subset); the community's eval harnesses are *research-grade benchmarks* (isolated, reproducible, scored, comparable). **Status:** the v0.5.0 recommendation landed — we now ship a SWE-bench-style **Docker-isolated task runner** (`pi-run eval --benchmark`, 5 seed tasks); growing the dataset and adding diff-grading are the follow-ons. It directly serves our anti-lock-in pitch ("run the same eval across any provider").

### B. Sandboxing / isolation

| Project | Feature |
|---|---|
| **Gemini CLI** | Mature OS-level sandboxing: macOS Seatbelt, gVisor, Docker/Podman, LXC, Landlock+seccomp; permissive/restrictive/strict tiers |
| **Codex CLI** | Clean tier model (read-only / workspace-write / danger-full-access); `codex sandbox` tool; filesystem separated from `--ask-for-approval` |
| **SWE-bench / terminal-bench** | Docker containers per task |

We have **no sandboxing** — Pi's bash tool runs in the working tree (with per-call `timeout:` now enforced by our agent wrappers, but no OS/network isolation). **Recommendation:** low-cost first step is a `pi-run eval --sandbox` that runs the eval suite inside Docker (matches SWE-bench), rather than sandboxing the daily-driver chat path.

### C. Permission models

| Project | Feature |
|---|---|
| **Claude Code** | 6 permission modes incl. `auto` (model-driven permission classifier) — unique; per-tool allow/deny, `/permissions`, `--dangerously-skip` |
| **Codex CLI** | 3-tier permission ladder; `--ask-for-approval untrusted/on-request/never` |
| **Gemini CLI** | 4 approval modes; plan mode |
| **OpenCode** | per-tool `ask/allow/deny` config |

We delegate permissions entirely to Pi/pi-subagents (the project's agent wrappers carry bash-timeout rules, but the *harness* has no permission surface). **Recommendation:** add a `pi-run chat --permission-mode` passthrough or a documented permission policy per agent; low effort, high perceived-safety value.

### D. Hooks / lifecycle events

| Project | Feature |
|---|---|
| **Claude Code** | 7 hook event types incl. PreCompact, conditional `if` hooks, veto capability |
| **Copilot CLI** | 6 hook types incl. `errorOccurred` (unique) |
| **Kiro** | `AgentSpawn`/`AgentStop` hooks, veto via exit code |

We have **no harness-level hooks** (pi-subagents has internal lifecycle, but nothing user-extensible from the Go CLI). **Recommendation:** `pi-run` hook config (e.g., `pre-eval`, `post-eval`, `pre-chat`) would be a differentiator for CI integration — note that `pi-run cost` already provides the pre-flight/budget gate that would sit inside such hooks.

### E. Session / memory / context tooling

| Project | Feature |
|---|---|
| **Claude Code** | Structured cross-session memory (user/feedback/project/ref categories, indexed `MEMORY.md`), `/context` visualization, `/export`, `/branch` fork |
| **Kiro** | Auto-generated `product.md` / `tech.md` / `structure.md` project understanding on first run (unique) |
| **Gemini / Copilot CLI** | Auto-compaction at 95%, `/compact`, `/memory` |
| **Codex CLI** | `codex resume <id>`, `codex fork` |

We have resume (`pi-run resume` → `pi --continue`) and Lore memory integration, but no `/context` visualization, no structured memory categories, no auto-generated project understanding. **Recommendation:** cheap win — document/curate a `CONTEXT.md`/project-understanding convention (Kiro-style) and a `pi-run doctor --context` view.

### F. Model portability / providers

| Project | Feature |
|---|---|
| **OpenCode** | 75+ providers via Models.dev; `baseURL` config; native OpenAI-compatible |
| **Codex CLI** | `config.toml` profiles, multi-provider (OpenAI, Anthropic, Azure, Bedrock, Ollama) |
| **Claude Code** | API key + Vertex/Bedrock setup wizards |
| **Gemini CLI** | native Gemini + Ollama (`--oss`) |

We have 7 providers — good, but **below** OpenCode (75+) and Codex (multi-provider + profiles). **Recommendation:** add Azure OpenAI + a `models.dev`-style catalog (or an import from Models.dev) to grow to 15-20 providers cheaply; keep explicit no-fallback semantics as a differentiator.

### G. Cloud / remote execution & interop

| Project | Feature |
|---|---|
| **Codex CLI** | `codex cloud` (cloud execution), `codex mcp-server` (run AS an MCP server), `--remote ws://` |
| **Copilot CLI** | ACP (Agent Client Protocol) support |
| **Claude Code** | `/remote-control`, `/desktop` |

We have none of these. **Recommendation:** skip for now (heavy), but note `codex mcp-server`-style interop as a stretch goal — could let other agents consume pi-run as a tool.

### H. Cost / usage tracking

| Project | Feature |
|---|---|
| **Claude Code** | `--max-budget-usd` budget cap |
| **pi-subagents** | per-run token/cost in results (we already surface this in subagent output) |

Our harness CLI previously had **no budget cap** and no cost dashboard. **Status:** both shipped in v0.5.0 — `pi-run cost` (per provider/model, `--json`, `--since`, `--reset`) and `pi-run chat|print --max-budget-usd` (pre-flight refusal, exit 6). High value for the anti-lock-in story ("know what each provider costs"); next step is wiring the cap into the CI scorecard (§5).

---

## 3. Convergent community patterns we should adopt (the "table stakes")

From the Q2 2026 landscape, every serious CLI agent converged on:
1. **Non-interactive mode** (`-p`/`exec`) — we have `pi-run print`. ✅
2. **MCP as the tool-extension protocol** — we have `pi-mcp-adapter` package. ✅ (but no harness-level MCP config)
3. **Project instruction file** (`CLAUDE.md`/`AGENTS.md`/`GEMINI.md`) — we have `AGENTS.md` + `.pi/SYSTEM.md`. ✅
4. **Permission tiering** (read-only / ask / auto-approve) — ❌ we delegate to Pi without a harness surface.
5. **Session persistence + resume** — we have `pi-run resume`. ✅
6. **SKILL.md conventions** — Pi uses skills; our `.pi/agents` wrappers carry skills. ✅

So we're strong on the *convergent* surface; the gaps are the *divergent bets* (sandboxing, hooks, budget, eval depth).

---

## 4. Positioning takeaways (what to lead with)

**Our moat is the "provider-agnostic harness + built-in eval" combo — now a full choose → measure → control loop:**
- **choose**: explicit multi-provider routing with reproducible eval (OpenCode has providers but no eval; SWE-bench has eval but no harness UX),
- **measure**: Docker-isolated benchmark (`pi-run eval --benchmark`) + real spend aggregation (`pi-run cost`),
- **control**: `--max-budget-usd` pre-flight cap with a dedicated exit code (6).

Plus: zero-key deterministic eval (unique), secret-manager pluggability, stdlib-only compiled binary with Homebrew distribution. No competitor — CLI or Pi extension — spans the whole loop; the Pi-native extensions (§1.5) only cover the *measure* step, and only interactively.

**Emphasize in README/marketing:**
- "One harness, any provider, with a built-in eval" (already our tagline — keep).
- **Now shipped:** "Run the same eval across OpenAI, Anthropic, DeepSeek, or local models — see which provider actually solves your tasks." (Docker-isolated benchmark + real cost per provider.)
- **Headline next differentiator:** the **CI provider scorecard** (`pi-run ci-benchmark`, §5) — ModelBench and pi-bench compare models interactively in a TUI; we turn provider comparison into a repeatable, budgeted, build-gating CI artifact.
- "No sandbox needed for trustworthy work" is NOT our story — be honest that sandboxing is a roadmap item, and lead with the eval + portability instead.

---

## 5. Roadmap recommendations (ranked)

| Priority | Feature | Effort | Source | Why |
|---|---|---|---|---|
| **SHIPPED (v0.5.0)** | `pi-run eval --benchmark`: Docker-isolated task runner (5 seed tasks); follow-ons: grow dataset, diff-grading | Medium | SWE-bench, terminal-bench | Closest to our core identity; enables provider comparisons; differentiator |
| **SHIPPED (v0.5.0)** | Cost/usage tracking: `pi-run cost` + `--max-budget-usd` (exit 6) | Low | Claude Code, pi-subagents | High user value; reinforces anti-lock-in |
| **P0 (new)** | Provider Scorecard in CI: `pi-run ci-benchmark` — run the same task suite against 2-3 providers in CI; produce a scorecard (pass rate, cost, latency); fail the build on regression or budget breach | Medium | Pi ModelBench, pi-bench, SWE-bench | ModelBench/pi-bench compare interactively; we uniquely can make provider comparison a repeatable, budgeted, gated CI artifact because we already have CLI + Docker + cost + budget |
| **P1** | Permission surface: `pi-run chat --permission-mode` passthrough + per-agent policy docs | Low | Claude Code, Codex, Gemini | Safety; table-stakes convergent pattern |
| **P1** | Hooks: `pre-eval`/`post-eval`/`pre-chat` in `pi-run` config | Low-Med | Claude Code, Copilot CLI | CI integration + extensibility |
| **P1** | Provider breadth: Azure OpenAI + Models.dev-style catalog (15-20 providers) | Low-Med | OpenCode, Codex | Closing the portability gap |
| **P2** | Session tooling: `/context`-style view, Kiro-style project understanding doc | Low | Claude Code, Kiro | DX |
| **P2** | `codex mcp-server`-style interop (run as MCP server) | High | Codex CLI | Stretch differentiator |

---

## 6. Sources

- wal.sh — *CLI Coding Agents: 2026 Q2 Comparison* (Claude Code, Copilot CLI, Gemini CLI, Codex CLI, Kiro, OpenCode, Goose, Aider; verified 2026-04-18/06-21)
- SWE-bench Evaluation Harness Reference (Docker, layered images, run_evaluation, cloud eval)
- terminal-bench / Harbor (continuous benchmark, task rubric, Docker sandboxes)
- aider Benchmarks (Exercism 133, edit formats, determinism)
- OpenHands Evaluation Harness docs (user_response_fn, run_evaluation, EvalOutput)
- pi.dev docs (Pi extensions/skills/packages); opencode.ai docs; goose docs
- Pi ModelBench (github.com/Deca/Pi-ModelBench) — Pi extension, `/benchmark` slash command; deterministic model comparison with profiles (coding, coding-agent, simplebench, reasoning, structured-data, customer-support, writing), per-model quality/latency/tokens/cost, `--runs`, HTML/JSON reports; runs inside a live Pi TUI session
- pi-bench (github.com/fornace/pi-bench) — `/bench` slash command + standalone script; probes every model with real streaming calls, ranks by latency/cost/quality, feeds model selection into pi-recap
- Agent Plugins 1.0.0 (agent-plugins.org) — vendor-neutral packaging spec for skills + MCP servers (Amazon/Cursor/Microsoft/OpenAI/Vercel/Google)
