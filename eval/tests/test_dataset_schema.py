"""Dataset v2 schema lint - hermetic (no keys, no network).

Validates eval/datasets/coding_samples.jsonl against the live-eval v2 schema
(spec §4.2 / §6 / review checklist "Dataset v2"):

- exactly 20 records with unique ids;
- required fields: id, input, expected_output, category, difficulty, grader,
  tags, reference;
- category/difficulty/grader are closed enums;
- tags are non-empty and include a ``regression-<thing>`` tag;
- graderRef is present iff grader == "deterministic";
- the reference file exists and is non-empty for every task;
- deterministic tasks: the reference PROVABLY passes its grader (Terminal-Bench
  oracle validation) via eval/grader.py;
- judge tasks: the reference file exists (non-empty expected answer).
"""

import json
import re
import sys
from collections import Counter
from pathlib import Path

import pytest

EVAL_DIR = Path(__file__).resolve().parents[1]
if str(EVAL_DIR) not in sys.path:
    sys.path.insert(0, str(EVAL_DIR))

from grader import DATASETS_DIR, run_grader  # noqa: E402

DATASET_PATH = EVAL_DIR / "datasets" / "coding_samples.jsonl"

CATEGORIES = {
    "code-gen",
    "bug-fix",
    "shell/ops",
    "concept",
    "negative-edge",
    "harness-routing",
}
DIFFICULTIES = {"easy", "medium", "hard"}
GRADERS = {"deterministic", "judge"}
# Spec §4.2 category budget (min, max).
CATEGORY_BUDGET = {
    "code-gen": (5, 10),
    "bug-fix": (10, 16),
    "shell/ops": (8, 10),
    "concept": (8, 10),
    "negative-edge": (7, 10),
    "harness-routing": (5, 10),
}
ID_RE = re.compile(r"^coding-\d{3}$")
REGRESSION_RE = re.compile(r"^regression-.+")
REQUIRED_FIELDS = (
    "id",
    "input",
    "expected_output",
    "category",
    "difficulty",
    "grader",
    "tags",
    "reference",
)


@pytest.fixture(scope="module")
def rows():
    with DATASET_PATH.open(encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def test_dataset_has_exactly_50_records(rows):
    assert len(rows) == 50, f"expected exactly 50 records, found {len(rows)}"


def test_required_fields_and_enum_values(rows):
    for row in rows:
        task_id = row["id"]
        for field in REQUIRED_FIELDS:
            assert field in row, f"{task_id}: missing required field {field!r}"
        assert isinstance(row["input"], str) and row["input"].strip(), (
            f"{task_id}: input must be a non-empty string"
        )
        assert isinstance(row["expected_output"], str) and row["expected_output"].strip(), (
            f"{task_id}: expected_output must be a non-empty string"
        )
        assert row["category"] in CATEGORIES, (
            f"{task_id}: invalid category {row['category']!r}"
        )
        assert row["difficulty"] in DIFFICULTIES, (
            f"{task_id}: invalid difficulty {row['difficulty']!r}"
        )
        assert row["grader"] in GRADERS, f"{task_id}: invalid grader {row['grader']!r}"
        assert isinstance(row["tags"], list) and row["tags"], (
            f"{task_id}: tags must be a non-empty list"
        )
        assert any(REGRESSION_RE.match(t) for t in row["tags"]), (
            f"{task_id}: tags must include a regression-<thing> tag"
        )
        for tag in row["tags"]:
            assert isinstance(tag, str) and tag, f"{task_id}: empty tag"


def test_ids_are_unique_and_well_formed(rows):
    ids = [row["id"] for row in rows]
    assert len(ids) == len(set(ids)), f"duplicate ids: {sorted(ids)}"
    for task_id in ids:
        assert ID_RE.match(task_id), f"{task_id}: id must match coding-\\d{{3}}"


def test_grader_ref_present_iff_deterministic(rows):
    for row in rows:
        task_id = row["id"]
        if row["grader"] == "deterministic":
            assert isinstance(row.get("graderRef"), str) and row["graderRef"], (
                f"{task_id}: deterministic task requires a graderRef"
            )
        else:
            assert "graderRef" not in row or not row.get("graderRef"), (
                f"{task_id}: judge task must omit graderRef"
            )


def test_category_budget(rows):
    counts = Counter(row["category"] for row in rows)
    for category, (lo, hi) in CATEGORY_BUDGET.items():
        assert lo <= counts[category] <= hi, (
            f"category {category!r}: {counts[category]} records, expected "
            f"{lo}-{hi} per spec §4.2"
        )


def test_reference_files_exist_and_are_non_empty(rows):
    for row in rows:
        ref = DATASETS_DIR / row["reference"]
        assert ref.is_file(), f"{row['id']}: reference missing: {ref}"
        assert ref.read_text(encoding="utf-8").strip(), (
            f"{row['id']}: reference file is empty"
        )


def test_grader_ref_files_exist(rows):
    for row in rows:
        if row["grader"] != "deterministic":
            continue
        path = DATASETS_DIR / row["graderRef"]
        assert path.is_file(), f"{row['id']}: graderRef missing: {path}"


def test_deterministic_references_provably_pass_their_grader(rows):
    """Terminal-Bench oracle rule: every deterministic reference solution must
    pass its own grader (spec §4.2 / review checklist)."""
    failures = []
    for row in rows:
        if row["grader"] != "deterministic":
            continue
        candidate = (DATASETS_DIR / row["reference"]).read_text(encoding="utf-8")
        passed, detail = run_grader(row, candidate)
        if not passed:
            failures.append(f"{row['id']}: reference failed its grader: {detail}")
    assert not failures, "oracle validation failures:\n" + "\n".join(failures)


DATASET_VERSION_RE = re.compile(r"^\d{4}-\d{2}-\d{2}\.\d+$")


def test_tasks_manifest_has_dataset_version():
    """Every dataset content change must bump datasetVersion (EVAL-3)."""
    tasks = json.loads((EVAL_DIR / "datasets" / "tasks.json").read_text(encoding="utf-8"))
    version = tasks.get("datasetVersion")
    assert version is not None, "tasks.json must carry a datasetVersion (bump on every dataset change)"
    assert DATASET_VERSION_RE.match(version), (
        f"datasetVersion {version!r} must match YYYY-MM-DD.N"
    )
