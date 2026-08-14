# Agent Plugins 1.0 — Positioning + Example (Document + Example Only)

**Date:** 2026-08 (wave 2 docs pass)
**Priority:** P3 (see [P2 research synthesis](p2-research-synthesis-2026-08.md) §3)
**Wave decision:** DOCUMENT + EXAMPLE ONLY. No runtime support until the
standard stabilizes. No Go code, no `providers.json`, no `eval/` changes in
this wave.

---

## 1. Status: what Agent Plugins 1.0 is and why we are not implementing it yet

Agent Plugins 1.0 is a vendor-neutral **packaging standard for portable agent
behaviors** — skills and MCP servers bundled into distributable, installable
units. It is shepherded by the agent-plugins.org / open-plugins effort, with
Amazon, Cursor, Microsoft, OpenAI, Vercel, and Google named as participating
vendors in the competitive landscape (see
[docs/competitive-gap-analysis-2026-08.md](competitive-gap-analysis-2026-08.md) §1.5
and §6).

**Standardization status: Working Draft.** The research sweep
([docs/p2-research-synthesis-2026-08.md](p2-research-synthesis-2026-08.md) §1.3)
explicitly flags Agent Plugins 1.0 / open-plugins as a *Working Draft*, and the
competitive gap analysis cites it as **Agent Plugins 1.0.0** — a version number
that itself signals how fast the draft is moving. Pre-stable means field names,
packaging layout, and host-reader behavior can change between revisions.

**Why we are NOT implementing runtime support yet (draft churn risk):**

- Any runtime loader we build now would be written against a moving target.
  A spec revision that renames manifest fields or changes the payload layout
  would silently invalidate every packaged plugin we produced, forcing either
  migration tooling or spec-snapshot pinning.
- We have direct precedent for this risk in-repo: the MCP research
  (research brief §1.2, summarized in the P2 synthesis) found that the
  2026-07-28 stateless MCP spec was breaking, and the safe move was to target
  the stable initialize-based `2025-03-26` spec. A draft packaging standard is
  the same hazard one layer up.
- The value of plugins is *portability across hosts*; that value only exists
  once the format is stable enough that other hosts can consume what we ship.

**Consequence:** this wave ships a positioning doc and one example manifest so
the shape of the idea is recorded, with an explicit review checkpoint before
any runtime work is scheduled (see [§7 Roadmap](#7-roadmap)).

---

## 2. Relationship to MCP

MCP and Agent Plugins are **complementary layers, not competitors**:

| Layer | What it standardizes | In pi-harness today |
|---|---|---|
| **MCP** (Model Context Protocol) | Tools a client (agent host) can call — a live, procedural interface | none — `pi-run mcp-server` shipped in v0.8.0 then **removed in v0.10.0** (cut list #100; see CHANGELOG) |
| **Agent Plugins 1.0** (draft) | Packaged, portable agent behaviors/workflows — skills and MCP servers distributed as installable units | none (documented only) |

MCP answers "how does an agent call a tool at runtime"; Agent Plugins answers
"how does a reusable behavior get packaged, distributed, and installed". An
upstream Agent Plugins package can *contain* an MCP server (per the draft's own
scope — skills + MCP servers), which makes the two standards stacked layers:
the plugin is the distribution envelope, MCP is one of the runtime surfaces
inside it. pi-harness already ships its own minimal MCP server; a future plugin
format would complement it, not replace it. This framing follows the P2
research synthesis §1.2 (MCP is the agent→tool winner) and §1.3 (Agent Plugins
is the packaging layer), and the competitive gap analysis §1.5 (packaging
spec, not a benchmark tool — "relevant to our plugin story, not to eval").

---

## 3. What pi-harness could ship as plugins (future)

If/when the standard stabilizes, the most natural first-class plugin types map
onto things pi-harness already produces. All examples below are concrete
artifacts in this repo today; *none of them are plugins yet* — this section is
a forward-looking inventory.

### 3.1 Benchmark task suites (`type: benchmark`)

`eval/benchmarks/*` already has a stable, self-describing layout
(see [docs/benchmarks.md](benchmarks.md)):

```
eval/benchmarks/fix-divide-by-zero/
├── task.json      # id, prompt, timeoutSecs, testScript, solution
├── src/           # task workspace the agent edits
├── tests/run.sh   # verification script; exit 0 = pass
└── solution/      # optional oracle
```

A benchmark plugin would package one such task directory (or a suite of them)
so an external suite can be dropped in without adaptation. The `task.json`
field set (`id`, `prompt`, `timeoutSecs`, `testScript`, `solution`) already
looks like a manifest payload; today it is validated hermetically by
`pi-run eval --benchmark-dry-run` (no Docker, no keys — CI-safe).

### 3.2 Skills bundles (`type: skills`)

Pi supports extensions/skills/packages (per pi.dev docs; see P2 research
synthesis §4 sources), and pi-harness already carries agent configuration and
skill-adjacent content under `.pi/` (`AGENTS.md`, `.pi/SYSTEM.md`,
`.pi/settings.json`, `.pi/agents/*.md` wrappers, `.pi/APPEND_SYSTEM.md`). A
skills plugin would bundle a named behavior — e.g. a reproducible agent
workflow document plus the supporting config — as a portable unit that can be
installed into a Pi environment.

### 3.3 Provider scorecard definitions (`type: scorecard`)

`pi-run ci-benchmark` (`internal/cli/scorecard.go`) gates on per-provider
scorecards: providers, models, `--fail-below` pass rates, `--baseline`
regression tolerance, `--runs`. A scorecard plugin would carry one such
definition (which providers/models to run, which pass-rate bar gates CI) so a
benchmark suite + scorecard pair ships as one portable unit instead of a
hand-assembled flag set.

---

## 4. Proposed layout (DRAFT proposal — not a spec)

> **Warning:** this layout and schema are a **pi-harness-local placeholder**
> for documentation purposes only. It deliberately does **not** claim
> conformance to the upstream Agent Plugins 1.0 draft, and it is expected to
> change when that standard stabilizes. Treat nothing here as stable.

A plugin is a **directory** containing a manifest plus a payload:

```
<plugin-name>/
├── manifest.json   # DRAFT schema below (required)
└── payload/        # type-specific content (required):
                    #   benchmark → task dir as in eval/benchmarks/<name>/
                    #   skills    → skill/package directory
                    #   scorecard → scorecard definition JSON
```

### Minimal manifest schema (DRAFT — every field subject to change)

| Field | Kind | Description (DRAFT) |
|---|---|---|
| `name` | required | Unique, safe id following the benchmark `id` convention (`[a-zA-Z0-9][a-zA-Z0-9._-]*`; see docs/benchmarks.md). Used as the install directory name. |
| `version` | required | Semantic version of the plugin payload. |
| `description` | required | One-line summary of what the plugin provides. |
| `author` | required | Maintainer or organization (free-form string; may be empty for community bundles). |
| `type` | required | Plugin kind; suggested enum `benchmark \| skills \| scorecard` (see §3). |
| `entry` | required | Relative path to the plugin's entry point, e.g. `payload/task.json` (benchmark), `payload/SKILL.md` (skills), `payload/scorecard.json` (scorecard). |

Open questions that are intentionally **not** answered here (they belong to
the upstream draft): whether `entry` will be a path or a URI; whether
dependencies/peer-requirements exist; how signatures/trust are modeled; how
plugin versions interact with host pinning.

---

## 5. Example

See [examples/plugins/manifest.example.json](../examples/plugins/manifest.example.json),
created alongside this doc. It demonstrates a `benchmark`-type plugin wrapping
the `fix-divide-by-zero` task shape from `eval/benchmarks/`:

```json
{
  "name": "example-fix-divide-by-zero",
  "version": "0.1.0",
  "description": "Example plugin manifest (DRAFT schema) showing how a pi-harness benchmark task could be packaged; no runtime consumes this yet.",
  "author": "testvalue-author",
  "type": "benchmark",
  "entry": "payload/task.json"
}
```

### Intended usage (HYPOTHETICAL — NOT IMPLEMENTED)

The commands below **do not exist** in `pi-run` today (the dispatch table in
`internal/cli/app.go` has no `plugin` case; `pi-run help` does not list one).
They are recorded here only to make the packaging idea concrete for reviewers:

```bash
pi-run plugin list               # enumerate installed plugins (e.g. ./plugins/*),
                                 # read each manifest.json, print name/version/type
pi-run plugin validate <dir>     # check manifest fields (name/version/type/entry)
                                 # and that entry resolves inside the plugin dir
```

Until a `plugin` subcommand exists, the only "validation" available is manual:
`python3 -m json.tool manifest.json` for JSON well-formedness and a visual
check of the field set in §4.

---

## 6. Adoption risks

| Risk | Description | Watch-point before committing runtime support |
|---|---|---|
| **Schema churn** | Draft revisions rename manifest fields / change payload layout; every revision invalidates packaged plugins or forces migration/pinning. | Spec reaches stable/1.0 **or** a widely-adopted freeze; draft schema unchanged for ≥ one full release cycle. |
| **Host-support fragmentation** | Pre-stable standard means hosts adopt different snapshots (or never adopt). Plugin portability is the entire point; without host adoption the format is a parallel, harness-only path. | A critical mass of hosts (Claude Code, Codex, Cursor, and peers) ships a compatible stable reader; broad, consistent adoption sustained over time. |
| **Runtime-surface mismatch** | pi-harness launches Pi as the agent runtime. A pi-run plugin loader that Pi itself cannot consume adds a parallel format with no user-visible value. | Verify Pi's own skills/packages surface can consume plugin content (or that plugin payloads degrade cleanly into the existing `eval/benchmarks` and `.pi/` mechanisms). |
| **Cost/benefit inversion** | The current benchmark suite is already hermetic, CI-safe, and portable by directory copy (docs/benchmarks.md). Plugin packaging only pays off when external suites can be dropped in without adaptation. | Hold unless an external ecosystem of suites actually forms around the stabilized standard. |

Net position: **watch and document now; implement only after the §7
acceptance criteria are met.** Nothing in this wave commits pi-harness to any
particular plugin format.

---

## 7. Roadmap

**Priority: P3** — "Agent Plugins 1.0 packaging of benchmark tasks/skills",
per [docs/p2-research-synthesis-2026-08.md](p2-research-synthesis-2026-08.md) §3.

| Stage | Scope | Status |
|---|---|---|
| **This wave (P3, docs)** | `docs/plugins.md` positioning + `examples/plugins/manifest.example.json`. No runtime code. | Done (this change) |
| **Revisit gate** | Re-evaluate at the next roadmap review, not reactively. | Pending |
| **Runtime support (P3→implement)** | Only when acceptance criteria below are met; scope would be defined then (likely `pi-run plugin list/validate` + a loader for the stabilized manifest). | Not started |

**Acceptance criteria for runtime support (any one is sufficient to open the
ticket):**

1. **Agent Plugins 1.0 reaches stable status (1.0)** at agent-plugins.org /
   open-plugins, **and** the schema has been stable for ≥ one release cycle;
   or
2. **A critical mass of hosts adopt a single stable version** of the spec —
   evidenced by multiple major coding agents shipping compatible readers and
   sustaining that compatibility across releases.

**Non-goals (this wave):** no `plugin` command, no loader, no
`providers.json` or `eval/` changes, no README/CHANGELOG edits (owned by the
docs pass separately).

---

## 8. Review checklist

A reviewer should verify:

- [ ] **Working-Draft status is accurate**: the doc says Agent Plugins 1.0 is
      a Working Draft, pre-stable, with churn risk; it makes **no** claim of
      conformance and labels §4's schema as a pi-harness-local DRAFT proposal.
- [ ] **No runtime claims**: `pi-run plugin list/validate` are explicitly
      marked hypothetical/not implemented and match the current command table
      (`internal/cli/app.go` has no `plugin` case).
- [ ] **Example JSON is valid**: `examples/plugins/manifest.example.json`
      passes `python3 -m json.tool` (or `jq`) and uses only `testvalue`-style
      placeholders — no secrets, no real-looking key material.
- [ ] **Citations are accurate**: P2 research synthesis §1.3/§3 (Working
      Draft + P3 row), §1.2 (MCP layer), competitive gap analysis §1.5/§6
      (vendor participation, packaging-spec-not-benchmark), docs/benchmarks.md
      (task layout), and `eval/benchmarks/*` + `internal/cli/scorecard.go`
      (concrete payload examples) — all referenced correctly.
- [ ] **House rules held**: docs-only lane — no Go, tests, `providers.json`,
      `eval/`, `README.md`, or `CHANGELOG.md` changes; no hardcoded user
      paths; `go build ./... && go test ./internal/cli/` still pass.
