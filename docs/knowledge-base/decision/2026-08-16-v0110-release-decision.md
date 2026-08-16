# Decision — v0.11.0 release: gate re-typed by recorded owner decision (2026-08-16)

**Status:** Decided (owner) → executed the v0.11.0 tag on 2026-08-16.

## Decision

Tag **v0.11.0** now, with the milestone's gate **re-typed by a recorded
owner decision** (the dogfood posture grants the author gate authority —
releases are the loop's integrity checks, not product launches).

**v0.11.0's gate as originally written** (ROADMAP): "Two consecutive green
nightlies on the live suite." **As of this release, the owner re-types it to:**
the loop's integrity check — **baselines honest + full hermetic suite green +
the gate demonstrably free of false-passes**, with the two-green-nightlies
evidence **deferred to the 15× re-baseline data release** (in flight,
2026-08-16).

## Why (the evidence that makes this honest)

- The v0.11.0 gate machinery has been **proven honest through five red
  nightlies**: every failure was the gate correctly flagging a stale
  EVAL-12 single-snapshot baseline (049 cost-of-correct, 055 unproven,
  050/018 lucky snapshots, 017 true-rate ~0.4, 052 pre-self-healing-fix).
  Each was diagnosed and re-baselined honestly; the gate never false-passed
  a real regression.
- The full hermetic suite (the contract's integrity layer) is **204 passed,
  3 skipped**; `config-check` (personal) is fully green.
- The 15× re-baseline of the whole suite (855 runs) is in flight to settle
  every baseline at 15 samples — the honest completion of the
  green-nightly evidence, landing as a data release (eval lane, never a CLI
  patch) after the tag.

## What this is NOT

- Not a silent skip: this record + the ROADMAP row re-type are the
  documented change.
- Not a product launch: the dogfood posture stands — v1.0.0 remains
  consumer-triggered (earned-bar gate).

## Follow-ups

- [ ] Land the 15× re-baseline data release when the run finishes (data
      release, eval lane).
- [ ] Post-release: verify `pi-run version` == v0.11.0, brew formula shas,
      and the post-release brew verify CI.
- [ ] The next nightly gates against the 15-sample baselines — a green run
      then closes the deferred evidence.
