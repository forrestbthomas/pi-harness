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
- Prefix: `feat(oss):`, `fix(cli):`, `docs:`, `chore(oss):`, `test:`, `ci:`.
- Reference issues/PRs where relevant.

## Security

See `SECURITY.md` for reporting vulnerabilities. Never commit API keys.

## Issue and PR conventions

- Use the issue templates (`.github/ISSUE_TEMPLATE/`) for bug reports and
  feature requests.
- Use the pull request template (`.github/PULL_REQUEST_TEMPLATE.md`) — fill in
  the summary, test plan, and checklist.
- Add a dated `## [x.y.z]` entry at the top of `CHANGELOG.md` for any
  user-visible change.
