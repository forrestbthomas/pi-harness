# Scope Contract
**Task:** #157 — Derive the Ollama `/model` picker from the local daemon catalog | **Plan:** Approved in chat (2026-08-15) | **Date:** 2026-08-15 | **Status:** ACTIVE

## In Scope
- **Files:**
  - `.pi/extensions/ollama.ts` (new) — register a Pi `ollama` provider via Pi's documented static import pattern; fetch and normalize the local daemon catalog with a bounded timeout and last-successful cache.
  - `.pi/extensions/lib/ollama-catalog.ts` (new) — pure discovery/normalization helpers with no Pi runtime imports, so Node's hermetic tests can exercise them without a daemon or runtime dependency.
  - `.pi/extensions/__tests__/ollama.test.ts` (new) — hermetic Node tests for discovery parsing, timeout, and cached-catalog fallback. The subdirectory prevents Pi's extension auto-discovery from treating the test as an extension.
  - `.pi/settings.json` — load the project-local Ollama extension.
  - `internal/cli/providers.go`, `internal/cli/pi.go`, `internal/cli/app.go`, `internal/cli/benchmark.go`, `internal/cli/eval.go`, `internal/cli/ollama.go`, `internal/cli/ollama_test.go`, `eval/conftest.py`, `providers.json` — route the harness `ollama` provider to Pi’s `ollama` provider while retaining the loopback OpenAI-compatible endpoint, permitting its local catalog refresh, supporting its keyless local-daemon authentication, and choosing a live installed model as the unflagged launch default.
  - `internal/cli/*_test.go` — hermetic regression tests for routing and catalog normalization/failure handling.
  - `CHANGELOG.md` — user-visible bug-fix entry.
- **Features:**
  - `pi-run chat --provider ollama` launches Pi under provider ID `ollama`.
  - Pi’s `/model` selector receives the exact model tags reported by the configured local Ollama daemon.
  - Discovery has a short timeout, does not need credentials, and retains the last successfully discovered catalog if the daemon is unavailable.
  - Tests use a local fake endpoint; CI never requires a real Ollama daemon or model download.
- **Boundaries:**
  - OpenAI-compatible request transport remains `http://localhost:11434/v1`.
  - The extension is project-local and uses Pi’s supported extension API; no fork or modification of the installed Pi runtime.
  - The issue and PR close #157.

## Out of Scope
- Upstream Pi-native Ollama provider/support (explicitly deferred).
- Dynamic catalogs for any provider other than Ollama.
- Global `~/.pi` configuration changes or user credential changes.
- Altering Pi’s global OpenAI catalog behavior.
- New Go/npm dependencies, model pulls, or network-dependent CI.
- Unrelated provider-catalog cleanup/refactoring.

# Scope Change Log
| # | Category | What | Why | Decision | Outcome |
| 1 | emergent | Add extension discovery tests | Discovery behavior belongs to the TypeScript extension; Go-only tests would not prove the `/model` catalog is correct | Permit (user approved) | Node built-in tests will use a fake loopback fetch; no daemon or new dependency |
| 2 | emergent | Add `internal/cli/pi.go`; omit Pi `--offline` for Ollama only | Pi's supported dynamic-provider cache will not perform fresh discovery while offline | Permit (user approved) | All other providers retain `--offline`; Ollama refreshes only its bounded loopback catalog |
| 3 | implementation safeguard | Place the extension test under `.pi/extensions/__tests__/` | Pi auto-loads every top-level `.ts` file in `.pi/extensions/`; a top-level test would be mistaken for an extension | Permit (covered by approved test addition) | Test remains project-local but is not auto-loaded by Pi |
| 4 | critical | Add explicit keyless-provider handling in launch and benchmark paths | Harness otherwise refuses the credential-free Ollama daemon before Pi starts | Permit (user approved) | A declarative provider field limits bypassing secret lookup to Ollama; all other providers keep existing key requirements |
| 5 | critical | Split pure helpers into `.pi/extensions/lib/ollama-catalog.ts` | Pi aliases static `@earendil-works/pi-ai` imports; dynamic imports can bypass the alias and fail. The lib keeps Node tests runtime-independent | **Auto-approved (user directive)** | Static imports in the extension; pure helpers in lib |
| 7 | emergent | Exclude `OLLAMA_API_KEY` from the eval live-key guard (Go + Python) | A stale placeholder key for the now-keyless provider flipped `anyProviderKeyEnv()` and made `TestRunEvalNoKeySkipsLive` run the live branch in this shell | **Auto-approved (user directive)** | Keyless providers no longer gate live evals; the failing test is now hermetic and green |
| 8 | emergent | Live default model for unflagged Ollama launches (`ollama.go`) | The static `ollama/llama3.1` placeholder is not installed locally, so a bare `pi-run chat --provider ollama` 404'd after the picker fix | **Auto-approved (user directive)** | Bounded loopback lookup picks the first chat-capable installed tag; static default is the fallback when the daemon is down; hermetic httptest coverage |
| 6 | process | Auto-approval directive | User directed autonomous fixes using persona-agent best practices (worker/reviewer/scout + playtest battery discipline) | **Auto-approved (user directive)** | Deviations are logged here as encountered; work proceeds without blocking prompts |

# Follow-up Tasks
- [ ] Propose upstream Pi-native Ollama support after the harness extension is validated.
