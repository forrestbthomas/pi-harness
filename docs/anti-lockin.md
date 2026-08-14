# Avoiding AI-Provider Vendor Lock-In with pi-harness

> Last verified 2026-08-14 (provider table, env vars, no-fallback invariant).

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
