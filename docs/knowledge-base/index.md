# Knowledge Base Index

| Entry | Type | Trigger Tags | Summary |
| --- | --- | --- | --- |
| [launchEnv ships non-interactive env (PR #59)](./constraint/launchenv-ships-non-interactive-env-pr-59.md) | constraint | type:constraint, prevention | Every spawned pi process gets GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true GIT_TERMINAL_PROMPT=0 PAGER=cat — prevents git-editor/pager hang class. |
| [Continuous self-evaluation of harness (parked idea)](./decision/continuous-self-evaluation-of-harness-parked-idea.md) | decision | type:decision, harness-health | User wants harness self-evaluation along axes beyond self-healing: audit skills present in a harness/project for compatibility; offer reconcile option to user. |
| [Self-healing W1 design decisions](./decision/self-healing-w1-design-decisions.md) | decision | type:decision, self-healing | Watchdog in pi-run (zero upstream): process-group kill + output-stall detection + git-state recovery + escalation packet. Per-tool timeout deferred to upstream pi-subagents #150 — *superseded by W5: the real trace is #978→#979 + #1076/#1077 (merged `a660ea3`, Part B).* |
| [Persona debate — scope, north star, keep/cut/split](./decision/2026-08-14-persona-debate-scope.md) | decision | type:decision, charter, scope | Six-persona debate (2026-08-14) adjusted the north star, defined CHARTER, and produced the keep/cut/split change-set that drove #98–#112. |
