# Scope Contract
**Task:** OSS-1 — Canonical module path + identity alignment + CI install check | **Plan:** BACKLOG OSS-1 (EPIC-6, RICE ~18, 0.25 pw) | **Date:** 2026-08-14 | **Status:** ACTIVE

## In Scope
- **Ground truth (verified):** canonical identity is **`forrestbthomas/pi-harness`** (gh API `nameWithOwner`, git remote, README clone URL, CONTRIBUTING, update-homebrew-formula.sh all agree). The defect: go.mod + Go imports + ldflags + docs say `forrestthomas1` (extra `1`, missing `b`), and CODEOWNERS says `@forrestthomas` (missing `b`). `go install github.com/forrestthomas/pi-harness@latest` fails on module-path mismatch.
- **Files:**
  - `go.mod` — `module github.com/forrestthomas1/pi-harness` → `github.com/forrestbthomas/pi-harness`
  - Go imports: `cmd/pi-run/main.go`, `internal/cli/app.go`, `internal/cli/app_test.go`
  - ldflags: `scripts/bootstrap.sh`, `scripts/build-release.sh`
  - Docs: `README.md` (go.mod comment + any module-path mentions), `AGENTS.md`, `.pi/SYSTEM.md`, `docs/understand/structure.md`
  - `.github/CODEOWNERS` — `@forrestthomas` → `@forrestbthomas` (canonical handle)
  - `eval/tests/test_harness_config.py` — `test_go_module_path` asserts canonical path; **add identity/install contract check** (module path matches repo URL; CODEOWNERS handle matches canonical user)
- **Features:**
  1. `go.mod` module path matches the canonical repo URL (`github.com/forrestbthomas/pi-harness`).
  2. All imports/ldflags/docs reference the canonical path; CODEOWNERS uses the canonical handle.
  3. Hermetic CI check: `go build` works with the new module path; a contract test pins module-path == repo identity (so it can't drift back).
- **Boundaries:**
  - Historical dated docs (`.lore.md` frozen record, `docs/superpowers/plans/`, `docs/superpowers/specs/`) keep their old module-path mentions — they are history, not current-state (same precedent as the cut-list PR).
  - `.pi-subagents/artifacts/`, `.pi/debate/` are gitignored runtime/process artifacts — untouched.
  - No behavior change; no release/tag change (module path is a pre-1.x... actually module path change is a build-time concern; verified by `go build`/`go test`).

## Out of Scope
- GOV-2 (PM doc relocation + spec archive) — separate item, ships later.
- GOV-1 (drift guard) — OSS-1's contract test is a *seed* of it, not the full guard.
- OSS-2 (on-ramp v2: first-issue path, SLA, MIT-in/out, PR carve-out) — separate item.
- Renaming the GitHub repo itself (the repo is already `forrestbthomas/pi-harness`; no rename needed).
- Homebrew tap repo name (already canonical `forrestbthomas/homebrew-tap`).

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
|---|----------|------|-----|----------|---------|

# Follow-up Tasks
- [ ] After merge: verify `go install github.com/forrestbthomas/pi-harness@<tag>` works on a clean checkout (needs a release tag; manual/CI).
