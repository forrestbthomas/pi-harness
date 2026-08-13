---
id: "kb-20260813050430-launchenv-ships-non-interactive-env-pr-59"
type: "constraint"
status: "active"
scope: "project"
components:
  - "pi-run"
tools: []
trigger_tags:
  - "type:constraint"
  - "prevention"
related_files: []
created_at: "2026-08-13T05:04:30+00:00"
updated_at: "2026-08-13T05:04:30+00:00"
---

# launchEnv ships non-interactive env (PR #59)

## Summary

Every spawned pi process gets GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true GIT_TERMINAL_PROMPT=0 PAGER=cat — prevents git-editor/pager hang class.

## Context

PR #59; the 10-min rebase hang root cause

## Applies When

Any agent-run prevention work; debugging editor/pager hangs

## Decision Or Root Cause

Prevention at env boundary beats detection for interactive deadlock class

## Resolution Or Rule

Do not remove these vars; extend with CI=true etc. if new prompt classes appear

## Validation

TestLaunchEnvNonInteractive; pi-run config-check

## Tags

- type:constraint
- prevention

## Related Files Or Systems

- None
