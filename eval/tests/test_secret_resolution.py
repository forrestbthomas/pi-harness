"""Unit tests for Bitwarden-aware key resolution (env var -> bw_get fallback)."""

import pytest
from conftest import SUPPORTED_PROVIDER_KEYS, get_secret, has_api_key


def _fake_bw_get(tmp_path, script: str) -> str:
    """Write a fake bw_get executable returning/failing per `script`."""
    path = tmp_path / "bw_get"
    path.write_text(script, encoding="utf-8")
    path.chmod(0o755)
    return str(path)


def test_get_secret_prefers_env_var(tmp_path, monkeypatch):
    monkeypatch.setenv("OPENROUTER_API_KEY", "sk-or-v1-env-value")
    # bw_get would fail if called; env must win.
    monkeypatch.setenv("BW_GET", _fake_bw_get(tmp_path, "#!/bin/sh\nexit 3\n"))
    assert get_secret("OPENROUTER_API_KEY") == "sk-or-v1-env-value"


def test_get_secret_falls_back_to_bw_get(tmp_path, monkeypatch):
    monkeypatch.delenv("OPENROUTER_API_KEY", raising=False)
    monkeypatch.setenv(
        "BW_GET",
        _fake_bw_get(tmp_path, "#!/bin/sh\nprintf '%s\\n' 'sk-or-v1-vault-value'\n"),
    )
    assert get_secret("OPENROUTER_API_KEY") == "sk-or-v1-vault-value"


def test_get_secret_returns_none_when_bw_get_fails(tmp_path, monkeypatch):
    monkeypatch.delenv("DEEPSEEK_API_KEY", raising=False)
    monkeypatch.setenv("BW_GET", _fake_bw_get(tmp_path, "#!/bin/sh\nexit 3\n"))
    assert get_secret("DEEPSEEK_API_KEY") is None


def test_get_secret_returns_none_when_bw_get_missing(tmp_path, monkeypatch):
    monkeypatch.delenv("KIMI_API_KEY", raising=False)
    monkeypatch.setenv("BW_GET", str(tmp_path / "does-not-exist"))
    assert get_secret("KIMI_API_KEY") is None


def test_has_api_key_true_when_openrouter_env_set(monkeypatch):
    monkeypatch.setenv("OPENROUTER_API_KEY", "sk-or-v1-x")
    assert has_api_key() is True


def test_has_api_key_false_when_nothing_available(tmp_path, monkeypatch):
    for key in SUPPORTED_PROVIDER_KEYS:
        monkeypatch.delenv(key, raising=False)
    monkeypatch.setenv("BW_GET", _fake_bw_get(tmp_path, "#!/bin/sh\nexit 3\n"))
    assert has_api_key() is False
