# Knowledge Base Index

| Entry | Type | Trigger Tags | Summary |
| --- | --- | --- | --- |
| [launchEnv ships non-interactive env (PR #59)](./constraint/launchenv-ships-non-interactive-env-pr-59.md) | constraint | type:constraint, prevention | Every spawned pi process gets GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true GIT_TERMINAL_PROMPT=0 PAGER=cat — prevents git-editor/pager hang class. |
| [Self-healing W1 design decisions](./decision/self-healing-w1-design-decisions.md) | decision | type:decision, self-healing | Watchdog in pi-run (zero upstream): process-group kill + output-stall detection + git-state recovery + escalation packet. Per-tool timeout deferred to upstream pi-subagents #150. |
