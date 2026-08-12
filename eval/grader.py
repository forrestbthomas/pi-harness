"""Shared deterministic grading harness for the live-eval v2 dataset.

The dataset (eval/datasets/coding_samples.jsonl, schema v2, spec §4.2) declares
a graderRef on every deterministic task pointing at a standalone executable
grader (eval/datasets/graders/<id>/grade.py). The grader reads the candidate
text on stdin and exits 0 (pass) or non-zero (fail, with an explanation on
stderr).

This module is the single entry point for running those graders:

    run_grader(task, candidate_text) -> (passed, detail)
    is_deterministic(task) -> bool
    grader_script_path(task) -> Path | None

``task`` is a dataset row dict (as loaded from coding_samples.jsonl) carrying
``id`` and ``graderRef``. Judge-graded tasks (grader == "judge") have no
graderRef and are graded by the live suite's LLM judge instead.

Used by eval/tests/test_dataset_schema.py (reference-solution oracle
validation, Terminal-Bench rule, spec §4.2/§6) and by the live suite's
deterministic lane. Stdlib-only.
"""

import subprocess
import sys
from pathlib import Path

#: Root of the dataset tree holding graders/<id>/grade.py and references/<id>/.
DATASETS_DIR = Path(__file__).resolve().parent / "datasets"


def is_deterministic(task) -> bool:
    """True when the dataset row is graded by a deterministic grader."""
    return task.get("grader") == "deterministic"


def grader_script_path(task) -> Path | None:
    """Absolute path to the task's grade.py, or None for judge-graded tasks.

    Accepts either a dataset row dict or a task identifier (id string like
    "coding-001", or a path to the graders/<id> directory).
    """
    if isinstance(task, (str, Path)):
        base = Path(task)
        if base.is_dir():
            return base / "grade.py"
        return DATASETS_DIR / "graders" / str(task) / "grade.py"
    ref = task.get("graderRef")
    if not ref:
        return None
    path = Path(ref)
    if not path.is_absolute():
        path = DATASETS_DIR / path
    return path


def run_grader(task, candidate_text: str, timeout: float = 60.0) -> tuple[bool, str]:
    """Run the task's deterministic grader with the candidate text on stdin.

    Returns (passed, detail): passed is True when the grader exits 0; detail is
    the grader's stderr (or a short status line) for reporting.
    """
    path = grader_script_path(task)
    if path is None:
        return False, "task is judge-graded: no deterministic graderRef"
    if not path.is_file():
        return False, f"grader not found: {path}"
    try:
        proc = subprocess.run(
            [sys.executable, str(path)],
            input=candidate_text,
            text=True,
            capture_output=True,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        return False, f"grader timed out after {timeout:g}s: {path}"
    detail = (proc.stderr or "").strip() or f"grader exited {proc.returncode}"
    if proc.returncode == 0:
        return True, detail
    return False, detail
