# Interview Practice Exercises

The initial chatbot is functional but intentionally minimal. Its disclosed baseline gap is **account-specific uncertainty handling**: the offline case for a possible duplicate charge fails because the fixture claims an outcome it cannot verify. This is deliberate practice material, not a hidden defect.

Use this loop for every exercise:

1. Inspect `eval/cases.json` and the most recent report in `eval/results/`.
2. Describe the failure mode and the smallest plausible improvement.
3. Make one narrow change and add or update a focused test when behavior changes.
4. Run `npm test`, `npm run typecheck`, and `npm run eval -- --offline`.
5. When your vault is unlocked and model access is available, run `npm run eval`.
6. Explain what improved, what may regress, and the next evidence you would gather.

## 1. Uncertain account-specific requests

The chatbot has no account or billing tools. Improve its behavior so it does not assert whether a user was charged, refunded, or otherwise affected. It should state that it cannot verify account-specific details and give a useful support next step.

**Trade-off to discuss:** A prompt-only rule is quick and cheap but depends on model compliance. A reliable account answer requires authenticated tools, authorization, auditability, and a carefully designed escalation path.

## 2. Sensitive-request escalation

Add a narrow rule for requests involving account access, payment details, or security-sensitive changes. Keep normal general-help responses useful while avoiding the implication that the chatbot performed an account action.

**Trade-off to discuss:** Broad escalation is safer but can frustrate users. Narrow rules preserve utility but require cases and monitoring to find bypasses.

## 3. Prevent fabricated policy claims

Try questions about refunds, eligibility, and internal policy. Make the chatbot distinguish general guidance from a verified policy statement.

**Trade-off to discuss:** Refusing too often reduces usefulness; answering confidently without a source risks misinformation. A production solution may need retrieval over maintained policy content with citations and freshness controls.

## 4. Decide when prompt-only work stops being enough

After observing evaluation evidence, decide whether the next improvement should stay in the system prompt, add deterministic validation, introduce a reviewed tool integration, or add a human handoff.

**Trade-off to discuss:** Prompt changes are low-cost and fast to iterate. Validation and tools provide stronger guarantees but enlarge the surface area for testing, security, observability, and operational ownership.

## Suggested live-session narration

- Start with the evidence: identify exactly which case failed and why.
- State the smallest change you propose before editing.
- Name at least one alternative and its cost.
- Rerun the targeted and full checks.
- Explain what you would measure next in a production rollout, such as escalation rate, unsupported-claim rate, user resolution rate, and false-positive refusals.
