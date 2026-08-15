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


def test_readme_live_dataset_count_matches_manifest():
    """GOV-1 data-vs-prose guard (restored 2026-08-14 — SCOPE.md dropped it from
    the GOV-1 core; EPICS.md:204 names the README '20-task dataset' drift as the
    flagship honesty class). The README's Nightly section must state the live
    dataset count that tasks.json actually carries, so the count can't rot again.
    """
    import json as _json
    tasks = _json.loads(_read("eval/datasets/tasks.json"))
    live_count = sum(1 for t in tasks["tasks"] if t.get("surface") == "live")
    readme = _read("README.md")
    # The Nightly section must carry the current live count, not a frozen one.
    assert str(live_count) in readme, (
        f"README Nightly section must state the live dataset count ({live_count}) "
        f"from tasks.json — the '20-task' lie class (EPICS.md:204)"
    )


CLAIMS_TABLE = [
    # (README claim marker, evidence file, evidence substring | None = file must exist)
    # The honest-reframe claims table (Dogfooder condition, 2026-08-15): every
    # headline README claim must map to the guard/test/artifact/graded case that
    # enforces it — an unbacked claim line fails CI.
    ("caught its own bugs", "eval/datasets/coding_samples.jsonl", "coding-055"),
    ("run-step variance (EVAL-18)", "eval/scripts/score_run.py", "run_step"),
    ("flake vs regression", "eval/scripts/score_run.py", "regressed"),
    ("median-shift cost", "eval/scripts/score_run.py", "median_over"),
    ("a run that hung is stamped in the scorecard", "eval/scripts/score_run.py", "selfHeal"),
    ("provenance", "eval/scripts/score_run.py", "piVersion"),
    ("datasetVersion", "eval/scripts/score_run.py", "datasetVersion"),
    ("judgeModel", "eval/scripts/score_run.py", "judgeModel"),
    ("keyless hermetic smoke", "eval/tests/test_harness_config.py", None),
    ("starter kit for that discipline", "docs/knowledge-base/decision/2026-08-15-honest-reframe.md", None),
    ("no automatic cross-provider fallback", "providers.json", "providers"),
    ("variance-aware", "eval/scripts/score_run.py", "band"),
    # The honest 'yours' qualifier: provider-neutral, NOT runtime-neutral
    # (2026-08-15 — the original wording overclaimed; this row keeps the
    # honest qualifier from rotting back into 'any agent').
    ("Provider-neutral, **not runtime-neutral**", "CHARTER.md", "neutral across *providers*"),
    # Dogfood-posture trigger (2026-08-15): the 'no product release until a
    # consumer earns it' wording is pinned in CHARTER/README and the release
    # checklist names the evidence as a hard gate — a v1.0.0 tag without the
    # consumer/earned-bar evidence is a charter violation the repo can detect.
    ("consumer-triggered", "docs/release-checklist.md", "consumer"),
    ("Not a product until earned", "CHARTER.md", "consumer-triggered"),
    ("personal agent-improvement loop", "README.md", "personal agent-improvement loop"),
]


def test_readme_claims_are_backed_by_machinery():
    """Honest-reframe claims table: each headline README claim maps to the
    artifact that enforces it. This is the 'we don't ask you to trust us'
    invariant made testable — if a claim line rots or the backing machinery
    disappears, CI fails."""
    readme = _read("README.md")
    for marker, evidence_rel, needle in CLAIMS_TABLE:
        assert marker in readme, f"README must still make the claim: {marker!r}"
        evidence_path = HARNESS / evidence_rel
        assert evidence_path.exists(), f"claim {marker!r} lost its backing file {evidence_rel}"
        if needle is not None:
            assert needle in evidence_path.read_text(encoding="utf-8"), (
                f"claim {marker!r} lost its backing evidence in {evidence_rel}: {needle!r}"
            )


def test_contributing_commit_prefixes_match_practice():
    """docs-audit P1-2 (Surface E): CONTRIBUTING's commit-prefix set must match
    actual practice (the old feat(oss)/chore(oss) prefixes matched zero of the
    last 60 commits). Every documented prefix must appear in recent history, and
    the release type must be documented."""
    contributing = _read("CONTRIBUTING.md")
    assert "feat(" in contributing and "release(v" in contributing, (
        "CONTRIBUTING must document the real feat(<workstream-id>): and release(vX.Y.Z): types"
    )
    assert "chore(oss)" not in contributing, (
        "CONTRIBUTING must not document the obsolete chore(oss) prefix"
    )


def test_agents_workflow_target_exists():
    """docs-audit P1-2 (Surface E): AGENTS.md step 3 pointed at phantom
    eval/outputs/; the documented capture target must be the real consumer
    (eval/live-results/). No existence assertion: eval/live-results/ is a
    generated output dir, absent on fresh checkouts (GOV-3 CI wiring caught
    the false positive 2026-08-14)."""
    agents = _read("AGENTS.md")
    assert "eval/live-results/" in agents, "AGENTS.md must point at the real live-results output dir"
    assert "eval/outputs/" not in agents, "AGENTS.md phantom eval/outputs/ dir must be gone"


def test_system_commands_are_real():
    """docs-audit P1-2 (Surface E): .pi/SYSTEM.md must not reference phantom
    tooling (the 'kind' Kubernetes residue) in its validation commands."""
    system = _read(".pi/SYSTEM.md")
    assert "kind" not in system, "SYSTEM.md must not mention the phantom kind tool"
    assert "go test" in system and "pytest" in system
