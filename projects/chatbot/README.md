# Terminal Chatbot Practice

A small TypeScript chatbot and Python evaluation harness for practicing the live technical format described in the interview guide: inspect a local chatbot, run evaluations, make one targeted improvement, rerun, and explain the trade-offs.

This is practice material. It does not reproduce or claim access to any internal interview repository.

## Prerequisites

- Node.js 20 or newer
- npm
- Python 3.11 or newer
- [uv](https://docs.astral.sh/uv/)
- Bitwarden CLI (`bw`) for online chat or evaluation

Check the local tooling:

```bash
node --version
npm --version
python3 --version
uv --version
bw --version
```

## Setup and offline verification

Install the Node dependencies, then run the deterministic checks:

```bash
npm install
npm run typecheck
npm test
npm run eval -- --offline
```

The final command intentionally reports **1/2 passed** and exits with status `1`. That is the disclosed baseline behavior you will investigate during practice; it is not a setup failure. It neither accesses Bitwarden nor calls a model.

Each evaluation run writes a local JSON report to `eval/results/`. Reports are gitignored.

## Online chat and evaluation

The default model identifier is `gpt-5.6-terra`. To choose another available model, set the non-secret `OPENAI_MODEL` environment variable for the command you run:

```bash
OPENAI_MODEL=your-model-id npm run chat
```

For online commands, unlock your Bitwarden vault yourself before starting. The project does not log in to or unlock Bitwarden. It reads the **password field** of the Bitwarden item named `OPENAI_API_KEY` with:

```bash
bw get password OPENAI_API_KEY
```

The returned value is used only in memory by the OpenAI client. Do **not** paste an API key into `.env`, source code, a terminal command, email, or chat.

Start an interactive multi-turn terminal chat:

```bash
npm run chat
```

Type `/exit` or send EOF to stop the chat.

Run the live-model evaluation suite:

```bash
npm run eval
```

Online evaluation invokes the local JSON adapter once per case. It requires an unlocked Bitwarden vault and access to the configured model. If either is unavailable, the command reports a safe error without printing credential output.

## Project map

- `src/chat.ts` — prompt, model adapter, and chat service
- `src/credentials.ts` — Bitwarden-only in-memory credential resolver
- `src/cli.ts` — terminal chat loop
- `eval/cases.json` — versioned inputs, requirements, and offline fixtures
- `eval/graders.py` — deterministic signal grader
- `INTERVIEW_EXERCISES.md` — suggested improvement loops and trade-offs

## Security boundaries

- `.env` is ignored and contains no API-key configuration.
- The code never runs `bw login` or `bw unlock`.
- Credential subprocess output is never logged or copied to evaluation reports.
- Unit tests use fake clients and injected command runners; they do not invoke Bitwarden or OpenAI.
