# Bulletin Workflow — multi-agent review with pi-bulletin

> **Scope (per BACKLOG #12 MSG-1, charter-conformance decision 2026-08-23):**
> pi-bulletin coordinates **harness-launched Pi sessions** only — sessions
> the author starts for work in this repo. It is a *coordination* channel,
> never a permission/config/execution path: bulletin messages can't approve
> permissions, change config, or run commands.

pi-bulletin is a bulletin-first coordination package (blackboard pattern)
for parallel agent work: agents post findings cheaply, the lead compresses
them into a digest **once per round**, and conflicts are reconciled only
when cheap symbolic checks say they matter. It replaces the peer-mailbox
model (pi-teams / Claude Agent Teams) that thrashes: every message was a
full LLM turn injected into the recipient's context, with no shared picture.

- Package: <https://github.com/forrestbthomas/pi-bulletin> (MIT)
- Eval evidence for the design: `docs/agent-teams-coordination-research-2026-08.md`

## When to use it

Use the bulletin workflow for **parallel, read-only, exploratory work**
where multiple lenses add value and the deliverable is a synthesized
report: parallel code review, research, competing-hypothesis debugging.
Do NOT use it for sequential work, same-file edits, or tasks with many
dependencies — a single session or pi-subagents is cheaper there.

## Launch

> **Setup:** `start-team.py` is NOT part of the npm package — it ships in the
> separate pi-bulletin checkout. Clone it once (or pull the latest) and keep it
> at the path referenced below:
>
> ```bash
> git clone git@github.com:forrestbthomas/pi-bulletin.git ~/Projects/pi-bulletin
> # verify the launcher is present:
> ls ~/Projects/pi-bulletin/scripts/start-team.py
> ```

```bash
# from any project (harness worktree or elsewhere):
python3 ~/Projects/pi-bulletin/scripts/start-team.py \
  --team <name> \
  --target <path-to-review> \
  --launch "pi-run chat --provider openrouter --model openai/gpt-5.6-terra"
```

The launcher opens N tmux panes (default 5 roles), gives each role its
instruction, attaches you to watch, and captures sessions + bulletin state
into `bulletin-runs/<team>-<ts>/` after you detach (Ctrl-b d).

Default roles: `lead-synthesizer`, `bug-hunter`, `complexity-analyst`,
`security-spotter`, `devil-advocate`. Customize with `--roles a,b,c` and
`--task-file`.

## Protocol (per `skills/bulletin.md` in the package)

1. **Fan out** — each agent works its lens, posting findings:
   `bulletin_post(kind=finding, ref=<topic>, claim=<one-liner>)`. Prefer
   structured `ref`/`claim` over prose; never direct-message teammates.
2. **Observe** — `bulletin_read(since_seq=...)` at natural checkpoints.
   Cheap (file read); the cost is only the context you choose to load.
3. **Sync round (lead only, ONE LLM call per round)** —
   `bulletin_conflicts` (cheap symbolic check) → compress everything since
   the last digest into one `bulletin_sync` digest. Teammates read the
   digest as the shared picture.
4. **Reconcile** — only on conflict: one resolver (usually the lead)
   decides, records the resolution in the next digest. No debate loops.

## Guardrails

- Harness-launched sessions only; never arbitrary Pi sessions.
- Bulletin is a coordination channel: no permission grants, no config
  changes, no command execution through it.
- Read-only review targets: agents must not modify files under review.
- Budget the run: per-team caps (e.g. ≤ $8, ≤ 30 min); the launcher's
  exit traps surface a dying agent instead of silently stalling.

## First use checklist (dogfood)

- [ ] Pick a real review target (e.g. a PR's diff, a module, a backlog
      candidate like HEAL-6).
- [ ] `start-team.py --team <name> --target <path>` and let it run.
- [ ] Verify the bulletins: `bulletin_status` event/digest counts; the
      lead's report is the deliverable.
- [ ] Fill the coordination-quality paragraph: did the team's communication
      help or thrash? Feed observations back to
      <https://github.com/forrestbthomas/pi-bulletin>.
