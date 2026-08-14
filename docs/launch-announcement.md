# Launch Announcement Drafts — pi-harness v0.3.0

> **Status: unpublished draft (as of v0.3.0, 2026-08-09) — NOT current.** The
> repo is at v0.10.0 (17 providers, benchmark runner, budget gate, self-heal);
> refresh this copy before any launch. Retained for reuse.

> Post where it fits your audience. Each draft is standalone. The core pitch:
> **one harness, any AI provider — and a built-in way to prove your agent works.**

---

## Hacker News (Show HN)

**Title:** Show HN: Pi-Harness — run the same coding agent on any LLM provider, with an eval suite

**Body:**

I built an open-source harness that treats "the coding agent" as a configurable
thing rather than a vendor lock-in. It wraps the [Pi coding agent](https://pi.dev/)
behind one Go CLI (`pi-run`) and lets you route the same agent config to any
provider — OpenAI, DeepSeek, Anthropic, Gemini, Groq, OpenRouter, or a local
Ollama/vLLM endpoint — by changing one flag:

```bash
pi-run chat                      # OpenAI by default
pi-run chat --provider deepseek # same agent, different model
```

The thing I couldn't find elsewhere: a harness that also **evaluates** the
agent's output. It ships a DeepEval-based pytest suite that runs your agent on
a dataset and scores the responses — so you can actually measure whether a
model swap or prompt change improves things, instead of vibes:

```bash
pi-run eval --quick   # deterministic smoke subset, no API key needed
pi-run eval           # full suite with LLM-as-judge
```

Design choices I care about:
- **Zero Go dependencies** (stdlib only) — `go build` just works
- **Data-driven providers** — add a provider by editing `providers.json`, no recompile
- **Keys via env or Bitwarden** — never committed, never logged
- **CI + releases automated** — cross-compiled binaries for all platforms,
  Homebrew tap for macOS
- **No lock-in**: the same agent config + eval suite routes to any provider

It's MIT, at https://github.com/forrestbthomas/pi-harness. Install:

```bash
brew install forrestbthomas/tap/pi-run   # macOS
# or: git clone https://github.com/forrestbthomas/pi-harness && bash scripts/bootstrap.sh
```

I'd love feedback on: the eval methodology, provider routing, and what a
"good" default eval dataset looks like.

---

## X / Twitter (thread)

**Post 1:**
Coding agents are great — until you're locked into one provider.
I built pi-harness: one harness, any LLM provider, with a built-in eval suite.
Open source, MIT. 🧵

**Post 2:**
The core: `pi-run` wraps the Pi coding agent. Swap providers with one flag:
`pi-run chat --provider deepseek` — same agent, different model. No recompile,
providers are data-driven in `providers.json`.

**Post 3:**
The part that's missing everywhere: evaluation.
It ships a DeepEval pytest suite that scores your agent's output.
`pi-run eval --quick` — deterministic, no API key needed.
`pi-run eval` — full suite with LLM-as-judge.

**Post 4:**
Built to be boring and reliable:
• Zero Go deps (stdlib only)
• Keys via env or Bitwarden, never logged
• CI + cross-platform releases + Homebrew tap
• MIT

**Post 5:**
Try it:
`brew install forrestbthomas/tap/pi-run`
or https://github.com/forrestbthomas/pi-harness
Feedback welcome — especially on eval methodology and default datasets.

---

## LinkedIn

**Title:** I open-sourced a harness for running and evaluating coding agents

**Body:**

Most coding-agent tooling locks you into one AI provider. I built
[pi-harness](https://github.com/forrestbthomas/pi-harness) to solve that: a
provider-agnostic harness (MIT, open source) that runs the same agent
configuration against any model — OpenAI, DeepSeek, Anthropic, Gemini, Groq,
OpenRouter, or a local endpoint — and, crucially, includes an evaluation suite
to measure whether your agent actually works.

Highlights:
- One Go CLI (`pi-run`) wraps the Pi coding agent; providers are data-driven
  (add one in `providers.json`, no recompile).
- DeepEval-based pytest suite scores agent output — deterministic smoke subset
  needs no API key; full suite uses an LLM judge.
- Keys resolve env-first then Bitwarden; never committed or logged.
- Automated CI, cross-platform release binaries, Homebrew tap for macOS.

Install: `brew install forrestbthomas/tap/pi-run`
Repo: https://github.com/forrestbthomas/pi-harness

I'd genuinely appreciate feedback from anyone working on agent evaluation —
it's the hardest open problem in this space, and I want to get the methodology right.

---

## Reddit (r/golang or r/LocalLLaMA)

**Title:** [Project] pi-harness — run the same coding agent on any LLM provider, with an eval suite (MIT, Go)

**Body:**

I've been frustrated that "coding agent" and "AI provider" are usually coupled.
I built an open-source harness (Go CLI, stdlib-only) around the Pi coding agent
that decouples them:

- One agent config routes to any provider: `pi-run chat --provider deepseek`
- Providers are data-driven in `providers.json` — add one without recompiling
- Ships a DeepEval pytest suite so you can measure agent output, not just vibe
- Keys via env or Bitwarden; never committed, never logged
- Automated CI + cross-platform release binaries + Homebrew tap

Would love thoughts on the eval methodology — what does a good default dataset
look like for coding-agent evaluation?

https://github.com/forrestbthomas/pi-harness
