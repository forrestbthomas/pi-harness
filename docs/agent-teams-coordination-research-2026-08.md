# Agent Teams & Coordination: Research Note

> **Date:** 2026-08-21 · **Status:** research / decision-input (no code landed)
> **Question:** Should the harness adopt pi-teams as-is, build its own team
> coordination layer, or neither — and what should the coordination design be?
> **Answer-in-one-line:** The thrash observed in pi-teams/Claude Agent Teams is
> structural (per-message LLM turns, no shared situational awareness). The
> research points to a **shared-bulletin (blackboard) + cheap eventing +
> batched/reconcile-on-conflict** design, not peer message passing. Building it
> belongs in a separate open-source Pi package, not in pi-harness.

## 1. What is in the harness today

- **pi-teams is NOT on `main`.** It exists only on the `try-pi-teams` branch /
  worktree (`.worktrees/try-pi-teams`), commit `be48e1b` ("chore(pi): add
  pi-teams 0.9.14 (agent-teams try-out)"). Diff = `.pi/settings.json`,
  `.pi/npm/package.json`, `.pi/npm/package-lock.json` only.
- **pi-teams** = third-party Pi package (`github.com/burggraf/pi-teams`, MIT,
  v0.9.14), a port of `claude-code-teams-mcp` (cs50victor). Terminal-pane
  (tmux/Zellij/iTerm2/WezTerm) + file-based coordination: team lead spawns full
  Pi agents into panes; shared task board + per-agent inboxes
  (`send_message`/`read_inbox` polling). Protocol inherited from Claude Code
  Agent Teams (mailbox + task list).
- **MSG-1 (Cross-session peer messaging)** is already queued in `BACKLOG.md`
  (rank 12, pitched 2026-08-18), scoped to *harness-launched sessions only*,
  with a charter-conformance decision required before promotion. Research note
  there: "no Pi Core or extension analog today — pi-subagents intercom is
  parent↔child only."

## 2. Reference implementations and why they thrash

Claude Code Agent Teams (official docs:
<https://code.claude.com/docs/en/agent-teams>; reverse-engineered protocol:
<https://nwyin.com/blogs/claude-code-agent-teams-reverse-engineered.html>):

- **Topology:** lead + flat peers; each teammate is a separate CLI process with
  its own full context window.
- **Coordination substrate:** files on disk — `teams/{name}/config.json`
  (membership), `inboxes/{agent}.json` (per-agent mailbox), `tasks/{name}/*.json`
  (task list, `flock` claiming, `blocks`/`blockedBy`).
- **Messaging:** append to inbox → recipient polls → message injected as a
  **synthetic conversation turn**. Types: `task_assignment`, `message`,
  `broadcast`, `shutdown_request/response`, `plan_approval_request/response`,
  `idle_notification`.
- **Thrash is structural, not incidental:**
  1. Every message costs a full LLM turn (context injection + evaluation); no
     cheap signal path.
  2. No shared situational awareness: teammates start fresh (spawn prompt only,
     no inherited history); each maintains its own context. Findings must be
     sent + injected + re-evaluated.
  3. Coordination overhead scales with team size (O(n²) peer messaging
     potential; broadcast tokens scale linearly).
  4. Measured token cost ≈ **7× baseline** in plan mode; docs recommend 3–5
     teammates and *parallel exploration (research/review) only* — explicitly
     not same-file edits or heavily dependent work.
  5. Known failure modes even in the reference: task-status lag, early
     stopping, permission friction, no nested teams, no session resumption.

pi-teams inherits all of the above. The tmux/UI plumbing is real value; the
coordination protocol (peer message passing) is the weak part.

## 3. Research landscape

### 3.1 Why multi-agent LLM systems fail — MAST (arXiv 2503.13657, Berkeley)

First empirically grounded failure taxonomy: 200+ traces, 7 frameworks, 14
failure modes, 3 categories — specification issues (41.8%), inter-agent
misalignment (36.9%), task verification (21.3%). Key findings:

- Inter-agent misalignment includes information withholding, proceeding on
  wrong assumptions instead of asking for clarification (11.7%), task
  derailment, ignoring other agents' input — the thrash/derail patterns seen in
  real team runs.
- Prompt-level interventions gave marginal gains; topology-level changes gave
  more. Many failures are **organizational-design problems, not LLM-capability
  problems** ("even organizations of sophisticated individuals can fail
  catastrophically if the organization structure is flawed").
- Fix direction: standardized communication protocols, persistent
  memory/state, strong multi-level verification.

### 3.2 The communication design space — "Beyond Self-Talk" (arXiv 2502.14321)

- **Architectures:** flat, hierarchical, team, society, hybrid. Flat (pi-teams /
  Claude Teams shape) scales poorly — communication overhead grows with agent
  count.
- **Communication strategies:** one-by-one (serial; latency linear in turns),
  simultaneous-talk (parallel but state-stale; needs arbitration),
  **simultaneous-talk-with-summarizer** (dedicated agent compresses concurrent
  outputs into a shared coherent summary — the "bulletin" idea with a
  mechanism).
- **Communication paradigms:** message passing (point-to-point/broadcast —
  what pi-teams does), speech act, **blackboard** (central shared repository;
  agents read/write it; no direct agent-to-agent contact).
- **Protocols:** MCP, A2A, ANP emerging as standards; no need to invent a wire
  protocol.

### 3.3 The blackboard result — LbMAS (arXiv 2507.01701)

The experiment implied by the "cue/bulletin" idea, done with LLM agents:
agents communicate **only through a shared blackboard**; a **control unit**
(LLM) selects which agents act per round from current blackboard content; a
**cleaner** prunes redundant messages; a **conflict-resolver** detects
contradictions and forces reconciliation.

- **Better accuracy than SOTA static/dynamic message-passing MAS and the
  second-lowest token cost** of all baselines.
- **Only 2.9–3.3 blackboard cycles on average** — reconciliation is *batched*,
  not continuous. Easy problems stop after one round.
- Cleaner ablated → performance drops (content management matters).
- Lineage: classic blackboard architecture (Hayes-Roth 1985; Hearsay-II) — the
  pre-LLM distributed-AI answer to "many experts, one shared workspace, a
  scheduler decides who acts."

### 3.4 Reconciliation-loop thinking, made agent-appropriate

- Kubernetes operator pattern: reconcile continuously because diffing is
  microseconds. LLM reconcile is expensive, so the literature's answer is
  **cheap detection, expensive resolution**:
  - **LatticeMind** (arXiv 2608.08236): conflict-aware memory with **cheap
    symbolic conflict checks at write time; LLM reconciliation only for
    unresolved semantic cases** (0.97 vs 0.61 accuracy vs aggregation
    baselines; removing checker or reconciler costs 12–14 points).
  - **Anthropic multi-agent research system**
    (<https://www.anthropic.com/engineering/multi-agent-research-system>):
    orchestrator-worker, *synchronous* spawn→collect rounds, lead-as-summarizer;
    explicitly notes async peer coordination is still open ("agents can't steer
    subagents; subagents can't coordinate"). Also: **write artifacts to
    filesystem, pass references** (avoids "game of telephone").
  - **MetaGPT** (arXiv 2308.00352): reduces free-form chatter via SOPs and
    shared structured artifacts (requirement doc → design doc → code) —
    synchronize on documents, not messages.

### 3.5 On "level-based design"

Read as hierarchical/layered design: the literature is unambiguous that
**hierarchy is the anti-thrash structure** (survey §3.1.2: ChatDev, HyperAgent,
Magentic-One; Claude best practices: lead synthesizes, teammates own disjoint
files). Second reading — *levels of abstraction in what is shared* (raw
messages vs. summarized state vs. structured artifacts) — equally important:
each aggregation level is a token-cost multiplier and a fidelity risk, so
choose the *highest* level that preserves fidelity.

## 4. Assessment of the reconciliation-loop / observability idea

**Diagnosis confirmed**: peer message passing with full-agent context is the
wrong substrate for coordinating 5 agents; thrash is structural.

**Refined shape** — not "k8s reconcile loop with LLMs" but a three-tier
separation:

1. **Cheap eventing (never an LLM call):** writes/updates to shared state
   generate cheap "state changed" signals, not prose. This is the fast
   observability loop.
2. **Shared aggregated picture (the bulletin):** a blackboard/summarizer layer
   periodically compresses what happened into a compact situational digest —
   one LLM call per sync round, not per message (LbMAS control unit + cleaner;
   survey simultaneous-talk-with-summarizer; Anthropic lead-as-summarizer).
3. **Expensive reconciliation only at convergence points:** LLM calls fire on
   conflict detection (cheap symbolic check first — LatticeMind), task
   completion (verification gates), and round boundaries (lead synthesis),
   never per message.

**One-liner:** reconcile the shared state, not the agents' conversations.

## 5. Recommendation

- **Do not merge pi-teams into the harness as-is.** It is a capable UI/plumbing
  implementation of a coordination protocol the research says is wrong for this
  goal, and it adds a third-party runtime dependency to a measurement product.
  Keep on `try-pi-teams` as a reference artifact.
- **Do not build a "pi-teams competitor" inside the harness** — charter
  expansion (general agent orchestration ≠ measurement). The bulletin design
  belongs in a **separate, open-source Pi package** (MIT, standalone,
  dogfoodable), not in pi-harness.
- **Cheapest high-value path:** the bulletin-first protocol as a Pi extension
  package (~200–400 lines TS + skill), evaluated against pi-teams on a small
  benchmark (same 5-agent research task; compare tokens, wall-clock, output
  quality). The eval harness is built for exactly this comparison.
- **Charter-compliant sliver already queued:** MSG-1 (cross-session peer
  messaging for harness-launched sessions) — fold the bulletin idea in (shared
  digest, not just raw inboxes), record the charter-conformance decision.
- **Contribute option:** pi-teams is MIT and maintained; a "blackboard mode" PR
  upstream (shared digest + cheap-change signals replacing inbox polling)
  would test the design with lower ownership cost.

## 6. Sources

| # | Source | Relevance |
|---|---|---|
| 1 | Claude Code Agent Teams docs — <https://code.claude.com/docs/en/agent-teams> | Reference implementation; best practices; limitations |
| 2 | nwyin, "Reverse-Engineering Claude Code Agent Teams" — <https://nwyin.com/blogs/claude-code-agent-teams-reverse-engineered.html> | On-disk protocol: mailboxes, task files, message types, 7× token cost |
| 3 | Pan et al., "Why Do Multi-Agent LLM Systems Fail?" (MAST) — arXiv 2503.13657 | Failure taxonomy; design-over-model conclusion |
| 4 | Zhou et al., "Beyond Self-Talk: A Communication-Centric Survey of LLM-Based MAS" — arXiv 2502.14321 | Architecture/strategy/paradigm taxonomy; blackboard vs message passing |
| 5 | Han & Zhang, "Exploring Advanced LLM Multi-Agent Systems Based on Blackboard Architecture" (LbMAS) — arXiv 2507.01701 | Blackboard beats message passing on accuracy AND tokens |
| 6 | Zhou et al., "LatticeMind: A Conflict-Aware Memory Primitive" — arXiv 2608.08236 | Cheap symbolic checks + LLM reconciliation only when unresolved |
| 7 | Anthropic, "How we built our multi-agent research system" — <https://www.anthropic.com/engineering/multi-agent-research-system> | Orchestrator-worker; summarizer; artifact-not-content handoff; 15× tokens |
| 8 | Hong et al., "MetaGPT" — arXiv 2308.00352 | SOP + shared structured artifacts; synchronize on documents |
| 9 | burggraf/pi-teams (package README + source in `.worktrees/try-pi-teams`) | The port under evaluation |
| 10 | `BACKLOG.md` MSG-1 | Existing charter-compliant cross-session messaging item |

## 7. Decision log

- 2026-08-21: Research note created; no code changed on `main` in this task.
- 2026-08-21: `BACKLOG.md` MSG-1 updated to fold in the bulletin idea (see item).
- 2026-08-21: Spike scaffolded in worktree `spike-bulletin-protocol` (Pi
  package `pi-bulletin`: shared bulletin file + cheap signals + skill).
