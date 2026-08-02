# Open-Source-Ready Harness — Design Spec

> **Status:** Approved (decision captured 2026-08-02)
> **Goal:** Turn the `harness` repository (pi-run CLI + DeepEval suite) into a
> public, clone-and-run open-source project that keeps users out of AI-provider
> vendor lock-in.

## Why this project is a good OSS candidate

The hard part of "avoid vendor lock-in" is already solved in the codebase:

- `pi-run` routes the same harness, same eval suite, same agent config to
  **OpenAI, OpenRouter, or DeepSeek** with `--provider` / `PI_PROVIDER` /
  `--model` pass-through (`internal/cli/providers.go`, `internal/cli/app.go`).
- Key resolution is env-first with a pluggable secret store
  (`internal/cli/keys.go`).
- The DeepEval pytest suite is cleanly separated under `eval/`.

The gap is **not functionality — it is portability and polish.** The repo is
currently personal: hardcoded user paths, a wrong system prompt, machine-specific
tests, Bitwarden-centric defaults, no license, no CI, no bootstrap.

## Decisions (locked)

| Decision | Choice |
|---|---|
| Scope | **Phased**: Phase 1 de-personalization → Phase 2 CI/CD + releases → Phase 3 provider-agnostic platform → Phase 4 docs/examples → Phase 5 hardening |
| Public home | **GitHub**, username **`forrestbthomas1`** (matches existing GitLab username) |
| Repo / module name | **`pi-harness`** → module `github.com/forrestbthomas1/pi-harness` |
| License | **MIT** |

> `gh` CLI is not installed on this machine; GitHub publishing will use
> `git push` to a new remote + web UI. No secrets are ever committed.

## Principles (from project conventions)

- **No external Go dependencies.** `pi-run` stays stdlib-only (no `go.sum`).
- **Deterministic, reproducible steps** over ad-hoc commands.
- **Never commit API keys, tokens, or kubeconfig contents.**
- **Keep changes minimal and focused**; do not refactor unrelated code.
- **Evidence over assertion**: every task ends with passing tests/builds.

---

## Phase 1 — De-personalization (clone-and-run)

### 1.1 LICENSE
- Add `LICENSE` (MIT) with current year + `forrestbthomas1`.

### 1.2 System prompt
- Rewrite `.pi/SYSTEM.md` from the leftover "Go + Kubernetes Controller
  Development Harness" prompt to an accurate description of **this** harness:
  a provider-agnostic coding-agent launcher + evaluation suite.
- Keep `.pi/APPEND_SYSTEM.md` guardrails (no keys in output, prefer venv, etc.)
  but remove the hardcoded `/Users/forrestthomas/...` file-path note and any
  personal assumptions.

### 1.3 Hardcoded user paths
- `.pi/settings.json` `skills` array: currently
  `["/Users/forrestthomas/Projects/tmp/agent-skills/skills"]` → make relative /
  documented / optional (do not ship a user-specific absolute path).
- `scripts/install-skills.sh:18`: `PROJECT_SETTINGS="/Users/forrestthomas/Projects/harness/..."`
  → derive from the script's own location (`git rev-parse --show-toplevel`).
- `internal/cli/config_check.go`: `~/Projects/tmp/agent-skills` check → make it
  optional / non-fatal on other machines.

### 1.4 Portable tests + health checks
- `pi-run config-check` / `doctor` must pass on a **fresh machine** that has
  Node + `pi` + a provider key. Personal checks (symlink into `~/bin`,
  `~/.zshrc` content, skills count) become **opt-in** via
  `PI_RUN_PERSONAL=1` or a `--personal` flag.
- `eval/tests/test_harness_config.py`: rewrite to test the **repo**, not the
  machine. Keep only: JSON validity, no literal keys, module path, defaults.
  Move personal-machine assertions behind a skip when `PI_RUN_PERSONAL` unset.

### 1.5 Key resolution
- Keep env-first as the **primary documented** path.
- Bitwarden (`bw_get`) becomes an optional, documented **example** of a
  pluggable secret store, not the default assumption.
- `internal/cli/keys.go` already supports `BW_GET` env override — document it.

### 1.6 Bootstrap
- Add `scripts/bootstrap.sh` (or extend `pi-run setup`) so a **fresh clone** can
  go from zero to `pi-run chat` with one command:
  1. Detect/install Node (nvm) + `pi` CLI.
  2. `go build -o bin/pi-run ./cmd/pi-run`.
  3. `pi-run setup` (venv + deps + model catalogs).
  4. Print a friendly "set your key" message.
- The README "Quick Start" must start from `git clone`, not from a pre-configured machine.

### 1.7 Cleanup of tracked personal artifacts
- Delete `.pi/settings.json.bak-pre-openrouter` (tracked, personal).
- Delete stub root `package-lock.json` (empty `packages: {}`, no purpose).
- Remove `github-repo-controller/bin/controller-gen` leftover (untracked; add to
  `.gitignore` or delete).
- Review the uncommitted `M AGENTS.md` change and commit it deliberately.

### 1.8 README rewrite
- "Quick Start" for a stranger: clone → bootstrap → set key (plain env var) →
  `pi-run chat`.
- Document env-var key resolution as the primary path; Bitwarden as optional.
- Add "Contributing" / "Security" pointers; remove personal-machine instructions.

---

## Phase 2 — CI/CD + releases

### 2.1 CI (GitHub Actions)
- `.github/workflows/ci.yml`: on push/PR
  - `go test ./...`, `go vet ./...`, `go build ./...`
  - `pytest --quick` (venv) — **skips live-LLM tests when no key** (already the behavior)
  - secret scan (gitleaks) — no key material in the repo
  - dependabot for Go + npm + GitHub Actions
- Optional `nightly-live-eval.yml` (key-gated, skips gracefully).

### 2.2 Releases
- Semver tags (`v0.1.0`, ...), goreleaser for multi-platform binaries + Homebrew
  tap, plus `go install github.com/forrestthomas1/pi-harness/cmd/pi-run@latest`.
- Document release flow in `CONTRIBUTING.md`.

---

## Phase 3 — Provider-agnostic platform (the lock-in killer)

### 3.1 Data-driven provider table
- Move the provider routing table (`internal/cli/providers.go` hardcoded slice)
  to data-driven config: `providers.yaml` (or env-driven) with
  `name / keyEnv / piProvider / defaultModel / optional baseURL`.
- Ship defaults: `openai`, `openrouter`, `deepseek`, **`anthropic`, `gemini`,
  `groq`, and local OpenAI-compatible (Ollama/vLLM)**.
- `pi-run providers` command lists configured providers/models.
- Unknown provider → actionable error (already present; keep).

### 3.2 Eval-judge provider flexibility
- DeepEval's LLM-as-a-judge currently defaults to OpenAI implicitly.
- Document + test `DEEPEVAL_MODEL` and provider key selection so eval can run
  against the same non-OpenAI providers.
- `eval/conftest.py` `get_secret()` already supports provider keys; make the
  judge model configurable and non-OpenAI-first.

### 3.3 Docs: "Avoid vendor lock-in"
- New `docs/anti-lockin.md`: BYO-key, BYO-model, local models, zero-code
  provider additions, side-by-side provider comparisons.

---

## Phase 4 — Docs, examples, community

- `examples/`:
  - custom eval (new metric)
  - add a provider (config-based)
  - run against a local model (Ollama/vLLM OpenAI-compatible)
  - compare providers side-by-side
- `docs/architecture.md`, ADRs, FAQ, changelog.
- README hero section + screenshots.

---

## Phase 5 — Hardening (optional later)

- Eval output reports + run caching (save $).
- Sandboxed Pi execution (containers).
- Benchmark leaderboard across providers.

---

## Out of Scope (this pass)

- No changes to the DeepEval test logic itself (suite stays pytest).
- No interactive provider picker.
- No config-file parser beyond the provider table (no `viper`).
- No removal of the personal rc-file key exports on this machine.
- GitHub publishing itself (push + web UI) is a manual step after Phase 2 CI is green.
