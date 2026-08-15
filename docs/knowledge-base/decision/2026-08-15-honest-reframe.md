# Decision — Honest reframe: the no-moat measurement layer (2026-08-15)

**Status:** Decided (7-persona debate) → adopted as the project's positioning.

## Decision

Adopt the honest reframe: this project is **a no-moat, no-new-science
measurement layer for coding agents** — a self-healing runtime whose product
is the score. The techniques are commodity (every major agent vendor has an
internal eval team that could build this in a week); the offering is the
**discipline made external and verifiable**: a versioned, reproducible,
variance-aware, provenance-stamped measurement of *your* agent configuration,
with the harness's own failures becoming graded cases.

**One-liner:** "The score is the product: a versioned, reproducible,
self-healing measurement of *your* agent configuration — we cannot cheat it,
and you can verify it. This repository is the demo."

## Why (the trigger)

The author, after ~2 weeks of building, was challenged on what the project
actually is versus Claude Code / Codex / Copilot / Pi Core / OpenCode / Deep
Code. Conclusion: **no moat, no new science, all of these companies could
build it in a week** — and they all have internal measurement systems that are
often very good. The difference is positioning, not technology. The
positioning must not overpromise.

## The debate (7 personas, `.pi/debate/honest-reframe/`)

Round-1 verdicts: **ADOPT unanimously (7/7), with modifications.** Strongest
convergence: **no-moat makes the discipline MORE load-bearing, not less** — the
versioned contract is the entire value when the techniques are commodity, so
the release gates (v0.11.0 two green nightlies; v1.0.0 ≥14 green, EVAL-16
enforced) are the product, not quality theater. **No gate threshold changes.**

Key modifications the personas required (all adopted):

1. **Lead with the verifiable claim, not the apology.** "No moat" lives in the
   charter + an honesty/limits block; the README hero leads with the checkable
   claim + the keyless commands (`pi-run config-check`, `pi-run eval --quick`,
   `pi-run doctor`). The published number is a **dated demo**; the **mechanism
   is the promise**.
2. **Kill the grudge wording.** "The vendors won't ship it because it's bad
   for them" is an unfalsifiable opinion about third parties (and flatters
   us). Replaced by the testable claim: **neutrality** — the vendors'
   measurement is welded to their agent; a neutral, cross-provider, externally
   verifiable seam for *your* config is an **uncontested space** (capability ≠
   choice). The vendor-incentive observation stays as one line of context in
   the charter's "why this exists."
3. **"Can't lie" softened everywhere to the checkable claim.** Not "the score
   can't lie" (an absolute that breaks on the first gate hiccup) but "the
   false-fail/false-pass classes we actually hit are handled (flake vs
   regression, run-step variance, median cost) and you can re-run the score
   yourself from the committed contract."
4. **The claims table.** Every README headline claim maps to the
   guard/test/artifact/graded case that enforces it; `test_docs_drift.py` was
   extended so an unbacked claim line fails CI. "We don't ask you to trust
   us" is a tested invariant.
5. **Non-promises extended.** Benchmark caveat: the 55-case suite is a demo
   of the discipline, not a benchmark corpus; rates are evidence of the
   mechanism, not of agent quality. No-support-commitment line. CHARTER's
   "we do not" list is reaffirmed and extended, never trimmed — no-moat means
   fewer features, deeper seam.
6. **Docs-only, before/with v0.11.0.** CHARTER + README + ROADMAP + STATUS +
   this record; every drift-guard invariant green in one CI pass; the release
   train is not delayed.

## What does NOT change

- **Gates:** v0.11.0 and v1.0.0 thresholds untouched; gate definitions are
  advertised as the trust surface.
- **In-flight work:** the v0.11.0 gate countdown (two consecutive green
  nightlies), EVAL-18 (shipped), coding-055 (0.2 → 0.8), the release runbook
  — unchanged. The reframe rides the train; it does not stop it.
- **EVAL-17 (agentic pump):** priority unchanged (6/7; Skeptic dissents on a
  consumer gate). Its DoD is re-expressed against the honest-measurement
  outcome.
- **OSS on-ramp / "distributable is earned":** untouched.

## Re-open triggers

A consumer asks for a different framing, or the neutrality claim is falsified
(e.g., a vendor ships a neutral cross-provider measurement surface and the
uncontested space closes), or the earned-bar gate on v1.0.0 fails and the
positioning needs revision.
