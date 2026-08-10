# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| v0.3.x  | ✅ Active          |
| < 0.3   | ❌ Not supported    |

## Reporting a Vulnerability

**Please do not open a public issue for security vulnerabilities.**

Use GitHub's **private vulnerability reporting** (recommended):

1. Go to the repo: https://github.com/forrestbthomas/pi-harness
2. **Security** tab → **Report a vulnerability** → **Create a report**.

This sends the report privately to the maintainers.

If private reporting is unavailable, email the maintainer privately at the
address shown in the repo (git config user.email / profile), or open a
**draft** security advisory from the Security tab.

### What to include

- Affected version / commit
- Steps to reproduce
- Impact (what an attacker could do)
- Suggested fix (if known)

You should receive a response within 7 days. If not, please follow up.

## Scope

In scope:

- The Go CLI (`pi-run`) and its handling of API keys
- The Python eval suite (does it leak secrets? does it run arbitrary code?)
- Build/CI configuration and release artifacts

Out of scope:

- Vulnerabilities in upstream dependencies (Pi CLI, DeepEval, Homebrew
  formulas) — report those upstream
- The [homebrew-tap](https://github.com/forrestbthomas/homebrew-tap) repo —
  report there separately

## Security posture notes

- API keys are resolved env-first, then from an optional secret store
  (`BW_GET`); values are **never logged** (diagnostics report presence only).
- The eval suite's live-LLM tests skip gracefully when no key is present; the
  deterministic subset runs keyless.
- No hardcoded credentials or user paths are shipped (enforced by tests).
