# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/).

## [Unreleased]

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

## [0.6.0] - 2026-08-11

### Added
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
