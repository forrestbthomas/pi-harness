"""Unit tests for Bitwarden-aware key resolution (env var -> bw_get fallback)."""

import os

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


def test_has_api_key_is_env_only(tmp_path, monkeypatch):
    """has_api_key must not invoke the secret backend during collection."""
    for key in SUPPORTED_PROVIDER_KEYS:
        monkeypatch.delenv(key, raising=False)
    # This fake returns a value if invoked; env-only key detection must ignore it.
    monkeypatch.setenv(
        "BW_GET",
        _fake_bw_get(tmp_path, "#!/bin/sh\nprintf '%s\\n' 'sk-should-not-be-called'\n"),
    )
    monkeypatch.setenv("PI_SECRET_BACKEND", "bitwarden")
    assert has_api_key() is False


def test_get_secret_env_first(monkeypatch):
    """Env var wins regardless of backend."""
    monkeypatch.setenv("OPENAI_API_KEY", "sk-env-value")
    assert get_secret("OPENAI_API_KEY") == "sk-env-value"


def test_get_secret_one_password(monkeypatch, tmp_path):
    """1password backend: op read resolves the secret."""
    op_bin = tmp_path / "op"
    op_bin.write_text("#!/bin/sh\nprintf '%s\\n' 'sk-op-value'\n")
    op_bin.chmod(0o755)
    monkeypatch.setenv("PATH", str(tmp_path) + os.pathsep + os.environ.get("PATH", ""))
    monkeypatch.setenv("PI_SECRET_BACKEND", "1password")
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    # Fake BW_GET that would fail if called (hermetic: never touch a real vault).
    monkeypatch.setenv("BW_GET", _fake_bw_get(tmp_path, "#!/bin/sh\nexit 3\n"))
    assert get_secret("OPENAI_API_KEY") == "sk-op-value"


def test_get_secret_env_only_no_fallback(monkeypatch, tmp_path):
    """env-only backend: no fallback, returns None when env var unset."""
    monkeypatch.setenv("PI_SECRET_BACKEND", "env-only")
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    # Fake BW_GET that would succeed if called (proves env-only never falls back).
    monkeypatch.setenv("BW_GET", _fake_bw_get(tmp_path, "#!/bin/sh\nprintf '%s\\n' 'sk-vault-value'\n"))
    assert get_secret("OPENAI_API_KEY") is None



def test_any_provider_key_env_includes_local(monkeypatch):
    for key in SUPPORTED_PROVIDER_KEYS:
        monkeypatch.delenv(key, raising=False)
    monkeypatch.setenv("LOCAL_API_KEY", "test-key")
    from conftest import any_provider_key_env

    assert any_provider_key_env() is True
