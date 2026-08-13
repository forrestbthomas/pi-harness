# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/).

## [Unreleased]

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
