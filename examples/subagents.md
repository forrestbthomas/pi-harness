# Subagent-Driven Development Example

The harness is subagent-capable via pi-subagents. Here's a minimal
scout → worker → reviewer loop using the harness's own files.

## 1. Scout (recon)

"Use scout to map the provider-routing code in this repo before we change it."

The scout child returns: relevant files, entry points, data flow, risks.

## 2. Worker (implement)

"Use worker to implement this plan task: add a `--timeout` flag to
`pi-run eval` per docs/superpowers/plans/2026-08-09-v0.3.0-robustness.md Task 2."

The worker child edits files, runs tests, and reports status + commit SHA.

## 3. Reviewer (verify)

"Use reviewer to review the diff from the worker. Check spec compliance,
tests, and simplicity."

The reviewer child returns findings; you (the parent) apply fixes or re-dispatch.

## Notes

- Keep the parent as orchestrator and final decision-maker.
- One writer per worktree; use isolated worktrees for parallel workers.
- Fresh-context reviewers are independent of the implementer.
