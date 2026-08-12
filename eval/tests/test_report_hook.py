"""Hermetic test for the self-contained eval report hook (conftest).

Verifies that PI_EVAL_REPORT makes conftest write a JSON report matching the
shape score_run.py parses (tests[] with nodeid/outcome/user_properties),
without relying on the pytest-json-report plugin.
"""

import json
import os
import subprocess
import sys
from pathlib import Path

import pytest

PYTHON = Path(sys.executable)


@pytest.fixture
def report_env(tmp_path, monkeypatch):
    """Run a tiny pytest session with PI_EVAL_REPORT set; return the report path."""
    report = tmp_path / "report.json"

    def _run(test_content: str, *extra_args):
        # The probe must live UNDER eval/ (a temp subdir of eval/tests) so
        # pytest loads eval/conftest.py — conftest applies only within its
        # directory tree. Use a unique name to avoid collection collisions.
        probe_dir = Path(__file__).parent / "probe_tmp"
        probe_dir.mkdir(parents=True, exist_ok=True)
        test_file = probe_dir / "probe_test.py"
        test_file.write_text(test_content, encoding="utf-8")
        env = dict(os.environ)
        env["PI_EVAL_REPORT"] = str(report)
        subprocess.run(
            [
                str(PYTHON), "-m", "pytest", str(test_file),
                "-p", "no:cacheprovider", "-q",
                *extra_args,
            ],
            cwd=Path(__file__).parent.parent,
            env=env,
            check=False,
            capture_output=True,
            text=True,
        )
        probe_dir.rename(tmp_path / f"probe_done_{test_file.name}")
        return json.loads(report.read_text(encoding="utf-8"))

    return _run


def test_report_hook_writes_shape(report_env):
    payload = report_env(
        "def test_ok():\n    assert True\n"
    )
    assert "tests" in payload
    assert payload["tests"][0]["nodeid"].endswith("probe_test.py::test_ok")
    assert payload["tests"][0]["outcome"] == "passed"
    assert "user_properties" in payload["tests"][0]


def test_report_hook_records_user_properties(report_env):
    payload = report_env(
        "def test_props(record_property):\n"
        "    record_property('pass', True)\n"
        "    record_property('costUsd', 0.01)\n"
    )
    props = payload["tests"][0]["user_properties"]
    assert props == [{"pass": True}, {"costUsd": 0.01}]


def test_report_hook_records_failed_outcome(report_env):
    payload = report_env(
        "def test_bad():\n    assert False\n"
    )
    assert payload["tests"][0]["outcome"] == "failed"
