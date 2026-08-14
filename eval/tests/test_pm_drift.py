"""GOV-1 (mechanical core) — PM-layer drift guards.

The planning layer self-audits mechanically, extending the test_docs_drift.py
pattern. Catches the drift classes that actually burned this project:

- EPIC-4/EPIC-5 surviving the charter that outlaws them (epics vs backlog).
- The v0.7.0 changelog gap (tag exists, no changelog section — 2026-08-14).
- STATUS/ROADMAP staleness after shipping.

These are the invariants the roadmap ritual maintains BY HAND; GOV-1 makes
them CI failures instead. Runs keyless in the deterministic suite (git tag is
available in CI — the repo is a git checkout).
"""

import re
import subprocess
from pathlib import Path

HARNESS = Path(__file__).resolve().parents[2]


def _read(rel: str) -> str:
    path = HARNESS / rel
    assert path.exists(), f"missing {rel}"
    return path.read_text(encoding="utf-8")


def _backlog_ids_and_epics() -> dict[str, str]:
    """BACKLOG ranked rows: id -> epic column (or ''). The ranked table is the
    single source of truth; each row carries its epic in the 4th column."""
    result: dict[str, str] = {}
    for line in _read("BACKLOG.md").splitlines():
        m = re.match(r"^\|\s*(\d+)\s*\|\s*\*\*([A-Z]+-\d+)\s*—", line)
        if not m:
            continue
        fields = [f.strip() for f in line.strip("|").split("|")]
        epic = fields[3] if len(fields) > 4 else ""
        result[m.group(2)] = epic
    return result


def _epics_item_ids() -> set[str]:
    """EPICS.md item-table ids (the index)."""
    ids = set()
    for line in _read("EPICS.md").splitlines():
        m = re.match(r"^\|\s*([A-Z]+-\d+)\s*\|", line)
        if m and m.group(1) not in ("ID",):
            ids.add(m.group(1))
    return ids


def _epics_parked_ids() -> set[str]:
    """EPICS.md 'Parked / deferred' table ids (deliberately not in the queue)."""
    ids = set()
    in_parked = False
    for line in _read("EPICS.md").splitlines():
        if line.startswith("## Parked / deferred"):
            in_parked = True
            continue
        if in_parked and line.startswith("## "):
            break
        if in_parked:
            m = re.match(r"^\|\s*([A-Z]+-\d+)\s*\|?", line)
            if m:
                ids.add(m.group(1))
    return ids


def test_epics_backlog_consistency():
    """Every BACKLOG row with an epic appears in EPICS.md, and vice versa —
    with the parked/deferred escape hatch (EVAL-7/HEAL-1/HEAL-4 are parked, so
    they appear in EPICS's Parked table, not the ranked queue)."""
    backlog = _backlog_ids_and_epics()
    epics_ids = _epics_item_ids()
    parked = _epics_parked_ids()

    backlog_ids = set(backlog)
    missing_from_epics = backlog_ids - epics_ids
    assert not missing_from_epics, (
        f"BACKLOG rows missing from EPICS.md index: {sorted(missing_from_epics)}"
    )
    accounted = epics_ids - backlog_ids
    unaccounted = accounted - parked
    assert not unaccounted, (
        f"EPICS.md items missing from BACKLOG ranked queue AND not in the parked "
        f"table: {sorted(unaccounted)}"
    )


def _git_tags() -> set[str]:
    try:
        out = subprocess.run(
            ["git", "tag", "--list"], cwd=HARNESS, capture_output=True, text=True, timeout=30
        )
    except (OSError, subprocess.SubprocessError):
        return set()
    if out.returncode != 0:
        return set()
    return {t.strip() for t in out.stdout.splitlines() if t.strip()}


def _changelog_versions() -> set[str]:
    """CHANGELOG section versions, e.g. [0.10.0] -> v0.10.0 (incl. -pre tags)."""
    versions = set()
    for line in _read("CHANGELOG.md").splitlines():
        m = re.match(r"^## \[(Unreleased|(\d+\.\d+\.\d+(?:-pre)?))\]", line)
        if m and m.group(2):
            versions.add("v" + m.group(2))
    return versions


def test_every_tag_has_changelog_section():
    """GOV-1: every git tag must have a CHANGELOG section — the v0.7.0 gap
    class (tag existed, no [0.7.0] section, dropped in the 0.8.0 release-notes
    commit) must never return."""
    tags = _git_tags()
    if not tags:
        return  # not a git checkout; skip silently (CI is always one)
    changelog = _changelog_versions()
    missing = sorted(tags - changelog)
    assert not missing, (
        f"git tags missing CHANGELOG sections: {missing} — every tag needs a "
        f"[x.y.z] entry (the v0.7.0 gap class, REL-2)"
    )


def test_changelog_sections_have_tags():
    """GOV-1 (inverse): released changelog sections must have a matching tag —
    a changelog entry for a version that was never tagged is a ledger lie."""
    tags = _git_tags()
    if not tags:
        return
    changelog = _changelog_versions()
    phantom = sorted(changelog - tags)
    assert not phantom, (
        f"CHANGELOG sections with no matching git tag: {phantom}"
    )


def test_status_now_refers_to_shipped_work():
    """GOV-1: STATUS 'Now' must not claim a workstream is committed while it's
    still 'In design' — the exact stale-status class that recurred before
    test_docs_drift existed."""
    status = _read("STATUS.md")
    roadmap = _read("ROADMAP.md")
    assert "SHIPPED" in status, "STATUS Now must mark shipped workstreams"
    assert "In design (spec in progress)" not in status, (
        "STATUS must not show a workstream as committed while it is in design"
    )
    # Sanity: the two docs share workstream vocabulary (W5/W6... present).
    for w in ("W5", "W6", "W7", "W8", "W9", "W10"):
        assert w in roadmap, f"ROADMAP missing {w} row"
