# Fix: Lore `.lore.md` Rewrites + Model-Catalog Refresh Timeout

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** (A) Stop the `@loreai/pi` extension from rewriting `.lore.md` (it treats each git worktree as a new empty project and mirrors its store into the file, deleting everything else; it also strips backticked text on import). (B) Stop `pi update --models` and pi startup from hanging on the machine's dead IPv6 route to `pi.dev` (pi's 15s `AbortController` then aborts with "Model catalog refresh timed out").

**Architecture:** (A) Commit `.lore.json` with `loreFile.enabled = false` — lore's documented off-switch for `.lore.md` export/import and its file watcher; knowledge stays in the DB and is still injected via LTM, and the `agentsFile` (AGENTS.md) export stays enabled. `.lore.md` becomes a static, hand-maintained doc. (B) In the `pi-run` Go CLI, every spawned `pi` process gets `NODE_OPTIONS=--dns-result-order=ipv4first` in its environment, forcing IPv4-first DNS so Node's fetches never land on the broken IPv6 path.

**Tech Stack:** Go 1.21 (stdlib only), existing `pi-run` CLI (`internal/cli/`), `@loreai/pi` extension config, shell + live-API verification.

## Global Constraints

- Go 1.21; **stdlib only** — no external Go dependencies.
- `NODE_OPTIONS=--dns-result-order=ipv4first` must be applied to **every** `pi` spawn: `execPi` (chat/print/setup) and `doctor.go`'s `piListModels`. Pre-existing `NODE_OPTIONS` in the environment must be **overridden**, not appended after (duplicate env keys are undefined; last-wins is not guaranteed).
- `.lore.json` config: `{ "loreFile": { "enabled": false } }` at the repo root, committed. Partial config — other keys untouched.
- `.lore.md` decision entry must be **plain text, no backticks** (import may be re-enabled later; backticks are stripped on import).
- Do not modify `AGENTS.md` in this effort; its lore-managed section is controlled by `agentsFile` (stays enabled) and must remain byte-identical after verification runs.
- Verification requires a live DeepSeek call (env `DEEPSEEK_API_KEY` is present) — same as the Task 11 smoke test.
- Worktree: execute on a branch in an isolated worktree (`.worktrees/lore-ipv6-fix`, gitignored) — create via the using-git-worktrees skill before Task 1. `$HARNESS` below = the worktree path. Note: `.lore.json` is committed, so the worktree itself will contain it — Task 1's worktree verification is the real test.

---

### Task 1: Make `.lore.md` static (disable lore file management)

**Files:**
- Create: `.lore.json`
- Modify: `.lore.md` (header comment + re-add the `pi-run` decision entry)

**Interfaces:**
- Produces: committed `.lore.json`; `.lore.md` with an updated header and a plain-text `pi-run` decision entry; verification that a `pi` run no longer rewrites `.lore.md` or `AGENTS.md`.

- [ ] **Step 1: Write `.lore.json`**

Create `.lore.json` at the repo root with exactly:
```json
{
  "loreFile": {
    "enabled": false
  }
}
```

- [ ] **Step 2: Verify it is valid JSON**

Run: `cd "$HARNESS" && node -e 'JSON.parse(require("fs").readFileSync(".lore.json","utf8")); console.log("valid")'`
Expected: `valid`.

- [ ] **Step 3: Update the `.lore.md` header comment**

In `.lore.md`, replace line 1:
```markdown
<!-- Managed by lore (https://github.com/BYK/loreai) — manual edits are imported on next session. -->
```
with:
```markdown
<!-- Static knowledge doc. lore file export/import is disabled via .lore.json (loreFile.enabled=false); lore memory lives in the DB and is injected via LTM. -->
```

- [ ] **Step 4: Re-add the `pi-run` decision entry**

In `.lore.md`, under the `### Decision` section, insert after the second existing decision bullet (after the "Decision to focus on live technical section" bullet):
```markdown
* **Decision to make pi-run the single harness entry point**: The Pi harness runtime is a Go CLI (pi-run, module github.com/forrestthomas/harness) - the single source of truth for provider routing (OpenAI default; OpenRouter and DeepSeek direct via --provider / PI_PROVIDER env), Bitwarden key resolution (env var first, then bw_get; never printed), Pi launching (nvm-aware), eval, setup, doctor. Replaced all shell functions (pi-harness / pi-harness-print in ~/.zshrc and ~/.bashrc) and Makefile targets; no cross-provider auto-fallback (explicit routing only). github-repo-controller and chatbot projects moved to standalone repos (~/Projects).
```
(Plain text, no backticks.)

- [ ] **Step 5: Commit**

```bash
cd "$HARNESS"
git add .lore.json .lore.md
git commit -m "fix(lore): disable .lore.md export/import; make .lore.md a static doc"
```

- [ ] **Step 6: Verify a pi run no longer rewrites the files**

Run pi in the worktree (the worktree has the committed `.lore.json`):
```bash
cd "$HARNESS"
cp .lore.md /tmp/lore-before.md
PI_OFFLINE=1 pi-run print --provider deepseek "Reply with exactly one word: ok" >/dev/null 2>&1
git status --short
diff .lore.md /tmp/lore-before.md && echo "LORE.md UNCHANGED"
```
Expected: `git status --short` shows **nothing** (no `.lore.md` or `AGENTS.md` modifications); `diff` prints nothing and echoes `LORE.md UNCHANGED`.

---

### Task 2: Force IPv4-first DNS for every `pi` spawn

**Files:**
- Modify: `internal/cli/pi.go` (add `childEnv`, use it in `execPi`)
- Modify: `internal/cli/doctor.go` (`piListModels` uses `childEnv`)
- Test: `internal/cli/pi_test.go`

**Interfaces:**
- Produces: `func childEnv(binDir string, extraEnv []string) []string` — returns `os.Environ()` with (1) any existing `NODE_OPTIONS` entry removed, (2) `PATH` prepended with `binDir`, (3) `NODE_OPTIONS=--dns-result-order=ipv4first` appended, (4) `extraEnv` appended. `execPi` and `piListModels` both use it.

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/pi_test.go`:
```go
func TestChildEnvForcesIPv4FirstDNS(t *testing.T) {
	t.Setenv("NODE_OPTIONS", "--max-old-space-size=4096")
	env := childEnv("/fake/node/bin", []string{"DEEPSEEK_API_KEY=sk-x"})
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "NODE_OPTIONS=--dns-result-order=ipv4first") {
		t.Fatal("child env must force IPv4-first DNS")
	}
	if strings.Contains(joined, "--max-old-space-size") {
		t.Fatal("pre-existing NODE_OPTIONS must be overridden, not appended")
	}
	if !strings.Contains(joined, "PATH=/fake/node/bin"+string(os.PathListSeparator)) {
		t.Fatal("PATH must have the node bin dir prepended")
	}
	if !strings.Contains(joined, "DEEPSEEK_API_KEY=sk-x") {
		t.Fatal("extra env must be present")
	}
}
```
Add `"strings"` to the imports in `pi_test.go`.

- [ ] **Step 2: Run tests — verify they fail**

Run: `cd "$HARNESS" && go test ./internal/cli/ -run ChildEnv -v`
Expected: FAIL — `undefined: childEnv`.

- [ ] **Step 3: Implement `childEnv` and wire it in**

In `internal/cli/pi.go`, add the helper and switch `execPi` to it:
```go
// childEnv builds the environment for spawned pi processes: the nvm node bin
// dir is prepended to PATH, NODE_OPTIONS forces IPv4-first DNS (the IPv6 route
// to pi.dev is broken on some networks and stalls fetch until timeouts), and
// extraEnv KEY_ENV=value pairs are appended. Any pre-existing NODE_OPTIONS is
// overridden, not duplicated.
func childEnv(binDir string, extraEnv []string) []string {
	env := make([]string, 0, len(os.Environ())+len(extraEnv)+2)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "NODE_OPTIONS=") {
			continue // replaced below
		}
		env = append(env, kv)
	}
	env = append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = append(env, "NODE_OPTIONS=--dns-result-order=ipv4first")
	return append(env, extraEnv...)
}
```
Add `"strings"` to pi.go's imports. Replace the env assembly in `execPi`:
```go
	path := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	// Use the absolute pi path: exec.Command resolves the binary against the
	// parent's PATH, not cmd.Env, so PATH alone is not enough.
	cmd := exec.Command(filepath.Join(binDir, "pi"), args...)
	cmd.Env = childEnv(binDir, extraEnv)
```
(Delete the now-unused `path` local.)

In `internal/cli/doctor.go`, `piListModels`:
```go
	cmd := exec.Command(filepath.Join(binDir, "pi"), "--offline", "--list-models")
	cmd.Env = childEnv(binDir, nil)
	return cmd.Output()
```

- [ ] **Step 4: Run tests, build, vet**

Run:
```bash
cd "$HARNESS" && go test ./internal/cli/ -v && go build ./... && go vet ./...
```
Expected: all tests pass (keys 3, providers 5, pi 6, repoRoot 1, app 8 = 23); build and vet exit 0.

- [ ] **Step 5: Rebuild and reinstall the binary**

```bash
cd "$HARNESS"
go build -o bin/pi-run ./cmd/pi-run
bin/pi-run install
```
Expected: `installed ~/bin/pi-run -> <worktree>/bin/pi-run`.

- [ ] **Step 6: Verify `pi update --models` completes (no 15s abort)**

```bash
cd "$HARNESS" && pi-run setup
```
Expected: venv check passes, pip install is idempotent, and **"refreshing model catalogs" completes** (no `Error: Model catalog refresh timed out.`); exit 0. (If pip or the refresh still stalls, run the refresh once directly to capture the error: `NODE_OPTIONS=--dns-result-order=ipv4first pi update --models`.)

- [ ] **Step 7: Verify pi startup no longer hangs (no PI_OFFLINE)**

```bash
cd "$HARNESS" && pi-run print --provider deepseek "Reply with exactly one word: ok" 2>&1 | tail -2; echo "exit=${PIPESTATUS[0]}"
```
Expected: completes in well under the previous 600s hang, prints `ok`, exit 0. (This makes a live API call.)

- [ ] **Step 8: Commit**

```bash
cd "$HARNESS"
git add internal/cli/pi.go internal/cli/pi_test.go internal/cli/doctor.go
git commit -m "fix(cli): force IPv4-first DNS for pi child processes (broken IPv6 route stalls fetch)"
```

---

### Task 3: Documentation and final verification

**Files:**
- Modify: `README.md`

**Interfaces:**
- Produces: README that reflects the two fixes (no stale "times out" troubleshooting bullet; `.lore.json` noted).

- [ ] **Step 1: Update the README troubleshooting section**

In `README.md`, replace the bullet:
```markdown
- **`pi update --models` times out?** Network to the model registry is
  unavailable; the stored catalog still resolves the default models
  (`pi-run doctor` reports this as informational).
```
with:
```markdown
- **Model catalog refresh / startup hangs?** The machine's IPv6 route to pi.dev
  is broken and Node's fetch can land on it. `pi-run` forces IPv4-first DNS
  (`NODE_OPTIONS=--dns-result-order=ipv4first`) for every pi process it spawns,
  so `pi-run setup` and pi startup use the stable IPv4 path.
```

- [ ] **Step 2: Note `.lore.json` in the README layout**

In the `## Project Layout` code block, add after the `.gitignore` line:
```
├── .lore.json                 # lore config: .lore.md export/import disabled (static doc)
```

- [ ] **Step 3: Final verification**

```bash
cd "$HARNESS"
go test ./...
go vet ./...
pi-run eval --quick
pi-run config-check
pi-run doctor
```
Expected: Go tests pass; eval quick 3/3; config-check all `[ok]`; doctor all hard checks pass.

- [ ] **Step 4: Commit**

```bash
cd "$HARNESS"
git add README.md
git commit -m "docs: document IPv4-first DNS fix and static .lore.md"
```

---

## Self-Review Checklist

- **Spec coverage:** Fix A → Task 1 (`.lore.json`, header, entry, pi-run rewrite verification); Fix B → Task 2 (TDD `childEnv`, execPi + piListModels, live refresh + startup verification); docs → Task 3. ✓
- **Placeholder scan:** no TBD/TODO; complete code in every code step. ✓
- **Type consistency:** `childEnv(binDir string, extraEnv []string) []string` — same signature in pi.go, pi_test.go, doctor.go. `execPi`/`piListModels` unchanged otherwise. ✓
- **Constraint check:** `NODE_OPTIONS` override (not duplicate) is unit-tested; `.lore.md` entry is backtick-free; `AGENTS.md` untouched by the plan; exit-code semantics unchanged. ✓
