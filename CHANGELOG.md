# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/).

## [Unreleased]

### Added
- **EVAL-13 — Cost-variance tolerance in the nightly gate** (EPIC-1,
  v0.11.0 "The gate that can't lie"): the cost gate is now flake-aware like
  the pass-rate gate (EVAL-2). A single run over 2× baseline is a reported
  **cost flake** (never a gate failure) — kills the 2026-08-14 coding-010
  single-run cost-spike false-fail class; a median over 2× baseline OR ≥2
  over-threshold runs still fails as a real regression. Scorecard JSON +
  markdown surface `costFlakes`. Hermetic tests: 48 score_run tests pass.
- **EVAL-14 — Benchmark provenance parity** (EPIC-1, v0.11.0): the
  ci-benchmark scorecard now carries a `provenance` block
  (`datasetVersion` / `agentModel` / `judgeModel` / `piVersion`) matching the
  live nightly surface (EVAL-3 schema) — closing the "provenance in **every**
  scorecard" DoD on both surfaces. Best-effort (missing env/tasks.json →
  "unknown", never a crash); provider-scorecard workflow records `PI_VERSION`;
  docs-drift guard pins both workflows. Golden fixture + 2 hermetic Go tests.
- **EVAL-15 — Split-seam verification** (EPIC-1, v0.11.0): `docs/
  benchmark-seam.md` pins the harness↔eval versioned contract (tasks.json +
  datasetVersion, score_run SCHEMA_VERSION, scorecard JSON shape, dataset/
  grader/reference/benchmark layout, known coupling, split trigger); new
  hermetic `test_benchmark_seam.py` (7 tests) pins the contract and runs a
  **self-containment dry-run** that inventories harness-root coupling into
  `eval/live-results/seam-report.json` (recorded honestly: 70 couplings / 14
  files today — the seam is real but not yet self-contained, which is exactly
  the handoff-kit fact the pi-bench split trigger needs).
- **EVAL-8 — Judge stabilization** (EPIC-1, v0.11.0): 3 easy concept cases
  (coding-002/004/013) converted judge→deterministic (new graders, oracle
  rule holds — references provably pass); judge-graded cases 8 → 5; remaining
  judge cases run **majority-of-3** (`EVAL_JUDGE_RUNS=3`). **Bugfix (scope
  change #1): judge-case passes never reached the gate** — `score_run` only
  collected `test_live_suite.py` nodeids, silently dropping the metrics
  layer's judge passes (they were unbaselined so nothing failed); now
  `test_live_metrics` nodeids are collected too, and the metrics layer only
  judges the 5 judge-graded cases (was all 50 — deterministic cases are
  double-counted otherwise).
- **REL-3 — Post-release brew verify** (EPIC-6, v0.11.0): new
  `scripts/verify-homebrew-formula.sh` + release.yml step installs the
  released formula into a temp prefix and asserts `pi-run version` == the
  tag — a tag shipped with a broken brew install now fails the release job
  (closes the fire-and-forget `TAP_PUSH_TOKEN` silent-distribution-break
  class).
- **REL-4 — Node-drift guard** (EPIC-6, v0.11.0): `pi-run doctor` now emits
  an informational warning when the resolved nvm Node major differs from the
  CI-pinned Node 22 LTS line (user machines running a Node CI never tested
  become visible); hermetic tests for the drift warning + no-warning-on-pin;
  docs-drift guard keeps the CI pin and the doctor reference in sync.
- **TAX-2 — Flag/env-var prune audit** (EPIC-6, v0.11.0): audited every
  README-documented flag and env var against actual usage. Outcome: **no dead
  flags or env vars** — but the watchdog env vars (`PI_SELF_HEAL`,
  `PI_STALL_TIMEOUT_SECS`, `PI_WATCHDOG_GRACE_SECS`) were real, tested Go
  knobs missing from the README env table (prose-only). Added the three rows +
  a docs-drift guard so future env vars can't silently miss the table.
- **EVAL-6 slice 1 — Agentic (tool-using) case family** (EPIC-1, v0.11.0):
  new `agentic` dataset category with 4 tool-using cases (coding-051..054) —
  the agent must use read-only tools (`--permission-mode plan`:
  read/grep/find/ls) to gather repo facts (go.mod version, grader count,
  provider count, Release-Milestones heading), graded deterministically on
  the tool-grounded value (a hallucinated answer fails). This is the first
  surface that can produce real `PI_SELF_HEAL` wedge observability (the data
  pump HEAL-2 is gated on) and it exercises harness differentiators, not just
  print mode. The 50-cap lint became "≥50 + per-category budgets" (new
  `agentic` (3,8) budget); taxonomy updated in Python + Go benchmark parser +
  tasks.json manifest. Oracle rule holds (references provably pass).

## [0.10.0] - 2026-08-14

### BREAKING
- **Removed `pi-run mcp-server` and the `PI_OTLP_ENDPOINT` OTLP telemetry
  export** (cut list, 2026-08-14): consumerless surfaces (zero external
  callers found in repo search per the persona debate). Anything calling
  `pi-run mcp-server` or setting `PI_OTLP_ENDPOINT` breaks. Deleted
  `internal/cli/mcp.go` (+ tests), `internal/cli/otel.go` (+ tests), Python
  contract tests, and `scripts/pdf2txt.sh`. Homebrew/release machinery kept
  (CHARTER: macOS/Homebrew is the shipped leg).
- **Go module path moved** `github.com/forrestthomas1/pi-harness` →
  `github.com/forrestbthomas/pi-harness` (OSS-1) — old import paths and
  `go install ...forrestthomas1...@latest` no longer resolve.

### Added
- **Version-truth stamp (REL-1)**: nightly + provider-scorecard CI builds now
  stamp `-ldflags "-X …/cli.Version=$(git describe --tags --always)"` so
  scorecards record a resolvable `piVersion`, never `dev` (the 2026-08-14
  version-milestone debate found every nightly scorecard was stamped `dev`).
- **Release milestones (v0.10.0 → v1.0.0)** — seven-persona version-milestone
  debate (2026-08-14) converged on a gated release plan: v0.10.0 "Identity,
  boundary, truth" (now), v0.11.0 "The gate that can't lie", v0.12.0
  "Resilience with receipts" (ride-or-slip on W5 Part C), v1.0.0 "The
  contract release" (gates, not dates: EPIC-1+6 DoDs, ≥14 green nightlies,
  EVAL-16 enforced, install-path CI-proven, consumer OR recorded earned-bar
  decision). Versioning policy codified (0.x minor = binary/contract delta;
  patches = fixes only; data releases ride the eval lane). New release-
  machinery items: REL-1 version-truth stamp, REL-2 changelog ledger repair,
  REL-3 post-release brew verify, REL-4 Node-drift guard, REL-5 baseline
  provenance. See `ROADMAP.md` §Release Milestones and `.pi/debate/
  milestones/synthesis.md` (local).
- **OSS-1 — Canonical install & identity alignment** (EPIC-6, 2026-08-14):
  Go module path `github.com/forrestthomas1/pi-harness` →
  `github.com/forrestbthomas/pi-harness` (matches the canonical repo URL —
  `go install github.com/forrestbthomas/pi-harness@latest` now resolves for
  a stranger); all imports/ldflags/current-state docs updated; CODEOWNERS
  handle → `@forrestbthomas`; new hermetic `test_canonical_identity` pins
  module-path == README URL == CODEOWNERS handle so it cannot drift back.
- **EPIC-6 — Repo maturity** (maturity-scan convergence 2026-08-14): new
  epic owning governance/community/debt work — OSS-1 canonical install+
  identity (~18), GOV-1 drift+charter-conformance guard, GOV-2 relocation+
  spec archive, OSS-2 contributor on-ramp, OWN-1 CODEOWNERS matrix, TAX-1
  usage evidence, TAX-2 flag prune, SECURITY ritual line, PORT-0 quarterly
  re-confirm. EPIC-4 parked (charter non-goal; PORT-0 only), EPIC-5 dissolved
  (DX-1/DX-2 → idea inbox), EPIC-1 trimmed (EVAL-10 removed, EVAL-7 parked,
  EVAL-5 capped at 50, +EVAL-13/14/15/16, EVAL-6 RICE 1.4, EVAL-12
  0-unbaselined DoD + standing cadence), HEAL-1 demoted to idea inbox, HEAL-2
  data-gated, COST-1 within-provider only. See
  `docs/knowledge-base/decision/2026-08-14-persona-debate-scope.md`.
- **Project charter + boundary contract** (scope debate 2026-08-14):
  [`CHARTER.md`](CHARTER.md) defines what this project is and explicitly what
  it is not — one product, the harness; the eval suite is its measurement
  layer, not a separate product today. North star adjusted: self-healing,
  measurable; "distributable" is earned (macOS/Homebrew shipped; further
  platforms or a pi-bench repo wait for a concrete consumer / second owner
  team). README headline is now single-product; `AGENTS.md` and `.pi/SYSTEM.md`
  carry the scope summary for agents. Decision record:
  `docs/knowledge-base/decision/2026-08-14-persona-debate-scope.md`.
- **Dataset growth — benchmark batch** (W10): +3 edit-based Docker
  benchmark tasks (fix-parse-dates, add-rate-limiter, fix-graceful-shutdown)
  — agents edit `src/`, hidden `tests/run.sh` grades the edited tree
  (patch-application grading); manifest updated, `datasetVersion` →
  `2026-08-14.4`. **EVAL-5 complete**: live suite 50 cases + 8 benchmark tasks.
- **Dataset growth Batch B** (W10): 35 → 50 live cases — +15
  (concept 036-040 judge-graded, shell/ops 041-045, negative-edge 046-048 +
  harness-routing 049-050 deterministic with verified hidden-test graders);
  lint exactly-50 + budgets updated; `datasetVersion` → `2026-08-14.3`.
- **Dataset growth Batch A** (W10): 20 → 35 live cases — +15 (code-gen 021-025,
  bug-fix 026-030, regression twins 031-035), each with a deterministic hidden-test
  grader and reference (schema lint proves every reference passes); category
  budgets and the exactly-35 lint updated; `datasetVersion` → `2026-08-14.2`.
  Authoring was parallelized across 3 subagent lanes with central verification
  (grader reference-pass + reject-side spot checks).
- **Self-heal events in the provider scorecard** (W9): `pi-run ci-benchmark`
  sets `PI_SELF_HEAL=1` in the provider-scorecard workflow and the Go
  scorecard JSON surfaces `selfHeal { nEvents, byKind }` (informational,
  backward-compatible `omitempty`), so weekly cross-provider runs are as
  observable as the nightly live eval.
- **Dataset versioning + scorecard provenance** (W8): `tasks.json` carries a
  guarded `datasetVersion` (bumped on every dataset change; enforced by the
  dataset schema lint); `score_run.py` records `provenance { datasetVersion,
  agentModel, judgeModel, piVersion }` in the summary + compact summary;
  the nightly captures `pi-run version` for provenance.
- **Flake-aware gate + evidence artifacts** (W7): nightly/provider-scorecard
  upload steps now run on every gate outcome (`if: always()`) and the nightly
  artifact includes `.pi/heal/events.jsonl` (EVAL-1); the gate treats a
  single failed run (1-of-N) as a flake — reported in the scorecard's
  `flakes` section, never a failure — while ≥2 failed runs still fail as a
  regression, and the nightly bumps to 5 runs per case (EVAL-2). Hermetic
  tests in `tests/test_score_run.py` + a workflow invariant in
  `tests/test_docs_drift.py`.
- **Scorecard self-heal observability** (W6): `nightly-live-eval.yml` live job
  sets `PI_SELF_HEAL=1` so watchdog/group-kill/recovery events are recorded
  to `.pi/heal/events.jsonl`; `score_run.py` surfaces `selfHeal {nEvents,
  byKind}` in the summary, compact `--json-summary`, and step summary
  (informational only — no gate change). Hermetic tests in
  `tests/test_score_run.py` run in the deterministic CI job.
- **Adopt upstream run-level timeout machinery** (W5 Part A): pin
  `npm:pi-subagents@0.45.1` → `0.48.0` so the harness finally installs the
  default wall-clock timeout for async subagent runs (our upstream #978/#979,
  shipped 0.47.0+). Lockfile re-resolved; config-check + pin test + subagent
  smoke verified.
- **`pi-run doctor` non-interactive env check** (BACKLOG #1): `doctor` now
  verifies the #59 hang-prevention launch env (`GIT_EDITOR`/`GIT_SEQUENCE_EDITOR`,
  `GIT_TERMINAL_PROMPT`, `PAGER`) is present and fails loudly if a required var
  is removed. `launchEnv` builds from a single `nonInteractiveEnv` source of
  truth; unit tests pin the exact set so the guard cannot be silently emptied.

## [0.9.2] - 2026-08-13

### Added
- **Real live-eval baseline** (W2): `eval/baselines/live-baseline.json` is now
  the honest baseline from the 2026-08-13 green nightly (17 deterministic
  cases × 3 runs, cheap tier + gpt-4.1-mini judge, overall 82.4%, four
  sub-1.0 cases recorded as-is) instead of an empty placeholder. The nightly
  gate now has real per-case regression bounds; verified to catch a
  deliberate regression (exit 1). `nightly-live-eval.yml` sets
  `PI_MODEL_TIER=cheap` so future re-baselines record the agent tier.
- **Pi-subagents pinned** (BACKLOG #2): `.pi/settings.json` pins
  `npm:pi-subagents@0.45.1`; `.pi/npm/package.json` + `package-lock.json` are
  now tracked (exact `0.45.1`) so a settings-triggered npm refresh cannot
  silently drift the subagent tooling. `pi-run config-check` verifies the pin.
- **Owner-only writer perms** (BACKLOG #3): escalation packets,
  `.pi/heal/events.jsonl`, the cost ledger, and the reset marker are now
  created 0700/0600 instead of 0755/0644, closing the world-readable
  secret-shaped artifact regression at the source. Go contract tests assert
  the permissions.

## [0.9.1] - 2026-08-13

### Added
- **Self-healing agent runs** (W1, spec
  `docs/superpowers/specs/2026-08-13-self-healing-design.md`):
  - `pi-run self-heal` detects in-progress git state and recovers it
    (`GIT_EDITOR=true git rebase --continue` when conflicts are resolved;
    reports `needs-attention` with conflict paths otherwise; `--abort` is
    explicit-only; worktree-safe detection, index.lock idempotency guard).
  - Process-group kill for timed-out runs: SIGTERM → 10s grace → SIGKILL
    reaps the whole tree (fixes the direct-child-only kill gap).
  - Output-stall watchdog for non-interactive spawns (`PI_STALL_TIMEOUT_SECS`,
    default 300s; chat excluded).
  - Escalation packet `.pi/heal/<timestamp>-report.json` + exit code `9`
    (watchdog terminated), honest resume handle for `--no-session` runs.
  - `PI_SELF_HEAL=1` observability events (`.pi/heal/events.jsonl`).
- **Contract test pin** for exit code `9` in `test_contract_exit_codes.py`.

### Changed
- `launchEnv` injects non-interactive env (`GIT_EDITOR=true`,
  `GIT_SEQUENCE_EDITOR=true`, `GIT_TERMINAL_PROMPT=0`, `PAGER=cat`) so child
  bash tools can never block on an interactive editor/pager (#59).
- Subagent wrappers forbid interactive editor/pager commands (#59).
## [0.9.0] - 2026-08-12

### Added
- **Deterministic eval hardening** (#40-#44): scorecard artifact is now
  byte-deterministic and fully CLI-owned — a `scorecardNow` time seam,
  `writeScorecardLatest` (the CI `cp` shell glue is gone), a golden JSON
  fixture with `-update`, and a strict `DisallowUnknownFields` round-trip
  (renamed fields fail loudly). Two drift guards: usage text ↔ command
  dispatch, and the MCP `providers` keyEnv list ↔ the Python mirror.
- **Python contract tests** against the real binary (37 hermetic tests):
  MCP JSON-RPC handshake over stdio (initialize → silent notification →
  tools/list → tools/call, stdout-is-sacred, EOF exits 0), `--model-tier`
  exit-code/argv contracts via a fake node+pi, OTel fake-collector
  (unset/one-POST/exit-attr/500/unreachable), `project-understand`
  determinism, and the `--exit-codes` table + ordering (usage 2 > missing
  key 3 > node 4). CI builds `pi-run` before running them (new
  `python-contract` job; nightly too) (#42-#44).
- **Live agent evaluation v2** (#45-#48): dataset grown 5 → 20 tasks with a
  schema-v2 JSONL (category, difficulty, grader, reference, regression tags)
  and a balanced category budget — code-gen, bug-fix, shell/ops, concept,
  **negative/edge (was zero)**, and harness-routing. Every task ships a
  reference that provably passes its deterministic grader (oracle rule,
  enforced by `test_dataset_schema.py`).
- **Nightly live-eval gate** (#47-#48): `nightly-live-eval.yml` is now a
  two-job split — a deterministic job (no key needed) plus a live job that
  **hard-fails with an explicit `::error::` when the provider key is
  missing** (a skipped eval is indistinguishable from a passed one), runs
  each case 3× (`EVAL_RUNS_PER_CASE`), and gates on per-case baselines
  (`eval/baselines/live-baseline.json`, 0.05 tolerance) via
  `eval/scripts/score_run.py` with cost-per-task regression (>2× baseline
  fails). `pytest-json-report~=1.5` added for the report JSON.
- **Cost-per-task metrics** (#45): the scorecard and nightly run JSON now
  carry `costPerTaskUsd`, `costPerSuccessfulTaskUsd` (div-by-zero guarded),
  `tokensPerTask`, `agentCostUsd`, `judgeCostUsd`. `chat`/`print` gain
  `--cost-mode <mode>` (chat|print|resume|backfill|benchmark|live-eval) so
  CI-tagged runs attribute spend explicitly; the ledger documents the
  `benchmark` and `live-eval` modes.
- **Unified task governance** (#47): `eval/datasets/tasks.json` manifest
  references both the live JSONL and the Docker benchmark seeds with a shared
  taxonomy; benchmark `task.json` files now carry
  `category`/`difficulty`/`grader` (validated by the Go parser and the
  manifest bijection tests).
- **Live metrics layer** (#48): `test_live_metrics.py` consumes the
  previously-dead `sample_cases` fixture — TaskCompletionMetric, a custom
  G-Eval rubric per category, and a deterministic `CodeQualityMetric` fast
  lane; judge cost via deepeval's `metric.evaluation_cost` (nightly-only).

### Changed
- The full hermetic pytest surface grew from ~28 to **100 tests**; CI runs a
  `python-contract` job that builds `pi-run` fresh before subprocess tests.

## [0.8.0] - 2026-08-12

### Added
- `pi-run project-understand [--out <dir>]`: deterministic Kiro-style
  project-understanding docs (product.md / tech.md / structure.md) generated
  from a repo checkout with stdlib Go only (no LLM). Language census with
  non-blank LOC, framework markers (go.mod, package.json, pyproject.toml,
  Dockerfile, GitHub Actions), 2-level pruned structure tree. Skips
  .git/node_modules/.venv/.pi/worktrees/bin/dist/build; deterministic output
  with no timestamps or absolute paths (#33).
- `pi-run mcp-server`: local READ-ONLY MCP server over stdio (spec 2025-03-26,
  line-delimited JSON-RPC 2.0). Three tools — `providers` (env-var NAMES only,
  never values), `cost` (aggregate spend, optional `since`), `benchmark_dry_run`
  (format validation, no Docker/keys). Tool failures are `isError` results, not
  JSON-RPC errors; -32700/-32601/-32600 handled; id echoed; notifications
  silent; exit 0 at clean EOF. Local-only and read-only by design (#34).
- OTel GenAI agent telemetry export: when `PI_OTLP_ENDPOINT` is set (e.g.
  http://localhost:4318), each `chat`/`print`/`resume` run emits one GenAI
  `invoke_agent` span to `<endpoint>/v1/traces` as OTLP/HTTP JSON (stdlib only,
  no protobuf). Best-effort: 2s timeout, one warning line, never changes the
  exit code; unset env is a complete no-op. Pins the Development-status GenAI
  semantic conventions; 16-byte trace id + 8-byte span id (#35).
- `pi-run chat|print --model-tier fast|balanced|cheap` (+ `PI_MODEL_TIER`
  env): cost-aware model routing. Design law: tier selection NEVER changes the
  provider and NEVER silently falls back — unknown or unmapped tiers are exit-2
  usage errors listing valid/available tiers; `--model-tier` + `--model` flags
  are mutually exclusive (exit 2); env tier + explicit `--model` → `--model`
  wins; `resume` rejects the flag and ignores the env. `pi-run providers` shows
  a TIERS column; `config-check` validates providers.json `modelTiers`;
  malformed tiers warn and fall back to built-in defaults (#36, #38).
- Docs: Agent Plugins 1.0 positioning (`docs/plugins.md`) + example manifest
  (`examples/plugins/manifest.example.json`) — Working Draft, no runtime
  support yet (#37). Cost-aware routing design spec
  (`docs/superpowers/specs/2026-08-12-cost-aware-routing-design.md`) (#36).
- `pi-run setup` fix: venv + requirements now resolve absolute from the repo
  root, and a missing `eval/requirements.txt` fails fast with a clear message
  instead of a confusing pip error (works from any cwd, incl. brew installs)
  (#31).
- `pi-run chat|print --permission-mode default|plan|acceptEdits|bypassPermissions`
  (+ `--read-only` alias, `PI_PERMISSION_MODE` env). Maps to Pi's real flags:
  plan → `--tools read,grep,find,ls`; bypassPermissions → `--approve`;
  default/acceptEdits → none. Per-agent permission policies in
  `.pi/agents/worker|reviewer|scout.md` (#29).
- Command hooks via `.pi/hooks.json`: pre-eval / post-eval / pre-chat hooks
  (cmd, timeoutSecs default 30, continueOnError); `pi-run hooks list` and
  `hooks run <event>`; missing config is a no-op (#29).
- Provider breadth: catalog grew from 7 to 17 providers — added azure,
  ollama, mistral, cohere, together, perplexity, fireworks, moonshot, xai,
  bedrock (baseURLs where applicable). No cross-provider auto-fallback;
  key-env lists stay in sync between Go and Python (#29).
## [0.7.0] - 2026-08-12

### Added
- `pi-run chat|print --permission-mode default|plan|acceptEdits|bypassPermissions`
  (+ `--read-only` alias, `PI_PERMISSION_MODE` env). Maps to Pi's real flags:
  plan → `--tools read,grep,find,ls`; bypassPermissions → `--approve`;
  default/acceptEdits → none. Per-agent permission policies in
  `.pi/agents/worker|reviewer|scout.md` (#29).
- Command hooks via `.pi/hooks.json`: pre-eval / post-eval / pre-chat hooks
  (cmd, timeoutSecs default 30, continueOnError); `pi-run hooks list` and
  `hooks run <event>`; missing config is a no-op (#29).
- Provider breadth: catalog grew from 7 to 17 providers — added azure,
  ollama, mistral, cohere, together, perplexity, fireworks, moonshot, xai,
  bedrock (baseURLs where applicable). No cross-provider auto-fallback;
  key-env lists stay in sync between Go and Python (#29).
  (Restored 2026-08-14 — this section was dropped in the v0.8.0 release-notes
  commit `02215b3`; content from `3c0a575`, REL-2 changelog ledger repair.)

## [0.6.0] - 2026-08-11

### Added
- `pi-run cost`: aggregate real spend from Pi session files (`usage.cost.total`
  per message, no price tables) — per provider/model table, `--json` machine
  output, `--since <date>` filter, `--reset` to archive the spend ledger and
  start a fresh budget period.
- `pi-run chat|print --max-budget-usd <n>` (env `PI_MAX_BUDGET_USD`): pre-flight
  budget check that refuses to launch when cumulative spend is already at/above
  the cap (exit code 6 = budget exceeded), plus an append-only spend ledger
  (`.pi/cost-ledger.jsonl`, gitignored) recording each run's provider, model,
  tokens, and cost.
- `pi-run eval --benchmark`: Docker-isolated, scored benchmark runner. Run the
  same task suite against any provider and compare results. Includes a seed
  benchmark suite under `eval/benchmarks/`, hermetic `--benchmark-dry-run`
  format validation (no Docker, no keys), per-task pass/fail grading with
  timing, and JSON reports under `eval/benchmark-results/` (gitignored). Exit
  code 7 when Docker is unavailable.
- `pi-run ci-benchmark`: provider scorecard in CI. Runs the benchmark suite
  against 2+ providers (`--providers openai,deepseek`, optional `--models`),
  writes a per-provider scorecard (`eval/benchmark-results/scorecard-<run>.json`:
  pass rate, cost, latency, tokens), and gates the build: exit 8 when any
  provider drops below `--fail-below <rate>` or regresses vs `--baseline <path>`
  (default tolerance 0.05), exit 6 when run cost hits `--max-budget-usd`.
  Includes `--runs <n>` median repeats for flaky runs and `--quick-profile` for
  cheap scheduled smoke runs.

## [0.4.3] - 2026-08-11

### Fixed
- Subagent children could hang forever on tool calls: a child `bash` call that
  starts a background process inheriting the terminal (no bash default timeout
  + no async wall-clock default) blocked the parent indefinitely. Added
  project agent wrappers (`.pi/agents/worker|reviewer|scout.md`) with a
  10-minute `timeoutMs` default and a working rule to always pass `timeout:`
  on `bash` calls and avoid background/inherit-terminal commands; documented
  the required explicit `timeoutMs` convention (#22).
- Upstream: filed pi-subagents issue #978 and PR #979 (default async timeout)
  with deterministic reproductions of the hang.
## [0.4.2] - 2026-08-10

### Changed
- Build toolchain bumped from Go 1.21 to Go 1.26.5 (CI, go.mod, README).
  Go 1.21 is past end-of-life; 1.26 includes security fixes and performance
  improvements (new GC). No CLI behavior change.
- Adopted modern Go language/stdlib features: `for i := range n` (1.22),
  `errors.AsType` (1.26), `slices.Index` (1.21+), `strings.Builder` and
  `bytes.Cut` via `go fix` modernizers; CI now runs the race detector.
## [0.4.1] - 2026-08-10

### Fixed
- `pi-run` repo-root resolution now falls back to the current working directory when the executable is not inside a harness checkout (e.g. a Homebrew-installed binary under `<prefix>/Cellar/...`), so `pi-run config-check` and friends work correctly from a project directory (#18).
- Cross-language `PI_SECRET_BACKEND` contract: Go now accepts the `bw` alias for bitwarden, matching the Python side; a new contract test shells out to the stdlib-only `eval/secret_backend.py` to keep both languages in sync (#17).
- `pi-run install` flag parsing is now tested at the command level (`--force`, `--help`, unknown flags) (#17).
- Secret resolution never echoes values: the eval-side availability probe reports via exit code instead of printing key material (#17).

### Changed
- Dependency bumps: `pypdf ~= 6.15`, `pytest-timeout ~= 2.4` (#8, #10).
- `eval/conftest.py` delegates `get_secret` to the shared `eval/secret_backend.py` (single identifier contract).
## [0.4.0] - 2026-08-10

### Added
- Pluggable secret manager backend (`PI_SECRET_BACKEND`: bitwarden, 1password, env-only) via a `SecretBackend` interface.
- Embedded seven-provider table (openai, openrouter, deepseek, anthropic, gemini, groq, local) with `PI_RUN_PROVIDERS_FILE` override and `baseURL` support.
- `pi-run resume` — continue the most recent session.
- Per-test timeout and no-key skip guard for the eval suite (`pi-run eval` never blocks on a locked vault).
- Build-time version via `-ldflags` (default `dev`).
- Consistent `pi-run: <cmd>: <reason>` error messages + `--exit-codes`.
- Cross-platform release workflow (`scripts/build-release.sh` + GitHub Actions + Homebrew tap auto-update).
- Issue/PR templates and this changelog.
- Subagent-driven development docs (pi-subagents).
- Least-privilege CI permissions + pip cache.

### Fixed
- `pi-run install` never silently overwrites a non-harness `~/bin/pi-run` (adds `--force`); symlink replacement is atomic.
- Explicit provider-table override failures warn with path+reason instead of failing silently.
- Secret-manager subprocesses bounded to 30s so a hung helper cannot block indefinitely.
- Python secret backend rejects unknown values instead of silently falling back to Bitwarden.
- `pi-run doctor` distinguishes required prereqs from optional key/backend status (no-key setup exits 0).
- `pi-run eval` argument parsing is strict: `--help`, unknown-flag diagnostics, `--` pytest pass-through.
- `pi-run clean` reports what it removed.
- Bootstrap script detects missing `nvm` and guides install; PATH export guidance added.
- README aligned with actual behavior (Node policy, env table, eval env-only keys, no quarantine `xattr` workaround).
- Eval collection no longer probes the secret manager (env-only `has_api_key`).

## [0.3.0] - 2026-08-09

### Added
- Session resume (`pi-run resume`), eval timeouts/skip guard, build-time version, `--exit-codes`.
- Release automation and Homebrew tap distribution.
- Contribution surface (issue/PR templates, changelog, subagent docs).

## [0.2.0-pre] - 2026-08-09

### Added
- Data-driven provider table (`providers.json`) + `pi-run providers`.
- DeepEval judge provider flexibility (`DEEPEVAL_MODEL`).
- Community docs, anti-lock-in guide, examples.
- GitHub Actions CI, bootstrap script, MIT license, module rename.
