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
