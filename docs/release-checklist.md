# Release checklist — v0.11.0 "The gate that can't lie"

Runbook for cutting v0.11.0. The gate is **evidence, not dates**: the tag is
cut only after the release gate closes. All steps are scripted/checked; the
ritual's hard-fails are there to prevent the v0.9.1/v0.9.2 class of mistake
(tagging a local commit that squash-merging later rewrote).

Milestone contract (ROADMAP.md): **Two consecutive green nightlies on the live
suite** (`tasks.json` is the count authority) + weekly provider scorecard
green with provenance fields.

---

## 0. Gate evidence (must exist BEFORE any release work)

- [ ] **Two consecutive green nightlies** on `main`. Evidence: the two
      workflow runs' `live` job conclusion == `success`, and each `live-results`
      report gates `PASS` under the current gate (EVAL-18 run-step band +
      EVAL-13 median cost). Record run IDs here:
      1. `________________` (green)
      2. `________________` (green)
- [ ] **Weekly provider scorecard green** with provenance fields (last
      `provider-scorecard.yml` run). Record: `________________`
- [ ] No open release-blocking PRs (EVAL-16 enforcement, EPIC-1/6 DoD items
      are v1.0.0 gates, not v0.11.0 gates).

## 1. Pre-flight (main)

- [ ] `git fetch github main` — the tag must point at fetched remote main tip.
- [ ] Working tree clean on `main`; no open PRs that change the CLI/gate.
- [ ] `go build ./... && go vet ./... && go test ./...` green locally.
- [ ] `eval/.venv/bin/python -m pytest eval/tests -q` green locally
      (incl. drift guards `test_docs_drift` / `test_pm_drift`).
- [ ] `pi-run config-check` passes (with `PI_RUN_PERSONAL=1` for the skills
      check).
- [ ] README scorecard numbers refreshed to the latest green nightly
      (invariants to preserve: "55" in the Nightly/measurement section;
      provider count line; exit-codes/watchdog/`9` on one line; env-table
      rows for `PI_SELF_HEAL`, `PI_STALL_TIMEOUT_SECS`, `PI_WATCHDOG_GRACE_SECS`).

## 2. Changelog flip (PR first — the ritual's first step)

- [ ] Move `## [Unreleased]` → `## [v0.11.0] - 2026-08-XX` (Keep a Changelog
      format), dated with the release day.
- [ ] Audit the section: every shipped PR since v0.10.0 appears once, in the
      right category (Added / Fixed / Changed); `BREAKING:` banner on any
      user-visible or measurement-contract delta (commands, flags, exit codes,
      install path, gate semantics, scorecard schema, runtime-changing pin
      bumps — versioning policy in CONTRIBUTING, enforced by GOV-1).
- [ ] If a data release (re-baseline, `datasetVersion`) rides along, note it
      — data releases ride the eval lane, never CLI patches.
- [ ] PR the changelog flip; wait for CI + merge to `main` **before** tagging.

## 3. Tag

- [ ] Run `scripts/tag-release.sh v0.11.0` (tags fetched remote `main` tip;
      refuses to re-tag; pushes the tag). It uses `github` as the default
      remote.
- [ ] Confirm `release.yml` ran and the **ancestor hard-fail** passed (it
      refuses to release from a tag that isn't an ancestor of `main`).
- [ ] Confirm the released binary's `pi-run version` == `v0.11.0` (the
      REL-1 ldflags stamp — never `dev`).

## 4. Formula + distribution (macOS/Homebrew — the shipped leg)

- [ ] Run `scripts/update-homebrew-formula.sh v0.11.0` and open the formula
      PR in the homebrew tap; confirm the 4 shas match the released artifact.
- [ ] Post-release brew verify CI job green (cold-tap install yields
      `pi-run version` == v0.11.0).

## 5. Post-release

- [ ] Node-drift guard + `doctor` warning verified on the released binary.
- [ ] Provider scorecard re-run green with provenance fields (`agentModel`,
      `judgeModel` not `unknown`).
- [ ] STATUS.md: move v0.11.0 to "Shipped recently"; add the tag line to the
      CHANGELOG index; prune stale remote branches per `docs/roadmap-workflow.md`.
- [ ] ROADMAP: mark v0.11.0 shipped; next gate = v1.0.0 ("The contract
      release", ≥14 consecutive green nightlies, EVAL-16 enforced, …).

## 6. Rollback notes

- No git-history rewrite. A bad tag is deleted (`git push github :v0.11.0`
  + local `git tag -d`), the fix lands on `main`, and the next tag is cut
  from the corrected tip. Formula SHAs are updated in the tap, never edited
  in place with drift.
