# Scope Contract
**Task:** Cut list from the 2026-08-14 persona debate (consumerless machinery + scope hygiene) | **Plan:** `.pi/debate/synthesis.md` + `docs/knowledge-base/decision/2026-08-14-persona-debate-scope.md` | **Date:** 2026-08-14 | **Status:** CLOSED — 3 changes logged

## In Scope
- **Files — MCP server (cut):**
  - `internal/cli/mcp.go` (384 lines), `internal/cli/mcp_test.go` (527 lines) — delete
  - `internal/cli/app.go` — remove `mcp-server` command case + usage line
  - `README.md` — remove MCP server section + command-table row + CLI list mention + `pi-mcp-adapter` package note if it exists only for this
  - `eval/tests/test_contract_mcp.py` (234 lines) — delete
  - `.github/workflows/nightly-live-eval.yml`, `.github/workflows/ci.yml` — remove `test_contract_mcp.py` from pytest invocations
  - `eval/conftest.py` — update `pi_run_bin` probe (remove "usage must document mcp-server" assertion); keep `SUPPORTED_PROVIDER_KEYS` mirror (used by other tests)
  - `docs/*.md` — remove `pi-run mcp-server` command references where they describe the command (keep historical spec/research text)
- **Files — OTel exporter (cut):**
  - `internal/cli/otel.go` (183 lines), `internal/cli/otel_test.go` (222 lines) — delete
  - `internal/cli/app.go` — remove `maybeExportOTLPSpan` call + `PI_OTLP_ENDPOINT` usage line
  - `README.md` — remove Telemetry (OTLP) section + env table row
  - `eval/tests/test_contract_otel.py` (106 lines) — delete
  - `.github/workflows/nightly-live-eval.yml`, `.github/workflows/ci.yml` — remove `test_contract_otel.py` from pytest invocations
  - `eval/conftest.py` — remove `PI_OTLP_ENDPOINT` from `_HERMETIC_ENV_VARS`
- **Files — pdf2txt (cut):**
  - `scripts/pdf2txt.sh` — delete (only referenced by the decision record itself)
- **Files — docs hygiene:**
  - `README.md` — already single-product (done in #98); clean remaining orphaned mentions
  - `AGENTS.md`, `.pi/SYSTEM.md`, `CHARTER.md` — update any MCP/OTel references to reflect removal
  - `docs/knowledge-base/decision/2026-08-14-persona-debate-scope.md` — update follow-up checklist (cut list done)

## Boundaries
- **No behavior change to remaining CLI surface.** Only removal; no refactor of code that stays.
- **Exit-code table unchanged** (0–9 are stable contract; no mcp/otel codes to remove).
- **Dataset case `coding-020`** (harness-routing, MCP protocol version) is a *knowledge* grader about MCP protocol negotiation — it references `pi-run mcp-server` in its prompt. Handle per decision below (flag).
- **No change to eval grading, baselines, budget, score_run, watchdog, self-heal, cost, providers, hooks, install, doctor, config-check.**
- **Hermetic tests stay green** after removal: `go build/vet/test`, pytest contract subset, config-check.

## Out of Scope (explicit)
- **Homebrew / release machinery** — CHARTER.md (just landed, #98) declares "macOS/Homebrew is the shipped leg." Cutting `update-homebrew-formula.sh` / `release.yml` Homebrew step contradicts the charter. Flagged for decision; NOT cut without explicit approval.
- **PM artifacts relocation to `docs/governance/`** — declared in CHARTER.md as a follow-up; it is a docs-move workstream with wide link churn (ROADMAP/BACKLOG/EPICS/STATUS/SCOPE references), not a code cut. Separate PR.
- **Spec shelf-ware consolidation** (19 specs → one decisions file) — shipped specs are referenced by ROADMAP rows; archiving them breaks roadmap links. Flagged for decision on depth (archive-only vs full consolidation); not cut blind.
- `install-skills.sh`, `bootstrap.sh`, `tag-release.sh`, `build-release.sh` — governance/release tooling, not consumerless product surface.
- `.pi/settings.json` Pi packages — runtime config, not harness product surface.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|
| 1 | ambiguity | `coding-020` dataset case references `pi-run mcp-server` in its prompt | Cutting the MCP server makes the case's premise stale (the case itself tests MCP protocol knowledge, not our server) | **Permit (user-approved 2026-08-14)** — rewrite the prompt to generic MCP protocol negotiation, keep case + grader | Done — prompt/expected_output rewritten, `datasetVersion` → 2026-08-14.5 |
| 2 | user-expansion | Homebrew/release machinery cut | Debate listed it; CHARTER.md (#98) says Homebrew is the shipped leg — direct conflict | **Decline (user-approved 2026-08-14)** — charter wins; Homebrew/release machinery kept as-is | Kept — no change |
| 3 | user-expansion | Spec shelf-ware consolidation depth | Debate said "archive to one decisions file"; 19 specs are ROADMAP-linked history | **Defer (no option selected 2026-08-14)** — specs untouched; separate workstream later | Deferred — follow-up task |

# Follow-up Tasks
- [ ] Resolve flagged decisions (coding-020, Homebrew, specs) — scope change #1–3
- [ ] After cuts: update CHANGELOG [Unreleased], run full verification
- [ ] PM artifacts relocation to `docs/governance/` — separate workstream (declared in CHARTER.md)
