# Cost-Aware Model Routing (`--model-tier`) — Design

**Date:** 2026-08-12
**Status:** SHIPPED — `--model-tier` + PI_MODEL_TIER (v0.8.0, PR #38)
**Target release:** v0.8.0 (feature)

## 1. Context & Motivation

**Landscape:** The 2026 pattern for model routing is "route in the gateway by
task tier, optimize **end-to-end task cost**" (P2 research synthesis
docs/p2-research-synthesis-2026-08.md §1.3, researcher brief `63686c9b`).
Competitive tools (Claude Code, LiteLLM, OpenRouter) all gained *some* form of
per-task-tier model selection; the research explicitly validates that our
explicit no-fallback routing + cost tracking is the right shape, and calls a
cost-aware router (per-task-tier model choice) the next step.

**What pi-harness already has:** an explicit routing table (`providers.json` +
embedded `defaultProviders`, 17 providers), a cost command and a budget cap
(`pi-run cost` / `--max-budget-usd`, shipped v0.5.0), and a hard safety
property: **the only routing inputs are `providers.json` /
`PI_RUN_PROVIDERS_FILE` and `--provider` / `PI_PROVIDER`; there is NO automatic
cross-provider fallback**. The gap is a *within-provider* model-tier selector:
today the user picks a provider and gets its single `defaultModel` unless they
type an exact `--model` id. There is no first-class "I want the cheap model /
the fast model for this provider" affordance, which is exactly the lever the
2026 cost-per-task pattern wants.

**Goal:** a `--model-tier <fast|balanced|cheap>` selector for `chat`/`print`
(env `PI_MODEL_TIER`) that picks a model **within the already-selected
provider**, with zero change to the provider-routing invariant.

## 2. Current State (verified)

- `Provider` struct — `internal/cli/providers.go:12-16`: `Name`, `KeyEnv`,
  `PiProvider`, `DefaultModel`, `BaseURL`. No tier concept today.
- Embedded routing table — `internal/cli/providers.go:28-47`: 17 providers;
  `openai` is the default. `providers.json` (repo root) mirrors the same table
  field-for-field and is pinned equal to the embedded table by
  `TestDefaultProvidersMatchRepoProvidersJSON` (`internal/cli/providers_json_test.go:88-113`,
  struct `==` comparison — **any new struct field must land in providers.json in
  the same change or this test fails**).
- Provider resolution — `ResolveProvider` (`internal/cli/providers.go:152-161`):
  `--provider` flag → `PI_PROVIDER` env → `"openai"`. `LookupProvider`
  (`internal/cli/providers.go:121-128`) errors with a "want one of:" list — the
  error-style template this spec reuses.
- Load/validation — `loadActiveProviders` (`internal/cli/providers.go:62-71`):
  any load error warns loudly on stderr and **falls back to built-in defaults**.
  `loadProviders` (`internal/cli/providers.go:105-119`): a missing *default*
  repo table falls back silently; a missing/malformed *explicit*
  (`PI_RUN_PROVIDERS_FILE`) table is an error. `ProvidersFromJSON`
  (`internal/cli/providers.go:85-94`) is the single JSON choke point.
- Flag parsing — `splitLaunchArgs` (`internal/cli/app.go:247-295`) parses
  `--provider`/`--model`/`--max-budget-usd`/`--permission-mode`/`--read-only`
  (with `=`-forms) and passes everything else through to pi.
- Launch wiring — `runLaunch` (`internal/cli/app.go:155-198`): resolves
  permission mode → budget cap → provider → **model** (`model := modelFlag; if
  model == "" { model = p.DefaultModel }`, `internal/cli/app.go:183-185`) →
  budget pre-flight (exit 6, `internal/cli/cost.go:20`) → key (exit 3) → node
  (exit 4) → hooks → `execPi(piArgs(...))`.
- Model injection into pi — `piArgs` (`internal/cli/pi.go:59-92`) always emits
  `--provider <PiProvider> --model <model> --offline`; the tier work only needs
  to produce the right `model` string.
- Flag-wins-over-env precedent — `resolveBudgetCap` (`internal/cli/cost.go:477-491`)
  and `resolvePermissionMode` (`internal/cli/app.go:306-317`, unknown value →
  usage error). `--model-tier` follows both patterns.
- Exit codes — usage errors are **exit 2** (usage text `internal/cli/app.go:44-70`,
  table `internal/cli/app.go:80-89`). No new exit code is needed; the
  `--exit-codes` table is unchanged.
- Display — `runProviders` (`internal/cli/app.go:451-460`) prints
  `name\tdefaultModel\tkeyEnv\t[baseURL]` tab-separated.
- Deterministic checks — `runConfigCheck` (`internal/cli/config_check.go:14-80`)
  prints `[ok]`/`[FAIL]` lines, exit 1 on any failure; the place for a
  providers.json tiers check.
- Budget — `resolveBudgetCap` + pre-flight refusal (exit 6) + post-run ledger
  (`internal/cli/cost.go:477-491`, `app.go:191-206`). Orthogonal to model
  selection by construction: it never reads the model id.

## 3. Scope

### In scope (this feature)

1. `--model-tier <fast|balanced|cheap>` flag for `chat`/`print`; env
   `PI_MODEL_TIER` (flag wins over env).
2. `ModelTiers map[string]string` field on `Provider` (`json:"modelTiers,omitempty"`),
   mirrored in `providers.json` and the embedded `defaultProviders` table.
3. Resolution: `resolveModelTier(p Provider, tier string) (string, error)` +
   `resolveLaunchModel(p Provider, tier, model string) (string, error)` wiring;
   empty/`balanced` → `p.DefaultModel`.
4. Strict validation: malformed tier maps (bad type, empty value, unknown tier
   key, reserved `balanced` key) make `providers.json` load fail → existing
   warn-and-fallback path.
5. `pi-run providers` shows per-provider available tiers.
6. `config-check` gains a deterministic providers.json tiers check.
7. Hermetic Go tests (no keys, no network): resolution, conflict, exit codes,
   schema validation, display.

### Explicitly OUT of scope

- **Cross-provider fallback** — the tier feature must never change the provider
  and must never silently substitute a model or provider (the invariant, §4.1).
- **eval / ci-benchmark tier routing** — benchmark runs keep
  `--provider`/`--model` only; `--model-tier` is a `chat`/`print` surface.
- **Budget auto-downgrade** — `--max-budget-usd` never changes the selected
  model; auto-downgrade is explicitly rejected for v1 (§4.7, future work).
- **Price-table or cost-based model choice** — tiers are a *static, explicit*
  table; nothing in v1 computes "cheapest model" from live prices.
- **Per-task runtime tier switching** (route different turns of one session to
  different tiers) — v1 selects one model per launch.
- **`resume` tier support** — `--model-tier` with `resume` is a usage error
  (exit 2); a resume should continue the session's model. `--model` remains the
  explicit override for resume, as today (`internal/cli/pi.go:59-92`).
- **README / CHANGELOG edits and the `providers.json` row edits** — owned by the
  parent (docs pass + providers.json must land in the same change as the struct
  field, see §6 / §9).

## 4. Design

### 4.1 Invariant — model-tier selection NEVER changes the provider, NEVER silently falls back (#1 design constraint)

**Constraint (design law #1):** `--model-tier` is a *within-provider* model
selector. It resolves against the provider selected by the existing, unchanged
routing chain (`--provider` / `PI_PROVIDER` / default `openai` →
`ResolveProvider`, `internal/cli/providers.go:152-161`; table from
`providers.json` / `PI_RUN_PROVIDERS_FILE` / embedded defaults). Under no
circumstance may the tier machinery:

1. change which provider is used (no "this tier is cheaper on DeepSeek, switch"
   behavior), or
2. silently substitute a different model or provider when the requested tier is
   unavailable (no "requested fast, got balanced" behavior).

**Rationale:** pi-harness's core safety property is **explicit provider
routing**. Changing providers silently would alter which API key is used, which
billing account is charged, data-residency/egress, and model behavior — the
exact surprise the no-fallback design exists to prevent. Silent model
substitution is the same surprise class one level down: the user asked for a
tier and gets a different one, so cost and capability results become
non-reproducible ("why did this run cost twice as much / answer differently?").
The research brief's own recommendation is *"our explicit no-fallback routing
+ cost tracking is the right shape"* — the tier feature must extend that shape,
not bend it. The cost of the invariant is a hard failure (exit 2, with a
helpful message listing what *is* available) when a tier is unavailable —
exactly the loud-failure contract `loadActiveProviders` already applies to
malformed tables (`internal/cli/providers.go:62-71`).

### 4.2 Interface

```
pi-run chat  --model-tier <fast|balanced|cheap> "task"
pi-run print --model-tier <fast|balanced|cheap> "task"
PI_MODEL_TIER=<fast|balanced|cheap> pi-run chat "task"   # env fallback
```

- Valid tier names: `fast`, `balanced`, `cheap`.
- `balanced` is a reserved alias for **the provider's `defaultModel`** (it is
  never stored in the tier map).
- Precedence: `--model-tier` flag wins over `PI_MODEL_TIER` env — the same
  flag-wins pattern as `resolveBudgetCap` (`internal/cli/cost.go:477-485`) and
  `resolvePermissionMode` (`internal/cli/app.go:306-317`).
- Parsing: `splitLaunchArgs` (`internal/cli/app.go:247-295`) gains
  `--model-tier` and `--model-tier=` forms, alongside the existing flags; all
  other args still pass through to pi unchanged.
- `resume` rejects the **flag** (exit 2) and **ignores `PI_MODEL_TIER` env entirely** — see §3 OUT. A resumed session continues with the model it was launched with; the env must not silently swap it.
- `--model` is unchanged and still takes precedence over tier resolution *as an
  explicit exact-id override* **unless both are given** (§4.3c).

### 4.3 Decision table — exact behavior and exit codes

| # | Situation | Exact behavior | Exit code |
|---|---|---|---|
| (a) | Tier is **unknown** (not `fast`/`balanced`/`cheap`, e.g. `--model-tier turbo`) | Stderr usage error naming the valid set — `pi-run: chat: unknown model tier "turbo" (valid: fast, balanced, cheap)` — before any key/node access, and no pi launch. Mirrors the unknown-permission-mode error (`internal/cli/app.go:306-317`). | **2** (usage) |
| (b) | Tier is **known** but the selected provider has **no model for it** (e.g. `--provider deepseek --model-tier cheap`, deepseek ships no `cheap` entry) | Stderr usage error listing the tiers that provider actually offers, e.g. `pi-run: chat: provider "deepseek" has no model for tier "cheap" (available: balanced, fast)`. **No fallback to another tier, model, or provider.** | **2** (usage) |
| (c) | `--model-tier` **and** `--model` **both** given as **flags** | Stderr usage error — `pi-run: chat: --model-tier and --model are mutually exclusive; pick one` — and no pi launch. Rejected rather than "`--model` wins" because silently ignoring an explicit flag is the same surprise class as fallback, and precedence rules that differ between flag and env forms are a footgun. | **2** (usage) |
| (c') | **env** `PI_MODEL_TIER` set **and** explicit `--model` flag given | `--model` wins (no error). The env var is a *default*, not an explicit flag, so an explicit `--model` overrides it — exactly the flag-wins-over-env precedent in `resolveBudgetCap` (`internal/cli/cost.go:477-491`) and `resolvePermissionMode` (`internal/cli/app.go:306-317`). A globally exported `PI_MODEL_TIER` must never break existing `--model` invocations. | 0 / normal launch |
| (d) | `--model-tier` given with **no provider** | Provider resolves exactly as today — default `openai` (`ResolveProvider`, `internal/cli/providers.go:152-161`) — and the tier resolves against `openai`'s tier map. Provider selection is untouched by the tier flag. | 0 / normal launch path (key/node/etc. as today) |

Failure order in `runLaunch` (all exit-2 checks happen before key resolution, so
usage errors always beat the missing-key exit 3 — same ordering rule already in
`internal/cli/app.go:158-162`): permission mode → budget cap → **tier/model
conflict** → provider → **tier resolution** → model → budget pre-flight (6) →
key (3) → node (4) → hooks → launch.

### 4.4 Data model

**Struct** (`internal/cli/providers.go:12-16` gains one field):

```go
type Provider struct {
    Name         string            `json:"name"`
    KeyEnv       string            `json:"keyEnv"`
    PiProvider   string            `json:"piProvider"`
    DefaultModel string            `json:"defaultModel"`
    BaseURL      string            `json:"baseURL,omitempty"`
    ModelTiers   map[string]string `json:"modelTiers,omitempty"` // tier name -> model id
}
```

**providers.json mirror:**

```json
{
  "name": "openai",
  "keyEnv": "OPENAI_API_KEY",
  "piProvider": "openai",
  "defaultModel": "openai/gpt-5.6-terra",
  "modelTiers": {
    "fast":  "openai/gpt-5.6-mini",
    "cheap": "openai/gpt-5.1-mini"
  }
}
```

Map semantics: `fast`/`cheap` → an **exact model id for that provider** (id
format identical to `defaultModel`, i.e. `provider/model` as passed to pi via
`--model`, `internal/cli/pi.go:59-92`). `balanced` is reserved and must never
appear as a key (§4.5). A provider without tiers simply omits the field (nil
map → only `balanced` available).

**Built-in `defaultProviders` tiers — which providers get them (v1):**

Five providers ship tier maps; the other twelve ship none (their
`defaultModel` is already the cheap/fast line — groq/together/fireworks llama
70B-class, ollama/local single-model endpoints, moonshot/kimi-k2, xai/grok-4 —
so an extra tier adds noise, not capability). Omitted tiers on a tiered
provider are intentional and demonstrate rule (b) in practice.

| provider | `defaultModel` (= balanced) | `fast` | `cheap` |
|---|---|---|---|
| openai | `openai/gpt-5.6-terra` | `openai/gpt-5.6-mini` ⚠ | `openai/gpt-5.1-mini` ⚠ |
| openrouter | `openai/gpt-5.6-terra` | `openai/gpt-5.6-mini` ⚠ (served via openrouter) | `deepseek/deepseek-v4-flash` ⚠ (served via openrouter) |
| deepseek | `deepseek/deepseek-v4-flash` | `deepseek/deepseek-v4-pro` ⚠ | *(omitted — the `-flash` default is already the cheapest line)* |
| anthropic | `anthropic/claude-sonnet-4` | `anthropic/claude-haiku-4` ⚠ | *(omitted — haiku is Anthropic's single fast/cheap line; nothing cheaper in the current lineup)* |
| gemini | `gemini/gemini-2.5-pro` | `gemini/gemini-2.5-flash` ⚠ | `gemini/gemini-2.5-flash-lite` ⚠ |

> ⚠ **VERIFY BEFORE RELEASE:** every tier model above except
> `deepseek/deepseek-v4-pro` (which already appears as a valid `--model` value
> in README.md:270) is a **placeholder string that must be confirmed against
> the pi model catalog** (`pi-run setup` / `pi update --models`) before v0.8.0
> ships. If a string is wrong, fix the tier entry — the invariant is unchanged;
> only the table content is corrected. `openrouter/auto` is deliberately NOT a
> tier value: it delegates model choice to OpenRouter, which is the
> silent-substitution class this design rejects as a *tier default* (it remains
> available today via an explicit `--model openrouter/auto`,
> README.md:273).

### 4.5 Validation of providers.json against the struct

`ProvidersFromJSON` (`internal/cli/providers.go:85-94`) remains the single JSON
choke point and gains a tier check (`validateTiers(p Provider) error`):

1. **Type-level malformation** (e.g. `"modelTiers": "not-an-object"`) — fails
   `json.Unmarshal` today, no new code: `ProvidersFromJSON` returns an error →
   `loadProviders` returns it → `loadActiveProviders` warns on stderr and falls
   back to built-in defaults (`internal/cli/providers.go:62-71`), identical to
   the existing malformed-table contract pinned by
   `TestLoadActiveProvidersMalformedDefaultWarns`
   (`internal/cli/providers_json_test.go`).
2. **Semantic validation** (new, in `validateTiers`):
   - key not in {`fast`, `cheap`} → error (catches typos like `"fasst"`, which
     would otherwise be *silently inert* — the user's "fast run" would quietly
     run balanced; that is exactly the silent-surprise class the invariant
     forbids);
   - key `balanced` present → error (reserved alias for `defaultModel`;
     storing it is a config bug);
   - value empty/whitespace → error.
   Any violation fails the whole table (warn + fall back to defaults) — **no
   partial tolerance**. Rationale: partially accepting a bad table would create
   a state where `providers.json` says one thing and the running binary does
   another, and the existing whole-table fallback is the established contract.

**Sync constraint:** because `TestDefaultProvidersMatchRepoProvidersJSON`
compares full structs (`providers_json_test.go:88-113`), the `providers.json`
rows for the five tiered providers **must be updated in the same change** as
the struct field and embedded table, or the suite fails. The parent owns the
`providers.json` edit (docs/JSON pass); this spec pins the exact field shape in
§4.4 so the edit is mechanical.

### 4.6 Resolution

Two functions in `internal/cli/providers.go` (testable without execPi):

```go
// knownModelTiers is the closed set of tier names. balanced is the
// default-model alias and is never stored in a tier map.
var knownModelTiers = map[string]bool{"fast": true, "balanced": true, "cheap": true}

// resolveModelTier returns the model id for a requested tier of provider p.
//   tier == ""         -> p.DefaultModel            (no tier requested)
//   tier == "balanced" -> p.DefaultModel            (reserved alias)
//   tier in p.ModelTiers -> that model id
//   tier known, not mapped -> error listing the tiers p DOES offer
//   tier unknown           -> error listing the valid tier names
// There is deliberately NO case that returns a different tier/model than the
// one requested: an unavailable tier is an error, never a fallback.
func resolveModelTier(p Provider, tier string) (string, error)

// availableTiers returns the sorted, deduped tier names a provider offers;
// "balanced" is always present.
func availableTiers(p Provider) []string

// resolveLaunchModel is the runLaunch wiring: (c) conflict, (a) unknown tier,
// (b) unmapped tier, and the existing --model / default-model behavior.
func resolveLaunchModel(p Provider, tier, modelFlag string) (string, error)
```

`resolveLaunchModel` logic, in order:

1. `tier != "" && modelFlag != ""` → error: `--model-tier and --model are mutually exclusive; pick one` (exit 2, rule (c)).
2. `tier != ""` → `resolveModelTier(p, tier)` (rules (a)/(b) surface here).
3. else `modelFlag != ""` → `modelFlag`; else `p.DefaultModel` (today's behavior, `internal/cli/app.go:183-185`).

`runLaunch` wiring: `splitLaunchArgs` returns the tier flag; `runLaunch`
computes `tier := tierFlag; if tier == "" { tier = os.Getenv("PI_MODEL_TIER") }`
then `model, err := resolveLaunchModel(p, tier, modelFlag)`, error → exit 2
(printed like the other usage errors, `internal/cli/app.go:163-174`). Because
the env is only consulted when the flag is absent, an exported `PI_MODEL_TIER`
never conflicts with an explicit `--model` (rule (c')) and never applies to
`resume` (which rejects the flag and never reads the env). The
resolved `model` flows into the existing `piArgs` call unchanged
(`internal/cli/pi.go:59-92`) — pi.go needs **no** edits.

Error messages follow the `LookupProvider` "want one of:" style
(`internal/cli/providers.go:121-128`), e.g.:
`provider "deepseek" has no model for tier "cheap" (available: balanced, fast)`.

### 4.7 Budget interaction — `--max-budget-usd` stays orthogonal

**Decision: the budget cap does NOT auto-downgrade the model tier in v1.**

- `--max-budget-usd` / `PI_MAX_BUDGET_USD` (`resolveBudgetCap`,
  `internal/cli/cost.go:477-491`) enforces a spend ceiling the only way it does
  today: pre-flight refusal when cumulative spend ≥ cap (exit 6,
  `internal/cli/app.go:191-206`) and a post-run warning. It never reads the
  model id, and `--model-tier` never reads the budget.
- **Why no auto-downgrade:** (1) it violates the explicit-routing principle —
  the user asked for `--model-tier fast`, and the system would silently launch
  `cheap`; (2) it makes runs non-reproducible — the same command line yields a
  different model depending on prior spend, so cost attribution and benchmark
  results become unstable; (3) the budget already has a clean, loud enforcement
  path (refusal + warning), and a user who wants cheap runs can simply pass
  `--model-tier cheap`.
- **Future work (documented, not built):** an explicit, opt-in
  `--auto-downgrade-tier-on-budget` flag *if* a real need appears — with
  mandatory loud stderr notice of the downgrade and ledger recording of the
  actual model used (the ledger already records the resolved model per run,
  `internal/cli/cost.go` `recordRunSpend`). It is rejected for v1 so the v1
  contract stays "the flag you pass is the model you get".

### 4.8 Display — `pi-run providers`

`runProviders` (`internal/cli/app.go:451-460`) gains a `TIERS` column inserted
after the default model (before `keyEnv`, so the key/URL tail stays stable):

```
name    defaultModel            tiers                  keyEnv
openai  openai/gpt-5.6-terra    balanced,cheap,fast    OPENAI_API_KEY
deepseek deepseek/deepseek-v4-flash balanced,fast       DEEPSEEK_API_KEY
groq    groq/llama-3.3-70b-versatile  balanced          GROQ_API_KEY
```

- `TIERS` = comma-joined `availableTiers(p)` (sorted; `balanced` always
  present), `-` when the provider ships no tier map (only `balanced` — display
  `balanced` for uniformity; decision: always print `balanced` since it is
  universally valid).
- Tab-separated, matching the existing line format so existing
  `runProviders`-style consumers and tests keep working with one new column.

## 5. Implementation Plan

1. `internal/cli/providers.go`: add `ModelTiers` field to `Provider`;
   add tier entries to the five `defaultProviders` (§4.4 table, ⚠ placeholders
   flagged in a code comment "verify before release"); add `knownModelTiers`,
   `resolveModelTier`, `availableTiers`, `resolveLaunchModel`, and
   `validateTiers` (called from `ProvidersFromJSON`).
2. `internal/cli/app.go`: `splitLaunchArgs` parses `--model-tier` /
   `--model-tier=`; usage text (`internal/cli/app.go:44-70`) documents the flag
   + `PI_MODEL_TIER`; `runLaunch` adds the conflict check, env fallback, and
   `resolveLaunchModel` call replacing `internal/cli/app.go:183-185`; `resume`
   rejects the flag (exit 2); `runProviders` adds the TIERS column.
3. `internal/cli/config_check.go`: add a deterministic check that the repo
   `providers.json` tier maps pass `validateTiers` (no keys/network; the
   `[ok]`/`[FAIL]` pattern at `config_check.go:14-80`).
4. Tests — see §6.
5. `providers.json`: add `modelTiers` to the five providers (**parent-owned**;
   must land with step 1 or the sync test fails).
6. Docs: README "Model Routing" section (**parent-owned**, §9) + CHANGELOG
   v0.8.0 entry (parent-owned). No new exit code, so the `--exit-codes` table
   (`internal/cli/app.go:80-89`) is unchanged.

## 6. Tests (hermetic — no keys, no network)

Follow the existing patterns in `internal/cli/providers_test.go`,
`internal/cli/providers_json_test.go`, and `internal/cli/app_test.go`
(`captureProvidersStderr`, `captureRunStdout`, `t.Setenv`, `t.TempDir()`).

**`internal/cli/providers_test.go`:**
- `TestResolveModelTier` — table-driven: `""` → `DefaultModel`; `"balanced"` →
  `DefaultModel`; `"fast"`/`"cheap"` → mapped id (openai, gemini); unknown tier
  `"turbo"` → error naming valid tiers; known-but-unmapped `"cheap"` on
  deepseek → error naming `balanced, fast`. Uses `LookupProvider` + the real
  embedded table.
- `TestResolveModelTierNeverFallsBack` — for every tiered provider and every
  known tier, the result is either an error or exactly the requested map
  entry/`DefaultModel` — asserts the no-silent-fallback invariant structurally.
- `TestDefaultProvidersTierMapsValid` — for every `defaultProviders` entry:
  keys ⊆ {`fast`, `cheap`}, no `balanced` key, all values non-empty.
- `TestAvailableTiers` — openai → `[balanced cheap fast]` (sorted); deepseek →
  `[balanced fast]`; groq (no map) → `[balanced]`.
- `TestResolveLaunchModel` — table-driven: tier-only, model-only, neither,
  conflict (`tier`+`modelFlag` → error), unknown tier → error.

**`internal/cli/providers_json_test.go`:**
- Extend `sampleProvidersJSON` with a `modelTiers` block; extend
  `TestProvidersFromJSON` to assert the parsed map.
- `TestProvidersFromJSONInvalidTiers` — table: `"modelTiers": "nope"` (type
  error), `{"fast": ""}`, `{"balanced": "..."}`, `{"fasst": "..."}` → all
  errors.
- `TestLoadActiveProvidersMalformedTiersWarns` — repo `providers.json` with a
  bad tier map → `captureProvidersStderr` shows the warning and the result
  equals `defaultProviders` (mirrors `TestLoadActiveProvidersMalformedDefaultWarns`).
- `TestDefaultProvidersMatchRepoProvidersJSON` — existing test automatically
  covers tier sync once providers.json lands; no new code.

**`internal/cli/app_test.go`:**
- `TestLaunchModelTierConflictExit2` — `chat --model-tier fast --model x` → 2.
- `TestLaunchUnknownTierExit2` — `chat --model-tier turbo` → 2.
- `TestLaunchUnavailableTierExit2` — `print --provider deepseek --model-tier cheap` → 2.
- `TestLaunchModelTierResumeRejected` — `resume --model-tier fast` → 2.
- `TestRunProvidersShowsTiers` — `providers` output contains the TIERS column
  values for a fixture table.
- `TestSplitLaunchArgsModelTier` — both `--model-tier fast` and
  `--model-tier=fast` parse; pass-through intact.

**config-check:** `runConfigCheck` with a valid repo providers.json passes the
new tier check; with a corrupted fixture (via `HARNESS_ROOT`) it prints
`[FAIL]`. (Pattern: existing config-check test setup in
`internal/cli/app_test.go` / `config_check`-adjacent tests.)

## 7. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Placeholder tier model strings don't exist in pi's catalog | High | All non-catalog strings marked ⚠ and gated: verify via `pi-run setup` catalog before release; wrong strings are a table edit, not a design change |
| `providers.json` and the embedded table drift apart (sync test) | Medium | Existing struct-equality test (`providers_json_test.go:88-113`) fails loudly; providers.json lands in the same change (§4.5) |
| Users expect tier = cross-provider cheapest | Medium | Invariant is #1 and stated in docs + error messages; `pi-run providers` shows per-provider tiers so the boundary is visible |
| `--model-tier` plus legacy `--model` habits cause friction | Low | Clear mutual-exclusion error (exit 2) listing both flags; README documents the choice |
| Auto-downgrade requests from budget users | Low-Med | Explicitly rejected in v1 with rationale (§4.7); the loud refusal (exit 6) + `--model-tier cheap` covers the real need |

## 8. Decision

**Recommend proceeding.** The P2 research names per-task-tier model routing as
the 2026 cost pattern and validates our explicit no-fallback shape; this spec
adds the tier lever **inside** that shape: a static, validated
`modelTiers` table per provider, a three-word flag (`--model-tier
fast|balanced|cheap`) + env var, strict exit-2 failures instead of any fallback,
an unchanged budget cap, and ~10 hermetic tests. No new exit codes, no changes
to `piArgs`/launch plumbing, no eval/benchmark surface. The only external
dependency is the parent-owned `providers.json` edit landing with the struct
field (mechanical, pinned in §4.4).

## 9. Docs Pass (parent-owned — NOT edited here)

- **README "Model Routing" section (README.md:256-273):** add `--model-tier`
  flag + `PI_MODEL_TIER` env; the tier table for the five providers; the
  invariant statement ("tier never changes provider, never falls back"); the
  (a)-(d) behaviors incl. the `--model` conflict rule; pointer to
  `pi-run providers` TIERS column. Do NOT edit README in this lane.
- **CHANGELOG:** v0.8.0 entry under the parent's release pass.
- **providers.json:** five `modelTiers` blocks (§4.4) — must land with the
  struct field change (sync test).
- **This spec's review gate:** §Review checklist below.

## Review checklist

A reviewer can verify the implementation against this spec by checking:

- [ ] **Invariant preserved:** `resolveModelTier`/`resolveLaunchModel` never
      touch `Provider` identity — grep that no code path in the feature calls
      `ResolveProvider` or mutates the provider after resolution, and that an
      unavailable tier returns an error rather than another model/`DefaultModel`
      (test `TestResolveModelTierNeverFallsBack` proves it structurally).
- [ ] **Exit codes:** (a) unknown tier → 2, (b) known-but-unmapped tier → 2,
      (c) `--model-tier`+`--model` conflict → 2, resume rejection → 2 — all
      before any key/node access; no new exit code added; `--exit-codes` table
      unchanged.
- [ ] **No silent fallback:** no case in `resolveModelTier` returns a tier
      different from the one requested; malformed tier maps warn + fall back to
      built-in defaults (existing `loadActiveProviders` path), never silently
      drop the map while keeping the rest of a custom table.
- [ ] **providers.json validation:** `validateTiers` rejects unknown keys,
      `balanced` key, empty values; type-level malformation fails the whole
      table load; `TestDefaultProvidersMatchRepoProvidersJSON` passes with the
      updated providers.json.
- [ ] **Budget orthogonal:** `--max-budget-usd` path (`resolveBudgetCap`, exit
      6) has no model-tier interaction; no auto-downgrade code exists.
- [ ] **Display:** `pi-run providers` shows a TIERS column
      (`balanced,cheap,fast` style, `balanced` for untiered providers).
- [ ] **Hermetic tests listed in §6** are present and pass with no keys and no
      network (`go test ./internal/cli/`).
- [ ] **Placeholder discipline:** every non-catalog tier model string is marked
      ⚠ (or in a code comment) as verify-before-release.
- [ ] **Scope:** no Go code outside `internal/cli/providers.go`, `app.go`,
      `config_check.go`, and the three test files was changed; README,
      CHANGELOG, providers.json, eval/ untouched by this lane.
