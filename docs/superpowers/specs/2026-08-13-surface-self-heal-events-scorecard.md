# Spec — Surface `PI_SELF_HEAL` Events in the Scorecard + Enable in CI — W6

**Date:** 2026-08-13 · **Status:** SHIPPED — W6 (2026-08-14, PR #83)
**Source:** BACKLOG #1 (RICE 1.40, Enabler, ~0.5 pw) — promoted to ROADMAP W6 by user decision.

## Goal

Make the self-healing layer observable in the live-eval scorecard and turn on its event collection in CI, so we accumulate evidence about how often the watchdog/group-kill/recovery paths actually fire in real agent runs — the data that unblocks BACKLOG #4/#5 (auto-open issue on N self-heals, model-catalog refresh triggers).

## Problem Statement

W1 shipped the self-healing layer (`PI_STALL_TIMEOUT_SECS`, process-group kill, escalation packets, exit 9) and `PI_SELF_HEAL=1` writes `.pi/heal/events.jsonl` (`logSelfHealEvent` in `internal/cli/escalation.go`). But:

1. **CI never enables it** — `nightly-live-eval.yml` does not set `PI_SELF_HEAL`, so no events are recorded during real agent runs.
2. **The scorecard doesn't surface it** — `eval/scripts/score_run.py` reports pass rate, cost, tokens, latency, and gate status, but no self-heal event count. A night where the watchdog killed wedged runs would look identical to a clean night in the scorecard.

## Context From Code

- Event writer: `logSelfHealEvent(dir, kind, detail)` (`internal/cli/escalation.go:223`) — appends `{"ts","kind","detail"}` JSON lines to `<dir>/.pi/heal/events.jsonl` (owner-only 0700/0600), no-op unless `PI_SELF_HEAL=1` (`selfHealEnabled`, `internal/cli/watchdog.go:53`). Callers: `internal/cli/pi.go:245` (`group-kill`, wall-clock timeout), `pi.go:252` (`group-kill`, output stall).
- Live suite: `eval/conftest.py:58` `run_pi_print` runs `pi-run print` with `cwd = repo root`, so events land at `<repo>/.pi/heal/events.jsonl` — same directory pattern as the cost ledger the scorer already reads indirectly.
- Scorer: `eval/scripts/score_run.py` — stdlib-only, hermetic (reads report + baseline JSONs only). Summary builder `build_summary` (line 386), compact `build_compact_summary` (line 408: totals + gate + unbaselined). Writes `--out` JSON and `--json-summary`; appends a human line to `GITHUB_STEP_SUMMARY`.
- CI: `nightly-live-eval.yml` — `deterministic` job runs `test_score_run.py` hermetically; `live` job runs the suite with `PI_MAX_BUDGET_USD`, `PI_MODEL_TIER`, etc. in `env:`.

## In Scope

- **CI enable:** `.github/workflows/nightly-live-eval.yml` — add `PI_SELF_HEAL: '1'` to the `live` job env so real agent runs record events.
- **Scorecard surface:** `eval/scripts/score_run.py` —
  - Parse `<repo>/.pi/heal/events.jsonl` (optional `--heal-events <path>` override for hermetic tests; default `repo_root()/".pi/heal/events.jsonl"`).
  - Count events by `kind`; tolerate missing file (0 events) and malformed lines (skip, best-effort like `_ledger_entries`).
  - Add a `selfHeal` block to the full summary and the compact `--json-summary`: `{ nEvents, byKind: { kind: count } }`.
  - Append a one-line self-heal summary to the `GITHUB_STEP_SUMMARY` output.
  - **Informational only — no gate change:** self-heal counts must not fail the gate or alter baseline math.
- **Tests (hermetic):** `eval/tests/test_score_run.py` — new cases: missing events file → 0 events; well-formed file counted by kind; malformed lines skipped; `--heal-events` override honored; summary + compact summary both carry the block.
- **Docs/records:** `docs/` summary-format note if one exists; `CHANGELOG.md` `[Unreleased]` entry; ROADMAP W6 row + STATUS + BACKLOG promotion note; SCOPE.md contract.

## Out Of Scope

- Auto-open issue on N self-heals (BACKLOG #5 — depends on this data).
- Nightly archive upload of events (BACKLOG #3).
- Any watchdog/self-heal behavior change or new event kinds (W1 shipped).
- Making self-heal counts a gate/threshold (data collection first).
- Reading events from anywhere other than the repo-root path (the live suite cwd contract).

## Acceptance Criteria

Gherkin:

```gherkin
Scenario: CI records self-heal events during the live eval
  Given nightly-live-eval.yml live job
  When the live suite runs pi-run print
  Then PI_SELF_HEAL=1 is in the live job env
  And any watchdog/group-kill events are appended to <repo>/.pi/heal/events.jsonl

Scenario: Scorecard surfaces the event count
  Given a report + baseline and a heal events file with N events by kind K
  When score_run.py runs
  Then the summary and --json-summary contain selfHeal { nEvents: N, byKind: { K: count } }
  And the gate result is unchanged by the presence/absence of events

Scenario: Missing or malformed events are tolerated
  Given no events file (or malformed lines)
  When score_run.py runs
  Then selfHeal is { nEvents: 0, byKind: {} } and the run still scores normally
```

Non-behavioral:

- Hermetic: no network/keys in tests; `--heal-events` override drives all test fixtures.
- `test_score_run.py` stays in the `deterministic` CI job list (already present).
- `go build ./...`, `go vet ./...`, `go test ./...`, and the eval pytest set stay green.

## Validation Plan

- `eval/.venv/bin/python -m pytest eval/tests/test_score_run.py -v` (new + existing cases).
- Full deterministic CI set locally: the `deterministic` job's pytest list.
- `pi-run config-check` unaffected (no config surface change).
- Manual hermetic run: `score_run.py --report <fixture> --baseline <fixture> --heal-events <fixture>` → summary contains the block.
- Docs drift test (`eval/tests/test_docs_drift.py`) still green.

## Execution Plan (ordered)

1. Write failing tests in `test_score_run.py` (TDD red).
2. Implement `score_run.py` self-heal parsing + summary/step-summary additions (green).
3. Add `PI_SELF_HEAL: '1'` to `nightly-live-eval.yml` live job env.
4. Update docs: CHANGELOG `[Unreleased]`; ROADMAP W6 row; STATUS; BACKLOG promotion note.
5. Land via PR (`BACKLOG #1 — surface PI_SELF_HEAL events in scorecard`), ff-only sync.
6. Verify the merge and close W6 DoD in a docs-reconciliation pass.

## Self-Check

Goal user-visible ✅ · problem stated ✅ · context from code present ✅ · in/out scope non-conflicting ✅ · acceptance criteria observable/testable ✅ · validation plan concrete ✅ · execution order matches the approved backlog item ✅
