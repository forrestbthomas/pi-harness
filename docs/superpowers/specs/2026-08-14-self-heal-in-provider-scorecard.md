# Spec — EVAL-4: Self-Heal Events in the Provider Scorecard — W9

**Date:** 2026-08-14 · **Status:** DRAFT (awaiting user approval at scope-lock gate)
**Source:** BACKLOG EVAL-4 (0.60, EPIC-1) — bet on by user (2026-08-14).

## Goal

Extend W6's self-heal observability to the **provider scorecard** (`pi-run ci-benchmark`): the benchmark workflow records `.pi/heal/events.jsonl` and the Go scorecard JSON surfaces a `selfHeal` block, so weekly cross-provider runs are as observable as the nightly live eval.

## Problem Statement

W6 wired `PI_SELF_HEAL=1` + a `selfHeal { nEvents, byKind }` block into the nightly (Python `score_run.py`). The weekly provider scorecard (`provider-scorecard.yml` → `pi-run ci-benchmark`, Go `internal/cli/scorecard.go`) neither sets `PI_SELF_HEAL` nor reports self-heal events — a stalled benchmark run leaves no scorecard-visible trace, and the two evals are inconsistent.

## Context From Code

- `internal/cli/scorecard.go`: `scorecard` struct (line ~100) is the on-disk artifact; built in `runScorecard` (`sc := scorecard{...}`, line 689), written by `writeScorecard` (line 722).
- Events writer: `logSelfHealEvent(dir, kind, detail)` (`internal/cli/escalation.go:223`) appends JSONL to `<dir>/.pi/heal/events.jsonl` when `PI_SELF_HEAL=1`; `selfHealEnabled` (`watchdog.go:53`).
- `provider-scorecard.yml`: env has `PI_NODE_VERSION` etc.; `pi-run ci-benchmark --providers openai,deepseek --fail-below 0.8 --max-budget-usd 5.0`.
- Tests: `internal/cli/scorecard_test.go` (table-driven, stdlib) — the home for a reader test.

## In Scope

- **`internal/cli/scorecard.go`**: add `scorecardSelfHeal { NEvents int; ByKind map[string]int }`; add `SelfHeal *scorecardSelfHeal` (`json:"selfHeal,omitempty"`) to `scorecard`; populate from `<root>/.pi/heal/events.jsonl` via a best-effort `readSelfHealEvents(root)` (missing/malformed → zero counts; never fatal).
- **`.github/workflows/provider-scorecard.yml`**: add `PI_SELF_HEAL: '1'` to the job env.
- **`internal/cli/scorecard_test.go`**: unit tests — reader missing file → zeros; counts by kind; malformed lines skipped; scorecard JSON contains `selfHeal`.
- **Docs/records:** CHANGELOG `[Unreleased]`; ROADMAP W9 row; STATUS; BACKLOG (EVAL-4 → active); SCOPE.md; spec.

## Out Of Scope

- Gating on self-heal counts (informational only, like W6).
- Any change to `ci-benchmark` grading, baselines, or budget logic.
- Python `score_run.py` selfHeal (already shipped in W6).
- EVAL-5 dataset growth (separate ticket in the same bet).

## Constraints

- Go stdlib-only; tests hermetic (no keys/network/Docker).
- The scorecard JSON schema addition must be backward-compatible (`omitempty`; golden fixture may need a re-gen if the scorecard struct gains a field — verify `TestScorecardGoldenJSON`).

## Assumptions

- `ci-benchmark` runs from the repo root, so `<root>/.pi/heal/events.jsonl` is the right path (same contract as the nightly).
- Whether events actually fire during benchmark runs is empirical; a `selfHeal: 0` block on a healthy run is a correct result.

## Blocking Questions

None.

## Acceptance Criteria

Gherkin:

```gherkin
Scenario: Provider scorecard surfaces self-heal events
  Given .pi/heal/events.jsonl exists with N events by kind K
  When ci-benchmark writes the scorecard
  Then the scorecard JSON contains selfHeal { nEvents: N, byKind: { K: count } }

Scenario: Missing events are tolerated
  Given no events file
  When ci-benchmark writes the scorecard
  Then selfHeal is omitted or zero and the run is unaffected

Scenario: Benchmark workflow records events
  Given provider-scorecard.yml
  When the job runs
  Then PI_SELF_HEAL=1 is in the job env
```

Non-behavioral: Go unit tests; golden fixture updated if the schema grows; docs-drift + full CI green.

## Validation Plan

- `go build ./... && go vet ./... && go test ./internal/cli/ -run 'Scorecard|SelfHeal'`.
- Full `go test ./...` + the deterministic pytest set + docs-drift.

## Execution Plan (ordered)

1. RED: failing Go tests (reader + scorecard JSON selfHeal).
2. GREEN: `readSelfHealEvents` + struct field + populate in `runScorecard`; add `PI_SELF_HEAL` to provider-scorecard.yml.
3. Docs: CHANGELOG, ROADMAP W9, STATUS, BACKLOG.
4. Verification-before-completion.
5. Land via PR (`BACKLOG EVAL-4 — self-heal events in provider scorecard`), ff-only sync.
6. Post-merge reconciliation + W9 DoD close.

## Open Questions

- None blocking.

## Self-Check

Goal user-visible ✅ · problem stated ✅ · context from code present ✅ · in/out scope non-conflicting ✅ · constraints actionable ✅ · assumptions visible ✅ · blocking questions none ✅ · acceptance criteria observable/testable ✅ · validation plan concrete ✅ · execution order matches EPIC-1 sequence ✅
