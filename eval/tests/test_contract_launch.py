"""Contract tests: pi-run launch wiring against a fake node/pi toolchain.

fake_launch_env builds the exact nvm layout internal/cli/pi.go expects: the
node file is only stat'ed (never executed) and the fake pi at the absolute
<binDir>/pi path receives the real argv plus the key via the environment.
Also data-driven checks of the --model-tier surface against the repo's real
providers.json (spec §4.12).
"""

import json
import subprocess
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]


def _run(pi_run_bin, env, args, cwd=None):
    return subprocess.run(
        [pi_run_bin, *args], env=env, capture_output=True, text=True, timeout=30, cwd=cwd
    )


def _args_line(fake_launch_env) -> str:
    lines = [line for line in fake_launch_env.log_text.splitlines() if line.startswith("ARGS:")]
    assert lines, f"fake pi was never launched; log:\n{fake_launch_env.log_text!r}"
    return lines[-1]


def _launch(pi_run_bin, hermetic_env, fake_launch_env, args, *, tier_env=None, fake_pi_exit=None):
    base = hermetic_env()
    if tier_env is not None:
        base = hermetic_env(PI_MODEL_TIER=tier_env)
    env = fake_launch_env.env(base, OPENAI_API_KEY="testvalue", fake_pi_exit=fake_pi_exit)
    return _run(pi_run_bin, env, args)


def test_launch_pi_argv_contract(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(pi_run_bin, hermetic_env, fake_launch_env, ["print", "hello"])
    assert result.returncode == 0, result.stderr
    assert _args_line(fake_launch_env) == (
        "ARGS:--provider openai --model openai/gpt-5.6-terra --offline -p --no-session hello"
    )


def test_launch_exit_code_passthrough(pi_run_bin, hermetic_env, fake_launch_env):
    # pi's exit code passes through pi-run unchanged.
    result = _launch(
        pi_run_bin, hermetic_env, fake_launch_env, ["print", "hello"], fake_pi_exit=7
    )
    assert result.returncode == 7


def test_launch_key_via_env_not_argv(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(pi_run_bin, hermetic_env, fake_launch_env, ["print", "hello"])
    assert result.returncode == 0, result.stderr
    log = fake_launch_env.log_text
    assert "KEY_OPENAI=testvalue" in log  # key reached pi via the environment
    assert "testvalue" not in _args_line(fake_launch_env)  # never in argv


def test_launch_model_tier_flag(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(
        pi_run_bin, hermetic_env, fake_launch_env, ["print", "--model-tier", "fast", "hello"]
    )
    assert result.returncode == 0, result.stderr
    assert "--model openai/gpt-5.4-mini" in _args_line(fake_launch_env)


def test_launch_model_tier_env(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(pi_run_bin, hermetic_env, fake_launch_env, ["print", "hello"], tier_env="cheap")
    assert result.returncode == 0, result.stderr
    assert "--model openai/gpt-5-mini" in _args_line(fake_launch_env)


def test_launch_model_flag_wins_over_tier_env(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(
        pi_run_bin,
        hermetic_env,
        fake_launch_env,
        ["print", "--model", "openai/gpt-5.6-terra", "hello"],
        tier_env="cheap",
    )
    assert result.returncode == 0, result.stderr
    args = _args_line(fake_launch_env)
    assert "--model openai/gpt-5.6-terra" in args
    assert "--model openai/gpt-5-mini" not in args


def test_launch_usage_errors_do_not_launch(pi_run_bin, hermetic_env, fake_launch_env):
    # Unknown tier is a usage error (2) before any key/node access: the fake
    # pi must never be invoked.
    result = _launch(
        pi_run_bin, hermetic_env, fake_launch_env, ["print", "--model-tier", "turbo", "x"]
    )
    assert result.returncode == 2
    assert not fake_launch_env.log.exists(), "usage error must not launch pi"


def test_launch_model_tier_and_model_conflict(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(
        pi_run_bin,
        hermetic_env,
        fake_launch_env,
        ["print", "--model-tier", "fast", "--model", "x", "hello"],
    )
    assert result.returncode == 2
    assert not fake_launch_env.log.exists()


def test_launch_model_tier_unavailable_exit2(pi_run_bin, hermetic_env, fake_launch_env):
    # deepseek only maps 'fast': 'cheap' is known but unmapped, so the error
    # lists the tiers the provider actually offers. No fallback, no launch.
    result = _launch(
        pi_run_bin,
        hermetic_env,
        fake_launch_env,
        ["print", "--provider", "deepseek", "--model-tier", "cheap", "hello"],
    )
    assert result.returncode == 2
    assert "(available: balanced, fast)" in result.stderr
    assert not fake_launch_env.log.exists()


def test_launch_resume_rejects_tier(pi_run_bin, hermetic_env, fake_launch_env):
    result = _launch(pi_run_bin, hermetic_env, fake_launch_env, ["resume", "--model-tier", "fast"])
    assert result.returncode == 2
    assert not fake_launch_env.log.exists()


def test_launch_catalog_mirror_providers_tiers(pi_run_bin, hermetic_env):
    """`pi-run providers` TIERS column mirrors the repo's real providers.json.

    Data-driven off the checkout's providers.json: HARNESS_ROOT is unset and
    cwd is the repo so the binary loads the real table. Every provider with
    modelTiers must advertise 'balanced,<sorted-tier-keys>'; providers without
    a tier map advertise 'balanced'.
    """
    env = hermetic_env(harness_root=None)
    result = _run(pi_run_bin, env, ["providers"], cwd=REPO_ROOT)
    assert result.returncode == 0, result.stderr
    rows = {}
    for line in result.stdout.splitlines():
        columns = line.split("\t")
        assert len(columns) >= 4, f"malformed providers row: {line!r}"
        rows[columns[0]] = columns
    catalog = json.loads((REPO_ROOT / "providers.json").read_text(encoding="utf-8"))["providers"]
    assert set(rows) == {provider["name"] for provider in catalog}
    for provider in catalog:
        tiers = provider.get("modelTiers") or {}
        expected = "balanced" + ("," + ",".join(sorted(tiers)) if tiers else "")
        assert rows[provider["name"]][2] == expected, (
            f"TIERS column for {provider['name']} is {rows[provider['name']][2]!r}, "
            f"want {expected!r}"
        )
