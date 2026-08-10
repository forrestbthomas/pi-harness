# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/).

## [Unreleased]

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
