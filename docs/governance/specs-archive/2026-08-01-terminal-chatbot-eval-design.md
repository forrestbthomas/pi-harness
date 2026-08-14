# Terminal Chatbot Practice Repository — Design

**Status:** Approved for specification review  
**Purpose:** A compact, local practice repository that mirrors the live technical format described in `Interviews/OpenAI/guide.txt`: inspect a chatbot and evaluator, run baseline evaluations, make narrow improvements, rerun, and explain trade-offs.

## Goals

- Provide a real terminal-based TypeScript chatbot backed by the OpenAI Responses API.
- Use the requested model identifier, `gpt-5.6-terra`, verbatim as the default.
- Include a Python 3.11+ evaluation harness runnable through `uv`.
- Keep the repository small enough to understand in a 30-minute setup period.
- Make baseline behavior intentionally imperfect but functional, with visible and documented improvement exercises.
- Keep credentials out of source, environment files, output, and logs.

## Non-goals

- A browser UI, persistence layer, user accounts, deployment infrastructure, or production traffic handling.
- Reproducing or claiming access to any internal interview repository.
- Automatically logging in to or unlocking Bitwarden.
- Calling a model as part of unit tests or offline smoke tests.

## Repository layout

```text
projects/chatbot/
├── src/
│   ├── chat.ts              # Prompt construction and OpenAI model adapter
│   ├── cli.ts               # Interactive, multi-turn terminal interface
│   ├── credentials.ts       # Secret-safe Bitwarden credential resolution
│   └── types.ts             # Narrow model-client contracts
├── tests/
│   ├── chat.test.ts         # Unit tests with injected fake client
│   └── credentials.test.ts  # Credential resolver behavior with mocked process calls
├── eval/
│   ├── cases.json           # Versioned prompts and deterministic expectations
│   ├── graders.py           # Required/forbidden phrase checks and scoring
│   ├── run_evals.py         # Online and offline evaluation runner
│   └── pyproject.toml       # uv project metadata and dependencies
├── .env.example             # Non-secret defaults only
├── .gitignore
├── package.json
├── tsconfig.json
├── README.md
└── INTERVIEW_EXERCISES.md   # Explicit practice scenarios and discussion prompts
```

## Architecture and data flow

### Chat path

1. `npm run chat` starts `src/cli.ts`.
2. The CLI maintains conversation history for its process lifetime and passes each user turn to `chat.ts`.
3. `chat.ts` obtains a model client and calls the OpenAI Responses API using `gpt-5.6-terra`, unless the non-secret `OPENAI_MODEL` environment variable overrides it.
4. The terminal renders only the assistant response and safe operational errors.

A narrow `ModelClient` interface separates chat logic from the API transport. Unit tests use a fake implementation, so tests are deterministic and do not need a key.

### Credential path

The credential resolver invokes:

```sh
bw get password OPENAI_API_KEY
```

It trims the result, rejects empty output, and passes the key only in memory to the model client. It does not log the command output, export the key to a file, place it in result reports, or print it in error messages.

The application does **not** run `bw login` or `bw unlock`; those are interactive authentication operations the user must perform beforehand. Failures identify the safe corrective action (install `bw`, log in/unlock the vault, or verify the non-secret item label) without revealing sensitive data.

### Evaluation path

1. `npm run eval` launches the Python runner through `uv`.
2. In `--offline` mode, the runner sends fixture responses through deterministic graders. It never invokes Bitwarden or OpenAI.
3. In online mode, the runner invokes the chatbot’s evaluation entry point, which resolves the key in memory and gets a live response.
4. `graders.py` scores expected and forbidden signals, then produces concise console output and a timestamped JSON report in an ignored local results directory.

## Baseline exercise design

The first runnable version will be deliberately limited, not broken. Its basic prompt will reliably answer general questions but will underperform on a documented support-assistant expectation such as acknowledging uncertainty before providing a next step. The evaluator will make that gap visible.

`INTERVIEW_EXERCISES.md` will disclose this pedagogical intent and contain suggested improvement paths, including:

- Improve uncertainty handling without making claims about unavailable policy or account data.
- Add a narrow escalation rule for sensitive or account-specific requests.
- Preserve useful answers while avoiding fabricated facts.
- Compare prompt-only adjustments against validation or tool-backed approaches.

The exercises are disclosed rather than hidden so this is ethical practice material, not a deceptive challenge.

## Configuration

`.env.example` contains only:

```dotenv
OPENAI_MODEL=gpt-5.6-terra
```

No API key is stored in `.env`. `OPENAI_MODEL` can be overridden for a local environment, while `gpt-5.6-terra` remains the default. The project will use the supplied model identifier verbatim and does not claim that its availability has been independently verified.

## Error handling and safety

- Input EOF exits the CLI cleanly.
- Empty user messages are rejected locally.
- API/Bitwarden subprocess errors are wrapped in action-oriented, secret-safe messages.
- Responses, eval inputs, and reports are treated as potentially sensitive; secret values are never included.
- Eval reports and all local environment files are gitignored.

## Verification and acceptance criteria

The completed project must support:

```sh
npm install
npm run typecheck
npm test
npm run eval -- --offline
npm run chat              # Requires an unlocked Bitwarden vault and model access
npm run eval              # Requires an unlocked Bitwarden vault and model access
```

Acceptance evidence:

- TypeScript compilation and unit tests pass without network access or Bitwarden.
- Offline evaluator produces a deterministic score and report.
- Credential resolver tests prove it never prints a returned password and produces safe errors.
- Documentation clearly separates offline setup from online requirements.
- Repository scans and staged diffs contain no API key or credential value.

## Trade-offs

| Decision | Benefit | Cost |
| --- | --- | --- |
| Terminal rather than web UI | Tight setup loop and small inspection surface | No browser/product UI practice |
| Deterministic evaluation signals | Reproducible and explainable results | Limited semantic nuance compared with model-graded evals |
| Bitwarden CLI lookup | No plaintext API key in the repository or `.env` | Vault must be unlocked for online operations |
| Fake client for unit tests | Fast, free, deterministic tests | Does not validate a provider call; online eval covers integration |

## Scope boundary

Only `projects/chatbot/` will be replaced during implementation. The existing unrelated `chatbot-project/` directory and root harness files will remain untouched.
