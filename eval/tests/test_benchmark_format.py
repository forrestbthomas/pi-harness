"""Benchmark task-format validation (hermetic: no Docker, no API keys).

Each task under eval/benchmarks/<name>/ must carry a task.json with at least
an id and a prompt (or an instruction.md file), plus a tests/run.sh
verification script. The CLI dry-run mode is the single source of truth for
format validation; this test mirrors its rules and runs it as an end-to-end
check.

Also validates the unified task manifest (eval/datasets/tasks.json, spec §4.6):
the JSONL live surface <-> tasks[] bijection, the benchmark directory <->
tasks[] bijection, the shared taxonomy, no duplicate ids, and non-empty tags
on live tasks. Benchmark task.json files now require category/difficulty/grader.
"""

import json
import subprocess
from pathlib import Path

import pytest

HARNESS = Path(__file__).resolve().parents[2]
BENCHMARKS = HARNESS / "eval" / "benchmarks"
MANIFEST = HARNESS / "eval" / "datasets" / "tasks.json"
DATASET = HARNESS / "eval" / "datasets" / "coding_samples.jsonl"
DEFAULT_TEST_SCRIPT = "tests/run.sh"
LOCAL_RELATIVE_KEYS = ("instruction", "testScript", "dockerfile", "solution")

CATEGORIES = ("code-gen", "bug-fix", "shell/ops", "concept", "negative-edge", "harness-routing", "agentic")
DIFFICULTIES = ("easy", "medium", "hard")
GRADERS = ("deterministic", "judge")


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

    # Schema extension (spec §4.6): the shared taxonomy makes live vs benchmark
    # scores comparable and is what makes the single manifest meaningful.
    # Benchmark grading is deterministic (spec §4.5: Docker grading has no
    # judge), so grader must be "deterministic".
    if data.get("category") not in CATEGORIES:
        problems.append(f"{task_path}: category must be one of {', '.join(CATEGORIES)}")
    if data.get("difficulty") not in DIFFICULTIES:
        problems.append(f"{task_path}: difficulty must be one of {', '.join(DIFFICULTIES)}")
    if data.get("grader") != "deterministic":
        problems.append(f"{task_path}: grader must be \"deterministic\" (benchmark tasks have no judge)")

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


# ---------------------------------------------------------------------------
# Unified task manifest (spec §4.6)
# ---------------------------------------------------------------------------


def _load_manifest() -> dict:
    assert MANIFEST.is_file(), f"missing manifest: {MANIFEST}"
    return json.loads(MANIFEST.read_text(encoding="utf-8"))


def _manifest_tasks(manifest: dict) -> list[dict]:
    tasks = manifest.get("tasks", [])
    assert tasks, "manifest must declare at least one task"
    return tasks


def _jsonl_ids() -> set[str]:
    assert DATASET.is_file(), f"missing live dataset: {DATASET}"
    ids = set()
    with DATASET.open(encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            ids.add(json.loads(line)["id"])
    return ids


def _benchmark_ids() -> set[str]:
    return {json.loads((d / "task.json").read_text(encoding="utf-8"))["id"] for d in _benchmark_dirs()}


def test_manifest_schema_and_taxonomy():
    manifest = _load_manifest()
    assert manifest.get("schemaVersion") == 1, "manifest schemaVersion must be 1"

    surfaces = manifest.get("surfaces", {})
    assert surfaces.get("live", {}).get("kind") == "jsonl"
    assert surfaces.get("live", {}).get("path") == "eval/datasets/coding_samples.jsonl"
    assert surfaces.get("benchmark", {}).get("kind") == "directory"
    assert surfaces.get("benchmark", {}).get("path") == "eval/benchmarks"

    taxonomy = set(manifest.get("categories", []))
    assert taxonomy == set(CATEGORIES), "manifest categories must match the shared taxonomy"

    problems = []
    for task in _manifest_tasks(manifest):
        if task.get("surface") not in ("live", "benchmark"):
            problems.append(f"{task.get('id')}: surface must be live or benchmark")
        if task.get("category") not in taxonomy:
            problems.append(f"{task.get('id')}: category {task.get('category')!r} not in taxonomy")
        if task.get("difficulty") not in DIFFICULTIES:
            problems.append(f"{task.get('id')}: difficulty {task.get('difficulty')!r} invalid")
        if task.get("grader") not in GRADERS:
            problems.append(f"{task.get('id')}: grader {task.get('grader')!r} invalid")
        if task.get("surface") == "live" and not task.get("tags"):
            problems.append(f"{task.get('id')}: live tasks must carry non-empty tags")
    assert not problems, "manifest problems:\n" + "\n".join(problems)


def test_manifest_live_rows_match_jsonl():
    """docs-audit P1-4: the manifest's live-task rows must mirror the JSONL
    (the grading truth) on category/difficulty/grader. 26 mismatches shipped
    with no guard (2026-08-14); the manifest was regenerated from the JSONL.
    """
    import json as _json
    manifest = _load_manifest()
    jsonl_by_id = {}
    with open(DATASET, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            d = _json.loads(line)
            jsonl_by_id[d["id"]] = d
    problems = []
    for task in _manifest_tasks(manifest):
        if task.get("surface") != "live":
            continue
        row = jsonl_by_id.get(task["id"])
        if row is None:
            problems.append(f"{task['id']}: manifest live row missing from jsonl")
            continue
        for field in ("category", "difficulty", "grader"):
            if task.get(field) != row.get(field):
                problems.append(
                    f"{task['id']}: manifest {field} {task.get(field)!r} != jsonl {row.get(field)!r}"
                )
    for cid in jsonl_by_id:
        if not any(t.get("id") == cid and t.get("surface") == "live" for t in _manifest_tasks(manifest)):
            problems.append(f"{cid}: jsonl case missing from manifest live rows")
    assert not problems, "manifest<->jsonl parity problems:\n" + "\n".join(problems)


def test_manifest_no_duplicate_ids():
    manifest = _load_manifest()
    ids = [task.get("id") for task in _manifest_tasks(manifest)]
    duplicates = sorted({task_id for task_id in ids if ids.count(task_id) > 1})
    assert not duplicates, f"duplicate task ids in manifest: {duplicates}"


def test_manifest_live_bijection():
    """JSONL records <-> manifest live tasks must be bijective (spec §4.6).

    NOTE: while the parallel dataset lane (feat/liveeval2-dataset) is in
    flight, coding_samples.jsonl carries only the v1 5 records while the
    manifest already declares all 20 (coding-001..coding-020, spec §4.2).
    The manifest->JSONL direction fails until that lane lands; the JSONL->
    manifest direction holds immediately.
    """
    manifest = _load_manifest()
    jsonl_ids = _jsonl_ids()
    manifest_live_ids = {
        task["id"] for task in _manifest_tasks(manifest) if task.get("surface") == "live"
    }
    problems = []
    missing_in_manifest = sorted(jsonl_ids - manifest_live_ids)
    if missing_in_manifest:
        problems.append(
            "JSONL records missing from manifest live tasks: " + ", ".join(missing_in_manifest)
        )
    missing_in_jsonl = sorted(manifest_live_ids - jsonl_ids)
    if missing_in_jsonl:
        problems.append(
            "manifest live tasks missing from coding_samples.jsonl "
            "(dataset lane not merged yet?): " + ", ".join(missing_in_jsonl)
        )
    assert not problems, "live surface is not bijective:\n" + "\n".join(problems)


def test_manifest_benchmark_bijection():
    """Benchmark task.json ids <-> manifest benchmark tasks must be bijective."""
    manifest = _load_manifest()
    benchmark_ids = _benchmark_ids()
    manifest_benchmark_ids = {
        task["id"] for task in _manifest_tasks(manifest) if task.get("surface") == "benchmark"
    }
    problems = []
    missing_in_manifest = sorted(benchmark_ids - manifest_benchmark_ids)
    if missing_in_manifest:
        problems.append(
            "benchmark task.json ids missing from manifest: " + ", ".join(missing_in_manifest)
        )
    missing_in_benchmarks = sorted(manifest_benchmark_ids - benchmark_ids)
    if missing_in_benchmarks:
        problems.append(
            "manifest benchmark tasks missing from eval/benchmarks: "
            + ", ".join(missing_in_benchmarks)
        )
    assert not problems, "benchmark surface is not bijective:\n" + "\n".join(problems)
