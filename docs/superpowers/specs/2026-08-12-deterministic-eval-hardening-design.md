# Deterministic Eval Hardening (Go scorecard tests + Python contract tests) — Design

**Date:** 2026-08-12
**Status:** SHIPPED — deterministic eval hardening (v0.9.0, PRs #40/#42–#44); NOTE: mcp-server/OTel contract-test sections superseded by the v0.10.0 cut-list removal (see CHANGELOG)
**Target release:** v0.8.0 (test & CI hardening — no user-facing features)

## 1. Context & Motivation

**Two verification gaps** sit between the shipped v0.6.0 scorecard feature
(`pi-run ci-benchmark`) and the shipped v0.7.x/0.8.0 CLI surface
(`mcp-server`, OTel export, `--model-tier`, `project-understand`), and both
undermine "measure, then trust the measurement":

1. **The scorecard artifact is not deterministically pinned.** `buildScorecard`
   stamps `Timestamp` and the run ID from `time.Now()` at build time
   (`internal/cli/scorecard.go:648`, `:673`), so the *same run data* produces a
   different JSON artifact every second. Today's tests cover aggregation math
   and gate logic (`scorecard_test.go`, 15 gate cases at `:121`) but there is no
   byte-stable "this is exactly what a scorecard looks like" contract, and the
   baseline-pointer file `scorecard-latest.json` is created by **untested shell
   glue** in CI (`.github/workflows/provider-scorecard.yml:54-66`, `cp` at
   `:64`) instead of by the CLI that owns the artifact directory.
2. **The real binary's cross-language contracts are untested.** The
   DeepEval suite exercises `pi-run` only via `config-check` and
   `eval --benchmark --benchmark-dry-run`
   (`eval/tests/test_harness_config.py`, `eval/tests/test_benchmark_format.py`).
   `mcp-server`'s JSON-RPC wire protocol, the `PI_OTLP_ENDPOINT` telemetry
   export, the `--model-tier` launch wiring, the exit-code contract, and
   `project-understand` determinism have Go unit tests but **no end-to-end
   check against the real binary** — and CI's `python-quick` job
   (`.github/workflows/ci.yml:30-48`) runs pytest *without building `pi-run`
   at all*, so even the existing subprocess tests only work if a binary happens
   to be preinstalled. `nightly-live-eval.yml` has the same latent gap.

**Goal:** (a) pin the scorecard artifact — timestamp, run ID, full JSON shape,
and the latest-pointer file — with hermetic Go tests behind a 2-line time seam;
(b) add a Python contract-test suite that exercises the **real `pi-run`
binary** for `mcp-server` / `--model-tier` / OTel / `project-understand` /
exit codes, and wire CI to build the binary before running it. **No live
evals, no dataset growth, no new Python dependencies** in this spec.

## 2. Current State (verified)

- **Scorecard builder** — `buildScorecard` (`internal/cli/scorecard.go:644`)
  sets `RunID: scorecardRunID(opts.providers)` and
  `Timestamp: time.Now().UTC().Format(time.RFC3339)` (`:648`).
  `scorecardRunID` (`:672`) is `time.Now().Format("20060102T150405") + "-" +
  strings.Join(providers, "-")` (`:673`) — **the determinism blocker: two
  `time.Now()` call sites inside the artifact path**. `writeScorecard` (`:678`)
  writes `eval/benchmark-results/scorecard-<run>.json` (dir gitignored,
  `.gitignore:8`).
- **Existing Go tests** (`internal/cli/scorecard_test.go`) —
  `TestEvaluateScorecardGates` (`:121`, table-driven, **15 cases**: fail-below,
  budget-at-cap/boundary, baseline regression/tolerance/provider-missing,
  incomplete-run precedence, `--runs` actual-spend accounting),
  `TestScorecardJSONRoundTrip` (`:398`, `reflect.DeepEqual`),
  `TestScorecardJSONOmitEmptyGates` (`:434`), `TestWriteScorecard` (`:462`,
  `t.TempDir()`), `TestParseBaselineScorecardShape` (`:239`),
  `TestExitCodesTableDocumentsScorecard` (`:585`). Helpers: `nearlyEqual`
  (`internal/cli/cost_test.go:50`), `captureRunStdout`/`captureRunStderr`
  (`internal/cli/app_test.go:537/:560`). The suite is **stdlib-only** — `go.mod`
  declares Go 1.26.5 with **zero `require` blocks**.
- **`scorecard-latest.json`** — created by a shell `cp` in the "Resolve
  baseline" step of `.github/workflows/provider-scorecard.yml:54-66` (`cp
  "$LATEST" eval/benchmark-results/scorecard-latest.json` at `:64`). This glue
  only exists in CI; it is untested, and the workflow's own comment calls it
  fragile (`ls -t | head -1` ordering).
- **CI precedent for building the binary** —
  `.github/workflows/provider-scorecard.yml:29-32`: `go build -o bin/pi-run
  ./cmd/pi-run` then `echo "$GITHUB_WORKSPACE/bin" >> "$GITHUB_PATH"`.
  `ci.yml:30-48` (`python-quick`) and `nightly-live-eval.yml` (live-eval job)
  run pytest **without** this step.
- **Hermetic Python precedents** — `eval/tests/test_harness_config.py`
  (subprocess `pi-run config-check`), `test_benchmark_format.py` (stale-binary
  skip probe, `:79-86`), `test_secret_resolution.py` (fake `bw_get`/`op` on a
  tmp PATH). `eval/conftest.py` carries `run_pi_print`, `sample_cases`, and
  `SUPPORTED_PROVIDER_KEYS` (17 keys) — a hand-maintained mirror of
  `internal/cli/eval.go:10-14` `supportedProviderKeyEnvs`. `eval/pytest.ini`
  already sets `timeout = 120` / `timeout_method = thread` (pytest-timeout is
  an existing dep). `eval/requirements.txt` has an explicit policy comment:
  new deps are deliberate, policy-documented decisions.
- **Features to contract-test (shipped):**
  - `mcp-server` — `runMCPServer` (`internal/cli/mcp.go:107-140`): line-delimited
    JSON-RPC 2.0 over stdio, EOF → **exit 0**; tools `providers`, `cost`,
    `benchmark_dry_run` (`:217`); `callProvidersTool` returns a JSON array of
    `{name, defaultModel, keyEnv, baseURL}` (`:299`). Go tests pin the
    handshake (`mcp_test.go:82` `TestMCPInitialize`, `:398`
    `TestMCPNotificationGetsNoResponse`).
  - OTel — `maybeExportOTLPSpan` (`internal/cli/otel.go`): env-gated on
    `PI_OTLP_ENDPOINT`, one POST to `<endpoint>/v1/traces`, best-effort
    (exactly one stderr warning, **never changes the exit code**), called from
    `runLaunch` after pi exits (`internal/cli/app.go:265`). Go tests use
    `httptest` (`otel_test.go:20-31` `captureOTLPRequest`, `:202` disabled-when-unset).
  - `--model-tier` — shipped (commit `6516b48`): `Provider.ModelTiers`
    (`internal/cli/providers.go:12-16`), `availableTiers` (`:235`),
    `resolveLaunchModel`, TIERS column in `runProviders`
    (`internal/cli/app.go:485`), tiers in `providers.json` (5 of 17 providers:
    openai, openrouter, deepseek, anthropic, gemini).
  - `project-understand` — `runProjectUnderstand` (`internal/cli/project_understand.go:110`):
    exit 0/1/2, writes `product.md`, `tech.md`, `structure.md` to `--out`
    (default `docs/understand`, `:15`), no `time.Now()`/randomness anywhere in
    the generator — reruns are byte-identical.
- **Launch plumbing (for the fake-node/fake-pi tests)** —
  `nodeBinDir` (`internal/cli/pi.go:34-42`) **only stats**
  `<HOME>/.nvm/versions/node/<v>/bin/node`; `execPiDir` (`:110-141`) runs the
  **absolute path** `<binDir>/pi` (`exec.Command(filepath.Join(binDir,
  "pi"))`); `resolveNodeVersion` (`:176-205`) honors `PI_NODE_VERSION` first.
  `piArgs` (`:59-92`) always emits `--provider <PiProvider> --model <model>
  --offline`; `print` adds `-p` (`--no-session` when no budget cap).
  `runLaunch` failure order (`internal/cli/app.go:155-248`): prompt/usage (2) →
  permission/tier conflicts (2) → provider/tier resolution (2) → budget
  pre-flight (6) → key (3) → node (4) → hooks → `execPi` (`:248`) → OTel
  export (`:265`). `--exit-codes` table (`app.go:81-97`): 0 ok · 1 generic ·
  2 usage · 3 missing key · 4 node/pi missing · 5 eval venv missing · 6 budget
  exceeded · 7 docker unavailable · 8 scorecard gate failed.

## 3. Scope

### In scope

1. **Go: deterministic scorecard tests** — a `scorecardNow` package-var seam,
   a golden-file test for the full scorecard JSON shape, a hand-rolled
   `-update` helper, and `writeScorecardLatest` making `scorecard-latest.json`
   CLI-owned and hermetic-testable.
2. **Go: drift guards** — `TestUsageDocumentsNewCommands` (usage text ↔
   command dispatch) and `TestMCPProvidersKeyEnvMatchesPythonMirror`
   (MCP providers keyEnv list ↔ `eval/conftest.py` `SUPPORTED_PROVIDER_KEYS`).
3. **Python: contract tests against the real `pi-run` binary** — new files
   under `eval/tests/` covering `mcp-server` (raw stdio JSON-RPC client),
   launch/`--model-tier` (fake node + fake pi), OTel (fake collector),
   `project-understand` (determinism), and the exit-code contract/ordering.
   Shared fixtures (`pi_run_bin`, `hermetic_env`, `fake_launch_env`,
   `fake_collector`) added to `eval/conftest.py`.
4. **CI wiring** — a new `python-contract` job in `ci.yml` that builds
   `pi-run` first (reusing the provider-scorecard build precedent) then runs
   the contract files; a "Build pi-run" step added to `nightly-live-eval.yml`.
   `provider-scorecard.yml` loses its shell `cp` glue.
5. **`scorecardNow` seam + `writeScorecardLatest`** — the only production-code
   changes in this spec (~30 lines total).

### Explicitly OUT of scope

- **Live evals** — no test runs a real provider; every contract test is
  hermetic (fake keys, fake node/pi, fake collector, tmp roots).
- **Dataset growth** — `eval/datasets/` and the benchmark task suite are
  untouched.
- **Cost-per-task / cost attribution changes** — the ledger and budget
  plumbing stay as shipped; only their *verification* is extended.
- **`pytest-json-report` (or any new Python dependency)** — deferred to the
  live-eval v2 spec; `eval/requirements.txt` is unchanged by this spec, and
  the 120 s pytest-timeout bound (`eval/pytest.ini`) is the hang guard.
- **Docker-in-nightly** — no Docker usage in any new test; the `ci-benchmark`
  live path stays in its own weekly/manual workflow.
- **README / CHANGELOG edits** — parent-owned docs pass.
- **Refactoring of the scorecard schema** — `schemaVersion` 1 and all field
  names are pinned as-is; the golden file freezes the current shape.

## 4. Design

### 4.1 Decision 1 — Go stays 100 % stdlib-only (no go-cmp, no goldie)

**Decision:** all new Go tests use `reflect.DeepEqual` + table-driven
assertions (the existing pattern at `scorecard_test.go:398`), plus **one
committed golden fixture** with a hand-rolled `-update` helper (~30 lines).
Inline assertions are preferred for gate logic; the golden is used for exactly
one thing — the full JSON shape of a built scorecard.

**Rationale:** `go.mod` has zero `require` blocks today; the project's stated
constraint (repeated in the benchmark/scorecard specs) is "stdlib-only Go".
`go-cmp`/`goldie` would add the first third-party test dependency for marginal
benefit: `reflect.DeepEqual` already covers struct equality, table-driven cases
cover the gate matrix, and a byte-comparison against a committed fixture is
trivially hand-rolled (`os.ReadFile` + `reflect.DeepEqual` on unmarshaled
values, or a byte compare; `-update` writes the fixture). Keeping the fixture
mechanism ~30 lines in-repo means every future contributor can reason about it
without a new library's semantics. One golden, not one per feature: golden
files rot when they proliferate, so the spec deliberately caps them.

### 4.2 Decision 2 — `scorecardNow` package-var seam (the only determinism fix)

**Decision:** add a package-level seam to `internal/cli/scorecard.go`:

```go
// scorecardNow is a package-level seam so tests can pin the scorecard
// timestamp and run ID. Production behavior is unchanged.
var scorecardNow = time.Now
```

and replace the two call sites — `scorecard.go:648` (`Timestamp:
scorecardNow().UTC().Format(time.RFC3339)`) and `scorecard.go:673`
(`scorecardNow().Format("20060102T150405") + "-" + ...`) — plus the test
restores it via `t.Cleanup`. **2-line production change** that makes
`buildScorecard`/`scorecardRunID` fully deterministic, which in turn makes the
golden test possible.

**Rationale:** the alternative — passing `now time.Time` through
`buildScorecard` → `scorecardRunID` — threads a parameter through every
caller and the existing signature; a package var is the established Go seam
pattern for clock injection, is invisible to production callers, and is exactly
bounded to the two offending call sites. Note for implementers: in tests pin a
**UTC** fixed time (`time.Date(2026, 8, 11, 15, 4, 5, 0, time.UTC)`) because
`scorecardRunID` formats with a **local-time layout** (`20060102T150405`, no
`UTC()` — existing behavior, unchanged) while `Timestamp` formats UTC; a UTC
fixed value keeps both consistent.

### 4.3 Decision 3 — `scorecard-latest.json` becomes CLI-owned

**Decision:** add `writeScorecardLatest(root string, sc scorecard) (string,
error)` to `internal/cli/scorecard.go` — mirrors `writeScorecard` (`:678`)
but writes the fixed name `eval/benchmark-results/scorecard-latest.json`.
`runScorecard` calls it after `writeScorecard` on every successful
`ci-benchmark` run (including the first run, seeding the chain). The "Resolve
baseline" step (`.github/workflows/provider-scorecard.yml:54-66`, the `cp` at
`:64`) is **deleted**; the workflow shrinks to: download artifact → run gate
(keeping the existing `[ -f scorecard-latest.json ]` conditional for
`--baseline`) → upload both `scorecard-*.json` and `scorecard-latest.json`.

**Rationale:** the `cp` glue is (a) untested — it executes only in CI, (b)
fragile — `ls -t | head -1` ordering and shell-quoting rules, (c) not
cross-platform. The CLI already owns the directory, the write path, and the
`<run>` naming; making the pointer file a Go function makes it
hermetic-testable with `t.TempDir()` (the `TestWriteScorecard` pattern,
`scorecard_test.go:462`), removes shell from the artifact chain, and keeps the
"latest is byte-identical to the most recent `scorecard-<run>.json`" property
enforceable by a unit test.

### 4.4 Decision 4 — CI builds `pi-run` before any Python subprocess test

**Decision:** add a dedicated **`python-contract`** job to
`.github/workflows/ci.yml` after `python-quick`: checkout → `setup-go 1.26.5`
→ **Build pi-run** (`go build -o bin/pi-run ./cmd/pi-run` +
`echo "$GITHUB_WORKSPACE/bin" >> "$GITHUB_PATH"`, copied from
`provider-scorecard.yml:42-46`) → `setup-python 3.11` → install
`eval/requirements.txt` into `eval/.venv` (unchanged policy, no new deps) →
`eval/.venv/bin/python -m pytest tests/test_contract_*.py -v`
(`working-directory: eval`). Also add the identical **Build pi-run** step to
`nightly-live-eval.yml` before "Run full eval" (its suite includes
`test_harness_config.py`, which already calls `pi-run` via PATH and today
relies on a preinstalled binary).

**Rationale:** `python-quick` stays the cheap DeepEval smoke (no ~30-60 s Go
build on every push); the contract suite is the one place a fresh binary is
mandatory, so it gets a job where the build is explicit and failures are
loud (a stale/missing binary **fails** the contract fixture — see §4.5 — never
skips). `nightly-live-eval` gets the same build step because the contract
files will run as part of its full `tests/ -v` invocation. This closes the
latent "binary tests only work if preinstalled" gap in both workflows.

### 4.5 Decision 5 — Python contract-test architecture (stdlib-only, four fixtures)

**Decision:** new tests live in `eval/tests/test_contract_*.py`; shared
fixtures go in `eval/conftest.py`. **No new Python dependencies** — the
JSON-RPC client, fake collector, and fake-launch environment are all stdlib
(`subprocess`, `json`, `select`, `http.server.ThreadingHTTPServer`,
`tempfile`, `shutil`); hang protection comes from the existing pytest-timeout
bound (`eval/pytest.ini`, 120 s, thread method).

**`pi_run_bin` (session-scoped):** resolves the binary in order —
`PI_RUN_BIN` env override → `shutil.which("pi-run")` → build-once
(`go build -o <session-tmp>/pi-run ./cmd/pi-run` from the repo root, cached
for the session). After resolution it **probes** the binary
(`pi-run --exit-codes` must exit 0 and print the `8  scorecard gate failed`
row; usage must mention `mcp-server`) and **fails loudly** with a
rebuild hint on mismatch — the opposite of `test_benchmark_format.py`'s skip
probe (`:31-40`), because a stale binary is a broken contract, not a skip
condition, and the CI job always builds fresh. `PI_RUN_BIN` is preserved
through every other fixture (it is the thing under test).

**`hermetic_env`:** clears every provider key env from
`SUPPORTED_PROVIDER_KEYS`, plus `PI_SECRET_BACKEND`, `BW_GET`,
`PI_OTLP_ENDPOINT`, `PI_MODEL_TIER`, `PI_RUN_PROVIDERS_FILE`,
`PI_NODE_VERSION`; sets `PI_SECRET_BACKEND=env-only`, `HOME=<tmp>`; optional
`HARNESS_ROOT=<tmp>` (default **on**; a test that needs the real checkout —
the providers.json data-driven test, §4.12 — requests `harness_root=None` and
passes `cwd=<repo>` explicitly, the `test_harness_config.py` pattern).

**`fake_launch_env`:** builds a tmp HOME shaped exactly like what
`pi.go` expects (§4.9): `.nvm/versions/node/v22.19.0/bin/` containing an empty
executable `node` and an executable `pi` shell script that appends `"ARGS:$*"`
plus key probes to `$FAKE_PI_LOG` and exits `${FAKE_PI_EXIT:-0}`. Sets
`PI_NODE_VERSION=v22.19.0`. Returns a namespace with the log path and a knob
for `FAKE_PI_EXIT`.

**`fake_collector`:** a `ThreadingHTTPServer` on `127.0.0.1:0` (ephemeral
port) running in a daemon thread; records every request's method, path,
headers, and body into a list; response status configurable (default 200);
teardown calls `shutdown()` + `server_close()`.

### 4.6 Decision 6 — exit-code contract: `--exit-codes` is the single source of truth, and ordering is pinned via subprocess

**Decision:** the `--exit-codes` table (`app.go:81-97`) is the canonical
contract for all 0..8 codes, and the **order** of failure precedence — usage
(2) beats missing-key (3) beats node-missing (4), with budget (6) after usage
checks — is pinned by subprocess tests against the real binary under
`hermetic_env`. Tests that assert a specific exit code must not hardcode
descriptions: `test_exit_codes_table_is_source_of_truth` parses the table and
the ordering tests assert observed codes equal the parsed rows.

**Rationale:** exit codes are the integration surface CI maps to red/green
builds; they are currently verified only by Go unit tests
(`TestExitCodesTableDocumentsScorecard`, `scorecard_test.go:585`). Pinning the
table text and the precedence order end-to-end catches the class of regression
where refactors reorder `runLaunch` checks (`app.go:155-248`) or mislabel a
code. The precedence order itself is a shipping contract: "usage errors always
win" is why `--model-tier turbo` with **no key configured** exits 2, not 3.

### 4.7 MCP JSON-RPC handshake + the stdout-is-sacred rule

**Handshake sequence pinned by the Python suite** (mirrors the Go tests
`mcp_test.go:82`, `:398`):

1. `initialize` (`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{}}}`)
   → **one** response line: `jsonrpc:"2.0"`, id echoed, `result.protocolVersion`
   `"2025-03-26"`, `result.serverInfo.name` `"pi-run"`,
   `result.capabilities.tools.listChanged` `false`.
2. `notifications/initialized` → **no response line** (silent). Proved by
   sending `tools/list` immediately after and asserting exactly one response
   line arrives, with the `tools/list` id.
3. `tools/list` → `tools` array contains exactly `providers`, `cost`,
   `benchmark_dry_run`, each with a description.
4. `tools/call providers` → `result.isError` false; `content[0].text` parses
   as JSON; entries carry `name`/`defaultModel`/`keyEnv`; `openai` and
   `OPENAI_API_KEY` present.
5. EOF (stdin closed) → process exits **0** (the `runMCPServer` contract,
   `mcp.go:130-138`).

Error resilience is also pinned: a malformed line yields a JSON-RPC parse
error (code −32700) and the server keeps serving; an unknown method yields a
JSON-RPC error; `tools/call` with an unknown tool yields a **tool result with
`isError: true`** (never a JSON-RPC error) — all followed by a working `ping`
to prove liveness.

**stdout-is-sacred framing rule (applies to every new test):** `pi-run`'s
stdout is a *protocol surface* per command — `mcp-server` owns stdout as
line-delimited JSON-RPC; `--exit-codes` owns stdout as the code table;
`providers` owns stdout as the tab-separated catalog. Diagnostics, warnings,
and telemetry failures go to **stderr only**. The suite asserts the boundary:
every stdout line from the MCP session parses as JSON-RPC and stderr is empty
on the happy path; the OTel warning test asserts the warning lands on stderr
and stdout stays clean (§4.10).

### 4.8 Golden-fixture mechanics

- **Fixture:** `internal/cli/testdata/scorecard-golden.json` (new `testdata/`
  dir — Go tooling ignores it at build time), committed.
- **Content:** the full `schemaVersion`-1 shape built via `buildScorecard`
  with the seam pinned to `2026-08-11T15:04:05Z` UTC, two provider rows
  (openai/deepseek), `failBelow`/budget/baseline configured, one regression —
  the same fixture values as `TestScorecardJSONRoundTrip` (`:398`) so the two
  tests corroborate each other.
- **Assertion:** `TestScorecardGoldenJSON` builds the scorecard with the
  pinned seam, marshals with `json.MarshalIndent(sc, "", "  ")` + `'\n'`
  (byte-identical to `writeScorecard`'s output), and compares **byte-for-byte**
  with the fixture; the mismatch message prints both sides. Byte-exactness is
  safe because the seam fixes the only nondeterminism and `MarshalIndent`
  emits struct-order-stable fields.
- **`-update` helper (~30 lines):** a package-level
  `var goldenUpdate = flag.Bool("update", false, "update golden fixture")` in
  `scorecard_test.go`; when set, the test writes the marshaled bytes to the
  fixture and returns. Rerun: `go test ./internal/cli/ -run TestScorecardGoldenJSON -update`.
  `go test` parses `-update` because the flag is registered at package scope
  before the testing framework's `flag.Parse`.

### 4.9 Fake node/pi mechanics (matching `pi.go` exactly)

The launch contract tests fake the entire Node/Pi toolchain, exploiting two
facts verified in `internal/cli/pi.go`:

- `nodeBinDir` (`:34-42`) **only stats** `<HOME>/.nvm/versions/node/<v>/bin/node` —
  the file must exist but is never executed, so an empty 755 file suffices.
- `execPiDir` (`:110-141`) executes the **absolute path**
  `<binDir>/pi` — `exec.Command(filepath.Join(binDir, "pi"))` — so a shell
  script at that exact path receives pi's real argv.
- `resolveNodeVersion` (`:176-205`) honors `PI_NODE_VERSION` first, so the
  fixture pins `v22.19.0` and never scans a real nvm install.

The fake pi script is therefore:

```sh
#!/bin/sh
echo "ARGS:$*" >> "$FAKE_PI_LOG"
echo "KEY_OPENAI=${OPENAI_API_KEY:-}" >> "$FAKE_PI_LOG"
exit "${FAKE_PI_EXIT:-0}"
```

Because `runLaunch` checks the key (exit 3) before the node (exit 4) and the
node before `execPi` (`app.go:228-248`), the fixture's `OPENAI_API_KEY=testvalue`
env var reaches pi via `launchEnv` (`pi.go:88`) — the contract asserts the key
is in pi's **environment, never in its argv** (no `sk-`/key substring in the
logged `ARGS:` line). `piArgs` (`pi.go:59-92`) pins the argv shape
(`--provider openai --model <model> --offline -p --no-session` for a
non-budgeted print run).

### 4.10 OTel fake-collector assertions

`fake_collector` (§4.5) is the Python counterpart of Go's
`captureOTLPRequest` (`otel_test.go:23`). Assertions, all under
`fake_launch_env` + `hermetic_env`:

1. **`PI_OTLP_ENDPOINT` unset → zero requests** (collector request count 0) —
   mirrors `otel_test.go:202`.
2. **Endpoint set → exactly one POST to `/v1/traces`** with
   `Content-Type: application/json`; payload decodes as OTLP
   `ExportTraceServiceRequest`: `resourceSpans[0].scopeSpans[0].spans[0]`
   has `name == "invoke_agent"`, `kind == 2` (CLIENT), `status.code == 1`
   (OK), and attributes `gen_ai.agent.name`, `gen_ai.provider.name ==
   "openai"`, `gen_ai.agent.model`, `pi_harness.run.mode == "print"`,
   `pi_harness.run.exit_code == 0`.
3. **Exit-code attribute reflects pi's exit:** `FAKE_PI_EXIT=7` →
   `pi_harness.run.exit_code == 7` and `status.code == 2` (ERROR); `pi-run`
   itself still exits **7** (telemetry never changes the exit code).
4. **Collector returns 500 → exactly one warning on stderr** matching
   `pi-run: warning: telemetry export failed: ... 500`, exit unchanged, stdout
   clean (stdout-is-sacred).
5. **Unreachable endpoint** (e.g. `http://127.0.0.1:1`) → same one-warning +
   unchanged-exit contract (mirrors `otel_test.go:184`).

### 4.11 Project-understand determinism

A fixture project (README.md with a distinctive paragraph, a go.mod, a couple
of source files) under a tmp `HARNESS_ROOT`. `pi-run project-understand
--out <tmp-out>` is run **twice**; assertions:

- exit 0; stdout announces the three written docs;
- `product.md`, `tech.md`, `structure.md` exist and contain the fixture's
  markers (README paragraph, `**Primary stack:**`, language census — the same
  markers the Go test checks, `project_understand_test.go:47-110`);
- **byte-identical reruns** for all three files (the determinism contract —
  the generator has no `time.Now()`/randomness, verified by grep over
  `project_understand.go`);
- output hygiene: no absolute tmp path and no placeholder secret value in the
  generated docs;
- `project-understand --bogus` → exit 2 (usage), the `:148` Go contract.

### 4.12 Model-tier contract tests, data-driven off `providers.json`

The tier surface is data-driven from the repo's real `providers.json` (17
providers, 5 with `modelTiers`: openai, openrouter, deepseek, anthropic,
gemini):

- **Catalog mirror:** run `pi-run providers` (cwd = repo, `HARNESS_ROOT`
  unset); for every provider with `modelTiers` in `providers.json`, assert its
  tab-separated row's TIERS column equals `balanced,` + comma-joined sorted
  tier keys (e.g. openai → `balanced,cheap,fast`); providers without a map →
  `balanced`.
- **Resolution into argv:** via `fake_launch_env` — `--model-tier fast` →
  `--model openai/gpt-5.6-mini`; `PI_MODEL_TIER=cheap` → `--model
  openai/gpt-5.1-mini`; explicit `--model` wins over the env tier.
- **Hard failures, no fallback:** `--model-tier turbo` → 2 before any key
  access; `--model-tier fast --model x` → 2 (mutual exclusion);
  `--provider deepseek --model-tier cheap` → 2 with a message listing
  `(available: balanced, fast)`; `resume --model-tier fast` → 2; none of these
  invoke the fake pi (log absent).

### 4.13 Drift guards

- **`TestUsageDocumentsNewCommands` (Go, `app_test.go`):** the usage
  `Commands:` block (`app.go:21-40`) and `Run`'s dispatch switch (`:104-149`)
  must never drift: every command word documented in usage must be a handled
  dispatch key, and every dispatch case must appear in usage. Supplements the
  existing one-directional hardcoded-list check
  `TestUsageMentionsAllCommands` (`app_test.go:645`); the new test derives the
  command list from the usage text (regex over the Commands block) and
  exercises the dispatch path (the `default` branch prints
  `pi-run: unknown command`, `app.go:149-152`). Adding a command without
  documenting it (or vice versa) fails.
- **`TestMCPProvidersKeyEnvMatchesPythonMirror` (Go, `mcp_test.go`):** the
  keyEnv set exposed by the MCP `providers` tool (`callProvidersTool`,
  `mcp.go:299`) must equal the `SUPPORTED_PROVIDER_KEYS` tuple parsed from
  `eval/conftest.py` (file read, regex extraction; no Python subprocess
  needed). This closes the mirror loop: `conftest.py` already documents that it
  mirrors `internal/cli/eval.go:10-14` (`supportedProviderKeyEnvs`), and
  `TestSupportedProviderKeyEnvsCoverCatalog` (`providers_test.go:70-83`) covers
  only the Go side. The Python side of the same mirror is asserted by
  `test_mcp_providers_keyenv_matches_conftest` (§6.2), so a catalog change
  without a conftest update fails on both sides.

## 5. Implementation Plan

1. **`internal/cli/scorecard.go` (prod, ~20 lines):** add `scorecardNow` var;
   swap call sites at `:648`/`:673`; add `writeScorecardLatest(root, sc)`
   (mirrors `writeScorecard` `:678`); call it from `runScorecard` after
   `writeScorecard`.
2. **Go tests:** `scorecard_test.go` — seam tests, golden test + `-update`
   flag + `internal/cli/testdata/scorecard-golden.json`, `TestWriteScorecardLatest`,
   cleanup-restore test; `app_test.go` — `TestUsageDocumentsNewCommands`;
   `mcp_test.go` — `TestMCPProvidersKeyEnvMatchesPythonMirror`.
3. **`eval/conftest.py`:** add `pi_run_bin`, `hermetic_env`, `fake_launch_env`,
   `fake_collector` fixtures (§4.5).
4. **`eval/tests/test_contract_*.py`:** five files per §6.2 (stdlib-only).
5. **CI:** `ci.yml` — new `python-contract` job (§4.4); `nightly-live-eval.yml`
   — add Build pi-run step; `provider-scorecard.yml` — delete the Resolve
   baseline step, upload `scorecard-latest.json` too.
6. **Verify:** `go test ./...`; `go vet ./...`; contract suite via
   `PI_RUN_BIN=bin/pi-run eval/.venv/bin/python -m pytest
   tests/test_contract_*.py` after a local `go build -o bin/pi-run
   ./cmd/pi-run`; `go test ./internal/cli/ -run TestScorecardGoldenJSON
   -update` to regenerate the fixture if the schema ever changes (schema is
   frozen in v1 — see §3).
7. **Docs:** README/CHANGELOG notes are parent-owned (§3).

## 6. Tests

### 6.1 Go tests (all hermetic — no keys, no network, no Docker)

| Test (file) | Asserts |
|---|---|
| `TestBuildScorecardDeterministicWithSeam` (scorecard_test.go) | With `scorecardNow` pinned to `2026-08-11T15:04:05Z` UTC: `buildScorecard` yields `RunID == "20260811T150405-openai-deepseek"` and `Timestamp == "2026-08-11T15:04:05Z"` exactly; two consecutive builds are `reflect.DeepEqual` |
| `TestScorecardRunIDDeterministic` (scorecard_test.go, extends `:508`) | Seam pinned → exact run-ID string; seam restored → `^\d{8}T\d{6}-providers$` shape |
| `TestScorecardGoldenJSON` (scorecard_test.go) | Full JSON shape byte-identical to `testdata/scorecard-golden.json`; `-update` flag rewrites the fixture (hand-rolled helper, ~30 lines) |
| `TestScorecardRoundTripRejectsUnknownFields` (scorecard_test.go) | Extend `TestScorecardJSONRoundTrip` (`:398`) with a `json.Decoder{DisallowUnknownFields: true}` decode so a renamed/removed scorecard field fails loudly instead of being silently ignored; every field the marshaler emits is known, so existing fixtures pass unchanged (precedent: `benchmark.go:81`). This also hardens the `parseBaseline` decode path, which would otherwise stay drift-blind |
| `TestWriteScorecardLatest` (scorecard_test.go) | `writeScorecardLatest(t.TempDir(), sc)` writes `eval/benchmark-results/scorecard-latest.json`; bytes equal `writeScorecard`'s output for the same struct; a second write overwrites (pointer semantics) |
| `TestScorecardNowRestored` (scorecard_test.go) | `t.Cleanup` restores `scorecardNow` — no cross-test pollution after a pinned test |
| `TestUsageDocumentsNewCommands` (app_test.go) | Usage `Commands:` block ↔ `Run` dispatch: no undocumented command, no undispatchable command (§4.13) |
| `TestMCPProvidersKeyEnvMatchesPythonMirror` (mcp_test.go) | `callProvidersTool` keyEnv set == `SUPPORTED_PROVIDER_KEYS` parsed from `eval/conftest.py` (§4.13) |

Existing tests continue to guard the seams this spec relies on:
`TestEvaluateScorecardGates` (`:121`, 15 cases), `TestScorecardJSONRoundTrip`
(`:398`), `TestScorecardJSONOmitEmptyGates` (`:434`), `TestWriteScorecard`
(`:462`), `TestParseBaselineScorecardShape` (`:239`), the MCP handshake tests
(`mcp_test.go:82/:398`), and `TestMaybeExportOTLPSpanDisabledWhenUnset`
(`otel_test.go:202`).

The baseline-regression boundary matrix is deliberately pinned by the existing
15-case `TestEvaluateScorecardGates` rather than duplicated in a new table:
`baselineRegressions` uses a strict inequality (`r.PassRate < b-tolerance`, see
`scorecard.go:577-598`), and float64 subtraction of decimal literals is
per-literal (e.g. `0.85-0.05 = 0.7999999999999999`, so a boundary row at
`0.8 < 0.7999999999999999` is *false*; `0.8-0.05 = 0.75` exactly, so `0.8` vs
`0.75` *does* pass). Any future editor adding a boundary row must compute
`b-tolerance` for the specific literals rather than assuming decimal exactness.

### 6.2 Python contract tests (real binary, hermetic; file → what each asserts)

**`eval/tests/test_contract_exit_codes.py`** (fixtures: `pi_run_bin`,
`hermetic_env`):

| Test | Asserts |
|---|---|
| `test_exit_codes_table_is_source_of_truth` | `pi-run --exit-codes` exits 0; parses exactly the 9 rows 0..8 with the documented descriptions (`app.go:81-97`); no gaps, no extra codes |
| `test_usage_error_beats_missing_key` | `pi-run print` (no prompt) → exit 2 (no key configured) |
| `test_usage_error_beats_missing_key_tier` | `pi-run print --model-tier turbo "x"` → exit 2 (no key configured) |
| `test_missing_key_beats_node_missing` | `pi-run print --provider openai "hello"` → exit 3 (key check precedes node check; HOME=tmp has no nvm) |
| `test_node_missing_after_key_present` | `OPENAI_API_KEY=testvalue`, no nvm → exit 4 |
| `test_observed_codes_match_table` | Observed codes from the ordering tests equal the rows parsed from `--exit-codes` (single source of truth enforced by the suite) |

**`eval/tests/test_contract_mcp.py`** (fixtures: `pi_run_bin`, `hermetic_env`
with `HARNESS_ROOT=tmp`; stdlib raw JSON-RPC client over
`subprocess.Popen(["pi-run", "mcp-server"], stdin/stdout/stderr pipes)`
using `select`-bounded reads):

| Test | Asserts |
|---|---|
| `test_mcp_initialize_handshake` | `initialize` → one response: jsonrpc 2.0, echoed id, `protocolVersion "2025-03-26"`, `serverInfo.name "pi-run"`, `capabilities.tools.listChanged false` |
| `test_mcp_initialized_notification_silent` | `notifications/initialized` then `tools/list` → exactly one response line (the `tools/list` response) |
| `test_mcp_tools_list` | Tools exactly `providers`, `cost`, `benchmark_dry_run`, each with a description |
| `test_mcp_call_providers` | `tools/call providers` → `isError` false; content parses as JSON; entries carry `name`/`defaultModel`/`keyEnv`; `openai` + `OPENAI_API_KEY` present |
| `test_mcp_providers_keyenv_matches_conftest` | KeyEnv set from the tool == `conftest.SUPPORTED_PROVIDER_KEYS` (Python side of the mirror, §4.13) |
| `test_mcp_unknown_tool_is_tool_error` | `tools/call bogus` → result with `isError true` (never a JSON-RPC error); a following `ping` still answers |
| `test_mcp_unknown_method_jsonrpc_error` | Unknown method → JSON-RPC error response; server keeps serving |
| `test_mcp_malformed_line_jsonrpc_error` | Garbage line → parse error (code −32700); next valid request works |
| `test_mcp_eof_exits_zero` | After `initialize`, closing stdin → process exits 0 |
| `test_mcp_stdout_is_sacred` | Across a mixed request sequence every stdout line parses as JSON with `jsonrpc "2.0"`; stderr empty on the happy path |

**`eval/tests/test_contract_launch.py`** (fixtures: `pi_run_bin`,
`hermetic_env`, `fake_launch_env`):

| Test | Asserts |
|---|---|
| `test_launch_pi_argv_contract` | Fake-pi log contains `--provider openai --model openai/gpt-5.6-terra --offline -p --no-session`; exit 0 |
| `test_launch_exit_code_passthrough` | `FAKE_PI_EXIT=7` → `pi-run` exits 7 (pi's code passes through) |
| `test_launch_key_via_env_not_argv` | Log has `KEY_OPENAI:testvalue`; the `ARGS:` line contains no key value |
| `test_launch_model_tier_flag` | `--model-tier fast` → `--model openai/gpt-5.6-mini` in argv |
| `test_launch_model_tier_env` | `PI_MODEL_TIER=cheap` → `--model openai/gpt-5.1-mini` in argv |
| `test_launch_model_flag_wins_over_tier_env` | `--model openai/gpt-5.6-terra` + `PI_MODEL_TIER=cheap` → argv model is the explicit flag |
| `test_launch_usage_errors_do_not_launch` | `--model-tier turbo` → exit 2; fake-pi log absent (no launch) |
| `test_launch_model_tier_and_model_conflict` | `--model-tier fast --model x` → exit 2; no launch |
| `test_launch_model_tier_unavailable_exit2` | `--provider deepseek --model-tier cheap` → exit 2, message lists `(available: balanced, fast)` |
| `test_launch_resume_rejects_tier` | `resume --model-tier fast` → exit 2 |

**`eval/tests/test_contract_otel.py`** (fixtures: `pi_run_bin`,
`hermetic_env`, `fake_launch_env`, `fake_collector`):

| Test | Asserts |
|---|---|
| `test_otel_no_endpoint_no_requests` | Endpoint unset → collector request count 0 |
| `test_otel_single_trace_post` | Endpoint set → exactly one POST `/v1/traces`, `Content-Type: application/json`; span `name "invoke_agent"`, `kind 2`, `status.code 1`; attributes `gen_ai.agent.name`, `gen_ai.provider.name "openai"`, `gen_ai.agent.model`, `pi_harness.run.mode "print"`, `pi_harness.run.exit_code 0` |
| `test_otel_exit_code_attribute_reflects_pi_exit` | `FAKE_PI_EXIT=7` → attribute `pi_harness.run.exit_code 7`, `status.code 2`; `pi-run` still exits 7 |
| `test_otel_collector_500_warns_and_exit_unchanged` | Collector status 500 → exit unchanged; exactly one `pi-run: warning: telemetry export failed` line on stderr mentioning 500; stdout clean |
| `test_otel_unreachable_endpoint_warns` | Endpoint `http://127.0.0.1:1` → exit unchanged + exactly one warning |

**`eval/tests/test_contract_project_understand.py`** (fixtures: `pi_run_bin`,
`hermetic_env` with `HARNESS_ROOT=<fixture project>`):

| Test | Asserts |
|---|---|
| `test_project_understand_writes_three_docs` | Exit 0; `product.md`/`tech.md`/`structure.md` written with the fixture's markers; stdout announces the docs |
| `test_project_understand_deterministic_reruns` | Two runs → all three files byte-identical |
| `test_project_understand_output_hygiene` | No absolute tmp path, no placeholder secret value in any generated doc |
| `test_project_understand_usage_error_exit2` | `--bogus` → exit 2 |
| `test_project_understand_default_out` | No `--out` → docs land in `<HARNESS_ROOT>/docs/understand` (the `:15` contract) |

## 7. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Golden fixture becomes a wall-to-wall string diff (float formatting, field order) | Low | Fixture is generated with `-update` and reviewed at creation; `MarshalIndent` + struct field order make output stable; the schema is frozen (v1, §3); `reflect.DeepEqual` fallback comparison isolates value vs. formatting drift |
| `scorecardNow` seam leaks across tests (flaky suite) | Low | `TestScorecardNowRestored` + `t.Cleanup` restore pattern enforced in the spec's test table |
| Stale local binary silently runs the contract suite | Medium | `pi_run_bin` probe fails loudly (not skips) on a binary without `mcp-server`/exit-8 support; CI job builds fresh (§4.4/§4.5) |
| MCP stdio tests hang on notification silence or EOF edge cases | Medium | `select`-bounded reads in the client; existing pytest-timeout 120 s global bound (`eval/pytest.ini`); every process killed in fixture teardown |
| Fake collector/server ports flake under CI load | Low | `127.0.0.1:0` ephemeral port (no fixed-port collisions); daemon threads + explicit `shutdown()` |
| `conftest.py`/`providers.json`/usage text drift past the guards | Low | Two-sided drift guards (§4.13) fail on either side; catalog growth already has the `TestSupportedProviderKeyEnvsCoverCatalog` precedent |
| contract suite slows push CI | Low | Dedicated job isolates the ~30-60 s Go build from `python-quick`; the ~40 new tests (8 Go + ~36 Python) are subprocess-fast and hermetic |
| `scorecard-latest.json` write failure breaks the artifact chain | Low | `runScorecard` treats a latest-write error as a run failure (same class as `writeScorecard`); unit test pins path/overwrite semantics |

## 8. Decision

**Recommend proceeding.** This spec converts two classes of untested behavior
into pinned contracts with zero user-facing change: the scorecard artifact
becomes byte-deterministic and fully CLI-owned (seam + golden + latest-pointer
all hermetic-tested), and the shipped CLI surface (`mcp-server`,
`--model-tier`, OTel, `project-understand`, exit codes) gets an end-to-end
hermetic Python suite running against the **real binary**, with CI jobs that
build that binary first and a workflow that loses its shell glue. Cost: ~30
lines of production Go, ~8 new Go tests + ~36 new Python tests (43 total),
one committed fixture, five Python
test files, one CI job. Constraints honored: stdlib-only Go, no new Python
deps, no live evals, no dataset growth — the deferred items (pytest-json-report
and friends) belong to the live-eval v2 spec.

## Review checklist

A reviewer can verify the implementation against this spec by checking:

- [ ] **Seam:** `scorecardNow` package var exists in `internal/cli/scorecard.go`;
      `:648` and `:673` are its only `time.Now()` call sites in the artifact
      path (`grep time.Now internal/cli/scorecard.go` shows only the seam + the
      benchmark-run timer at `:404`); production behavior unchanged with the
      var unset; `TestScorecardNowRestored` restores via `t.Cleanup`.
- [ ] **Golden:** `internal/cli/testdata/scorecard-golden.json` is committed;
      `TestScorecardGoldenJSON` passes with the seam pinned and matches
      `writeScorecard`'s byte output; `-update` rewrites it
      (`go test ./internal/cli/ -run TestScorecardGoldenJSON -update`); no
      other golden files added.
- [ ] **Stdlib-only:** `go.mod` still has zero `require` blocks; no go-cmp /
      goldie anywhere in the diff.
- [ ] **Latest-pointer:** `writeScorecardLatest` exists, is called by
      `runScorecard`, and `TestWriteScorecardLatest` pins path + byte-equality
      with the run file + overwrite semantics; `provider-scorecard.yml` no
      longer contains a `cp` step (diff shows only artifact upload/download
      around the gate).
- [ ] **CI builds the binary:** `ci.yml` `python-contract` job builds
      `pi-run` before pytest (same `go build -o bin/pi-run` +
      `$GITHUB_PATH` pattern as `provider-scorecard.yml:42-46`);
      `nightly-live-eval.yml` has a Build pi-run step before "Run full eval".
- [ ] **No new Python deps:** `eval/requirements.txt` is unchanged; the JSON-RPC
      client and fake collector are stdlib (`subprocess`, `json`, `select`,
      `http.server`).
- [ ] **MCP handshake:** the Python suite pins initialize → silent
      notification → tools/list → tools/call and EOF-exit-0; every MCP stdout
      line parses as JSON-RPC (stdout-is-sacred); error cases (malformed line,
      unknown method, unknown tool) keep the server alive.
- [ ] **Exit codes:** `test_exit_codes_table_is_source_of_truth` parses 0..8;
      subprocess tests pin 2-before-3-before-4 (usage beats missing key beats
      missing node) and observed codes match the table.
- [ ] **Fake launch:** `fake_launch_env` only stats
      `<HOME>/.nvm/versions/node/v22.19.0/bin/node` and executes the absolute
      `<binDir>/pi`; argv contract (`--provider/--model/--offline`) asserted;
      key travels via env, never argv.
- [ ] **OTel:** unset → zero requests; set → one POST `/v1/traces` with the
      `invoke_agent` span and `pi_harness.run.exit_code`; 500/unreachable →
      exactly one stderr warning and unchanged exit code.
- [ ] **Determinism:** two `project-understand` runs produce byte-identical
      docs; model-tier failures exit 2 and never launch the fake pi.
- [ ] **Drift guards:** `TestUsageDocumentsNewCommands` and
      `TestMCPProvidersKeyEnvMatchesPythonMirror` exist and fail on one-sided
      drift (add a command case / add a provider keyEnv without updating the
      mirror).
- [ ] **Scope:** no changes outside `internal/cli/scorecard.go`,
      `internal/cli/scorecard_test.go`, `app_test.go`, `mcp_test.go`,
      `eval/conftest.py`, `eval/tests/test_contract_*.py`, and the three
      workflow files; README, CHANGELOG, `providers.json`, `go.mod`, and
      `eval/requirements.txt` untouched by this lane.
