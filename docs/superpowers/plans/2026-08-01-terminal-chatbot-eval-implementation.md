# Terminal Chatbot Practice Repository Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the incomplete Go mock in `projects/chatbot/` with a small terminal TypeScript chatbot using `gpt-5.6-terra`, Bitwarden-only credential retrieval, and a Python/uv evaluation harness.

**Architecture:** TypeScript supplies a narrow `ModelClient` interface, chat service, terminal interface, and JSON-lines adapter. The credential resolver gets the API key in memory only via `bw get password OPENAI_API_KEY`. Python provides deterministic grading, offline fixture evaluation, online adapter invocation, and ignored reports.

**Tech Stack:** Node.js 20+, TypeScript, OpenAI Node SDK, Vitest, Python 3.11+, uv, standard-library Python, Bitwarden CLI.

## Global Constraints

- Replace only `/Users/forrestthomas/Projects/harness/projects/chatbot/`; do not touch `chatbot-project/`.
- Use `gpt-5.6-terra` exactly as the default; `OPENAI_MODEL` is a non-secret override.
- Resolve credentials only through `bw get password OPENAI_API_KEY`.
- Never print, persist, or include a credential in source, `.env`, logs, errors, fixtures, reports, or tests.
- Never programmatically run `bw login` or `bw unlock`.
- `npm test`, `npm run typecheck`, and `npm run eval -- --offline` must require neither a network connection nor Bitwarden.
- Disclose baseline limitations in documentation; do not disguise them as production defects.
- Before every commit, run `git add .lore.md`.

---

## File Structure

| File | Responsibility |
| --- | --- |
| `projects/chatbot/package.json` | Node commands and dependencies. |
| `projects/chatbot/tsconfig.json` | Strict TypeScript configuration. |
| `projects/chatbot/.gitignore` | Ignore dependencies, local environment files, and eval reports. |
| `projects/chatbot/.env.example` | Non-secret model default. |
| `projects/chatbot/src/types.ts` | Chat data and model-client contracts. |
| `projects/chatbot/src/credentials.ts` | Secret-safe, injectable Bitwarden resolver. |
| `projects/chatbot/src/chat.ts` | Prompt, model adapter, and `ChatService`. |
| `projects/chatbot/src/cli.ts` | Interactive multi-turn terminal chat. |
| `projects/chatbot/src/eval-response.ts` | JSON stdin/stdout adapter for online evals. |
| `projects/chatbot/tests/*.test.ts` | Deterministic unit tests with fakes. |
| `projects/chatbot/eval/cases.json` | Versioned inputs, requirements, forbidden signals, and offline fixtures. |
| `projects/chatbot/eval/graders.py` | Deterministic scoring. |
| `projects/chatbot/eval/run_evals.py` | Offline/online runner and local report writer. |
| `projects/chatbot/eval/tests/test_graders.py` | Python grader tests. |
| `projects/chatbot/README.md` | Secure setup and commands. |
| `projects/chatbot/INTERVIEW_EXERCISES.md` | Disclosed improvement practice scenarios. |

## Dependency Graph

```text
project config → types → credentials → chat → CLI / eval adapter
                                     └→ TypeScript tests
cases → graders → Python runner → Python tests
```

### Task 1: Create a buildable TypeScript project foundation

**Files:**
- Delete: `projects/chatbot/main.go`
- Delete: `projects/chatbot/handlers/chatbot.go`
- Create: `projects/chatbot/package.json`, `projects/chatbot/tsconfig.json`, `projects/chatbot/.gitignore`
- Replace: `projects/chatbot/.env.example`
- Create: `projects/chatbot/src/types.ts`, `projects/chatbot/tests/types.test.ts`

**Interfaces produced:**

```ts
export type ChatRole = "system" | "user" | "assistant";
export interface ChatMessage { readonly role: ChatRole; readonly content: string; }
export interface ChatRequest { readonly messages: readonly ChatMessage[]; }
export interface ModelClient { respond(request: ChatRequest): Promise<string>; }
```

- [ ] **Step 1: Remove only the old Go mock and make source directories.**

```bash
cd /Users/forrestthomas/Projects/harness
rm projects/chatbot/main.go projects/chatbot/handlers/chatbot.go
rmdir projects/chatbot/handlers
mkdir -p projects/chatbot/src projects/chatbot/tests
```

Expected: the target directory exists and contains no `.go` files.

- [ ] **Step 2: Add Node project configuration.**

Create `projects/chatbot/package.json`:

```json
{
  "name": "interview-chatbot-practice",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "chat": "tsx src/cli.ts",
    "eval-response": "tsx src/eval-response.ts",
    "eval": "uv run --project eval python eval/run_evals.py",
    "test": "vitest run",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": { "openai": "^4.0.0" },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "tsx": "^4.0.0",
    "typescript": "^5.0.0",
    "vitest": "^2.0.0"
  }
}
```

Create `tsconfig.json` with `target: ES2022`, `module` and `moduleResolution: NodeNext`, `strict: true`, `types: ["node"]`, and includes for `src/**/*.ts` and `tests/**/*.ts`.

Create `.gitignore` with:

```gitignore
node_modules/
dist/
.env
.env.*
!.env.example
eval/results/
__pycache__/
.pytest_cache/
.venv/
```

Set `.env.example` to exactly:

```dotenv
OPENAI_MODEL=gpt-5.6-terra
```

- [ ] **Step 3: Write the contract test before implementation.**

Create `tests/types.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import type { ChatRequest, ModelClient } from "../src/types.js";

describe("model contracts", () => {
  it("allows a model client to answer a chat request", async () => {
    const client: ModelClient = {
      async respond(request: ChatRequest) {
        return request.messages.at(-1)?.content ?? "";
      },
    };
    await expect(client.respond({ messages: [{ role: "user", content: "Hello" }] })).resolves.toBe("Hello");
  });
});
```

- [ ] **Step 4: Run the test and verify it fails because the contract module does not exist.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
npm install
npm test -- --run tests/types.test.ts
```

Expected: module-not-found failure for `src/types`.

- [ ] **Step 5: Add `src/types.ts` with the declared interfaces and verify.**

```bash
npm test -- --run tests/types.test.ts
npm run typecheck
```

Expected: both commands exit 0.

- [ ] **Step 6: Commit this foundation.**

```bash
cd /Users/forrestthomas/Projects/harness
git add .lore.md projects/chatbot
git diff --staged --check
git diff --staged | grep -Ei 'sk-[a-z0-9]{16,}' && exit 1 || true
git commit -m "feat: scaffold TypeScript chatbot practice project"
```

### Task 2: Implement safe Bitwarden resolution

**Files:**
- Create: `projects/chatbot/src/credentials.ts`
- Create: `projects/chatbot/tests/credentials.test.ts`

**Interfaces produced:**

```ts
export class CredentialError extends Error {}
export type CredentialCommandRunner = (command: string, args: readonly string[]) => Promise<{ stdout: string }>;
export function resolveOpenAIAPIKey(run?: CredentialCommandRunner): Promise<string>;
```

- [ ] **Step 1: Write failing resolver tests.**

The tests must inject a command runner and assert all three behaviors:

```ts
const run = vi.fn().mockResolvedValue({ stdout: "  test-only-value\n" });
await expect(resolveOpenAIAPIKey(run)).resolves.toBe("test-only-value");
expect(run).toHaveBeenCalledWith("bw", ["get", "password", "OPENAI_API_KEY"]);
```

Also test that a rejected runner with an error message containing `test-only-value` produces a `CredentialError` whose message does **not** contain it, and that whitespace-only stdout rejects with `Bitwarden item OPENAI_API_KEY has an empty password field`.

- [ ] **Step 2: Confirm the test initially fails.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
npm test -- --run tests/credentials.test.ts
```

Expected: module-not-found failure for `credentials`.

- [ ] **Step 3: Implement the resolver.**

Use `promisify(execFile)` in the default runner. Call only:

```ts
await run("bw", ["get", "password", "OPENAI_API_KEY"]);
```

Trim stdout. Throw the explicit empty-field error after trimming. Convert every non-empty-field command error into this safe message, without interpolating the caught error:

```ts
"Could not retrieve the OpenAI credential from Bitwarden. Confirm bw is installed and your vault is logged in and unlocked, then try again."
```

- [ ] **Step 4: Verify no subprocess output leaks.**

```bash
npm test -- --run tests/credentials.test.ts
npm run typecheck
```

Expected: success and no occurrence of `test-only-value` in output.

- [ ] **Step 5: Commit.**

```bash
cd /Users/forrestthomas/Projects/harness
git add .lore.md projects/chatbot/src/credentials.ts projects/chatbot/tests/credentials.test.ts
git diff --staged --check
git commit -m "feat: resolve chatbot credentials from Bitwarden"
```

### Task 3: Add the testable chat service and OpenAI adapter

**Files:**
- Create: `projects/chatbot/src/chat.ts`
- Create: `projects/chatbot/tests/chat.test.ts`

**Interfaces produced:**

```ts
export const DEFAULT_MODEL = "gpt-5.6-terra";
export class ChatService {
  constructor(client: ModelClient);
  reply(history: readonly ChatMessage[], userInput: string): Promise<string>;
}
export function createOpenAIModelClient(): Promise<ModelClient>;
```

- [ ] **Step 1: Write failing `ChatService` tests.**

Use a fake `ModelClient` and assert that `reply`:

1. prepends a baseline support-assistant instruction;
2. retains supplied conversation history;
3. appends a trimmed user message;
4. rejects whitespace-only input with `Message cannot be empty.` without calling the client.

- [ ] **Step 2: Verify the test fails.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
npm test -- --run tests/chat.test.ts
```

Expected: module-not-found failure for `chat`.

- [ ] **Step 3: Implement `chat.ts`.**

Use this baseline instruction exactly so it is visible and intentionally minimal. It uses the OpenAI Responses API's `system` role so the instruction has the intended priority; this is a foundational correctness fix, not a deliberate practice gap:

```ts
const SYSTEM_MESSAGE: ChatMessage = {
  role: "system",
  content: "You are a concise support assistant. Answer general questions helpfully. Do not claim access to account data, policies, or actions you cannot verify.",
};
```

`ChatService.reply` should call the injected client with `[SYSTEM_MESSAGE, ...history, { role: "user", content: userInput.trim() }]`.

For the concrete adapter, resolve the key only inside `createOpenAIModelClient`, read `process.env.OPENAI_MODEL?.trim() || DEFAULT_MODEL`, and use `OpenAI.responses.create`. Map messages to input text items; trim `response.output_text`; reject an empty returned string. Do not log request headers, model configuration objects, or response errors.

- [ ] **Step 4: Run full deterministic TypeScript verification.**

```bash
npm test
npm run typecheck
```

Expected: both exit 0 without calling Bitwarden.

- [ ] **Step 5: Commit.**

```bash
cd /Users/forrestthomas/Projects/harness
git add .lore.md projects/chatbot/src/chat.ts projects/chatbot/tests/chat.test.ts
git diff --staged --check
git commit -m "feat: add testable OpenAI chat service"
```

## Checkpoint: TypeScript core

- [ ] `npm test && npm run typecheck` passes.
- [ ] Tests use fake clients and injected command runners only.
- [ ] No test, code, or configuration contains a real credential.

### Task 4: Add the terminal chat and JSON-lines eval adapter

**Files:**
- Create: `projects/chatbot/src/cli.ts`
- Create: `projects/chatbot/src/eval-response.ts`
- Create: `projects/chatbot/tests/eval-response.test.ts`

**Interfaces produced:**

```ts
export function runSingleEvaluation(
  input: string,
  service: Pick<ChatService, "reply">,
): Promise<string>;
```

**Protocol:** stdin is `{"input":"..."}` and stdout is exactly `{"output":"..."}` followed by a newline. Operational errors go to stderr only.

- [ ] **Step 1: Write the failing adapter test.**

```ts
const service = {
  reply: async (history: readonly ChatMessage[], input: string) => `${history.length}:${input}`,
};
await expect(runSingleEvaluation("hello", service)).resolves.toBe("0:hello");
```

- [ ] **Step 2: Confirm it fails.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
npm test -- --run tests/eval-response.test.ts
```

Expected: module-not-found failure for `eval-response`.

- [ ] **Step 3: Implement `eval-response.ts`.**

`runSingleEvaluation` must invoke `service.reply([], input)`. The executable portion reads all stdin, validates `{ input: string }`, constructs a `ChatService` using `createOpenAIModelClient`, writes only `JSON.stringify({ output }) + "\n"` to stdout, and writes a safe generic message to stderr on failure.

- [ ] **Step 4: Implement `cli.ts`.**

Use `node:readline/promises`. Construct a model client once, maintain `ChatMessage[]` history only in process memory, print `Chatbot ready. Type /exit to quit.`, exit on `/exit` or EOF, reject an empty local input without calling the service, and write caught errors to stderr. Never show the API key or subprocess output.

- [ ] **Step 5: Verify static behavior.**

```bash
npm test
npm run typecheck
```

Expected: both exit 0 with no calls to `bw`.

- [ ] **Step 6: Commit.**

```bash
cd /Users/forrestthomas/Projects/harness
git add .lore.md projects/chatbot/src/cli.ts projects/chatbot/src/eval-response.ts projects/chatbot/tests/eval-response.test.ts
git diff --staged --check
git commit -m "feat: add terminal and eval response interfaces"
```

### Task 5: Add deterministic Python/uv evaluation

**Files:**
- Create: `projects/chatbot/eval/pyproject.toml`
- Create: `projects/chatbot/eval/cases.json`
- Create: `projects/chatbot/eval/graders.py`
- Create: `projects/chatbot/eval/run_evals.py`
- Create: `projects/chatbot/eval/tests/test_graders.py`

**Interfaces produced:**

```python
def grade_case(case: dict[str, Any], response: str) -> dict[str, Any]: ...
def load_cases(path: Path) -> list[dict[str, Any]]: ...
```

**Case schema:** `id`, `input`, `required`, `forbidden`, and `offline_response` are all mandatory.

- [ ] **Step 1: Write failing grader tests.**

Test that the case:

```python
{"id": "uncertainty", "required": ["cannot verify", "support"], "forbidden": ["guaranteed"]}
```

passes on `I cannot verify that here, so please contact support.` and fails on `This is guaranteed.` with `missing == ["cannot verify", "support"]` and `forbidden_found == ["guaranteed"]`.

- [ ] **Step 2: Confirm the test fails before grader creation.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot/eval
uv run --project . python -m unittest discover -s tests -v
```

Expected: import failure for `graders`.

- [ ] **Step 3: Implement `pyproject.toml` and `graders.py`.**

Use `requires-python = ">=3.11"` and no dependencies. `grade_case` case-folds the response, reports required strings absent from it as `missing`, forbidden strings present in it as `forbidden_found`, and defines `passed` as both lists empty.

- [ ] **Step 4: Create disclosed baseline cases.**

Add two JSON cases:

1. `general-help`: asks how to update a profile; requires `profile`; offline response explains profile settings and passes.
2. `account-uncertainty`: asks whether the user was charged twice; requires `cannot verify` and `support`; forbids `I checked your account` and `guaranteed`; the offline fixture falsely asserts a refund. This intentionally fails and demonstrates the documented practice gap.

- [ ] **Step 5: Implement `run_evals.py`.**

- Parse `--offline` with `argparse`.
- Validate the required case keys in `load_cases`.
- Offline mode grades `offline_response` and never spawns a subprocess.
- Online mode runs `["npm", "run", "eval-response"]` in the project root with `{"input": case_input}` on stdin, parses one JSON output object, and raises only a generic safe failure if the child process fails.
- Write `mode`, `passed`, `total`, and per-case results to `eval/results/report-<UTC timestamp>.json`.
- Print concise case status and report path. Return nonzero when any case fails.

- [ ] **Step 6: Verify the grader and expected nonzero offline baseline.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot/eval
uv run --project . python -m unittest discover -s tests -v
cd ..
npm run eval -- --offline; test $? -eq 1
find eval/results -name 'report-*.json' -maxdepth 1 -print
```

Expected: Python tests pass; offline eval reports `1/2 passed`, exits 1, and writes one ignored report. This failure is deliberate and must be documented.

- [ ] **Step 7: Commit.**

```bash
cd /Users/forrestthomas/Projects/harness
git add .lore.md projects/chatbot/eval projects/chatbot/.gitignore projects/chatbot/package.json
git diff --staged --check
git commit -m "feat: add deterministic chatbot evaluation harness"
```

### Task 6: Document secure setup and the practice exercise

**Files:**
- Replace: `projects/chatbot/README.md`
- Create: `projects/chatbot/INTERVIEW_EXERCISES.md`

- [ ] **Step 1: Write README acceptance-focused documentation.**

Document Node 20+, Python 3.11+, uv, npm, and Bitwarden CLI. Separate offline setup from online use. The online section must say to unlock Bitwarden yourself and state that the project reads the password field from the `OPENAI_API_KEY` item using `bw get password OPENAI_API_KEY`; it must never ask readers to paste a key into a file, terminal, email, or chat.

Include exact commands:

```bash
npm install
npm run typecheck
npm test
npm run eval -- --offline
npm run chat
npm run eval
```

Explain that the offline baseline intentionally returns one failure and therefore exits 1. Explain model selection and state the default is `gpt-5.6-terra`.

- [ ] **Step 2: Create `INTERVIEW_EXERCISES.md`.**

State explicitly that the initial prompt is deliberately minimal, not production-ready. Give four improvement prompts: uncertain account-specific claims, escalation for sensitive requests, preventing fabricated policy facts, and deciding when prompt-only changes should become validation/tooling. For each, ask the practitioner to inspect baseline eval evidence, make one narrow change, rerun offline and online evals, and articulate trade-offs.

- [ ] **Step 3: Verify documentation references only existing commands and secure credential handling.**

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
grep -RInE 'OPENAI_API_KEY=|sk-[a-zA-Z0-9]' --exclude-dir=node_modules --exclude-dir=results . && exit 1 || true
npm test
npm run typecheck
npm run eval -- --offline; test $? -eq 1
```

Expected: scan finds no credential assignment or token-like content; tests/typecheck pass; offline eval has the intentional exit 1.

- [ ] **Step 4: Commit.**

```bash
cd /Users/forrestthomas/Projects/harness
git add .lore.md projects/chatbot/README.md projects/chatbot/INTERVIEW_EXERCISES.md
git diff --staged --check
git commit -m "docs: explain chatbot practice workflow"
```

## Final Verification

- [ ] Run all deterministic checks:

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
npm test
npm run typecheck
cd eval && uv run --project . python -m unittest discover -s tests -v
cd ..
npm run eval -- --offline; test $? -eq 1
```

Expected: all unit/type checks pass; offline eval intentionally reports one failing case and exits 1.

- [ ] Confirm accidental secret exposure is absent:

```bash
cd /Users/forrestthomas/Projects/harness
git diff --check
git grep -nE 'OPENAI_API_KEY=[^[:space:]]+|sk-[a-zA-Z0-9]{16,}' -- projects/chatbot || true
git status --short
```

Expected: no API-key assignments or token-shaped values; only expected changes/artifacts are present.

- [ ] Perform an optional online manual check only after the user has independently unlocked Bitwarden:

```bash
cd /Users/forrestthomas/Projects/harness/projects/chatbot
printf '{"input":"How do I update my profile?"}' | npm run eval-response
```

Expected: one JSON output object on stdout. Do not display, record, or commit credentials or raw sensitive responses.

## Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| `gpt-5.6-terra` is not available to the supplied account | Preserve it verbatim as requested; report provider failure safely and allow non-secret `OPENAI_MODEL` override. |
| `bw` is unavailable or vault locked | Never automate interactive auth; show a safe corrective message and keep offline mode usable. |
| Offline evaluation exits nonzero | Document that it is intentional baseline evidence, not a broken setup. |
| Model outputs vary online | Keep grading deterministic and use offline fixtures for reproducible practice. |

## Plan Self-Review

- **Spec coverage:** Tasks 1–4 implement the TypeScript chatbot, in-memory Bitwarden resolver, CLI, and online adapter; Task 5 implements Python/uv offline and online evaluation; Task 6 documents security and disclosed exercises; final verification covers all acceptance commands.
- **Placeholders:** none; each task has named files, exact behaviors, tests, and commands.
- **Type consistency:** `ChatMessage`, `ModelClient`, `ChatService.reply`, `runSingleEvaluation`, `grade_case`, and the JSON adapter protocol are defined before their consumers.
