"""Contract tests: pi-run --exit-codes table and failure-precedence order.

The --exit-codes table (internal/cli/app.go) is the single source of truth
for exit codes 0..8. Subprocess tests pin the precedence order: usage (2)
beats missing-key (3) beats missing-node (4). The ordering assertions never
hardcode descriptions: observed codes are matched against the rows parsed
from the binary's own table.
"""

import re
import subprocess

# Documented descriptions from internal/cli/app.go exitCodesText. The table
# test below re-parses them from the binary and compares — this dict records
# the shipped contract, it does not define it.
EXIT_CODE_ROWS = {
    0: "ok",
    1: "generic error",
    2: "usage error",
    3: "missing API key",
    4: "node/pi not found",
    5: "eval venv missing",
    6: "budget exceeded",
    7: "docker unavailable (benchmarks)",
    8: "scorecard gate failed (ci-benchmark)",
}


def _parse_exit_codes(text: str) -> dict[int, str]:
    codes = {}
    for line in text.splitlines():
        match = re.match(r"^\s*(\d+)\s+(.+)$", line)
        if match:
            codes[int(match.group(1))] = match.group(2).strip()
    return codes


def _run(pi_run_bin, env, args):
    return subprocess.run(
        [pi_run_bin, *args], env=env, capture_output=True, text=True, timeout=30
    )


def _table(pi_run_bin, env):
    result = subprocess.run(
        [pi_run_bin, "--exit-codes"], env=env, capture_output=True, text=True, timeout=30
    )
    assert result.returncode == 0, f"--exit-codes failed:\n{result.stdout}\n{result.stderr}"
    return _parse_exit_codes(result.stdout)


def test_exit_codes_table_is_source_of_truth(pi_run_bin, hermetic_env):
    parsed = _table(pi_run_bin, hermetic_env())
    # Exactly the 9 rows 0..8 with the documented descriptions: no gaps, no
    # extra codes, no drifted text.
    assert parsed == EXIT_CODE_ROWS


def test_usage_error_beats_missing_key(pi_run_bin, hermetic_env):
    # `print` without a prompt is a usage error (2) even though no provider
    # key is configured (which would otherwise exit 3).
    result = _run(pi_run_bin, hermetic_env(), ["print"])
    assert result.returncode == 2


def test_usage_error_beats_missing_key_tier(pi_run_bin, hermetic_env):
    # An unknown --model-tier is a usage error (2) before any key access.
    result = _run(pi_run_bin, hermetic_env(), ["print", "--model-tier", "turbo", "x"])
    assert result.returncode == 2


def test_missing_key_beats_node_missing(pi_run_bin, hermetic_env):
    # Key resolution precedes node resolution: no key -> 3, not 4.
    result = _run(pi_run_bin, hermetic_env(), ["print", "--provider", "openai", "hello"])
    assert result.returncode == 3


def test_node_missing_after_key_present(pi_run_bin, hermetic_env):
    # With a key present but no nvm node under HOME, the launch stops at 4.
    env = hermetic_env(OPENAI_API_KEY="testvalue")
    result = _run(pi_run_bin, env, ["print", "--provider", "openai", "hello"])
    assert result.returncode == 4


def test_observed_codes_match_table(pi_run_bin, hermetic_env):
    # Every observed exit code from the precedence scenarios must be a row of
    # the --exit-codes table (single source of truth enforced by the suite).
    table = _table(pi_run_bin, hermetic_env())
    scenarios = [
        (["print"], {}, 2),
        (["print", "--model-tier", "turbo", "x"], {}, 2),
        (["print", "--provider", "openai", "hello"], {}, 3),
        (["print", "--provider", "openai", "hello"], {"OPENAI_API_KEY": "testvalue"}, 4),
    ]
    for args, extra, expected in scenarios:
        result = _run(pi_run_bin, hermetic_env(**extra), args)
        assert result.returncode == expected
        assert result.returncode in table, (
            f"observed exit code {result.returncode} is not documented in the "
            f"--exit-codes table"
        )
