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
