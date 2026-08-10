# Plan: Address Multi-Axis Review Findings

**Date:** 2026-08-10
**Base:** `main` @ `8178599`
**Source:** 5-axis review sweep (gpt-5.6-terra) — full reports in `.review-sweep/` (gitignored)
**Scope note:** The security **Critical** (world-readable transcripts in `.pi/sessions/` + `.pi-subagents/`) is **skipped** per user decision — both dirs are gitignored, nothing sensitive is tracked. The remaining 24 Important + 10 Minor findings are consolidated into the tasks below.

---

## Task 1 — Make eval truly offline (no vault probing during collection)
**Addresses:** Security-Imp #1, Performance-Imp #1, Ergonomics-Imp #2, Extensibility-Imp #5, Usability-Imp #6

**Problem:** `eval/conftest.py:167-170` `has_api_key()` calls `get_secret()` per provider, which shells out to `bw_get`/`op` (30s timeout each) at pytest import and in `skipif` decorators — even for `pi-run eval` deterministic runs that already decided env-only.

**Fix:**
1. `eval/conftest.py:167-170` — change `has_api_key()` to use the existing env-only `any_provider_key_env()`:
   ```python
   def has_api_key() -> bool:
       """True if any supported provider key is present in the environment (presence only)."""
       return any_provider_key_env()
   ```
2. Add a hermetic test (`eval/tests/test_secret_resolution.py` or new) asserting `has_api_key()` returns False when all 6 keys are unset even with a fake `BW_GET` on PATH (proving no subprocess call).

**Verification:** `eval/.venv/bin/python -m pytest eval/tests/test_secret_resolution.py -q` passes; a strace/probe confirms no `bw_get`/`op` during collection.

---

## Task 2 — Bootstrap/onboarding reliability
**Addresses:** Usability-Imp #1-3, Ergonomics-Imp #5, Security-Imp #2

**Problem:** `scripts/bootstrap.sh` builds `bin/pi-run` but prints `pi-run chat` (not on PATH); nvm-missing fails opaquely; auto-installs latest global node/pi.

**Fix:**
1. `scripts/bootstrap.sh`:
   - After building, detect nvm: if `~/.nvm/nvm.sh` missing → print guided install (the curl command) and exit 1 (not raw `nvm: command not found`).
   - Print a copy-pastable PATH export for `$ROOT/bin` (or invoke `bin/pi-run install`) and verify the command runs.
   - Announce global installs (`nvm install`, `npm install -g pi`) with explicit "this installs globally" messaging; keep auto-install but make it loud.
2. `README.md:72-94` — use `bin/pi-run` in source-install instructions consistently, or document the PATH export.

**Verification:** fresh-clone simulation: `bash scripts/bootstrap.sh` succeeds on a machine with nvm; on a machine without nvm, prints the guided install and exits nonzero with a clear message.

---

## Task 3 — Hardening `pi-run install` (non-destructive)
**Addresses:** Ergonomics-Critical #1

**Problem:** `internal/cli/app.go:184-190` — `_ = os.Remove(link)` silently deletes any existing `~/bin/pi-run` (file or foreign symlink).

**Fix:**
1. Before removing, `os.Lstat(link)`:
   - If it's a symlink pointing at this installation's target → replace atomically.
   - If it's a symlink to something else, or a regular file → refuse with a clear message + `--force` escape hatch.
2. Add test: `TestRunInstallRefusesForeignSymlink` (hermetic, temp HOME).

**Verification:** `go test ./internal/cli/ -run TestRunInstall -v` passes; manual: create foreign symlink, `pi-run install` refuses.

---

## Task 4 — Strict per-command argument parsing + complete help
**Addresses:** Ergonomics-Imp #1, Usability-Imp #4, Extensibility-Minor #1, Ergonomics-Imp #4

**Problem:** `app.go:73-75` — `eval --help` runs full eval; unknown flags ignored; `--provider`/`--model` dangling. `app.go:37-39` help lists only 3 providers (7 configured).

**Fix:**
1. `app.go` `Run()`: parse each subcommand's flags strictly:
   - `eval`: support `--help`, `--quick`, and a `--` pass-through to pytest (e.g. `pi-run eval -- tests/test_x.py::test_y`).
   - Unknown flags/trailing args → usage error, exit 2.
2. Help text: replace hardcoded provider list with `--provider <name> (see pi-run providers)` or dynamic list from `LoadProviders`.
3. README command table: add `resume`, `providers`, `--exit-codes`, `eval --help`.
4. Tests: `TestRunEvalHelpDoesNotRunSuite` (argv-recording fake python), `TestRunEvalUnknownFlagExit2`, `TestRunEvalPytestPassThrough`, `TestUsageMentionsAllCommands`.

**Verification:** `go test ./internal/cli/ -run 'TestRunEval|TestUsage' -v` passes; `pi-run eval --help` prints usage, exits 0; `pi-run eval --bogus` exits 2.

---

## Task 5 — Portable provider table + baseURL application
**Addresses:** Extensibility-Imp #1-3

**Problem:** Released binaries fall back to only 3 of 7 providers when `providers.json` absent; `baseURL` parsed never applied; `LOCAL_API_KEY` omitted from eval key lists.

**Fix:**
1. `internal/cli/providers.go` — embed the full 7-provider table as the fallback (in code or embedded JSON), so released binaries get all providers.
2. Add `PI_RUN_PROVIDERS_FILE` env override for external provider tables (documented).
3. Apply `Provider.BaseURL`: in `runLaunch`, if `baseURL` set, pass the appropriate env var to the Pi child (e.g. the OpenAI-compatible base URL var for the local provider). Add hermetic test asserting the env is set only when configured.
4. Sync key lists: add `LOCAL_API_KEY` to `internal/cli/eval.go` `supportedProviderKeyEnvs` and `eval/conftest.py` `SUPPORTED_PROVIDER_KEYS`.

**Verification:** `go test ./internal/cli/ -run 'TestProviders|TestRunPrint' -v` passes; built binary without providers.json still shows 7 providers in `pi-run providers`.

---

## Task 6 — Go secret-backend subprocess timeouts
**Addresses:** Security-Imp #4, Extensibility-Imp #4 (partially)

**Problem:** `internal/cli/secret.go:60-110` — `bw_get`, `op read`, `op account list`, `Status()` use unbounded `exec.Command`; Python has 30s.

**Fix:**
1. `internal/cli/secret.go` — use `exec.CommandContext` with a bounded timeout (e.g. 30s, matching Python). On timeout, return a backend-agnostic error naming only the backend.
2. Add test: fake slow `bw_get` script + short timeout → error, no hang.

**Verification:** `go test ./internal/cli/ -run 'TestSecretBackend|TestResolveSecret' -v` passes.

---

## Task 7 — Go/Python secret-backend behavior parity
**Addresses:** Extensibility-Imp #4

**Problem:** Unknown `PI_SECRET_BACKEND`: Go errors (`secret.go:25-35`), Python silently uses bitwarden (`conftest.py:96-141`).

**Fix:**
1. `eval/conftest.py` `get_secret()`: unknown backend → return `None` (or raise), not bitwarden default. Match Go.
2. Add cross-language contract test: for each of `["", "bitwarden", "1password", "op", "env-only", "env", "bogus"]`, assert Go and Python agree on behavior.

**Verification:** Go + Python secret tests both pass; new contract test green.

---

## Task 8 — Python dependency policy + CI pip cache + release parallelization
**Addresses:** Extensibility-Imp #6, Performance-Imp #2, Performance-Minor #1

**Problem:** `eval/requirements.txt` doesn't reflect the stated pyyaml/openai-only ceiling; no pip cache in CI; release cross-compiles serial.

**Fix:**
1. Document the ACTUAL dependency policy (DeepEval/pytest/pypdf/pytest-timeout stack) in README/CONTRIBUTING, or redesign. Simplest: document + pin with `~=` (already done in hardening); add a comment explaining the ceiling is aspirational. **Decision needed:** document-as-is vs. enforce.
2. `.github/workflows/ci.yml` + `nightly-live-eval.yml` — add `actions/setup-python` `cache: pip` with `cache-dependency-path: eval/requirements.txt`.
3. `scripts/build-release.sh` — keep serial (minor); optionally add bounded parallelism with `xargs -P` if release latency matters.

**Verification:** CI passes; pip cache key present in workflow YAML.

---

## Task 9 — Doctor semantics: required vs optional
**Addresses:** Usability-Imp #2, Ergonomics-Minor #1

**Problem:** `doctor.go:41-58` marks every absent provider key + unavailable backend as `[FAIL]` → new users with no key get "FAILURES FOUND" exit 1 despite correct setup.

**Fix:**
1. `doctor.go` — distinguish required (node, pi, eval venv) from optional (provider keys, secret backend availability):
   - Absent optional provider keys → `[info]` not `[FAIL]`.
   - Show one clear line: "configure one provider key to run chat/live evals".
   - Exit 1 only for required-runtime failures.
2. Update `doctor_test.go` + README no-key guidance.

**Verification:** `pi-run doctor` with no keys exits 0 (or 1 only for real breakage); tests pass.

---

## Task 10 — Docs alignment (Node, env vars, quarantine, eval semantics)
**Addresses:** Extensibility-Minor #2, Usability-Minor #1-2, Security-Imp #5, Ergonomics-Minor #3

**Fix:**
1. `README.md:30,197-200,286-294` — document actual Node policy: "highest nvm-installed semver; override with `PI_NODE_VERSION`"; remove "default Node 22".
2. Add env-var config table: `PI_PROVIDER`, `PI_SECRET_BACKEND`, `PI_NODE_VERSION`, `PI_RUN_PERSONAL`, `HARNESS_ROOT`, `BW_GET`, `OP_VAULT` (+ new `PI_RUN_PROVIDERS_FILE`).
3. `README.md:125-127` — remove quarantine-removal advice; keep Homebrew as primary macOS path.
4. `README.md:142-176` — align eval key semantics with implementation (env-only at CLI dispatch; document the env-only requirement or the Task 1 fix).

**Verification:** grep confirms no quarantine/xattr advice, no "Node 22" claims, env table present.

---

## Task 11 — CI least-privilege permissions
**Addresses:** Security-Minor #1

**Fix:** Add `permissions: contents: read` to `.github/workflows/ci.yml` and `nightly-live-eval.yml` (top-level). Keep `contents: write` only in `release.yml`.

**Verification:** workflows parse; CI passes.

---

## Task 12 — Clean command output
**Addresses:** Ergonomics-Minor #2

**Fix:** `internal/cli/app.go` `runClean()` — print status lines ("removed eval/.venv", "nothing to clean"), optional `--dry-run`.

**Verification:** `pi-run clean` prints meaningful output.

---

## Dependency / Sequencing

- Tasks 1-4 are the highest-value (they fix the cross-cutting eval/onboarding/CLI issues and the one Critical ergonomics).
- Tasks 5-7 improve extensibility/security (provider portability, timeouts, parity).
- Tasks 8-12 are polish/docs/CI.
- Tasks 1 and 7 both touch `eval/conftest.py` — do Task 1 first, then Task 7 (avoid conflicts).
- Task 4 touches `app.go` flag parsing — do before Task 3 (install) if both touch `runInstall` (they don't overlap much, but sequencing avoids churn).

## Execution

Subagent-driven development, same workflow as v0.3.0:
- Each task in an isolated branch/worktree, worker implements (TDD), reviewer approves.
- Final whole-branch review before merge.
- Model: `openai/gpt-5.6-terra` for implementation/review (now authenticated), or deepseek as fallback.

## Out of Scope (accepted)

- **Security Critical (transcripts):** skipped — gitignored per user decision.
- Full dependency redesign to pyyaml/openai-only ceiling: decision needed (Task 8.1).
- Windows native bootstrap support: documented limitation, not fixed here.
