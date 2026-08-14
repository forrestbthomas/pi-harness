# Contributing to pi-harness

Thanks for helping make pi-harness better!

## Development Setup

```bash
git clone https://github.com/forrestbthomas/pi-harness.git
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
- Prefix: `feat(<workstream-id>):`, `fix(<area>):`, `docs(<area>):`,
  `chore(<area>):`, `test:`, `ci:`, `release(vX.Y.Z):`. The scope is usually
  the backlog/workstream id (`feat(eval-6):`, `docs(tax-2):`, `docs(pm):`,
  `docs(audit):`) — see `git log --format=%s` for the current practice.
- Reference issues/PRs where relevant.

## Versioning policy

- **0.x minor** — any user-visible change (feature, removal, behavior change,
  install path, gate semantics); BREAKING changes are flagged in CHANGELOG.
- **0.x patch** — fixes to shipped behavior only, never features.
- **Data releases** (dataset re-baselines, `datasetVersion` bumps) ride the
  eval lane and never spawn a CLI tag.
- **Post-1.0** — strict semver: breaking = major, additive = minor, fix =
  patch. SECURITY's supported-versions table is bumped in the same PR as the
  CHANGELOG entry.

## Security

See `SECURITY.md` for reporting vulnerabilities. Never commit API keys.

## Issue and PR conventions

- Use the issue templates (`.github/ISSUE_TEMPLATE/`) for bug reports and
  feature requests.
- Use the pull request template (`.github/PULL_REQUEST_TEMPLATE.md`) — fill in
  the summary, test plan, and checklist.
- Add a dated `## [x.y.z]` entry at the top of `CHANGELOG.md` for any
  user-visible change.

## Releases

Main is branch-protected and uses squash merges, which rewrite commit hashes.
**The release tag must be created from the merged main tip, never from a local
commit that has not yet landed** — otherwise the tag is not an ancestor of
`main` and `git describe`/ancestry walks break (this happened on v0.9.1 and
v0.9.2). Correct order:

1. Merge all release commits (including the CHANGELOG entry) via PR.
2. Run the hardened script — it tags the fetched `github/main` tip and refuses
   re-tags:
   ```bash
   bash scripts/tag-release.sh vX.Y.Z github
   ```
   (Manual fallback only if you first ensure local main is current:
   `git fetch github && git pull --ff-only github main && git tag -a vX.Y.Z main && git push github vX.Y.Z`.)
3. The Release workflow builds, publishes, updates the Homebrew tap, and now
   verifies the installed formula reports `pi-run version` == the tag (REL-3);
   its first step verifies the tag is an ancestor of `main` and fails otherwise.

## Syncing after a merge

After a PR is squash-merged, sync local `main` with a **fast-forward only**:
`git fetch github && git checkout main && git pull --ff-only github main`.
Never use `git reset --hard` as a routine sync — it destroys local work if
`main` has diverged. Reset is only justified when upstream rewrote local
history (squash collapse) with no unique local commits, and only with explicit
user confirmation.
