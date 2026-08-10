# Design: Pluggable Secret Manager + De-personalize Public Docs

**Date:** 2026-08-09
**Status:** Approved (both parts)

## Goal

1. Make the harness's secret resolution pluggable — support Bitwarden (current),
   1Password (common in enterprise), and env-only, via a backend interface.
2. Remove personal/system-specific content from public docs and code comments
   so the repo is useful to others (not just the author's machine).

## Part 1 — Pluggable Secret Manager

### Architecture

New file `internal/cli/secret.go` defines a `SecretBackend` interface and
built-in adapters, selected by the `PI_SECRET_BACKEND` env var (default
`bitwarden`, preserving current behavior).

```go
type SecretBackend interface {
    Name() string
    Resolve(name string) (string, error)
    Status() (string, error) // human-readable, for `pi-run doctor`
}

func newSecretBackend() (SecretBackend, error)
```

### Built-in backends

| Backend | Env value | Command | Notes |
|---|---|---|---|
| Bitwarden | `bitwarden` (default) | `bw_get <name>` via `BW_GET` override; default `~/bin/bw_get` | Existing behavior unchanged |
| 1Password | `1password` / `op` | `op read "op://<vault>/<name>/credential"` | Requires `op` CLI signed in; vault/item naming documented |
| Env-only | `env-only` / `env` | none | No fallback; env var only |

### Selection logic

```go
backend := os.Getenv("PI_SECRET_BACKEND") // "" -> "bitwarden"
switch backend {
case "bitwarden", "":
    return &bitwardenBackend{...}
case "1password", "op":
    return &onePasswordBackend{...}
case "env-only", "env":
    return &envOnlyBackend{}
default:
    return nil, fmt.Errorf("unknown secret backend %q", backend)
}
```

### Integration points

- `internal/cli/keys.go:resolveSecret` — replace hardcoded `bw_get` call with
  `newSecretBackend().Resolve(name)` (env-first stays in `resolveSecret`).
- `internal/cli/doctor.go:38` — replace `bw_get --status` with
  `backend.Status()`.
- `eval/conftest.py:get_secret` — mirror Go logic: env first, then backend
  (respect `PI_SECRET_BACKEND`; default `bitwarden` via `BW_GET`).

### 1Password item naming convention

`op read "op://<Vault>/<ITEM_NAME>/credential"`. Vault defaults to the user's
`Personal` vault; documented in README. The `op` CLI must be installed and
signed in; `pi-run doctor` reports its status.

### Backward compatibility

`PI_SECRET_BACKEND` unset → `bitwarden` → byte-identical behavior to today.
All existing tests (which use fake `BW_GET` scripts) keep passing.

### Testing

- Hermetic fake scripts for `bw_get` and `op` (pattern from `keys_test.go`).
- `TestSecretBackendSelection` — each `PI_SECRET_BACKEND` value maps to the
  correct adapter.
- `TestResolveSecretFallsBackToOnePassword` — fake `op` script returns a value.
- `TestUnknownSecretBackend` — error message for unknown backend.

## Part 2 — De-personalize Docs & Code

### Principle

Keep functional behavior; remove machine-specific detail that only makes sense
on the author's machine. The portability test (`TestNoHardcodedUserPaths`)
already enforces no hardcoded user paths in code — extend the same discipline
to public docs.

### README changes (personal paths → generic)

| Current | Change to |
|---|---|
| `~/bin/pi-run` (lines 13, 94, 210) | "a `pi-run` binary on your PATH" |
| `v22.19.0` default (line 30) | "a Node 22+ install (default `22.x`, override `PI_NODE_VERSION`)" |
| "Dev API Keys" Bitwarden folder, `~/bin/bw_get`, `~/.bashrc` (131–138) | Generic: "API keys resolve env-first, then from a secret manager (Bitwarden via `bw_get`, 1Password via `op`)" |
| `~/Projects/tmp/agent-skills/`, `git -C ~/Projects/tmp/...` (226–232) | "a skills directory" — generic |

### Code comment changes

- `keys.go:13` — comment `default ~/bin/bw_get` → "default resolves from
  `BW_GET` or the user's home `bin/`".
- `app.go:28` — usage text `symlink ~/bin/pi-run` → "symlink pi-run onto your
  PATH".
- `app.go:123`, `setup.go:70`, `doctor.go:29` — `nodeVersion = "v22.19.0"` →
  named const `defaultNodeVersion = "22"` (a documented default, not a personal
  version).
- `config_check.go:61`, `doctor.go:72` — `~/bin/pi-run` → "`pi-run` on PATH".

### Not changing (intentional, not personal)

- `github.com/forrestthomas1/pi-harness` — the Go module path (required by
  go.mod and tests).
- `~/bin/pi-run` symlink as a *feature* (`pi-run install` intentionally
  symlinks) — we de-personalize the *docs*, not the feature.
- `~/.nvm` node-path resolution — real behavior; we de-personalize the *docs*
  examples.
- `~/bin/bw_get` helper (not tracked in this repo) — the author's local
  helper; docs just shouldn't hardcode it as the only option.

### Testing

- Update `paths_test.go` if needed; verify the README scrub keeps the
  lore-exemption working.
- `go test ./...` + `config-check` stay green.
- Grep check: no `~/Projects`, `Dev API Keys`, or personal machine paths in
  public docs.

## Out of scope

- Adding other secret managers beyond Bitwarden/1Password/env-only (the
  interface makes adding more trivial later).
- Migrating the author's local `~/bin/bw_get` helper.
- Renaming the Go module path.
