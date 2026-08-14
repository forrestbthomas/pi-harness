"""Deterministic docs-drift guards: high-signal invariants that catch the
staleness classes this project actually hit — README exit-code omissions
(missed `9`), provider-count drift (9-vs-8 parenthetical), stale version
claims (SECURITY v0.3.x), and roadmap statuses out of sync with shipped work
(W1 "In design" after shipping).

These run keyless in the deterministic suite (CI python-quick + the nightly
deterministic job). The post-merge docs-review ritual (docs/roadmap-workflow.md)
runs them after any behavior-changing merge.
"""

import json
import re
from pathlib import Path

HARNESS = Path(__file__).resolve().parents[2]


def _read(rel: str) -> str:
    path = HARNESS / rel
    assert path.exists(), f"missing {rel}"
    return path.read_text(encoding="utf-8")


def test_readme_exit_codes_include_watchdog_9():
    readme = _read("README.md")
    # Target the Command Reference table ("0 ok ... 9 watchdog"), not the
    # ci-benchmark-specific subset (6/7/8) which correctly omits 9.
    line = next(l for l in readme.splitlines() if l.startswith("Exit codes: `0` ok"))
    assert "9" in line and "watchdog" in line, f"exit-code line missing watchdog/9:\n{line}"


def test_readme_provider_count_matches_providers_json():
    readme = _read("README.md")
    providers = json.loads(_read("providers.json"))["providers"]
    assert f"{len(providers)} providers in total" in readme, (
        f"README provider count != providers.json ({len(providers)} rows)"
    )


def test_readme_source_platforms_is_not_all_platforms():
    readme = _read("README.md")
    assert "From source (macOS/Linux/WSL)" in readme
    assert "From source (all platforms)" not in readme


def test_changelog_has_unreleased_section():
    changelog = _read("CHANGELOG.md")
    assert "## [Unreleased]" in changelog, "CHANGELOG must keep an [Unreleased] section"


def test_security_supported_versions_are_current():
    security = _read("SECURITY.md")
    assert "v0.9.x" in security and "✅ Active" in security


def test_roadmap_shipped_workstreams_not_stale():
    roadmap = _read("ROADMAP.md")
    assert "W5" in roadmap, "ROADMAP missing W5 row"
    assert "SHIPPED" in roadmap, "ROADMAP must mark shipped workstreams"
    # The exact stale-status class that recurred before the workflow existed.
    assert "In design (spec in progress)" not in roadmap


def test_agents_md_has_repository_navigation():
    agents = _read("AGENTS.md")
    assert "Repository navigation" in agents, "AGENTS.md must contain the navigation map"


def test_eval_workflows_upload_on_any_outcome():
    """Upload steps must run on every gate outcome (EVAL-1), not just success."""
    for rel in (".github/workflows/nightly-live-eval.yml", ".github/workflows/provider-scorecard.yml"):
        text = _read(rel)
        assert "actions/upload-artifact" in text, f"{rel} lost its artifact upload"
        assert "if: always()" in text, f"{rel} upload step must carry if: always()"
