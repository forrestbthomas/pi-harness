"""Benchmark task-format validation (hermetic: no Docker, no API keys).

Each task under eval/benchmarks/<name>/ must carry a task.json with at least
an id and a prompt (or an instruction.md file), plus a tests/run.sh
verification script. The CLI dry-run mode is the single source of truth for
format validation; this test mirrors its rules and runs it as an end-to-end
check.
"""

import json
import subprocess
from pathlib import Path

import pytest

HARNESS = Path(__file__).resolve().parents[2]
BENCHMARKS = HARNESS / "eval" / "benchmarks"
DEFAULT_TEST_SCRIPT = "tests/run.sh"
LOCAL_RELATIVE_KEYS = ("instruction", "testScript", "dockerfile", "solution")


def _benchmark_dirs() -> list[Path]:
    assert BENCHMARKS.is_dir(), f"missing benchmark suite: {BENCHMARKS}"
    dirs = sorted(p for p in BENCHMARKS.iterdir() if (p / "task.json").is_file())
    assert dirs, f"no benchmark tasks under {BENCHMARKS}"
    return dirs


def _validate_task(task_dir: Path) -> list[str]:
    """Return a list of format problems for one task directory."""
    problems = []
    task_path = task_dir / "task.json"
    try:
        data = json.loads(task_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return [f"{task_path}: invalid JSON: {exc}"]

    if not data.get("id"):
        problems.append(f"{task_path}: missing required field 'id'")
    if not (data.get("prompt") or data.get("instruction")):
        problems.append(f"{task_path}: missing 'prompt' (or 'instruction')")
    if data.get("prompt") and data.get("instruction"):
        problems.append(f"{task_path}: set either 'prompt' or 'instruction', not both")

    timeout = data.get("timeoutSecs", 0)
    if not isinstance(timeout, int) or timeout < 0:
        problems.append(f"{task_path}: timeoutSecs must be a non-negative integer")

    for key in LOCAL_RELATIVE_KEYS:
        path = data.get(key)
        if path is None:
            continue
        if not isinstance(path, str) or Path(path).is_absolute():
            problems.append(f"{task_path}: {key} must be a relative path")
            continue
        if not (task_dir / path).is_file():
            problems.append(f"{task_path}: missing {key} file {path}")

    test_script = data.get("testScript") or DEFAULT_TEST_SCRIPT
    if not (task_dir / test_script).is_file():
        problems.append(f"{task_path}: missing testScript {test_script}")
    return problems


def test_benchmark_task_formats():
    problems = []
    for task_dir in _benchmark_dirs():
        problems.extend(_validate_task(task_dir))
    assert not problems, "benchmark format problems:\n" + "\n".join(problems)


def test_benchmark_dry_run_passes():
    """End-to-end dry-run must pass against the current binary.

    Skips when the pi-run on PATH is older than the benchmark feature (e.g. a
    Homebrew-installed release binary without --benchmark support), so the test
    does not fail in environments that have not yet installed the new binary.
    """
    probe = subprocess.run(
        ["pi-run", "eval", "--benchmark", "--benchmark-dry-run"],
        cwd=HARNESS,
        capture_output=True,
        text=True,
    )
    if "unknown flag" in probe.stderr and "--benchmark" in probe.stderr:
        pytest.skip("pi-run on PATH predates --benchmark support; install the new binary")
    assert probe.returncode == 0, (
        f"pi-run eval --benchmark --benchmark-dry-run failed "
        f"(exit {probe.returncode}):\n{probe.stdout}\n{probe.stderr}"
    )
