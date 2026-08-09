# Pluggable Secret Manager + De-personalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make pi-harness's secret resolution pluggable (Bitwarden, 1Password, env-only via `PI_SECRET_BACKEND`) and remove personal/system-specific content from public docs and code comments.

**Architecture:** Introduce a `SecretBackend` interface in `internal/cli/secret.go` with three built-in adapters (bitwarden via `bw_get`, 1password via `op`, env-only), selected by `PI_SECRET_BACKEND` (default `bitwarden`). `resolveSecret` (Go) and `get_secret` (Python) become backend-aware while staying env-first. Then de-personalize README paths and code comments.

**Tech Stack:** Go 1.21 (stdlib only), Python 3.11 + pytest, GitHub Actions.

## Global Constraints

- Go 1.21, stdlib-only (no new Go deps).
- Python deps: `pyyaml` + `openai` only (plus pytest/pytest-mock dev) — no new Python deps.
- **Never log or echo secret values** — env-var names only, presence-only diagnostics.
- **Never commit hardcoded user paths** — enforced by `TestNoHardcodedUserPaths`.
- Backward compatible: `PI_SECRET_BACKEND` unset → `bitwarden` → identical behavior.
- Hermetic tests: fake `bw_get`/`op` scripts; no ambient credentials.
- Keep the Go module path `github.com/forrestthomas1/pi-harness` unchanged.
- Keep the `~/bin/pi-run` symlink *feature*; de-personalize only the *docs*.

---

### Task 1: `SecretBackend` interface + selection (Go)

**Files:**
- Create: `internal/cli/secret.go`
- Test: `internal/cli/secret_test.go`

**Interfaces:**
- Produces: `type SecretBackend interface { Name() string; Resolve(name string) (string, error); Status() (string, error) }`, `func newSecretBackend() (SecretBackend, error)`, `type BitwardenBackend struct{...}`, `type OnePasswordBackend struct{...}`, `type EnvOnlyBackend struct{...}`

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/secret_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeScript writes an executable shell script and returns its path.
func fakeScript(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSecretBackendSelectionBitwarden(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "")
	be, err := newSecretBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if be.Name() != "bitwarden" {
		t.Fatalf("got %q, want %q", be.Name(), "bitwarden")
	}
}

func TestSecretBackendSelectionOnePassword(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "1password")
	be, err := newSecretBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if be.Name() != "1password" {
		t.Fatalf("got %q, want %q", be.Name(), "1password")
	}
}

func TestSecretBackendSelectionEnvOnly(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "env-only")
	be, err := newSecretBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if be.Name() != "env-only" {
		t.Fatalf("got %q, want %q", be.Name(), "env-only")
	}
}

func TestSecretBackendUnknown(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "vault")
	if _, err := newSecretBackend(); err == nil {
		t.Fatal("expected error for unknown backend")
	} else if !strings.Contains(err.Error(), "vault") {
		t.Fatalf("error should mention backend name, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestSecretBackend -v`
Expected: FAIL (compile error: `undefined: newSecretBackend`)

- [ ] **Step 3: Write minimal implementation**

Create `internal/cli/secret.go`:

```go
package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SecretBackend resolves API keys from a secret manager. Implementations must
// never log or print secret values — only presence and status.
type SecretBackend interface {
	Name() string
	// Resolve returns the secret value for name, or an error if unavailable.
	Resolve(name string) (string, error)
	// Status returns a human-readable backend status for pi-run doctor.
	Status() (string, error)
}

// newSecretBackend returns the configured SecretBackend, selected by the
// PI_SECRET_BACKEND env var. The empty string means "bitwarden" (backward
// compatible with the pre-pluggable behavior).
func newSecretBackend() (SecretBackend, error) {
	switch os.Getenv("PI_SECRET_BACKEND") {
	case "", "bitwarden":
		return &bitwardenBackend{}, nil
	case "1password", "op":
		return &onePasswordBackend{}, nil
	case "env-only", "env":
		return &envOnlyBackend{}, nil
	default:
		return nil, fmt.Errorf("unknown secret backend %q (want bitwarden, 1password, or env-only)", os.Getenv("PI_SECRET_BACKEND"))
	}
}

// bitwardenBackend resolves via bw_get (BW_GET override; default ~/bin/bw_get).
type bitwardenBackend struct{}

func (b *bitwardenBackend) Name() string { return "bitwarden" }

func (b *bitwardenBackend) bwGetPath() (string, error) {
	if p := os.Getenv("BW_GET"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home dir: %w", err)
	}
	return filepath.Join(home, "bin", "bw_get"), nil
}

func (b *bitwardenBackend) Resolve(name string) (string, error) {
	p, err := b.bwGetPath()
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	var out bytes.Buffer
	cmd := exec.Command(p, name)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("resolve %s: bw_get failed (vault locked? run `bw unlock`): %w", name, err)
	}
	v := strings.TrimSpace(out.String())
	if v == "" {
		return "", fmt.Errorf("resolve %s: bw_get returned an empty value", name)
	}
	return v, nil
}

func (b *bitwardenBackend) Status() (string, error) {
	p, err := b.bwGetPath()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(p, "--status").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// onePasswordBackend resolves via `op read "op://<Vault>/<name>/credential"`.
// The op CLI must be installed and signed in.
type onePasswordBackend struct{}

func (b *onePasswordBackend) Name() string { return "1password" }

func (b *onePasswordBackend) Resolve(name string) (string, error) {
	vault := os.Getenv("OP_VAULT")
	if vault == "" {
		vault = "Personal"
	}
	ref := fmt.Sprintf("op://%s/%s/credential", vault, name)
	var out bytes.Buffer
	cmd := exec.Command("op", "read", ref)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("resolve %s: op read failed (is 1Password CLI signed in? run `op signin`): %w", name, err)
	}
	v := strings.TrimSpace(out.String())
	if v == "" {
		return "", fmt.Errorf("resolve %s: op read returned an empty value", name)
	}
	return v, nil
}

func (b *onePasswordBackend) Status() (string, error) {
	out, err := exec.Command("op", "account", "list").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// envOnlyBackend resolves only from the environment; no fallback.
type envOnlyBackend struct{}

func (b *envOnlyBackend) Name() string { return "env-only" }

func (b *envOnlyBackend) Resolve(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("resolve %s: not set in environment (env-only backend)", name)
}

func (b *envOnlyBackend) Status() (string, error) {
	return "env-only (no secret manager)", nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestSecretBackend -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/secret.go internal/cli/secret_test.go
git commit -m "feat(secret): add pluggable SecretBackend interface (bitwarden/1password/env-only)"
```

---

### Task 2: Wire `resolveSecret` to the backend (Go)

**Files:**
- Modify: `internal/cli/keys.go:1-40` (full rewrite of the fallback logic)
- Modify: `internal/cli/keys_test.go` (add 1Password test)

**Interfaces:**
- Consumes: `newSecretBackend() (SecretBackend, error)` from Task 1
- Produces: updated `resolveSecret(name string) (string, error)` — same signature, backend-aware

- [ ] **Step 1: Write the failing test (1Password fallback)**

Append to `internal/cli/keys_test.go`:

```go
func TestResolveSecretFallsBackToOnePassword(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "1password")
	// fake op on PATH
	opBin := fakeScript(t, "op", "#!/bin/sh\nprintf '%s\\n' 'sk-op-value'\n")
	t.Setenv("PATH", filepath.Dir(opBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := resolveSecret("OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-op-value" {
		t.Fatalf("got %q, want %q", got, "sk-op-value")
	}
}

func TestResolveSecretEnvOnlyNoFallback(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "env-only")
	if _, err := resolveSecret("OPENAI_API_KEY"); err == nil {
		t.Fatal("expected error when env var unset with env-only backend")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestResolveSecret -v`
Expected: FAIL — `TestResolveSecretFallsBackToOnePassword` fails (the current code always calls `bw_get`, ignoring `PI_SECRET_BACKEND`), `TestResolveSecretEnvOnlyNoFallback` fails (current code falls back to `bw_get`).

- [ ] **Step 3: Rewrite `resolveSecret` to use the backend**

Replace the body of `resolveSecret` in `internal/cli/keys.go` with:

```go
// resolveSecret returns the value of env var name, falling back to the
// configured secret backend (PI_SECRET_BACKEND; default bitwarden via bw_get).
// Never logs values.
func resolveSecret(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	be, err := newSecretBackend()
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	return be.Resolve(name)
}
```

Remove now-unused imports (`bytes`, `os/exec`, `path/filepath`, `strings`) from `keys.go` — they move to `secret.go`. The file should only import `fmt` and `os`.

- [ ] **Step 4: Run all secret tests to verify they pass**

Run: `go test ./internal/cli/ -run TestResolveSecret -v`
Expected: PASS (env-priority, bitwarden fallback via `BW_GET`, 1Password fallback, env-only no-fallback, bw_get-fails)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/keys.go internal/cli/keys_test.go
git commit -m "feat(secret): wire resolveSecret to the pluggable SecretBackend"
```

---

### Task 3: `doctor` uses backend status (Go)

**Files:**
- Modify: `internal/cli/doctor.go:36-42` (vault status block)

**Interfaces:**
- Consumes: `newSecretBackend()`, `SecretBackend.Status()` from Task 1

- [ ] **Step 1: Write the failing test**

The doctor status logic is currently hard to test directly (it prints to stdout). Add a small helper in `doctor.go` and test it:

Create `internal/cli/doctor_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackendStatusBitwarden(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "bitwarden")
	fake := fakeScript(t, "bw_get", "#!/bin/sh\nprintf '%s\\n' 'unlocked'\n")
	t.Setenv("BW_GET", fake)
	be, err := newSecretBackend()
	if err != nil {
		t.Fatal(err)
	}
	status, err := be.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "unlocked" {
		t.Fatalf("got %q, want %q", status, "unlocked")
	}
}

func TestBackendStatusOnePassword(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "1password")
	opBin := fakeScript(t, "op", "#!/bin/sh\nprintf '%s\\n' 'account list output'\n")
	t.Setenv("PATH", filepath.Dir(opBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	be, err := newSecretBackend()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := be.Status(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they pass (they use Task 1's backend)**

Run: `go test ./internal/cli/ -run TestBackendStatus -v`
Expected: PASS (the backend `Status()` methods already exist from Task 1)

- [ ] **Step 3: Wire `doctor` to use `backend.Status()`**

In `internal/cli/doctor.go`, replace lines 36-42 (the `bw_get --status` block):

```go
	// Secret backend status (informational; never a value).
	be, err := newSecretBackend()
	if err != nil {
		check("secret backend", false)
	} else if status, err := be.Status(); err == nil {
		fmt.Printf("  [info] %s backend: %s", be.Name(), status)
	} else {
		check(be.Name()+" backend reachable", false)
	}
```

Remove the now-unused `exec` import from `doctor.go` if it was only used for `bw_get --status` (check the file's other uses first).

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/doctor.go internal/cli/doctor_test.go
git commit -m "feat(secret): doctor reports configured secret backend status"
```

---

### Task 4: Python `get_secret` backend-aware (Python)

**Files:**
- Modify: `eval/conftest.py:96-119` (`get_secret`)
- Test: `eval/tests/test_secret_resolution.py`

**Interfaces:**
- Consumes: the Go backend conventions (`PI_SECRET_BACKEND`, `BW_GET`, `op`)
- Produces: updated `get_secret(name) -> str | None` — env first, then backend

- [ ] **Step 1: Write the failing test**

Append to `eval/tests/test_secret_resolution.py` (create if missing — check existing patterns first):

```python
import os
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import conftest


def test_get_secret_env_first(monkeypatch):
    monkeypatch.setenv("OPENAI_API_KEY", "sk-env-value")
    assert conftest.get_secret("OPENAI_API_KEY") == "sk-env-value"


def test_get_secret_one_password(monkeypatch, tmp_path):
    # Fake op CLI on PATH
    op_bin = tmp_path / "op"
    op_bin.write_text("#!/bin/sh\nprintf '%s\\n' 'sk-op-value'\n")
    op_bin.chmod(0o755)
    monkeypatch.setenv("PATH", str(tmp_path) + os.pathsep + os.environ.get("PATH", ""))
    monkeypatch.setenv("PI_SECRET_BACKEND", "1password")
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    assert conftest.get_secret("OPENAI_API_KEY") == "sk-op-value"


def test_get_secret_env_only_no_fallback(monkeypatch):
    monkeypatch.setenv("PI_SECRET_BACKEND", "env-only")
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    assert conftest.get_secret("OPENAI_API_KEY") is None
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd eval && .venv/bin/python -m pytest tests/test_secret_resolution.py -v`
Expected: FAIL — `test_get_secret_one_password` fails (current `get_secret` ignores `PI_SECRET_BACKEND`), `test_get_secret_env_only_no_fallback` fails.

- [ ] **Step 3: Rewrite `get_secret`**

Replace the body of `get_secret` in `eval/conftest.py`:

```python
def get_secret(name: str) -> str | None:
    """Return the secret for ``name``: env var first, then the configured backend.

    Backend is selected by PI_SECRET_BACKEND (default bitwarden). Bitwarden
    uses ``bw_get`` (BW_GET override, default ~/bin/bw_get); 1password uses
    ``op read "op://<vault>/<name>/credential"``; env-only never falls back.
    Never logs the value. Returns None when unavailable.
    """
    value = os.environ.get(name)
    if value:
        return value

    backend = os.environ.get("PI_SECRET_BACKEND", "bitwarden")

    if backend in ("env-only", "env"):
        return None

    if backend in ("1password", "op"):
        vault = os.environ.get("OP_VAULT", "Personal")
        try:
            result = subprocess.run(
                ["op", "read", f"op://{vault}/{name}/credential"],
                capture_output=True,
                text=True,
                timeout=30,
            )
        except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
            return None
        if result.returncode != 0:
            return None
        return result.stdout.strip() or None

    # bitwarden (default)
    bw_get = os.environ.get("BW_GET", str(Path.home() / "bin" / "bw_get"))
    try:
        result = subprocess.run(
            [bw_get, name],
            capture_output=True,
            text=True,
            timeout=30,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
        return None
    if result.returncode != 0:
        return None
    return result.stdout.strip() or None
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd eval && .venv/bin/python -m pytest tests/test_secret_resolution.py -v`
Expected: PASS (env-first, 1Password fallback, env-only no-fallback, bitwarden fallback)

- [ ] **Step 5: Run the full eval quick suite (no key)**

Run: `cd eval && .venv/bin/python -m pytest tests/test_code_quality.py "tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty" -q`
Expected: 3 passed, no key needed

- [ ] **Step 6: Commit**

```bash
git add eval/conftest.py eval/tests/test_secret_resolution.py
git commit -m "feat(secret): Python get_secret respects PI_SECRET_BACKEND"
```

---

### Task 5: README de-personalization

**Files:**
- Modify: `README.md` (lines 13, 30, 94, 131-138, 210, 226-232, 334)

**Interfaces:**
- Consumes: nothing from prior tasks; standalone docs change

- [ ] **Step 1: Edit README personal paths → generic**

Make these edits (using exact current text — verify with grep before editing):

1. Line 13: `a compiled Go binary (repo \`bin/pi-run\`, symlinked into \`~/bin/pi-run\`)` → `a compiled Go binary (repo \`bin/pi-run\`)`
2. Line 30: `Node.js 22.19+ (managed via \`nvm\`; \`pi-run\` resolves \`v22.19.0\` by default, override with \`PI_NODE_VERSION\`)` → `Node.js 22+ (managed via \`nvm\`; \`pi-run\` resolves a default Node 22 version, override with \`PI_NODE_VERSION\`)`
3. Line 94: `add \`bin/\` to your PATH or run \`pi-run install\` to symlink it into \`~/bin/\`` → `add \`bin/\` to your PATH or run \`pi-run install\` to symlink it into a directory on your PATH`
4. Lines 131-138 (API Key Resolution): replace the Bitwarden-specific block with a generic secret-manager description:

```markdown
### API Key Resolution (Secret Manager)

API keys are resolved **env-first** (e.g. `export OPENAI_API_KEY=...`), then
from a configured secret manager. The backend is selected by `PI_SECRET_BACKEND`
(default `bitwarden`):

- `bitwarden` — via the `bw_get` helper (override its path with `BW_GET`).
  Requires an unlocked vault (`bw unlock`).
- `1password` — via the `op` CLI (`op read "op://<Vault>/<ITEM_NAME>/credential"`).
  Requires `op` CLI installed and signed in. Vault defaults to `Personal`,
  override with `OP_VAULT`.
- `env-only` — no fallback; env var only.

Every `pi-run` path resolves keys in the same order: env var first, then the
backend. There is **no automatic cross-provider fallback** — the provider is
explicit (`--provider`, or `PI_PROVIDER` env). A missing key is an error that
tells you what to do:

```
no DEEPSEEK_API_KEY available: export it, or check your secret manager
```

`pi-run doctor` reports the configured backend's status (never values).
```

5. Line 210 (usage table): `Build \`bin/pi-run\` and symlink \`~/bin/pi-run\`` → `Build \`bin/pi-run\` and symlink it onto your PATH`
6. Lines 226-232 (Skills): replace `~/Projects/tmp/agent-skills/`, `git -C ~/Projects/tmp/...` with generic:

```markdown
Two curated collections are pre-installed:

- **Superpowers** (`obra/superpowers`): a skills collection (brainstorming,
  writing-plans, executing-plans, systematic-debugging,
  test-driven-development, ...). Refresh from
  https://github.com/obra/superpowers.
- **Addy Osmani's agent-skills**: cloned into a local skills directory and
  wired via the `skills` array in `.pi/settings.json`. Includes
  spec-driven-development, code-review-and-quality, test-driven-development,
  and more.

For a durable copy, run `bash scripts/install-skills.sh` once (with network).
It clones both collections into a Pi auto-discovered location, points the
settings `skills` arrays at the durable clone, and is idempotent — re-run it
any time to `git pull` both collections.
```

7. Line 334: `fetch a provider key via \`bw_get\`` → `fetch a provider key from your configured secret manager`

- [ ] **Step 2: Verify no personal paths remain**

Run: `grep -rn "~/Projects\|Dev API Keys\|forrestthomas@gmail\|v22.19.0\|~/bin/pi-run" README.md | grep -v "github.com/forrestthomas1" || echo "(clean)"`
Expected: only the module-path line remains (or empty)

- [ ] **Step 3: Verify Go tests + config-check still pass**

Run: `go test ./... && go run ./cmd/pi-run config-check`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: de-personalize README (generic paths, pluggable secret manager)"
```

---

### Task 6: Code-comment de-personalization + named node version const

**Files:**
- Modify: `internal/cli/keys.go:13` (comment), `internal/cli/app.go:28,123,164`, `internal/cli/setup.go:13,70`, `internal/cli/doctor.go:29,68,72`, `internal/cli/config_check.go:61`

**Interfaces:**
- Consumes: nothing; standalone cleanup
- Produces: `defaultNodeVersion` const = `"22"` (with the `v` prefix added at use sites to preserve behavior)

- [ ] **Step 1: Add a `defaultNodeVersion` const**

Add near the top of `internal/cli/app.go`:

```go
// defaultNodeVersion is the Node version pi-run resolves when PI_NODE_VERSION
// is unset. It is a documented default, not machine-specific.
const defaultNodeVersion = "22"
```

- [ ] **Step 2: Replace hardcoded `v22.19.0` with the const**

In `app.go:123`, `setup.go:70`, `doctor.go:29` — change:
```go
nodeVersion = "v22.19.0"
```
to:
```go
nodeVersion = "v" + defaultNodeVersion
```
Verify each site uses the `v` prefix correctly (the current string includes it, and `nodeBinDir`/`piPath` expect it).

- [ ] **Step 3: De-personalize code comments**

- `keys.go:13`: `// resolveSecret returns the value of env var name, falling back to the configured secret backend (PI_SECRET_BACKEND; default bitwarden via bw_get). Never logs values.` (already done in Task 2 — verify)
- `app.go:28`: `symlink ~/bin/pi-run` → `symlink pi-run onto your PATH`
- `app.go:164`: `// runInstall builds the binary and symlinks it into ~/bin.` → `// runInstall builds the binary and symlinks it onto the user's PATH.`
- `setup.go:13`: `// path is resolved through symlinks so \`~/bin/pi-run -> <root>/bin/pi-run\`` → `// path is resolved through symlinks so the installed pi-run points at <root>/bin/pi-run`
- `doctor.go:68`: `// Symlink into ~/bin is a personal-machine convention; informational` → `// Symlink onto PATH is an install convention; informational`
- `doctor.go:72`: `check("~/bin/pi-run symlink present", pathExists(link))` → `check("pi-run symlink present", pathExists(link))`
- `doctor.go:74`: `fmt.Println("  [info] ~/bin/pi-run symlink check skipped ...")` → `fmt.Println("  [info] pi-run symlink check skipped ...")`
- `config_check.go:61`: `check("~/bin/pi-run symlinks to <root>/bin/pi-run",` → `check("pi-run symlinks to <root>/bin/pi-run",`

- [ ] **Step 4: Update tests that assert on the node version**

In `internal/cli/pi_test.go`, the test uses `"v22.19.0"` — update to use the const where it tests the default:

```go
nodeDir := filepath.Join(home, ".nvm", "versions", "node", "v"+defaultNodeVersion, "bin")
...
got, err := nodeBinDir(home, "v"+defaultNodeVersion)
```
(Keep the explicit-version test `TestNodeBinDirMissing` with `v99.0.0` as-is.)

- [ ] **Step 5: Run full Go test suite + vet**

Run: `go test ./... && go vet ./...`
Expected: PASS, clean

- [ ] **Step 6: Verify portability scan still passes**

Run: `go test ./internal/cli/ -run TestNoHardcodedUserPaths -v`
Expected: PASS (README de-personalization shouldn't affect it, but confirm)

- [ ] **Step 7: Commit**

```bash
git add internal/cli/ README.md
git commit -m "refactor: de-personalize code comments; add defaultNodeVersion const"
```

---

### Task 7: Final verification

**Files:**
- None (verification only)

**Interfaces:**
- Consumes: all prior tasks

- [ ] **Step 1: Full Go build + test + vet**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: all pass, vet clean

- [ ] **Step 2: Full eval quick suite (no key)**

Run: `cd eval && .venv/bin/python -m pytest tests/ -q`
Expected: deterministic subset passes; live tests skip gracefully (no key)

- [ ] **Step 3: `config-check` from built binary**

Run: `go build -o bin/pi-run ./cmd/pi-run && ./bin/pi-run config-check`
Expected: all checks pass

- [ ] **Step 4: Grep for residual personal paths**

Run: `grep -rn "~/Projects\|Dev API Keys\|forrestthomas@gmail\|v22.19.0" --include="*.go" --include="*.md" --include="*.py" --include="*.sh" --include="*.json" . 2>/dev/null | grep -v ".venv\|node_modules\|.git/\|.pi-subagents\|.lore.md\|docs/superpowers\|.superpowers\|launch-announcement\|github.com/forrestthomas1"`
Expected: no matches (or only the module path)

- [ ] **Step 5: Final commit (if any lint/residual fixes)**

Run: `git status --short` — if clean, done. Otherwise commit residual fixes.

---

## Self-Review Notes

- **Spec coverage:** Part 1 (SecretBackend, selection, integration, 1Password naming, backward compat, tests) → Tasks 1-4. Part 2 (README, code comments, not-changing items, tests) → Tasks 5-6. Final verification → Task 7.
- **Placeholder scan:** All code blocks are complete; no TBD/TODO.
- **Type consistency:** `newSecretBackend() (SecretBackend, error)` used consistently; `SecretBackend` interface has `Name()/Resolve()/Status()` everywhere; `defaultNodeVersion = "22"` used with `"v"+` prefix consistently.
