"""Deterministic harness-configuration checks (no API keys, no network).

Validates that the Pi harness is wired OpenRouter-first with OpenAI GPT models
preferred by default, skills are installed, and the shell aliases exist.

These tests only read local config files; they do not require any provider
credentials or network access, so they are safe to run in CI or locally via
`make pi-config-check`.
"""

import json
import re
from pathlib import Path

HARNESS = Path(__file__).resolve().parents[2]
PROJECT_SETTINGS = HARNESS / ".pi" / "settings.json"
GLOBAL_SETTINGS = Path.home() / ".pi" / "agent" / "settings.json"
AGENT_SKILLS = Path.home() / ".agents" / "skills"
OSMANI_SKILLS = Path.home() / "Projects" / "tmp" / "agent-skills" / "skills"
ZSH_RC = Path.home() / ".zshrc"
BASH_RC = Path.home() / ".bashrc"

EXPECTED_DEFAULT_PROVIDER = "openrouter"
EXPECTED_DEFAULT_MODEL = "openai/gpt-4o"
EXPECTED_PREFIX_PATTERNS = [
    "openai/*",
    "anthropic/*",
    "google/*",
    "meta-llama/*",
    "openrouter/*",
]
EXPECTED_GPT_PATTERNS = ["gpt-4o*", "gpt-4.1*", "gpt-5*"]
MIN_SUPERPOWERS_SKILLS = 14


def _load_json(path: Path) -> dict:
    assert path.exists(), f"missing expected config file: {path}"
    return json.loads(path.read_text(encoding="utf-8"))


def _has_literal_secret(text: str) -> bool:
    # Looks for API-key shaped strings; used to guard against committed keys.
    return re.search(r"sk-[A-Za-z0-9_-]{8,}", text) is not None


# --- project settings -------------------------------------------------------

def test_project_default_provider_is_openrouter():
    settings = _load_json(PROJECT_SETTINGS)
    assert settings["defaultProvider"] == EXPECTED_DEFAULT_PROVIDER


def test_project_default_model_is_openai_gpt():
    settings = _load_json(PROJECT_SETTINGS)
    assert settings["defaultModel"] == EXPECTED_DEFAULT_MODEL


def test_project_enabled_models_prefer_openai():
    settings = _load_json(PROJECT_SETTINGS)
    enabled = settings["enabledModels"]
    for pattern in EXPECTED_PREFIX_PATTERNS + EXPECTED_GPT_PATTERNS:
        assert pattern in enabled, f"missing enabled model pattern: {pattern}"
    # OpenAI GPT models are preferred: the openai/* pattern must come first.
    assert enabled.index("openai/*") < enabled.index("openrouter/*")


def test_project_skills_wire_osmani_skills():
    settings = _load_json(PROJECT_SETTINGS)
    skills = settings.get("skills", [])
    assert skills, "project settings skills array must not be empty"
    assert any("agent-skills" in s for s in skills), (
        "expected an agent-skills path in project settings skills"
    )


def test_project_settings_have_no_literal_keys():
    text = PROJECT_SETTINGS.read_text(encoding="utf-8")
    assert not _has_literal_secret(text), "literal API key found in project settings"


# --- global settings --------------------------------------------------------

def test_global_default_provider_and_model():
    settings = _load_json(GLOBAL_SETTINGS)
    assert settings["defaultProvider"] == EXPECTED_DEFAULT_PROVIDER
    assert settings["defaultModel"] == EXPECTED_DEFAULT_MODEL


def test_no_broken_moonshot_ai_provider_string():
    for path in (PROJECT_SETTINGS, GLOBAL_SETTINGS):
        text = path.read_text(encoding="utf-8")
        assert "moonshot-ai" not in text, f"broken provider key 'moonshot-ai' in {path}"


# --- skills -----------------------------------------------------------------

def test_superpowers_skills_installed():
    assert AGENT_SKILLS.is_dir(), f"missing skills dir: {AGENT_SKILLS}"
    count = sum(1 for p in AGENT_SKILLS.iterdir() if p.is_dir())
    assert count >= MIN_SUPERPOWERS_SKILLS, (
        f"expected >= {MIN_SUPERPOWERS_SKILLS} superpowers skills, found {count}"
    )


def test_osmani_agent_skills_clone_exists():
    assert OSMANI_SKILLS.is_dir(), (
        f"missing Addy Osmani agent-skills clone: {OSMANI_SKILLS}"
    )


# --- shell aliases ----------------------------------------------------------

def test_dotfiles_have_harness_functions():
    for rc in (ZSH_RC, BASH_RC):
        text = rc.read_text(encoding="utf-8")
        assert "pi-harness" in text, f"{rc.name} missing pi-harness"
        assert "pi-harness-print" in text, f"{rc.name} missing pi-harness-print"
        assert "--provider" in text, f"{rc.name} missing --provider flag"
        assert "openai/gpt-4o" in text, f"{rc.name} missing default model"


MIGRATED_SECRET_VARS = (
    "OPENROUTER_API_KEY",
    "OPENAI_API_KEY",
    "DEEPSEEK_API_KEY",
    "KIMI_API_KEY",
)


def test_dotfiles_wire_bitwarden_without_echoing():
    # Functions must resolve keys via bw_get (Bitwarden) without printing key
    # material to stdout.
    for rc in (ZSH_RC, BASH_RC):
        text = rc.read_text(encoding="utf-8")
        assert "bw_get" in text, (
            f"{rc.name}: harness should resolve keys via bw_get (Bitwarden)"
        )
        assert 'echo "$OPENROUTER_API_KEY"' not in text, (
            f"{rc.name}: must not echo the OpenRouter key"
        )
        assert 'echo "$OPENAI_API_KEY"' not in text, (
            f"{rc.name}: must not echo the OpenAI key"
        )


def test_dotfiles_have_no_static_secret_exports():
    # Migrated keys must not be hard-coded in rc files (static values).
    for rc in (ZSH_RC, BASH_RC):
        for line in rc.read_text(encoding="utf-8").splitlines():
            if not line.startswith("export "):
                continue
            var = line.split("=", 1)[0].replace("export ", "").strip()
            if var in MIGRATED_SECRET_VARS and "bw_get" not in line:
                assert False, (
                    f"{rc.name}: {var} must come from bw_get, not a static export: {line}"
                )
