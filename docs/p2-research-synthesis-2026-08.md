# P2 Research Synthesis + context-engine Decision — 2026-08-12

**Status:** Research complete (3 parallel researcher lanes) + context-engine decision brief.
**Purpose:** Turn the P2 roadmap into concrete, researched scope. Supersedes the stale
memory guidance in docs/competitive-gap-analysis-2026-08.md §E (which assumed Lore).

---

## 1. Research sweep (what the 2026 landscape says)

### 1.1 Session / context tooling (researcher brief 323467d2)

Every major 2026 coding agent converged on a four-layer context stack:

1. **Live context view** — `/context` (Claude Code: colored token grid + per-category
   breakdown incl. system prompt, tools, memory files, skills; optimization
   suggestions), `/status` (Codex), `/stats` (Gemini). Shows token usage by category.
2. **File-based persistent memory** — `CLAUDE.md` + typed `MEMORY.md` index (4
   categories: `user`/`feedback`/`project`/`reference`; capped 200 lines/25KB), layered
   under the shared `AGENTS.md` standard (Linux Foundation, 60K+ repos). Kiro
   auto-generates `product.md`/`tech.md`/`structure.md` on first run.
3. **Compaction** — manual `/compact` + auto-compact thresholds everywhere; Gemini's
   union-find episode-clustering is the 2026 frontier (88-92% fact recall vs 72-82%
   flat, 21% cheaper tokens, still experimental).
4. **Session lifecycle as a first-class artifact** — JSONL persistence, resume/fork/
   archive (Codex `archive`/`unarchive`; Pi resume/fork/`/tree`), export/import/share.

**Key insight for pi-harness:** Pi itself already provides ~60% of this (durable JSONL
session trees, resume/fork/`/tree`, footer token/cost/context totals, `/session`,
`/compact` with structured checkpoints, `/export`/`/import`/`/share`, AGENTS.md context
layering). So the highest-value pi-run work is **instrumentation and generation around
Pi's artifacts**, not rebuilding Pi's session engine:

- `pi-run context` (live token/cost breakdown by parsing session JSONL — reuse the
  `cost` command's scanner) — **P2**
- `pi-run project-understand` (Kiro-style product/tech/structure docs; deterministic
  stdlib-Go analysis + optional LLM pass) — **P1/P2**
- `pi-run handoff` (compact session exporter with copyable context — an unmet need
  flagged in a closed Codex issue) — **P2**

### 1.2 MCP-server interop (researcher brief 13eb6e98)

- **MCP is the clear agent→tool winner** (Linux Foundation/AAIF, ~half a billion
  monthly SDK downloads; first-class in Claude, Codex, Cursor, Gemini, ChatGPT).
- **`codex mcp-server` is the reference pattern** pi-harness should imitate: a coding
  agent exposed as an MCP server over stdio with a deliberately minimal tool surface,
  speaking spec `2025-03-26` (initialize-based). The 2026-07-28 stateless spec is
  breaking — **do not target it for an MVP** (12-month support window; every current
  host + codex use initialize-based).
- **ACP (Agent Client Protocol) is a different layer** (editor↔agent), not a competitor
  for this use case.
- A stdio-only MCP server in stdlib Go is ~200-300 lines (JSON-RPC 2.0 line-delimited:
  `initialize`, `tools/list`, `tools/call`, `ping`) — consistent with pi-run's
  zero-dep philosophy. Tool failures return content with `isError: true`, not JSON-RPC
  errors.
- **Recommended pi-run MCP tools:** `providers` (list), `eval --benchmark-dry-run`
  (validate), `cost` (spend), and optionally `chat`/`print` (delegate a task) — **P3**
  (medium/high effort, stretch differentiator).

### 1.3 P2-level landscape (researcher brief 63686c9b)

- **"The model is the engine, the harness is the car"** — mature players treat context
  assembly, tool exposure, the agent loop, and evaluation as the product; the newest
  competitive metric is **token efficiency and cost-per-task**, not raw benchmark score.
- **OTel GenAI agent telemetry** is a concrete (but Development-status) standard:
  `create_agent`/`invoke_agent`/`plan`/`execute_tool` spans with `gen_ai.*` attributes.
  Worth building now, pin versions. — **P2**
- **Sandboxing-as-a-service commoditized** (E2B microVMs, Modal gVisor/VM) — pi-harness
  could offer cloud eval backends as an adapter, not a platform. — **P2/P3**
- **Agent Plugins 1.0 / open-plugins** packaging standard (Working Draft) — pi-harness
  could ship its benchmark tasks/skills as a portable plugin. — **P3**
- **Model routing / cost optimization** is the 2026 pattern: route in the gateway by
  task tier, optimize **end-to-end task cost** (retries + escalations + wait time), not
  naive token cost. Our explicit no-fallback routing + cost tracking is the right
  shape; a cost-aware router (per-task-tier model choice) is the next step. — **P2/P3**
- **Budget/cost is a control-plane feature** (OpenRouter workspaces + budgets,
  LiteLLM per-team spend) — our cost tracking is the right shape; gaps are
  per-session/per-benchmark attribution + enforcement hooks.

---

## 2. context-engine decision (researcher brief fd2fe497)

**Verdict: ADD to P2 (not scrap, not keep-as-is).**

### Why not scrap
- **Runtime-healthy and actively used**: daemon served MCP `CallToolRequest` through
  2026-08-11 22:26; watch-loop offsets updated through 2026-08-11T14:25Z; wired in
  `~/.pi/agent/mcp.json` (`directTools: true`); the `using-context-engine` skill is
  installed and the agent calls the tools.
- **Fills a real, unserved gap**: temporal knowledge-graph memory (FalkorDB +
  Graphiti), decision recall (`recent_decisions`), project briefing (`get_briefing`),
  session-start prompt injection (`recall_for_prompt`). None of Pi (transcripts only),
  Lore (disabled, FTS5/three-tier, no graph), or the P2 items provide this.
- **The surviving memory backend by explicit user action**: Lore disabled
  (`LORE_DISABLED=1`), context-engine kept. Its compact opt-in recall is the antidote
  to the failure mode that killed Lore (huge auto-injected recap).
- **Landscape validates the category** (Graphiti: Thoughtworks Radar "Trial", Zep
  temporal-KG paper, Graphiti vs mem0 studies). Graphiti-core 0.29.2 is above the
  GHSA-gg5m-55jj-8m5g advisory threshold (0.28.2).
- **Cheap to keep, expensive to recreate**: the FalkorDB volume holds months of
  ingested sessions with no replacement.

### The one real risk (mitigated by the P2 scope)
- **Repo dormancy**: ~100 commits in a 13-day burst (Jul 19-Aug 2), then no commits
  10+ days; no tags/CI/CHANGELOG. The P2 adoption includes a version+CI commitment.

### Integration options
- **(a) Status quo** (external daemon + MCP) — keep as the operating mode. Zero cost,
  harness-agnostic. Gap: invisible to fresh clones; no health surfacing.
- **(b) Thin pi-run integration** — **P2**: `pi-run doctor` context-engine health check
  (probe :8377 + FalkorDB :6379, opt-in/personal-gated like `PI_RUN_PERSONAL`).
  **P3**: `pi-run context --recall/--decisions` thin client.
- **(c) Vendor into Go** — **rejected** (Python + Docker + Ollama; multi-month; orphans
  existing data; violates zero-dep philosophy worse than an external daemon).
- **(d) Document as the recommended memory backend** — **primary action** (replaces
  Lore in the roadmap).

### P2 scope (all small, stdlib-only Go + docs)
1. **Docs**: `docs/memory-backend.md` + README "Memory" section — what context-engine
   is, one-line install, the 6 MCP tools, the skill, security constraint (stay on
   127.0.0.1 — no auth), replaces Lore.
2. **`pi-run doctor` health check** (opt-in/personal-gated).
3. **Version + CI the engine**: tag `v0.2.0`, add a GitHub Actions pytest job
   (`uv sync` + `pytest -m 'not integration'`), CHANGELOG entry. De-risks dormancy.
4. **`pi-run context` thin client (P3)**: read-only; also a deterministic config-check
   that the documented MCP contract matches `~/.pi/agent/mcp.json`.
5. **Optional eval synergy (P3)**: assert `recall_for_prompt` is non-noise (context
   block < N tokens); correlate briefing-before-answer with DeepEval faithfulness.

### Operational notes to carry forward
- Keep `graphiti-core >= 0.28.2` (currently 0.29.2, safe).
- Fix README drift: `context-engine/README.md` claims native KNN but code uses cosine
  scan + BM25 + GraphRAG — align docs with code.
- Address launchd restart-race noise (port-bind "address already in use") if it bothers
  ops.
- GitLab/GitHub connectors are built but dormant — document as optional cron, don't
  promise them.

---

## 3. Updated P2/P3 roadmap (supersedes gap analysis §5 P2 rows)

| Priority | Feature | Effort | Source |
|---|---|---|---|
| **P2** | `pi-run context` — live token/cost/context view by parsing session JSONL (reuse cost scanner) | Low | Claude `/context`, Codex `/status` |
| **P2** | `pi-run project-understand` — Kiro-style product/tech/structure doc generator | Low-Med | Kiro |
| **P2** | context-engine as documented optional memory backend (docs + doctor check + version/CI) | Low | context-engine decision brief |
| **P2** | OTel GenAI agent telemetry export (pin semconv versions; Development-status) | Med | 63686c9b |
| **P2/P3** | Cost-aware routing (per-task-tier model choice, end-to-end task cost) | Med | 63686c9b |
| **P3** | `pi-run mcp-server` — expose providers/eval/cost as MCP server (stdio, spec 2025-03-26) | Med-High | codex mcp-server |
| **P3** | `pi-run context --recall/--decisions` thin MCP client to context-engine | Med | context-engine brief |
| **P3** | Agent Plugins 1.0 packaging of benchmark tasks/skills | Med | 63686c9b |
| **P3** | Cloud eval backend adapter (E2B/Modal) | Med | 63686c9b |

---

## 4. Sources

- Researcher briefs (this repo's `.pi-subagents/artifacts/`): `323467d2` (context
  tooling), `13eb6e98` (MCP server), `63686c9b` (P2 landscape), `fd2fe497`
  (context-engine decision).
- modelcontextprotocol.io; codex mcp-server docs; agent-plugins.org; open-telemetry
  semantic-conventions-genai; mnem.dev state-of-agent-memory-2026.
- context-engine repo (local): README, git logs, daemon logs, uv.lock; harness
  `~/.pi/agent/mcp.json`.
