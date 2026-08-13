---
id: "kb-20260813050559-continuous-self-evaluation-of-harness-parked-idea"
type: "decision"
status: "active"
scope: "project"
components:
  - "pi-run"
tools: []
trigger_tags:
  - "type:decision"
  - "harness-health"
related_files: []
created_at: "2026-08-13T05:05:59+00:00"
updated_at: "2026-08-13T05:05:59+00:00"
---

# Continuous self-evaluation of harness (parked idea)

## Summary

User wants harness self-evaluation along axes beyond self-healing: audit skills present in a harness/project for compatibility; offer reconcile option to user.

## Context

Sidenote 2026-08-13; parked, not to build now. Example: skill collision audit (duplicate names across ~/.agents/skills, ~/.pi/agent/skills, project paths), compatible/incompatible classification, user chooses reconciliation.

## Applies When

Future harness-health work; extending doctor/config-check; new pi-run self-eval command

## Decision Or Root Cause

Separate workstream from self-healing W1; user-driven reconcile (option offered), never automatic mutation

## Resolution Or Rule

Record and revisit; natural home is pi-run doctor/config-check or a new self-eval command; reuse duplicate-skill collision detection from 2026-08-13 PM-skill install

## Validation

TBD when unparked

## Tags

- type:decision
- harness-health

## Related Files Or Systems

- None
