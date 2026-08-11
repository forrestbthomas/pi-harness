# Subagent-Driven Development Example

The harness is subagent-capable via pi-subagents. Here's a minimal
scout → worker → reviewer loop using the harness's own files.

> **Timeout rule (required).** Always pass an explicit `timeoutMs` on every
> subagent launch. Async subagent runs have **no default timeout**, and a
> child `bash` tool call can hang forever if a command starts a background
> process that inherits the terminal (no default timeout in the bash tool
> either). Project agent wrappers in `.pi/agents/` default to 10 minutes, but
> explicit values are preferred and make the intent visible.

## 1. Scout (recon)

"Use scout to map the provider-routing code in this repo before we change it."

```js
subagent({ workflowScript: `return runs.run("recon", { agent: "scout", task: "Map provider routing code in this repo: files, entry points, data flow, risks.", timeoutMs: 600000 })` })
```

The scout child returns: relevant files, entry points, data flow, risks.

## 2. Worker (implement)

"Use worker to implement this plan task: add a `--timeout` flag to
`pi-run eval` per docs/superpowers/plans/2026-08-09-v0.3.0-robustness.md Task 2."

```js
subagent({ workflowScript: `return runs.run("implement", { agent: "worker", task: "Implement the --timeout flag for pi-run eval per the plan. Make edits, run go build/test/vet, report status.", timeoutMs: 900000 })` })
```

The worker child edits files, runs tests, and reports status + commit SHA.

## 3. Reviewer (verify)

"Use reviewer to review the diff from the worker. Check spec compliance,
tests, and simplicity."

```js
subagent({ workflowScript: `return runs.run("review", { agent: "reviewer", task: "Review the diff from the worker for spec compliance, tests, and simplicity.", timeoutMs: 600000 })` })
```

The reviewer child returns findings; you (the parent) apply fixes or re-dispatch.

## Notes

- Keep the parent as orchestrator and final decision-maker.
- One writer per worktree; use isolated worktrees for parallel workers.
- Fresh-context reviewers are independent of the implementer.
- Children (via `.pi/agents/` wrappers) are instructed to pass `timeout:` on
  every `bash` call and never start long-running/background commands that
  inherit the terminal — this is the second layer of defense against hangs.
