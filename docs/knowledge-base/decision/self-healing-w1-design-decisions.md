---
id: "kb-20260813050430-self-healing-w1-design-decisions"
type: "decision"
status: "active"
scope: "project"
components:
  - "pi-run"
tools: []
trigger_tags:
  - "type:decision"
  - "self-healing"
related_files: []
created_at: "2026-08-13T05:04:30+00:00"
updated_at: "2026-08-13T05:04:30+00:00"
---

# Self-healing W1 design decisions

## Summary

Watchdog in pi-run (zero upstream): process-group kill + output-stall detection + git-state recovery + escalation packet. Per-tool timeout deferred to upstream pi-subagents #150.

## Context

Research synthesis docs/self-healing-research-2026-08.md; spec docs/governance/specs-archive/2026-08-13-self-healing-design.md; incident: git rebase --continue spawned vi, hung 10min silently

## Applies When

Implementing W1 self-healing; any agent-run hang/stall work

## Decision Or Root Cause

Watchdog belongs in pi-run CLI (we control spawn); no upstream pi changes needed for gaps 2-5; #150 is the only piece that truly needs upstream

## Resolution Or Rule

Process-group kill via Setpgid+SIGTERM/SIGKILL; stall detection only for non-interactive spawns; git recovery only when ls-files -u empty; never auto-abort conflicted rebase; bounded 1 auto-recovery

## Validation

go test + hermetic pytest; PI_RUN_BIN fresh build

## Tags

- type:decision
- self-healing

## Related Files Or Systems

- None
