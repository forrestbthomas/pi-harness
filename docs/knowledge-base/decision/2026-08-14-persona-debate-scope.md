---
id: "kb-20260814-persona-debate-scope"
type: "decision"
status: "active"
scope: "project"
components:
  - "pi-run"
  - "eval"
tools: []
trigger_tags:
  - "type:decision"
  - "charter"
  - "scope"
related_files:
  - "CHARTER.md"
  - "ROADMAP.md"
created_at: "2026-08-14T08:00:00+00:00"
updated_at: "2026-08-14T08:00:00+00:00"
---

# Persona debate — project scope, north star, and the keep/cut/split decision

## Summary

A six-persona internal debate (Product Purist, Platform Architect, Portfolio
Manager/TPM, OSS Community Maintainer, Cost/Complexity Skeptic, Dogfooder/Meta
Lens) ran in two rounds over the question: *does the north star — "a
self-healing, measurable, distributable coding-agent harness" — need
adjusting, and what stays / gets cut / gets split?*

**Verdict: ADJUST — unanimous.** All six personas independently converged on
three repairs, and the disagreement narrowed to one question: the *trigger*
for splitting the eval suite into a separate benchmark repo (pi-bench).

Raw transcripts: `.pi/debate/` (forum-context, `round1/*.md`, `round2/*.md`,
`synthesis.md`) — kept local, not committed. This record is the durable
outcome; `CHARTER.md` is the enforceable boundary contract.

## Decision

1. **North star adjusted** (in `ROADMAP.md` and `CHARTER.md`):
   - Add an explicit **"we do not"** clause (the star previously named what the
     project does but never what it is *not* — the root cause of the
     everythingitis that grew four heads).
   - **"Distributable" is earned**: macOS/Homebrew is the shipped leg; any
     further platform or benchmark repo waits for a concrete consumer or a
     second owner team.
   - The harness is **measured by** the benchmark, it does not **own** it; the
     eval suite stays in-repo today as the measurement layer.
2. **Keep:** Go core (13.8k, 47 files) + hermetic tests + minimal CI; the
   versioned seam (`tasks.json` → `score_run.py` → scorecard JSON) as the
   jointly-owned contract; the eval suite in-repo with its honesty machinery
   (baseline gate, provenance, flake-aware gate, always-upload artifacts);
   ROADMAP.md (Now/Next/Later) + one conventions page; minimal release path;
   governance skeleton.
3. **Cut (consumerless, with evidence):** MCP server (384 lines), OTel exporter
   (183 lines) — zero callers found in repo search; `pdf2txt.sh`; spec
   shelf-ware → one `decisions.md`; PM planning artifacts → `docs/governance/`
   (off the contributor surface); README dual-product headline (rewritten);
   Homebrew upkeep → ~10-line `make release` until a tapper exists.
4. **Split (triggered, not big-bang):** eval suite → **pi-bench** (own repo)
   when **EPIC-1's DoD closes AND an external consumer appears**, or when a
   release is actually blocked by cross-layer coupling. Until then: declare the
   portfolio shape, adopt A-PR/B-PR separate-train discipline in-repo. OSS's
   dataset-first extraction is the acceptable partial step if dataset cadence
   trips first.

## Rationale

- PMI/CNCF research: a charter is purpose + in-scope + **explicit out-of-scope**;
  unowned open decisions breed scope creep; "everythingitis" kills projects.
- The seam already exists and is versioned (`datasetVersion 2026-08-14.4`
  calendar vs CLI `v0.9.x` semver = two version schemes = two lifecycles), so
  both keep *and* split stay cheap.
- Conway's Law conceded by all: one maintainer → one repo is natural; a split
  is an org decision, not an architecture fiat.
- The honesty machinery (flake gate, provenance, sub-1.0 baselines) is
  untouchable — defended by every persona without exception.

## Follow-ups

- [x] Charter + north star + README headline landed (PR #98/#99).
- [x] Cut list decision (MCP/OTel/pdf2txt cut; Homebrew kept per charter;
      specs deferred) — PR landing this change.
- [ ] Move PM artifacts under `docs/governance/`.
- [ ] Spec shelf-ware consolidation (deferred 2026-08-14; separate decision).
- [ ] Record split triggers in BACKLOG so EPIC-1's DoD doubles as pi-bench's
      maturity gate.
