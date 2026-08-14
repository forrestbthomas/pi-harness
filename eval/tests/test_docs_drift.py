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
    assert "v0.10.x" in security and "✅ Active" in security


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


def test_scorecard_surfaces_record_pi_version():
    """Both eval surfaces must record pi-run version for scorecard provenance
    (EVAL-3 nightly + EVAL-14 ci-benchmark) — otherwise piVersion is 'unknown'
    and the 'every scorecard is attributable' DoD silently regresses."""
    nightly = _read(".github/workflows/nightly-live-eval.yml")
    assert "PI_VERSION=$(pi-run version" in nightly, "nightly must record PI_VERSION"
    provider = _read(".github/workflows/provider-scorecard.yml")
    assert "PI_VERSION=$(pi-run version" in provider, "provider-scorecard must record PI_VERSION (EVAL-14)"


def test_node_pin_matches_doctor_reference():
    """REL-4: the CI Node pin (PI_NODE_VERSION in the nightly) must stay in
    sync with the doctor drift-guard reference — a silent CI pin bump that
    didn't update the doctor parity check is exactly the drift class this
    guard exists to catch."""
    nightly = _read(".github/workflows/nightly-live-eval.yml")
    doctor = _read("internal/cli/doctor.go")
    assert "PI_NODE_VERSION: 'v22.19.0'" in nightly, "nightly must pin Node 22.19.0"
    assert "v22." in doctor, "doctor drift guard must reference the Node 22 LTS line"


def test_readme_env_table_documents_watchdog_env_vars():
    """TAX-2: the watchdog env vars (PI_SELF_HEAL / PI_STALL_TIMEOUT_SECS /
    PI_WATCHDOG_GRACE_SECS) are real, tested Go knobs — they must appear in the
    README env table, not just prose (they silently were not, 2026-08-14)."""
    readme = _read("README.md")
    for var in ("PI_SELF_HEAL", "PI_STALL_TIMEOUT_SECS", "PI_WATCHDOG_GRACE_SECS"):
        assert f"| `{var}` |" in readme, f"README env table must document {var}"
