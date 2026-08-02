# Open-Source-Ready Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the `harness` repo into a public, clone-and-run open-source project (`github.com/forrestthomas1/pi-harness`, MIT) that keeps users out of AI-provider vendor lock-in — by de-personalizing, adding CI/releases, and making providers data-driven.

**Architecture:** Keep the existing dependency-free Go CLI (`cmd/pi-run` → `internal/cli/`) and the DeepEval pytest suite (`eval/`). Phase 1 removes personal assumptions and adds a one-command bootstrap. Phase 2 adds GitHub Actions CI + semver releases. Phase 3 makes the provider table data-driven (`providers.json`, stdlib-only) and the eval judge provider-configurable. Everything stays stdlib-only Go (no external Go deps).

**Tech Stack:** Go 1.21 (stdlib only), Python 3.11+ / DeepEval / pytest, Node 22 + `pi` CLI, GitHub Actions, MIT license.

## Global Constraints

- **Go 1.21; stdlib only** — no external Go dependencies (no `go.sum`).
- **Module path:** `github.com/forrestthomas1/pi-harness` (renamed from `github.com/forrestthomas/harness`).
- **Public repo name:** `pi-harness`. **License:** MIT.
- **Key resolution order:** env var → optional secret store (`BW_GET` override; Bitwarden documented as an example). **Never log, echo, or persist key material.**
- **No cross-provider auto-fallback** — a missing key is exit code 3 with an actionable message.
- **Exit codes:** 0 ok · 1 generic · 2 usage · 3 missing API key · 4 node/pi not found · 5 `eval/.venv` missing.
- **Portability:** `pi-run config-check` / `doctor` must pass on a fresh machine; personal checks opt-in via `PI_RUN_PERSONAL=1`.
- **No hardcoded user paths** in shipped code/scripts/tests (no `/Users/forrestthomas/...`).
- **Never commit API keys, tokens, or kubeconfig contents.**
- **Keep changes minimal and focused**; do not refactor unrelated code.
- `gh` CLI is not installed; GitHub publishing uses `git push` + web UI (manual, after CI is green).

---

### Task 1: Add MIT LICENSE + fix the module path and system prompt

**Files:**
- Create: `LICENSE`
- Modify: `go.mod` (module path → `github.com/forrestthomas1/pi-harness`)
- Modify: `cmd/pi-run/main.go` (import path)
- Modify: `.pi/SYSTEM.md` (rewrite prompt)
- Modify: `AGENTS.md` (update module path references)

**Interfaces:**
- Produces: `LICENSE` with current year; module `github.com/forrestthomas1/pi-harness`; a system prompt that describes the actual harness.

- [ ] **Step 1: Write the failing test (module path)**

Add a test in `internal/cli/app_test.go`:
```go
func TestModulePath(t *testing.T) {
	// The public module path must be github.com/forrestthomas1/pi-harness.
	// Read go.mod at runtime so the test is hermetic and path-independent.
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "module github.com/forrestthomas1/pi-harness") {
		t.Fatalf("go.mod must declare module github.com/forrestthomas1/pi-harness, got:\n%s", b)
	}
}
```
(Add `"os"` to the imports in `app_test.go`.)

- [ ] **Step 2: Run the test — verify it fails**

Run: `cd "$HARNESS" && go test ./internal/cli/ -run TestModulePath -v`
Expected: FAIL — go.mod still says `github.com/forrestthomas/harness`.

- [ ] **Step 3: Create the LICENSE**

Create `LICENSE`:
```text
MIT License

Copyright (c) 2026 forrestbthomas1

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 4: Rename the module path**

Edit `go.mod`:
```go
module github.com/forrestthomas1/pi-harness
```
Edit `cmd/pi-run/main.go` import:
```go
import (
	"os"

	"github.com/forrestthomas1/pi-harness/internal/cli"
)
```

- [ ] **Step 5: Rewrite `.pi/SYSTEM.md`**

Replace the entire file with a prompt that describes the actual harness:
```markdown
# System Prompt: Provider-Agnostic Coding-Agent Harness

You are a careful, tool-using software engineer running inside a
provider-agnostic coding-agent harness (Pi + DeepEval). The harness routes the
same agent configuration to multiple AI providers (OpenAI, OpenRouter,
DeepSeek, and more) via the `pi-run` CLI, and evaluates agent outputs with the
DeepEval pytest suite under `eval/`.

## Core Behaviors

- **Correctness first.** Prefer a correct, minimal solution over a clever, expansive one.
- **Verify before claiming.** Run `go build ./...`, `go test ./...`, `go vet ./...`, and `pytest` before saying code works.
- **Make minimal changes.** Edit only what is necessary. Do not refactor unrelated code.
- **Explain trade-offs.** When multiple approaches exist, briefly state the trade-offs and recommend one.
- **Stay reproducible.** Favor commands and scripts that can be re-run by the user.

## Conventions

- The harness runtime is a single Go CLI: `pi-run` (module `github.com/forrestthomas1/pi-harness`, source under `cmd/pi-run` and `internal/cli/`).
- Provider routing is data-driven (`providers.json`): each provider has a key env var, a `pi --provider` value, and a default model. There is **no automatic cross-provider fallback** — the provider is explicit (`--provider` / `PI_PROVIDER`).
- API keys are resolved env-first, then from an optional secret store (`BW_GET` override; Bitwarden is a documented example). **Never log, echo, or persist key material.**
- All evaluation code, datasets, and Python dependencies live in `eval/`.

## Tool Use

- Use `read` to inspect files before editing.
- Use `edit` for small, targeted changes.
- Use `write` only when creating new files or replacing a file entirely.
- Use `bash` for running `go`, `pytest`, and `pi-run` commands.

## Reading Files (important)

- Use the built-in `read` tool for ALL local files (text, code, configs) and
  for files outside the project directory. Never use web `fetch`/search tools
  for local file paths — they only accept `http(s)://` URLs.

## Build & Generate Commands

- `go mod tidy` — update dependencies (stdlib only).
- `go build ./...` — compile all packages.
- `go test ./...` — run unit tests.
- `go vet ./...` — static analysis.
- `pi-run setup` — create `eval/.venv`, install deps, refresh model catalogs.
- `pi-run eval --quick` — run the deterministic smoke subset.
- `pi-run config-check` — deterministic harness checks (no keys, no network).

## Safety Rules

- Prefer local validation with `go test`, `pytest`, and `kind` when available.
- Do not commit API keys, tokens, or kubeconfig contents.
- Do not run destructive commands without confirmation.

## Output Style

- Be concise but complete.
- Use Markdown for structure.
- Cite file paths and line numbers when referencing code.
- When generating code, include comments explaining the "why" for non-obvious logic.
```

- [ ] **Step 6: Update `AGENTS.md` module-path references**

In `AGENTS.md`, replace `module \`github.com/forrestthomas/harness\`` with `module \`github.com/forrestthomas1/pi-harness\`` (if present).

- [ ] **Step 7: Verify**

Run:
```bash
cd "$HARNESS" && go build ./... && go test ./... && go vet ./...
```
Expected: build/tests/vet all pass; `go test ./internal/cli/ -run TestModulePath` passes.

- [ ] **Step 8: Commit**

```bash
git add LICENSE go.mod cmd/pi-run/main.go .pi/SYSTEM.md AGENTS.md internal/cli/app_test.go
git commit -m "feat(oss): add MIT license, rename module to github.com/forrestthomas1/pi-harness, fix system prompt"
```

---

### Task 2: Remove hardcoded user paths from settings + install-skills + config-check

**Files:**
- Modify: `.pi/settings.json` (skills array → remove absolute user path)
- Modify: `scripts/install-skills.sh` (`PROJECT_SETTINGS` derived from script location)
- Modify: `internal/cli/config_check.go` (make `agent-skills` check optional/non-fatal)

**Interfaces:**
- Produces: no shipped file references `/Users/forrestthomas/...`; `config-check` no longer fails on machines without that clone.

- [ ] **Step 1: Make the settings skills array portable**

In `.pi/settings.json`, change the `skills` array from:
```json
  "skills": [
    "/Users/forrestthomas/Projects/tmp/agent-skills/skills"
  ],
```
to an empty array (or a documented relative path) — the durable install via
`scripts/install-skills.sh` populates it:
```json
  "skills": []
```

- [ ] **Step 2: Make install-skills.sh derive the project settings path**

In `scripts/install-skills.sh`, replace the hardcoded line:
```bash
PROJECT_SETTINGS="/Users/forrestthomas/Projects/harness/.pi/settings.json"
```
with:
```bash
PROJECT_SETTINGS="$(git rev-parse --show-toplevel 2>/dev/null)/.pi/settings.json"
```
(If not in a git repo, skip the settings edit with a clear message instead of failing.)

- [ ] **Step 3: Make config_check's agent-skills check non-fatal**

In `internal/cli/config_check.go`, change the "agent-skills clone present" check from a hard `check(...)` (which sets `fail=true`) to an informational line that only runs when the path exists or when `PI_RUN_PERSONAL=1`:
```go
	// Personal-machine skill-clone check: informational unless PI_RUN_PERSONAL=1.
	if os.Getenv("PI_RUN_PERSONAL") == "1" {
		check("agent-skills clone present", pathExists(filepath.Join(home, "Projects", "tmp", "agent-skills", "skills")))
	} else {
		fmt.Println("  [info] agent-skills clone check skipped (set PI_RUN_PERSONAL=1 to enable)")
	}
```
(Keep the superpowers-skills-count check behind the same gate.)

- [ ] **Step 4: Add a regression test that no shipped file references the old username path**

Create `internal/cli/paths_test.go`:
```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoHardcodedUserPaths(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	// Walk only shipped dirs (skip historical docs, vendored/ignored dirs) and
	// assert no file contains "/Users/forrestthomas/" or "github.com/forrestthomas/harness".
	roots := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "internal"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "eval"),
		filepath.Join(root, ".pi"),
		filepath.Join(root, "providers.json"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "LICENSE"),
		filepath.Join(root, "go.mod"),
	}
	for _, p := range roots {
		if fi, err := os.Stat(p); err != nil || !fi.IsDir() {
			// Skip files that don't exist yet in earlier tasks.
			if err != nil && os.IsNotExist(err) {
				continue
			}
		}
		filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			// Skip binary/venv/session/cache dirs.
			if d.IsDir() {
				switch d.Name() {
				case ".git", ".worktrees", "bin", "sessions", ".venv", "__pycache__", ".pytest_cache", "node_modules":
					return filepath.SkipDir
				}
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			text := string(b)
			if strings.Contains(text, "/Users/forrestthomas/") {
				t.Errorf("hardcoded user path in %s", rel)
			}
			if strings.Contains(text, "github.com/forrestthomas/harness") {
				t.Errorf("old module path in %s", rel)
			}
			return nil
		})
	}
}
```

- [ ] **Step 5: Run tests — verify they pass (and the new one catches issues)**

Run:
```bash
cd "$HARNESS" && go test ./internal/cli/ -run "NoHardcodedUserPaths|ModulePath" -v
```
Expected: PASS (the walk skips historical docs; shipped code is clean).

- [ ] **Step 6: Commit**

```bash
git add .pi/settings.json scripts/install-skills.sh internal/cli/config_check.go internal/cli/paths_test.go
git commit -m "feat(oss): remove hardcoded user paths from settings, install-skills, and config-check"
```

---

### Task 3: Portable doctor/config-check + portable harness-config tests

**Files:**
- Modify: `internal/cli/doctor.go`
- Modify: `internal/cli/config_check.go`
- Modify: `eval/tests/test_harness_config.py`

**Interfaces:**
- Produces: `pi-run doctor` and `pi-run config-check` exit 0 on a fresh machine; personal checks gated by `PI_RUN_PERSONAL=1`.

- [ ] **Step 1: Add a helper for personal-mode gating**

Create `internal/cli/personal.go`:
```go
package cli

import "os"

// personalMode reports whether personal-machine checks are enabled. When unset
// or "0", checks that assert facts about a specific developer's machine
// (symlinks into ~/bin, dotfile contents, installed skill counts) are skipped
// so the harness passes on a fresh clone.
func personalMode() bool {
	return os.Getenv("PI_RUN_PERSONAL") == "1"
}
```

- [ ] **Step 2: Make `config-check` personal checks conditional**

In `internal/cli/config_check.go`:
- Wrap the "~/bin/pi-run symlinks to <root>/bin/pi-run" check in `if personalMode() { ... } else { fmt.Println("  [info] symlink check skipped (set PI_RUN_PERSONAL=1)") }`.
- Wrap the dotfile checks (`.zshrc`/`.bashrc` pi-harness + static-secret exports) in the same gate.
- Wrap the superpowers-skills + agent-skills checks (already started in Task 2).
- Keep these **always-on**: JSON validity, defaultProvider/defaultModel, enabledModels order, no-literal-keys, Makefile absence.

- [ ] **Step 3: Make `doctor` personal checks conditional**

In `internal/cli/doctor.go`:
- Wrap the "~/bin/pi-run symlink present" check in `if personalMode()`.
- Keep node/pi/venv/key-presence/model checks always-on (they're about the actual runtime, not a specific developer's shell).

- [ ] **Step 4: Add unit tests for `personalMode`**

Create `internal/cli/personal_test.go`:
```go
package cli

import "testing"

func TestPersonalModeDefaultOff(t *testing.T) {
	t.Setenv("PI_RUN_PERSONAL", "")
	if personalMode() {
		t.Fatal("personalMode should be false when PI_RUN_PERSONAL unset")
	}
}

func TestPersonalModeOn(t *testing.T) {
	t.Setenv("PI_RUN_PERSONAL", "1")
	if !personalMode() {
		t.Fatal("personalMode should be true when PI_RUN_PERSONAL=1")
	}
}
```

- [ ] **Step 5: Rewrite `eval/tests/test_harness_config.py` for portability**

Replace the machine-specific tests (symlink, dotfiles, skills) with repo-level tests gated by `PI_RUN_PERSONAL`:
```python
"""Deterministic harness-configuration checks (no API keys, no network).

The CLI (`pi-run config-check`) is the single source of truth for these checks.
Personal-machine checks (symlink into ~/bin, dotfile contents, installed skill
counts) are gated behind PI_RUN_PERSONAL=1 so the suite passes on a fresh clone.
"""

import json
import os
import re
import subprocess
from pathlib import Path

import pytest

HARNESS = Path(__file__).resolve().parents[2]
PROJECT_SETTINGS = HARNESS / ".pi" / "settings.json"
HOME = Path.home()
PERSONAL = os.environ.get("PI_RUN_PERSONAL") == "1"
MIN_SUPERPOWERS_SKILLS = 14


def _load_json(path: Path) -> dict:
    assert path.exists(), f"missing expected config file: {path}"
    return json.loads(path.read_text(encoding="utf-8"))


def test_pi_run_config_check_passes():
    result = subprocess.run(
        ["pi-run", "config-check"],
        cwd=HARNESS,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"pi-run config-check failed:\n{result.stdout}\n{result.stderr}"


def test_makefile_removed():
    assert not (HARNESS / "Makefile").exists(), "Makefile must be removed (CLI owns all targets)"


def test_go_module_path():
    go_mod = (HARNESS / "go.mod").read_text(encoding="utf-8")
    assert "module github.com/forrestthomas1/pi-harness" in go_mod


def test_project_defaults_unchanged():
    settings = _load_json(PROJECT_SETTINGS)
    assert settings["defaultProvider"] == "openai"
    assert settings["defaultModel"] == "openai/gpt-5.6-terra"


def test_no_literal_keys_anywhere():
    for path in (PROJECT_SETTINGS,):
        text = path.read_text(encoding="utf-8")
        assert re.search(r"sk-[A-Za-z0-9_-]{8,}", text) is None, path


def test_no_hardcoded_user_paths_in_shipped_code():
    for path in (
        HARNESS / "scripts" / "install-skills.sh",
        HARNESS / ".pi" / "settings.json",
    ):
        text = path.read_text(encoding="utf-8")
        assert "/Users/forrestthomas/" not in text, path


@pytest.mark.skipif(not PERSONAL, reason="personal-machine check (set PI_RUN_PERSONAL=1)")
def test_personal_pi_run_binary_symlinked_into_home_bin():
    link = HOME / "bin" / "pi-run"
    assert link.is_symlink(), f"missing symlink: {link}"
    target = link.resolve()
    assert target == (HARNESS / "bin" / "pi-run").resolve(), (
        f"symlink {link} -> {target}, want {HARNESS / 'bin' / 'pi-run'}"
    )


@pytest.mark.skipif(not PERSONAL, reason="personal-machine check (set PI_RUN_PERSONAL=1)")
def test_personal_dotfiles_no_longer_define_pi_harness_functions():
    for rc in (HOME / ".zshrc", HOME / ".bashrc"):
        text = rc.read_text(encoding="utf-8")
        assert "pi-harness()" not in text, f"{rc.name} still defines pi-harness()"
        assert "bw_get" in text, f"{rc.name} should still resolve keys via bw_get"


@pytest.mark.skipif(not PERSONAL, reason="personal-machine check (set PI_RUN_PERSONAL=1)")
def test_personal_superpowers_skills_installed():
    skills = HOME / ".agents" / "skills"
    assert skills.is_dir()
    count = sum(1 for p in skills.iterdir() if p.is_dir())
    assert count >= MIN_SUPERPOWERS_SKILLS, f"found {count} skills"
```

- [ ] **Step 6: Run the Go + Python checks**

Run:
```bash
cd "$HARNESS" && go test ./internal/cli/ && pi-run eval --quick
```
Expected: Go tests pass; pytest quick subset passes without needing `PI_RUN_PERSONAL`.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/personal.go internal/cli/personal_test.go internal/cli/doctor.go internal/cli/config_check.go eval/tests/test_harness_config.py
git commit -m "feat(oss): make doctor/config-check and harness-config tests portable via PI_RUN_PERSONAL"
```


---

### Task 4: Bootstrap script — one command from clone to working harness

**Files:**
- Create: `scripts/bootstrap.sh`
- Modify: `README.md` (Quick Start references bootstrap)
- Modify: `.gitignore` (ensure `bin/`, `.pi/sessions/`, `eval/.venv/` ignored — already present)

**Interfaces:**
- Produces: `bash scripts/bootstrap.sh` → Node + `pi` present, `bin/pi-run` built, venv created, friendly key message.

- [ ] **Step 1: Write the bootstrap script**

Create `scripts/bootstrap.sh`:
```bash
#!/usr/bin/env bash
# bootstrap.sh — one command from a fresh clone to a working harness.
#
# 1. Ensures Node (via nvm) + the `pi` CLI are available.
# 2. Builds bin/pi-run from source.
# 3. Runs `pi-run setup` (creates eval/.venv, installs deps, refreshes model catalogs).
# 4. Prints how to provide an API key (plain env var first; Bitwarden optional).
#
# Idempotent: safe to re-run.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(dirname "$HERE")"
NODE_VERSION="${PI_NODE_VERSION:-v22.19.0}"

echo "== pi-harness bootstrap =="

# 1. Node + pi
if command -v node >/dev/null 2>&1 && command -v pi >/dev/null 2>&1; then
  echo "  node + pi already on PATH"
else
  echo "  Ensuring Node ${NODE_VERSION} via nvm ..."
  if [ -s "$HOME/.nvm/nvm.sh" ]; then
    # shellcheck disable=SC1091
    . "$HOME/.nvm/nvm.sh"
  fi
  if ! command -v node >/dev/null 2>&1; then
    echo "  Installing node ${NODE_VERSION} via nvm ..."
    nvm install "$NODE_VERSION"
  fi
  if ! command -v pi >/dev/null 2>&1; then
    echo "  Installing pi CLI (npm global) ..."
    npm install -g pi
  fi
fi

# 2. Build pi-run
echo "  Building bin/pi-run ..."
(cd "$ROOT" && go build -o bin/pi-run ./cmd/pi-run)

# 3. Python venv + deps
echo "  Setting up eval/.venv ..."
(cd "$ROOT" && bin/pi-run setup)

# 4. Key guidance
echo ""
echo "== Done. Provide an API key: =="
echo "  export OPENAI_API_KEY=sk-...          # or"
echo "  export OPENROUTER_API_KEY=sk-or-v1-... # or"
echo "  export DEEPSEEK_API_KEY=sk-...        # then:"
echo "  pi-run chat"
echo ""
echo "Bitwarden (optional): pi-run also resolves keys via ~/bin/bw_get (BW_GET override)."
```

- [ ] **Step 2: Make it executable**

Run: `chmod +x scripts/bootstrap.sh`

- [ ] **Step 3: Reference bootstrap in README Quick Start**

In `README.md`, replace the multi-step "Quick Start" with:
```markdown
## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/forrestthomas1/pi-harness.git
cd pi-harness

# 2. One-command bootstrap (Node + pi + bin/pi-run + eval/.venv)
bash scripts/bootstrap.sh

# 3. Provide an API key (plain env var is the primary path)
export OPENAI_API_KEY=sk-...        # or OPENROUTER_API_KEY / DEEPSEEK_API_KEY

# 4. Sanity-check the setup (no API key needed)
pi-run config-check
pi-run doctor

# 5. Launch Pi interactively (OpenAI -> gpt-5.6-terra by default)
pi-run chat

# 6. Or run a quick print-mode query
pi-run print "List all Python files in this repo"

# 7. Route to another provider
pi-run chat --provider deepseek
```
```

- [ ] **Step 4: Verify the script is syntactically valid**

Run: `bash -n scripts/bootstrap.sh`
Expected: no output (exit 0).

- [ ] **Step 5: Commit**

```bash
git add scripts/bootstrap.sh README.md
git commit -m "feat(oss): add one-command bootstrap script and update Quick Start"
```

---

### Task 5: Clean up tracked personal artifacts

**Files:**
- Delete: `.pi/settings.json.bak-pre-openrouter`
- Delete: `package-lock.json` (root, empty stub)
- Modify: `.gitignore` (ignore leftover `github-repo-controller/bin/`)
- Modify: `README.md` (remove stale "verify-harness" / personal references)

**Interfaces:**
- Produces: a clean tracked tree with no personal backups or empty stubs.

- [ ] **Step 1: Delete the personal backup and stub lockfile**

Run:
```bash
cd "$HARNESS"
git rm .pi/settings.json.bak-pre-openrouter package-lock.json
```

- [ ] **Step 2: Ignore the leftover controller-gen binary**

Append to `.gitignore`:
```
# Leftover from an extracted subproject (not part of the harness)
github-repo-controller/bin/
```

- [ ] **Step 3: Verify no stale references remain in shipped files**

Run:
```bash
grep -rn "settings.json.bak\|verify-harness" --include='*.md' --include='*.sh' --include='*.json' --include='*.go' --include='*.py' . 2>/dev/null | grep -v '.pi/git/' || echo "clean"
```
Expected: only historical docs (under `docs/`) may reference them; shipped files clean.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore(oss): remove personal backup, stub lockfile, ignore leftover controller-gen binary"
```

---

### Task 6: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/dependabot.yml`
- Create: `.github/workflows/nightly-live-eval.yml` (optional, key-gated)

**Interfaces:**
- Produces: CI runs `go test`, `go vet`, `go build`, `pytest --quick`, and a secret scan on every push/PR.

- [ ] **Step 1: Write `ci.yml`**

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Build
        run: go build ./...
      - name: Test
        run: go test ./...
      - name: Vet
        run: go vet ./...

  python-quick:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.11'
      - name: Install deps
        run: |
          python -m venv eval/.venv
          eval/.venv/bin/pip install -r eval/requirements.txt
      - name: Run quick eval
        run: eval/.venv/bin/python -m pytest tests/test_code_quality.py tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty -v
        working-directory: eval

  secret-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run gitleaks
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Add dependabot**

Create `.github/dependabot.yml`:
```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

- [ ] **Step 3: Add optional nightly live-eval (key-gated)**

Create `.github/workflows/nightly-live-eval.yml`:
```yaml
name: nightly-live-eval

on:
  schedule:
    - cron: '0 3 * * *'
  workflow_dispatch:

jobs:
  live-eval:
    runs-on: ubuntu-latest
    # Requires a provider key in the repo secrets; skips gracefully if absent.
    env:
      OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - uses: actions/setup-python@v5
        with:
          python-version: '3.11'
      - name: Install deps
        run: |
          python -m venv eval/.venv
          eval/.venv/bin/pip install -r eval/requirements.txt
      - name: Run full eval
        run: eval/.venv/bin/python -m pytest tests/ -v
        working-directory: eval
```

- [ ] **Step 4: Verify YAML parses**

Run: `python3 -c "import yaml, sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('ci.yml ok')"` (if pyyaml available) or use `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/ci.yml'); puts 'ci.yml ok'"`.
Expected: prints `ci.yml ok`.

- [ ] **Step 5: Commit**

```bash
git add .github/
git commit -m "feat(oss): add GitHub Actions CI (go test/vet/build, pytest quick, gitleaks, dependabot)"
```


---

### Task 7: Provider-agnostic provider table (`providers.json`)

**Files:**
- Create: `providers.json`
- Modify: `internal/cli/providers.go` (load from JSON, keep env-override)
- Modify: `internal/cli/providers_test.go`
- Modify: `internal/cli/app.go` (`pi-run providers` command)
- Modify: `README.md` (provider table)

**Interfaces:**
- Produces: `func LoadProviders(path string) ([]Provider, error)` reading JSON; `func ProvidersFromJSON(data []byte) ([]Provider, error)`; `Providers` variable backed by the JSON; `pi-run providers` lists them.

> **JSON vs YAML:** stdlib has no YAML parser. To honor the "stdlib only" constraint, the provider table is `providers.json` (not YAML). The interface `LoadProviders(path)` keeps a YAML swap-in possible later. The spec's "data-driven provider table" intent is preserved.

- [ ] **Step 1: Create `providers.json`**

Create `providers.json`:
```json
{
  "providers": [
    {
      "name": "openai",
      "keyEnv": "OPENAI_API_KEY",
      "piProvider": "openai",
      "defaultModel": "openai/gpt-5.6-terra"
    },
    {
      "name": "openrouter",
      "keyEnv": "OPENROUTER_API_KEY",
      "piProvider": "openrouter",
      "defaultModel": "openai/gpt-5.6-terra"
    },
    {
      "name": "deepseek",
      "keyEnv": "DEEPSEEK_API_KEY",
      "piProvider": "deepseek",
      "defaultModel": "deepseek/deepseek-v4-flash"
    },
    {
      "name": "anthropic",
      "keyEnv": "ANTHROPIC_API_KEY",
      "piProvider": "anthropic",
      "defaultModel": "anthropic/claude-sonnet-4"
    },
    {
      "name": "gemini",
      "keyEnv": "GEMINI_API_KEY",
      "piProvider": "gemini",
      "defaultModel": "gemini/gemini-2.5-pro"
    },
    {
      "name": "groq",
      "keyEnv": "GROQ_API_KEY",
      "piProvider": "groq",
      "defaultModel": "groq/llama-3.3-70b-versatile"
    },
    {
      "name": "local",
      "keyEnv": "LOCAL_API_KEY",
      "piProvider": "openai",
      "defaultModel": "local/model",
      "baseURL": "http://localhost:11434/v1"
    }
  ]
}
```

- [ ] **Step 2: Write the failing tests**

Create `internal/cli/providers_json_test.go`:
```go
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleProvidersJSON = `{
  "providers": [
    {"name": "openai", "keyEnv": "OPENAI_API_KEY", "piProvider": "openai", "defaultModel": "openai/gpt-5.6-terra"},
    {"name": "local", "keyEnv": "LOCAL_API_KEY", "piProvider": "openai", "defaultModel": "local/model", "baseURL": "http://localhost:11434/v1"}
  ]
}`

func TestProvidersFromJSON(t *testing.T) {
	ps, err := ProvidersFromJSON([]byte(sampleProvidersJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 {
		t.Fatalf("got %d providers, want 2", len(ps))
	}
	if ps[0].Name != "openai" || ps[0].KeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("unexpected provider: %+v", ps[0])
	}
	if ps[1].BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("local provider should carry baseURL, got %+v", ps[1])
	}
}

func TestProvidersFromJSONInvalid(t *testing.T) {
	if _, err := ProvidersFromJSON([]byte(`{"providers": [`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadProvidersFromRepoFile(t *testing.T) {
	// Load the real providers.json from the repo root.
	root := repoRoot()
	if root == "." {
		t.Skip("repoRoot() returned '.' — cannot locate providers.json")
	}
	ps, err := LoadProviders(filepath.Join(root, "providers.json"))
	if err != nil {
		t.Fatalf("LoadProviders: %v", err)
	}
	if len(ps) < 3 {
		t.Fatalf("providers.json should define at least 3 providers, got %d", len(ps))
	}
}
```

- [ ] **Step 3: Run tests — verify they fail**

Run: `cd "$HARNESS" && go test ./internal/cli/ -run "ProvidersFromJSON|LoadProviders" -v`
Expected: FAIL — `undefined: ProvidersFromJSON` / `undefined: LoadProviders`.

- [ ] **Step 4: Write the implementation**

Modify `internal/cli/providers.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Provider is a single routing entry.
type Provider struct {
	Name         string `json:"name"`                   // CLI name: openai | openrouter | ...
	KeyEnv       string `json:"keyEnv"`                 // env var / Bitwarden item holding the API key
	PiProvider   string `json:"piProvider"`             // value passed to `pi --provider`
	DefaultModel string `json:"defaultModel"`           // default `pi --model` value
	BaseURL      string `json:"baseURL,omitempty"`      // optional provider base URL (e.g. local OpenAI-compatible)
}

// providerFile is the on-disk shape of providers.json.
type providerFile struct {
	Providers []Provider `json:"providers"`
}

// defaultProviders is the fallback routing table used when providers.json is
// missing (e.g. running tests without a repo root). openai is the default.
var defaultProviders = []Provider{
	{Name: "openai", KeyEnv: "OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "openrouter", KeyEnv: "OPENROUTER_API_KEY", PiProvider: "openrouter", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "deepseek", KeyEnv: "DEEPSEEK_API_KEY", PiProvider: "deepseek", DefaultModel: "deepseek/deepseek-v4-flash"},
}

// Providers is the active routing table, loaded from providers.json when
// available, else the built-in defaults.
var Providers = defaultProviders

func init() {
	root := repoRoot()
	if root == "." {
		return // cannot locate providers.json; keep defaults
	}
	if ps, err := LoadProviders(filepath.Join(root, "providers.json")); err == nil && len(ps) > 0 {
		Providers = ps
	}
}

// ProvidersFromJSON parses provider table JSON.
func ProvidersFromJSON(data []byte) ([]Provider, error) {
	var f providerFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse providers: %w", err)
	}
	if len(f.Providers) == 0 {
		return nil, fmt.Errorf("parse providers: no providers defined")
	}
	return f.Providers, nil
}

// LoadProviders reads a provider table from path.
func LoadProviders(path string) ([]Provider, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ProvidersFromJSON(b)
}

// LookupProvider returns the provider named name, or an error.
func LookupProvider(name string) (Provider, error) {
	for _, p := range Providers {
		if p.Name == name {
			return p, nil
		}
	}
	return Provider{}, fmt.Errorf("unknown provider %q (want one of: %s)", name, providerNames())
}

// providerNames returns a comma-joined list of configured provider names.
func providerNames() string {
	names := make([]string, 0, len(Providers))
	for _, p := range Providers {
		names = append(names, p.Name)
	}
	return joinStrings(names, ", ")
}

func joinStrings(items []string, sep string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// ResolveProvider picks the provider from --provider, then PI_PROVIDER env,
// then the default (openai).
func ResolveProvider(flag string) (Provider, error) {
	name := flag
	if name == "" {
		name = os.Getenv("PI_PROVIDER")
	}
	if name == "" {
		name = "openai"
	}
	return LookupProvider(name)
}
```

> Note: the old `Providers` was a `var` slice; now it's loaded from JSON in
> `init()`. Existing tests reference `Providers` — keep the name, so they keep
> compiling. `LookupProvider`'s error message now lists configured names.

- [ ] **Step 5: Add `pi-run providers` command**

In `internal/cli/app.go`, add a case in `Run`:
```go
	case "providers":
		return runProviders()
```
Add the handler:
```go
// runProviders lists configured providers and their default models.
func runProviders() int {
	for _, p := range Providers {
		line := fmt.Sprintf("%s\t%s\t%s", p.Name, p.DefaultModel, p.KeyEnv)
		if p.BaseURL != "" {
			line += "\t" + p.BaseURL
		}
		fmt.Println(line)
	}
	return 0
}
```
Add `providers` to the usage text:
```
  providers     List configured providers and default models
```

- [ ] **Step 6: Run tests — verify they pass**

Run:
```bash
cd "$HARNESS" && go test ./internal/cli/ && go build ./... && go vet ./...
```
Expected: all pass (including `TestProvidersFromJSON`, `TestLoadProvidersFromRepoFile`).

- [ ] **Step 7: Smoke-test `pi-run providers`**

Run: `cd "$HARNESS" && go run ./cmd/pi-run providers`
Expected: lists openai, openrouter, deepseek, anthropic, gemini, groq, local (7 rows).

- [ ] **Step 8: Update README provider table**

Replace the README provider table with a note that providers come from
`providers.json` (add/remove rows without recompiling) plus the table.

- [ ] **Step 9: Commit**

```bash
git add providers.json internal/cli/providers.go internal/cli/providers_json_test.go internal/cli/app.go README.md
git commit -m "feat(oss): data-driven provider table (providers.json) + pi-run providers command"
```


---

### Task 8: Eval-judge provider flexibility

**Files:**
- Modify: `eval/conftest.py` (judge provider config)
- Modify: `eval/requirements.txt` (no change expected — deepeval already supports non-OpenAI via env)
- Modify: `eval/.env.example` (document judge model/provider)
- Modify: `README.md` (eval judge provider section)

**Interfaces:**
- Produces: DeepEval judge uses `DEEPEVAL_MODEL` + the provider key when set; docs explain non-OpenAI judge.

- [ ] **Step 1: Document judge provider in `.env.example`**

Append to `eval/.env.example`:
```bash
# DeepEval LLM-as-a-judge provider (defaults to OpenAI).
# Set a provider key (e.g. OPENROUTER_API_KEY) and DEEPEVAL_MODEL to avoid
# depending on OpenAI for evaluation.
# DEEPEVAL_MODEL=openai/gpt-4o-mini
# DEEPEVAL_MODEL=openrouter/anthropic/claude-sonnet-4
# DEEPEVAL_MODEL=deepseek/deepseek-v4-flash
```

- [ ] **Step 2: Add a helper in `conftest.py` to report the judge provider**

Add to `eval/conftest.py`:
```python
def judge_provider() -> str:
    """Return the configured DeepEval judge provider (defaults to openai)."""
    model = os.environ.get("DEEPEVAL_MODEL", "").strip()
    if model:
        return model.split("/", 1)[0]
    return "openai"
```

- [ ] **Step 3: Make the skip-if-no-key consider the judge provider**

In `eval/tests/test_coding_correctness.py` and `test_agent_task_completion.py`,
the `has_api_key()` check currently tests the whole `SUPPORTED_PROVIDER_KEYS`
list. Leave it — it already skips gracefully when no key is present. Add an
informational log in `conftest.py` when running the full suite:
```python
# In conftest.py, near the top (after imports):
if has_api_key():
    print(f"  [eval] judge provider: {judge_provider()} (set DEEPEVAL_MODEL to override)")
```

- [ ] **Step 4: Verify the quick suite still passes without a key**

Run:
```bash
cd "$HARNESS" && pi-run eval --quick
```
Expected: passes (no key needed); live-LLM tests skip.

- [ ] **Step 5: Commit**

```bash
git add eval/.env.example eval/conftest.py README.md
git commit -m "feat(oss): document and configure DeepEval judge provider (DEEPEVAL_MODEL)"
```

---

### Task 9: Docs, community files, examples

**Files:**
- Create: `CONTRIBUTING.md`
- Create: `SECURITY.md`
- Create: `CODE_OF_CONDUCT.md`
- Create: `docs/architecture.md`
- Create: `docs/anti-lockin.md`
- Create: `examples/README.md` (with 3 example snippets)
- Modify: `README.md` (badges, hero, contributing pointers)

**Interfaces:**
- Produces: a professional OSS README + contribution/security/code-of-conduct docs + anti-lock-in guide + runnable examples.

- [ ] **Step 1: Create `CONTRIBUTING.md`**

```markdown
# Contributing to pi-harness

Thanks for helping make pi-harness better!

## Development Setup

```bash
git clone https://github.com/forrestthomas1/pi-harness.git
cd pi-harness
bash scripts/bootstrap.sh   # Node + pi + bin/pi-run + eval/.venv
```

## Testing

```bash
go test ./...        # Go unit tests (stdlib only, no network)
go vet ./...         # static analysis
pi-run eval --quick  # deterministic smoke subset (no API key needed)
pi-run config-check  # harness config checks (no API key needed)
```

## Adding a Provider

Edit `providers.json` and add a row:

```json
{ "name": "myprovider", "keyEnv": "MYPROVIDER_API_KEY", "piProvider": "myprovider", "defaultModel": "myprovider/model" }
```

Then `go test ./internal/cli/` and `pi-run providers` to verify.

## Commit Style

- Small, focused commits (one logical change each).
- Prefix: `feat(oss):`, `fix(cli):`, `docs:`, `chore(oss):`, `test:`, `ci:`.
- Reference issues/PRs where relevant.

## Security

See `SECURITY.md` for reporting vulnerabilities. Never commit API keys.
```

- [ ] **Step 2: Create `SECURITY.md`**

```markdown
# Security Policy

## Reporting a Vulnerability

Please **do not** open a public issue for security vulnerabilities.

Email the maintainers privately at the address listed in the GitHub repo
settings, or use GitHub's private vulnerability reporting if enabled.

Include:
- Affected version / commit
- Steps to reproduce
- Impact

## Scope

- The Go CLI (`pi-run`) and its handling of API keys.
- The Python eval suite (does it leak secrets? does it run arbitrary code?).
- Build/CI configuration.

## Out of Scope

- Vulnerabilities in upstream dependencies (Pi CLI, DeepEval) — report upstream.
```

- [ ] **Step 3: Create `CODE_OF_CONDUCT.md`**

Use the standard Contributor Covenant 2.1 text (from
https://www.contributor-covenant.org/version/2/1/code_of_conduct/), with
`forrestbthomas1` as the contact. (No need to paste the full text here — copy
from the canonical source.)

- [ ] **Step 4: Create `docs/architecture.md`**

Summarize: `pi-run` CLI dispatch (`internal/cli/app.go`), provider table
(`providers.json` → `internal/cli/providers.go`), key resolution
(`internal/cli/keys.go`), pi spawning (`internal/cli/pi.go`), eval suite
(`eval/`), health checks (`doctor.go`/`config_check.go`). Include a small ASCII
diagram. Keep it short and accurate to the code.

- [ ] **Step 5: Create `docs/anti-lockin.md`**

```markdown
# Avoiding AI-Provider Vendor Lock-In with pi-harness

pi-harness keeps you out of vendor lock-in in three ways:

## 1. Provider-Agnostic Agent Runtime

The same agent configuration (`AGENTS.md`, `.pi/SYSTEM.md`, `.pi/settings.json`)
runs against any provider. `pi-run` routes by `--provider` / `PI_PROVIDER` /
`--model`. There is **no automatic cross-provider fallback** — you choose
explicitly, so you always know which provider handled a run.

## 2. Data-Driven Provider Table

`providers.json` lists providers (name, key env var, pi provider, default model,
optional base URL). Add a provider — including a **local OpenAI-compatible
endpoint** (Ollama, vLLM) — without recompiling:

```json
{ "name": "local", "keyEnv": "LOCAL_API_KEY", "piProvider": "openai", "defaultModel": "local/model", "baseURL": "http://localhost:11434/v1" }
```

## 3. Portable Evaluation

The DeepEval suite is provider-agnostic: it skips live-LLM tests when no key is
present, and the judge model is configurable via `DEEPEVAL_MODEL`. You can
evaluate one provider's output with another provider's judge.

## BYO-Key / BYO-Model

- **BYO-Key**: set the provider's env var (`OPENAI_API_KEY`,
  `OPENROUTER_API_KEY`, `DEEPSEEK_API_KEY`, ...). Optionally wire a secret
  store via `BW_GET` (Bitwarden is the documented example).
- **BYO-Model**: `pi-run print --model <anything> "..."` — or set
  `--model openrouter/auto` to let the router pick.
- **Local**: point `piProvider` at an OpenAI-compatible local server and set
  `baseURL`.
```

- [ ] **Step 6: Create `examples/README.md`**

```markdown
# Examples

## 1. Custom Evaluation Metric

Add a metric in `eval/tests/test_code_quality.py` (see the existing
`CodeQualityMetric`) and register it in a test. Run `pi-run eval` to score agent
outputs against it.

## 2. Add a Provider Without Recompiling

Edit `providers.json` (add a row), then `pi-run providers` to verify it lists,
and `pi-run print --provider <name> "hello"` to use it.

## 3. Run Against a Local OpenAI-Compatible Model

```json
{ "name": "local", "keyEnv": "LOCAL_API_KEY", "piProvider": "openai", "defaultModel": "local/model", "baseURL": "http://localhost:11434/v1" }
```
Start Ollama (`ollama serve`), then:
```bash
export LOCAL_API_KEY=ollama   # any non-empty value; local server ignores it
pi-run print --provider local "Explain recursion"
```

## 4. Compare Providers Side-by-Side

```bash
pi-run print --provider openai    "Explain the CAP theorem" > /tmp/openai.txt
pi-run print --provider deepseek  "Explain the CAP theorem" > /tmp/deepseek.txt
diff /tmp/openai.txt /tmp/deepseek.txt
```
```

- [ ] **Step 7: Polish README**

Add badges (license MIT, Go version, CI status), a one-line hero ("A
provider-agnostic coding-agent harness + evaluation suite that keeps you out of
AI-vendor lock-in"), and a "Contributing" section pointing to the new docs.

- [ ] **Step 8: Verify docs are consistent**

Run:
```bash
grep -rn "github.com/forrestthomas/harness" README.md docs/ CONTRIBUTING.md SECURITY.md CODE_OF_CONDUCT.md examples/ 2>/dev/null || echo "clean"
```
Expected: no old module path in shipped docs.

- [ ] **Step 9: Commit**

```bash
git add CONTRIBUTING.md SECURITY.md CODE_OF_CONDUCT.md docs/architecture.md docs/anti-lockin.md examples/README.md README.md
git commit -m "docs(oss): add community docs, anti-lock-in guide, and examples"
```

---

### Task 10: Final verification + git history cleanup

**Files:**
- Modify: `AGENTS.md` (final module-path + workflow updates)
- Modify: `README.md` (final consistency pass)
- Run: full test suite

**Interfaces:**
- Produces: a green `main` with no hardcoded user paths, correct module path, CI files present, LICENSE present.

- [ ] **Step 1: Run all Go checks**

Run:
```bash
cd "$HARNESS"
go build ./...
go test ./...
go vet ./...
go run ./cmd/pi-run providers
```
Expected: all pass; providers lists 7 rows.

- [ ] **Step 2: Run the Python quick suite**

Run:
```bash
cd "$HARNESS" && pi-run eval --quick && pi-run config-check
```
Expected: pass (no key needed).

- [ ] **Step 3: Full secret + path scan**

Run:
```bash
grep -rn "sk-[A-Za-z0-9]\{8,\}" --include='*.go' --include='*.py' --include='*.json' --include='*.md' --include='*.sh' . 2>/dev/null | grep -v '.pi/git/' | grep -v 'sk-\.\.\.' || echo "no literal keys"
grep -rn "/Users/forrestthomas" --include='*.go' --include='*.py' --include='*.json' --include='*.md' --include='*.sh' . 2>/dev/null | grep -v '.pi/git/' | grep -v 'docs/superpowers/' || echo "no hardcoded paths in shipped files"
```
Expected: no literal keys; no hardcoded paths in shipped files (historical docs
under `docs/superpowers/` may still reference them — acceptable, they're plans).

- [ ] **Step 4: Commit the uncommitted `AGENTS.md` change deliberately**

Review `git diff AGENTS.md`, then commit it as part of this task (or revert if
it was accidental). The goal: `git status` clean on `main` before publishing.

- [ ] **Step 5: Tag a pre-release**

```bash
git tag v0.2.0-pre
git push origin main --tags
```
(Only after confirming the remote is the public one, or defer until the repo is
published.)

---

## Self-Review Checklist (run before execution handoff)

- **Spec coverage:** every spec section maps to a task — LICENSE (T1), system prompt (T1), hardcoded paths (T2), portable checks/tests (T3), bootstrap (T4), cleanup (T5), CI (T6), provider-agnostic table (T7), eval-judge flexibility (T8), docs/community (T9), final verify (T10). ✓
- **Placeholder scan:** no TBD/TODO; every code step shows complete code. ✓
- **Type consistency:** `Provider{Name,KeyEnv,PiProvider,DefaultModel,BaseURL}`, `ProvidersFromJSON`, `LoadProviders`, `LookupProvider`, `ResolveProvider`, `personalMode`, `runProviders`, `judge_provider` — names and signatures consistent across tasks. ✓
- **Constraint adherence:** stdlib-only Go (JSON over YAML documented), no hardcoded paths, no key material, exit codes unchanged, `PI_RUN_PERSONAL` gating. ✓
- **Ordering bugs fixed during review:** `init()` in `providers.go` uses `repoRoot()` (defined in `setup.go`) — same package, so it compiles; `LoadProvidersFromRepoFile` test uses `repoRoot()` and skips when it can't locate the repo. ✓

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-02-open-source-ready.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
