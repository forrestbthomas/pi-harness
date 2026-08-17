# Spec — OBS-1 `pi-run sessions` + OBS-2 chat-path heal events — W11

**Date:** 2026-08-16 · **Status:** DRAFT → ACTIVE (promoted W11, EPIC-2)
**Source:** BACKLOG OBS-1/OBS-2 (EPIC-2) — promoted by owner 2026-08-16;
debate: `.pi/debate/resilience-observability/synthesis.md`; decision:
`docs/knowledge-base/decision/2026-08-16-observability-first-resilience.md`.

## Goal

Make the harness's observability cover **all modes**, closing the chat-path
blind hole: (1) a `pi-run sessions` fleet view (live = recent mtime + recent
sessions) over the existing session transcript seam, and (2) chat-path heal
events — aggregated `connection-flap` events written to the existing
`.pi/heal/events.jsonl` / `selfHeal` seam when a session suffers repeated
transport failures.

## Problem Statement

The observability audit found **asymmetric observability**: cost
(`.pi/cost-ledger.jsonl`) and transcripts (`.pi/sessions/*.jsonl`) are rich,
but the *heal* surface is gated to the non-interactive watchdog path
(`print`/`benchmark` with `PI_SELF_HEAL=1`). Heal events on disk: **0**. The
author ran 3 chat sessions across providers that suffered repeated
`Connection error.` episodes while the harness measured `self-heal events: 0`
— a false statement about chat mode. The most failure-prone mode is the
least-observed mode.

## Context From Code

- **Chat spawn does not tee output.** `execPiDir` (`internal/cli/pi.go:183`)
  wires `cmd.Stdout=os.Stdout`, `cmd.Stderr=os.Stderr` — the harness cannot see
  chat-path failures at spawn time. Only `execPiDirTimeout` (print/benchmark)
  tees stdout for the watchdog. **Therefore OBS-2 must observe after the fact,
  from the transcript — not by teeing the interactive TUI.**
- **Transcripts already record the failure class.** `.pi/sessions/*.jsonl`
  (schema v3: `{"type":"session","version":3,"id":...,"timestamp":...,"cwd":...}`
  then event lines) contains `type:"message"` events with
  `message.stopReason:"error"` (2,432 occurrences across history) and
  `message.errorMessage` (3,189), including the literal `"Connection error."`
  — the exact string observed in the wild. Non-connection failures appear too
  (e.g. `401 {"error":{"type":"authentication_error",...}}`, `502 ... "fetch
  failed"`).
- **The heal seam exists and is scorecard-fed.** `logSelfHealEvent(dir, kind,
  detail)` (`internal/cli/escalation.go:226`) appends JSONL to
  `<dir>/.pi/heal/events.jsonl` when `selfHealEnabled()` (`PI_SELF_HEAL=1`);
  the scorecard surfaces it as `selfHeal {nEvents, byKind}` (`scorecard.go`,
  `readSelfHealEvents`). **OBS-2 must write through this same file/format** so
  chat-path events surface in `selfHeal` without any scorecard change.
- **Command template:** `runProviders` (`internal/cli/app.go:563`) is the
  reader-command shape (stdlib, tab-separated, exit 0). `app.go` dispatch is a
  flat `switch args[0]` (line 100–150) — a new `case "sessions"` slot.
- **Session file naming:** `<ISO-timestamp>_<id>.jsonl`; file mtime is the
  liveness signal (a session actively streaming is touched continuously).
  `session_info` events carry a human `name`.

## Design Decision — OBS-2 is a transcript reader, not a TUI tee or Pi extension

The hard part of OBS-2 was "how does the harness see chat-path failures?"
Three options were weighed:

| Option | Verdict |
|---|---|
| **Tee chat stdout/stderr at spawn** (like the watchdog tees print) | **Rejected.** Chat is an interactive TUI; teeing risks breaking raw-mode rendering, and errors may be rendered by the TUI layer rather than plain stdout. |
| **Pi extension that reports provider errors** | **Rejected.** Adds a TS extension surface, a second test harness, and coupling to Pi's extension API — against "stdlib-only core" and "no installs." |
| **Transcript reader (chosen)** | **Adopted.** The session seam already records `stopReason:"error"` + `errorMessage` (incl. `"Connection error."`). OBS-2 scans the same files OBS-1 lists, applies the aggregation boundary, and writes `connection-flap` events through the existing `events.jsonl` seam. One reader module, two commands, zero new schema, zero new deps. |

This also keeps OBS-1 and OBS-2 in the same command family (`pi-run sessions`),
matching the debate's "one reader module" convergence and Skeptic's "no second
observability system" condition.

## In Scope

- **`internal/cli/sessions.go` (new):**
  - `runSessions(args)` — list sessions: for each `.pi/sessions/*.jsonl`,
    parse the first `session` line for `id`, `timestamp`, `cwd`, plus file
    mtime; render tab-separated rows. Flags: `--recent <duration>` (default
    view: last 24h), `--active` (mtime within a liveness window, default 5m),
    `--json` (parseable), `--help`.
  - `--heal` mode: scan the session transcripts (recent window), find
    connection-class error events, apply the aggregation boundary, and write
    `connection-flap` events to `.pi/heal/events.jsonl` (same file/format as
    `logSelfHealEvent`, **bypassing the `PI_SELF_HEAL` env gate because the
    scan is an explicit user action** — spec'd deviation, see Constraints).
  - Connection-class classifier: mirror Pi's `isRetryableAssistantError`
    *transport* subset — `connection.?error|connection.?lost|network.?error|
    socket hang up|fetch failed|reset before headers|other side closed|
    upstream.?connect|timed? out|timeout` — applied to `message.errorMessage`
    on `stopReason:"error"` messages. Non-transport errors (401 auth, quota)
    are **not** connection-flaps.
  - Aggregation boundary: **≥ N connection-class errors within window W per
    session id → exactly 1 `connection-flap` event** (`kind:"connection-flap"`,
    detail includes session id + count). Defaults N=3 (mirrors Pi's
    `retry.maxRetries`), W=10m; env-tunable (`PI_HEAL_FLAP_THRESHOLD`,
    `PI_HEAL_FLAP_WINDOW`).
- **`internal/cli/app.go`:** dispatch `case "sessions": return runSessions(args[1:])`; usage text row.
- **`internal/cli/sessions_test.go` (new):** hermetic tests (below).
- **Docs/records:** README one row for `pi-run sessions`; `docs/reference.md`
  command row; CHANGELOG `[Unreleased]`; ROADMAP W11 DoD checkboxes; STATUS;
  BACKLOG (OBS-1/OBS-2 → active); SCOPE.md; this spec.

## Out Of Scope

- **No dashboard, metrics endpoint, or OTel** (charter non-goal #4; CLI-only DoD).
- **No live tail / daemon / watch loop** — `sessions` is an on-demand command;
  "live" means recent-mtime view, not a background watcher.
- **No teeing the chat TUI, no Pi extension** (rejected above).
- **No HEAL-1 (liveness heartbeat) or HEAL-4 (stuck detection)** — they remain
  deferred and *consume* OBS-2's data later; this spec does not build them.
- **No changes to `score_run.py`** — `selfHeal {nEvents, byKind}` already
  reads `events.jsonl`; OBS-2 feeds it.
- **No auto-restart / auto-resume** (HEAL-3 decision ticket, not code).
- **No changes to the chat spawn path or the watchdog.**

## Constraints

- Go stdlib-only; tests hermetic (no keys/network/Docker); no new deps.
- `events.jsonl` written with the **same format and permissions** as
  `logSelfHealEvent` (0600 file, 0700 dir, `{"ts":...,"kind":...,"detail":...}`
  lines) so `readSelfHealEvents`/scorecard picks it up unchanged.
- **Spec'd deviation:** the `--heal` scan bypasses the `PI_SELF_HEAL` env gate
  (unlike ambient watchdog events) because it is an explicit user action; it
  still writes the *same file/format*. This is deliberate and recorded here.
- Backward-compatible: new command only; no existing command/flag/exit-code
  changes; unknown `sessions` flags → exit 2 usage error (house style).
- Malformed session lines and unreadable files are **skipped, never fatal**
  (mirror the `readSelfHealEvents` tolerance).

## Assumptions

- Sessions live in `<root>/.pi/sessions/*.jsonl` (the `sessionDir` setting).
- File mtime is a trustworthy liveness signal (Pi touches the file while
  streaming; a session idle > window is "not active").
- Defaults N=3/W=10m are a starting contract; tuning from real OBS-2 data is a
  follow-up (data-gate discipline), not a reason to hold the spec.

## Blocking Questions

None. (Aggregation defaults are spec'd; the scorecard needs no change.)

## Acceptance Criteria

Gherkin:

```gherkin
Scenario: Fleet view lists active and recent sessions
  Given .pi/sessions/ contains transcripts with mixed mtimes
  When pi-run sessions runs
  Then it prints one tab-separated row per session (id, timestamp, mtime-relative age, cwd)
  And --active shows only sessions whose file mtime is within the liveness window
  And --recent 24h is the default view

Scenario: Connection-flap aggregation boundary
  Given a session transcript with 3 connection-class errors within a 10-minute window
  When pi-run sessions --heal runs
  Then exactly 1 connection-flap event is appended to .pi/heal/events.jsonl
  And its kind is "connection-flap" and its detail names the session and count

Scenario: Below-threshold errors produce no event
  Given a session transcript with 2 connection-class errors within a 10-minute window
  When pi-run sessions --heal runs
  Then no connection-flap event is appended for that session

Scenario: Non-transport errors are not counted
  Given a session transcript with only 401 authentication errors
  When pi-run sessions --heal runs
  Then no connection-flap event is appended

Scenario: Malformed transcripts are tolerated
  Given a session file with unparseable lines
  When pi-run sessions runs
  Then it skips the malformed lines and exits 0

Scenario: Canary — events surface through the heal seam
  Given a session transcript with N connection-class errors within the window
  When pi-run sessions --heal runs
  Then .pi/heal/events.jsonl contains a connection-flap line in the same
    {"ts","kind","detail"} format readSelfHealEvents consumes
```

## TDD Order (red → green)

1. `sessions_test.go`: fleet-view tests (list/active/recent/json) — red → green.
2. `sessions_test.go`: connection-flap aggregation boundary + below-threshold +
   non-transport + malformed — red → green.
3. `sessions_test.go`: canary (event written in seam format) — red → green.
4. `go build ./...`, `go vet ./...`, `go test ./...` green; `test_pm_drift.py`
   + `test_docs_drift.py` green after docs edits.
5. PR citing `BACKLOG OBS-1/OBS-2 — W11`.