"""Deterministic harness-configuration checks (no API keys, no network).

The CLI (`pi-run config-check`) is the single source of truth for these checks;
this file shells out to it and adds a few direct assertions that are cheapest
to check here (symlink, dotfiles, Makefile absence, module path).
"""

import json
import re
import subprocess
from pathlib import Path

HARNESS = Path(__file__).resolve().parents[2]
PROJECT_SETTINGS = HARNESS / ".pi" / "settings.json"
HOME = Path.home()
MIN_SUPERPOWERS_SKILLS = 14


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


def test_pi_run_binary_symlinked_into_home_bin():
    link = HOME / "bin" / "pi-run"
    assert link.is_symlink(), f"missing symlink: {link}"
    target = link.resolve()
    assert target == (HARNESS / "bin" / "pi-run").resolve(), (
        f"symlink {link} -> {target}, want {HARNESS / 'bin' / 'pi-run'}"
    )


def test_makefile_removed():
    assert not (HARNESS / "Makefile").exists(), "Makefile must be removed (CLI owns all targets)"


def test_dotfiles_no_longer_define_pi_harness_functions():
    for rc in (HOME / ".zshrc", HOME / ".bashrc"):
        text = rc.read_text(encoding="utf-8")
        assert "pi-harness()" not in text, f"{rc.name} still defines pi-harness()"
        assert "bw_get" in text, f"{rc.name} should still resolve keys via bw_get"


def test_go_module_path():
    go_mod = (HARNESS / "go.mod").read_text(encoding="utf-8")
    assert "module github.com/forrestthomas1/pi-harness" in go_mod


def test_project_defaults_unchanged():
    settings = _load_json(PROJECT_SETTINGS)
    assert settings["defaultProvider"] == "openai"
    assert settings["defaultModel"] == "openai/gpt-5.6-terra"


def test_no_literal_keys_anywhere():
    for path in (PROJECT_SETTINGS, HOME / ".pi" / "agent" / "settings.json"):
        text = path.read_text(encoding="utf-8")
        assert re.search(r"sk-[A-Za-z0-9_-]{8,}", text) is None, path


def test_superpowers_skills_installed():
    skills = HOME / ".agents" / "skills"
    assert skills.is_dir()
    count = sum(1 for p in skills.iterdir() if p.is_dir())
    assert count >= MIN_SUPERPOWERS_SKILLS, f"found {count} skills"
