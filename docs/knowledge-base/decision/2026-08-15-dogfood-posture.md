# Decision — Dogfood posture: personal agent-improvement loop, public as demo (2026-08-15)

**Status:** Decided (short 7-persona debate) → adopted as the project's
positioning, replacing the product-shape framing.

## Decision

This project is the **author's personal agent-improvement loop, built in
public as a demo of the discipline** — measure → diagnose → improve →
re-measure, with receipts. **No product release until earned**: v1.0.0 (or
any product release) is **consumer-triggered** — it ships only when someone
outside the author wants the discipline for their own agent setup (the
recorded earned-bar gate). Until then, releases are the loop's integrity
check, not product launches.

## Why (the trigger)

After establishing there is no moat and no new science, and discovering Pi's
ecosystem already ships `pi-bench` (a benchmark runner for the Pi agent), the
author concluded: "This is just for me, and I'm finding it useful. Keep it
public, don't ship anything until someone actually wants this."

## The debate (short, `.pi/debate/dogfood-posture/`)

Round-1 verdicts: **ADOPT unanimously (7/7)** — the most honest statement the
project has made. Converged conditions, all adopted:

1. The loop is the product; today its only customer is the author, and it
   demonstrably works (coding-055 0.2 → 0.8).
2. The discipline binds HARDER for an audience of one — drift guards, claims
   table, nightly, provenance stay enforced (a loop that stops measuring
   itself is a hobby script).
3. Gates re-mean, not remove: v0.11.0 = the loop's integrity check (same
   threshold); v1.0.0 = consumer-triggered, never a date.
4. DoD redefined: "the loop closed with receipts" replaces "ship-ready for
   strangers."
5. The consumer trigger is machine-checked (`test_docs_drift.py` pins the
   wording; `docs/release-checklist.md` names the evidence as a hard gate) —
   a v1.0.0 tag without the trigger is a charter violation the repo can
   detect.
6. Wording discipline: no reintroduction of absolute "can't lie" claims.
7. Door stays open: public "so the discipline is visible, improvable, and a
   stranger can adopt it for their own setup" — as a demo, not a promise of
   support.

## What does NOT change

- Gate thresholds; in-flight work (EVAL-17, EVAL-16 enforcement, EPIC-6);
  the runtime/curriculum welds (provider-neutral, not runtime-neutral); the
  public demo (scorecard, claims table, receipts).

## Re-open triggers

A stranger asks to adopt the loop for their own setup (the consumer trigger);
or the discipline visibly erodes (skipped gates, unguarded claims).
