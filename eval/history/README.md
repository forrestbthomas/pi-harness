# Committed eval-run history

Per-run live-eval scorecards, committed so the harness keeps a transparent,
auditable history of evaluations — including runs whose gate failed. Raw
per-run artifacts stay in `eval/live-results/` (gitignored, uploaded by CI);
this directory holds the curated, permanent record: scorecard + delta report +
raw pytest report + caveats.

## Convention

Each run adds one row to the table below and four files, named
`<date>-<provider>-<model>.{scorecard,delta,report}.{json,md}`:

- `*.scorecard.json` — output of `eval/scripts/score_run.py` (the seam contract).
- `*.delta.json` / `*.delta.md` — output of `eval/scripts/score_delta.py` vs the
  committed baseline (report-only).
- `*.report.json` — raw pytest report (input to score_run), so the record is
  self-contained and re-runnable.

Provenance must be truthful when generating a scorecard: set `PI_MODEL_TIER`
(the tier name actually launched), `OPENAI_MODEL_NAME` (the judge model), and
`PI_VERSION` (the built pi-run version) on the score_run invocation. The
scorecard's `agentModel` is the **tier name** by design (score_run.py NOTE) —
record the resolved model id in the caveats when a run deviates from the
provider default.

## Runs

| Date (local) | Provider | Agent model | Tier | Judge model | n/case | Cases | Pass | Pass rate | Cost | Gate |
|---|---|---|---|---|---|---|---|---|---|---|
| 2026-08-21 14:39–14:59 | openrouter | openai/gpt-5.6-terra (remapped) | cheap | openai/gpt-5.6-terra (via OpenRouter) | 1 | 57 | 50 | 87.7% | $0.905 | FAIL (cost-only) |

---

## 2026-08-21 — OpenRouter URL + gpt-5.6-terra (local full live suite)

Files: `2026-08-21-openrouter-gpt5.6-terra.*`

**What ran:** the full live suite (`tests/test_live_suite.py`,
`tests/test_live_metrics.py`, `tests/test_agent_task_completion.py`), locally
from repo `main` (HEAD `54f0ee6`, pi-run built fresh as
`v0.11.1-3-g54f0ee6`). 59/59 pytest tests executed in 20:20.

### Configuration

- **Provider:** `openrouter` — Pi's native OpenRouter provider
  (`https://openrouter.ai/api/v1`); `OPENROUTER_API_KEY` from Bitwarden.
- **Agent model:** `openai/gpt-5.6-terra` — **reached via a temporary,
  uncommitted `providers.json` remap** (openrouter `modelTiers.cheap` →
  `openai/gpt-5.6-terra`) because the live suite hardcodes
  `--model-tier cheap`. Restored immediately after the run (`git diff
  providers.json` empty). The scorecard's `agentModel: "cheap"` therefore
  means *tier cheap, resolved to openai/gpt-5.6-terra for this run only*.
- **Judge:** DeepEval routed through the same OpenRouter URL
  (`OPENAI_BASE_URL=https://openrouter.ai/api/v1`, `OPENAI_API_KEY` = OR key,
  `OPENAI_MODEL_NAME=openai/gpt-5.6-terra`).
- **Scale:** `EVAL_RUNS_PER_CASE=1` (one agent run per case), `EVAL_JUDGE_RUNS=3`.
- **Budget:** `PI_MAX_BUDGET_USD=5`. The budget cap reads the **cumulative**
  spend ledger (was $225 historical), so the budget period was reset first
  (`pi-run cost --reset`; old ledger archived to
  `.pi/cost-ledger-20260821T213840Z.archive.jsonl`).
- **Dataset:** `2026-08-15.4` (55 live cases + 2 task-completion tests).

### Results

- **57 cases, 50 passed → 87.7%** overall pass rate; 7 failed.
- **Cost:** $0.905 agent spend (543K tokens, ~$0.0159/task), well under the $5 cap.
- **Self-heal events:** 0 (healthy run).
- **Gate: FAIL — cost-only.** 43 cases exceeded 2× baseline
  `costPerTaskUsd` (premium model vs the cheap-tier baseline; expected). The
  pass-rate gate **passed**: all 7 failures were classified as flakes (n=1 →
  variance band 1.0 → never a gate failure by design).

### Caveats (read before citing this scorecard)

1. **n=1 per case is not statistically comparable** to the 5-run baseline.
   Treat 87.7% as a signal, not a measurement; single-run failures
   (coding-013, coding-015, coding-044, coding-050 vs baselines 0.5–1.0) may
   be flake or signal — the EVAL-18 band correctly refused to call them
   regressions. Single-run *improvements* (coding-011 +0.4, coding-017 +0.6,
   coding-029 +0.8, coding-055 +0.2) are equally noisy.
2. **`agentModel` is a tier name, not the model id.** The scorecard records
   `"cheap"`; the actual model was `openai/gpt-5.6-terra` via the temporary
   tier remap described above.
3. **Judge cost records $0** — deepeval 4.1.8 has no pricing entry for
   `openai/gpt-5.6-terra`, so `evaluation_cost` never accrued. This is a known
   config gap (spec §4.5), not a free judge.
4. **Gate failure is expected and is the point.** The committed baseline was
   measured on cheap-tier models (gpt-5-mini / deepseek-v4-flash); a premium
   model will fail the cost gate by construction. This run is evidence of
   "you get what you pay for," not a regression in harness behavior.
5. **Run provenance:** provider=openrouter, agent=openai/gpt-5.6-terra,
   judge=openai/gpt-5.6-terra (via OpenRouter), EVAL_RUNS_PER_CASE=1,
   pi-run v0.11.1-3-g54f0ee6, datasetVersion 2026-08-15.4, run window
   2026-08-21 14:39:04–14:59:24 local.
6. **Repo state after the run:** clean. `providers.json` restored; a stray
   untracked `eval/providers.json` (byte-identical copy of the catalog, no
   secrets) that appeared at session end was removed.

### Reproduce

```bash
# 1. (temporary, uncommitted) remap the cheap tier for this provider/model:
#    providers.json -> openrouter.modelTiers.cheap = "openai/gpt-5.6-terra"
# 2. reset the budget period so PI_MAX_BUDGET_USD counts only this run:
pi-run cost --reset
# 3. run the live suite (from eval/), judge through OpenRouter:
PI_PROVIDER=openrouter \
OPENAI_BASE_URL=https://openrouter.ai/api/v1 \
OPENAI_MODEL_NAME=openai/gpt-5.6-terra \
EVAL_RUNS_PER_CASE=1 EVAL_JUDGE_RUNS=3 \
PI_MAX_BUDGET_USD=5 PI_SELF_HEAL=1 PI_MODEL_TIER=cheap \
PI_EVAL_REPORT=live-results/report.json \
.venv/bin/python -m pytest \
  tests/test_live_suite.py tests/test_live_metrics.py tests/test_agent_task_completion.py
# 4. score it with truthful provenance (env vars on the invocation):
PI_MODEL_TIER=cheap OPENAI_MODEL_NAME=openai/gpt-5.6-terra PI_VERSION=$(pi-run version) \
eval/.venv/bin/python eval/scripts/score_run.py --report eval/live-results/report.json \
  --baseline eval/baselines/live-baseline.json --tolerance 0.05 --runs 1 \
  --budget-usd 5.0 --out eval/live-results/live-<ts>.json
# 5. restore providers.json, then copy scorecard/delta/report here + add a README row.
```
