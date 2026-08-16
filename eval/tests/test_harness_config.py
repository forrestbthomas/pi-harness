"""Deterministic harness-configuration checks (no API keys, no network).

The CLI (`pi-run config-check`) is the single source of truth for these checks.
Personal-machine checks (symlink into ~/bin, dotfile contents, installed skill
counts) are gated behind PI_RUN_PERSONAL=1 so the suite passes on a fresh clone.
"""

import json
import os
import re
import subprocess
from pathlib import Path

import pytest

HARNESS = Path(__file__).resolve().parents[2]
PROJECT_SETTINGS = HARNESS / ".pi" / "settings.json"
HOME = Path.home()
PERSONAL = os.environ.get("PI_RUN_PERSONAL") == "1"
MIN_SUPERPOWERS_SKILLS = 13  # matches the current superpowers collection (13 skills, 2026-08-15)


def _load_json(path: Path) -> dict:
    assert path.exists(), f"missing expected config file: {path}"
    return json.loads(path.read_text(encoding="utf-8"))


def test_pi_run_config_check_passes():
    result = subprocess.run(
        ["pi-run", "config-check"],
        cwd=HARNESS,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, f"pi-run config-check failed:\n{result.stdout}\n{result.stderr}"


def test_makefile_removed():
    assert not (HARNESS / "Makefile").exists(), "Makefile must be removed (CLI owns all targets)"


def test_go_module_path():
    go_mod = (HARNESS / "go.mod").read_text(encoding="utf-8")
    # OSS-1 (canonical install/identity): the Go module path must match the
    # canonical repo URL so `go install github.com/forrestbthomas/pi-harness@latest`
    # works for a stranger. Historical docs may keep the old path, but go.mod
    # is the install contract.
    assert "module github.com/forrestbthomas/pi-harness" in go_mod


def test_canonical_identity():
    """OSS-1: the documented repo identity must be one spelling everywhere a
    stranger sees it — go.mod module path, README clone URL, CODEOWNERS handle.
    The old drift (go.mod `forrestthomas1` vs remote `forrestbthomas` vs
    CODEOWNERS `@forrestthomas`) broke `go install` against the README URL.
    """
    go_mod = (HARNESS / "go.mod").read_text(encoding="utf-8")
    readme = (HARNESS / "README.md").read_text(encoding="utf-8")
    codeowners = (HARNESS / ".github" / "CODEOWNERS").read_text(encoding="utf-8")
    canonical = "github.com/forrestbthomas/pi-harness"
    assert "module " + canonical in go_mod, "go.mod module path must match canonical repo URL"
    assert "git clone https://" + canonical + ".git" in readme, (
        "README clone URL must match canonical repo URL"
    )
    assert "@forrestbthomas" in codeowners, "CODEOWNERS must use the canonical handle"
    assert "forrestthomas1/" not in go_mod and "forrestthomas1/" not in readme, (
        "no stale forrestthomas1 module path in current-state install docs"
    )


def test_project_defaults_unchanged():
    settings = _load_json(PROJECT_SETTINGS)
    assert settings["defaultProvider"] == "openai"
    assert settings["defaultModel"] == "openai/gpt-5.6-terra"


def test_pi_subagents_pinned_exact_version():
    """pi-subagents must be pinned to an exact version so a settings-triggered
    npm refresh cannot silently drift the subagent tooling (BACKLOG #2)."""
    settings = _load_json(PROJECT_SETTINGS)
    packages = settings["packages"]
    specs = [p for p in packages if p.startswith("npm:pi-subagents")]
    assert len(specs) == 1, f"expected exactly one pi-subagents package spec, got {specs}"
    spec = specs[0]
    rest = spec.removeprefix("npm:pi-subagents")
    assert rest.startswith("@"), f"pi-subagents must be pinned with @version, got {spec!r}"
    version = rest[1:]
    assert version and version[0].isdigit(), f"pin must be an exact semver, got {spec!r}"
    assert "^" not in version and "~" not in version, f"pin must be exact (no ^/~), got {spec!r}"


def test_no_literal_keys_anywhere():
    for path in (PROJECT_SETTINGS,):
        text = path.read_text(encoding="utf-8")
        assert re.search(r"sk-[A-Za-z0-9_-]{8,}", text) is None, path


def test_no_hardcoded_user_paths_in_shipped_code():
    for path in (
        HARNESS / "scripts" / "install-skills.sh",
        HARNESS / ".pi" / "settings.json",
    ):
        text = path.read_text(encoding="utf-8")
        assert "/Users/forrestthomas/" not in text, path


_PI_INSTALL_SITES = (
    HARNESS / ".github" / "workflows" / "nightly-live-eval.yml",
    HARNESS / ".github" / "workflows" / "provider-scorecard.yml",
    HARNESS / "scripts" / "bootstrap.sh",
)


def test_pi_installs_use_scoped_coding_agent_package():
    """The Pi coding agent is installed from its scoped npm package.

    The unscoped `pi` npm package is "pi-number" (IonicaBizau's pi-digits
    calculator): its `pi` binary prints digits of pi and exits 0, so a
    harness that installs it silently neuters every agent run — each run
    "succeeds" in ~50ms with a non-empty "3", zero spend, and failed
    grading (nightly live-eval run 31652076075: costUsd 0 across the board
    while judge cost was real). Every install site must use the scoped
    package name (@earendil-works/pi-coding-agent), which ships the real
    coding agent CLI as `pi`.
    """
    for path in _PI_INSTALL_SITES:
        lines = [ln.strip() for ln in path.read_text(encoding="utf-8").splitlines()]
        assert "npm install -g pi" not in lines, (
            f"{path} installs the unscoped `pi` package (pi-number) instead of "
            "the coding agent — every agent run would print pi digits and "
            "record zero spend"
        )
        assert any(
            "@earendil-works/pi-coding-agent" in ln for ln in lines
        ), f"{path} must install the scoped @earendil-works/pi-coding-agent package"


@pytest.mark.skipif(not PERSONAL, reason="personal-machine check (set PI_RUN_PERSONAL=1)")
def test_personal_pi_run_binary_symlinked_into_home_bin():
    link = HOME / "bin" / "pi-run"
    assert link.is_symlink(), f"missing symlink: {link}"
    target = link.resolve()
    assert target == (HARNESS / "bin" / "pi-run").resolve(), (
        f"symlink {link} -> {target}, want {HARNESS / 'bin' / 'pi-run'}"
    )


@pytest.mark.skipif(not PERSONAL, reason="personal-machine check (set PI_RUN_PERSONAL=1)")
def test_personal_dotfiles_no_longer_define_pi_harness_functions():
    for rc in (HOME / ".zshrc", HOME / ".bashrc"):
        text = rc.read_text(encoding="utf-8")
        assert "pi-harness()" not in text, f"{rc.name} still defines pi-harness()"
        assert "bw_get" in text, f"{rc.name} should still resolve keys via bw_get"


@pytest.mark.skipif(not PERSONAL, reason="personal-machine check (set PI_RUN_PERSONAL=1)")
def test_personal_superpowers_skills_installed():
    # The registered durable location (scripts/install-skills.sh clones into
    # ~/.pi/agent/skills/superpowers). ~/.agents/skills is the user's PERSONAL
    # skill dir (game-design, context-engine, ...) — not the superpowers
    # collection; a pre-fix version of this test checked the wrong path.
    skills = HOME / ".pi" / "agent" / "skills" / "superpowers" / "skills"
    assert skills.is_dir()
    count = sum(1 for p in skills.iterdir() if p.is_dir())
    assert count >= MIN_SUPERPOWERS_SKILLS, f"found {count} skills"
