"""EVAL-15 — harness↔eval seam verification (hermetic dry-run).

Two jobs:

1. **Contract pins** (hard pass/fail): the seam's versioned surfaces must be
   present and self-consistent — tasks.json + datasetVersion, score_run
   SCHEMA_VERSION, >= 50 live cases, grader/reference count floors matching
   the manifest, benchmark dirs.

2. **Self-containment dry-run** (records, never fails): scan eval/ for
   harness-root coupling (repo_root() path assumptions, .pi/ reads,
   cmd/pi-run / internal/cli references, pi-run subprocess spawns) and write
   the inventory to eval/live-results/seam-report.json. A failing-on-coupling
   test would block every PR today — the point is to MEASURE the gap so the
   pi-bench split decision (charter: EPIC-1 DoD AND external consumer) has a
   factual handoff kit, per docs/benchmark-seam.md.

Hermetic: no keys, no network, no Go build. Runs in the deterministic suite.
"""

import json
import re
from pathlib import Path

EVAL_DIR = Path(__file__).resolve().parents[1]
HARNESS_ROOT = EVAL_DIR.parent
SEAM_REPORT = EVAL_DIR / "live-results" / "seam-report.json"

# Coupling signatures that tie the eval layer to the harness repo.
HARNESS_COUPLING_PATTERNS = {
    "repo_root()": r"repo_root\s*\(",
    "dot-pi-heal read": r"\.pi[\\/]heal",
    "harness cmd ref": r"cmd[\\/]pi-run",
    "harness internal ref": r"internal[\\/]cli",
    "pi-run subprocess spawn": r"subprocess\.[A-Za-z]+\([^)]*\[?\"?pi-run|\bpi-run\b",
}


def _load_manifest() -> dict:
    path = EVAL_DIR / "datasets" / "tasks.json"
    assert path.is_file(), "tasks.json missing — the seam manifest is required"
    return json.loads(path.read_text(encoding="utf-8"))


def _count_jsonl(path: Path) -> int:
    if not path.is_file():
        return 0
    return sum(1 for line in path.read_text(encoding="utf-8").splitlines() if line.strip())


def test_seam_manifest_and_dataset_version():
    manifest = _load_manifest()
    version = manifest.get("datasetVersion")
    assert version and re.match(r"^\d{4}-\d{2}-\d{2}\.\d+$", version), (
        f"datasetVersion must be YYYY-MM-DD.N, got {version!r}"
    )
    tasks = manifest.get("tasks", [])
    assert len(tasks) >= 50, f"manifest should carry >= 50 tasks, got {len(tasks)}"


def test_seam_score_run_schema_version():
    text = (EVAL_DIR / "scripts" / "score_run.py").read_text(encoding="utf-8")
    assert "SCHEMA_VERSION = 1" in text, "score_run.py must pin SCHEMA_VERSION = 1"


def test_seam_live_dataset_is_50_cases():
    # EVAL-6 added the agentic category; the floor is the 50-case benchmark
    # (per-category budgets enforced by test_dataset_schema).
    n = _count_jsonl(EVAL_DIR / "datasets" / "coding_samples.jsonl")
    assert n >= 50, f"coding_samples.jsonl must have >= 50 live cases, got {n}"


def test_seam_grader_and_reference_dirs_exist():
    graders = EVAL_DIR / "datasets" / "graders"
    references = EVAL_DIR / "datasets" / "references"
    assert graders.is_dir() and references.is_dir(), "graders/ and references/ must exist"
    assert len(list(graders.iterdir())) >= 40, "graders dir looks thin (< 40)"
    assert len(list(references.iterdir())) >= 50, "references dir must cover 50 cases"


def test_seam_benchmark_dirs_exist():
    benchmarks = EVAL_DIR / "benchmarks"
    assert benchmarks.is_dir() and len(list(benchmarks.iterdir())) >= 8, (
        "benchmarks/ must contain the edit-based benchmark tasks"
    )


def _scan_for_couplings(root: Path) -> list[dict]:
    """Scan eval/ for harness-root coupling signatures; return an inventory."""
    findings: list[dict] = []
    py_files = sorted(root.rglob("*.py"))
    for path in py_files:
        # Skip .venv and live-results artifacts.
        if ".venv" in path.parts or "live-results" in path.parts:
            continue
        rel = path.relative_to(root).as_posix()
        text = path.read_text(encoding="utf-8", errors="replace")
        for label, pattern in HARNESS_COUPLING_PATTERNS.items():
            for lineno, line in enumerate(text.splitlines(), 1):
                if re.search(pattern, line):
                    findings.append({
                        "file": rel,
                        "line": lineno,
                        "kind": label,
                        "snippet": line.strip()[:120],
                    })
    return findings


def test_seam_dry_run_writes_coupling_report():
    """EVAL-15 dry-run: write the coupling inventory. Never fails on findings —
    it records the gap so the split decision has facts (docs/benchmark-seam.md)."""
    findings = _scan_for_couplings(EVAL_DIR)
    SEAM_REPORT.parent.mkdir(parents=True, exist_ok=True)
    report = {
        "schemaVersion": 1,
        "generated": "2026-08-14",  # deterministic; no clock dependency
        "couplings": findings,
        "nCouplings": len(findings),
        "note": "Inventory of harness-root coupling in eval/. The seam is real "
                "(schema-versioned, provenance-carrying) but not yet "
                "self-contained; see docs/benchmark-seam.md for the split "
                "trigger and decoupling list.",
    }
    SEAM_REPORT.write_text(json.dumps(report, indent=2), encoding="utf-8")
    # The report must exist and be parseable; the count is informational.
    assert SEAM_REPORT.is_file(), "seam-report.json must be written"


def test_seam_dry_run_detects_known_coupling():
    """Sanity check the scanner itself: score_run.py's repo_root() coupling
    must be in the report (it is the load-bearing coupling we document)."""
    if not SEAM_REPORT.is_file():
        test_seam_dry_run_writes_coupling_report()
    report = json.loads(SEAM_REPORT.read_text(encoding="utf-8"))
    files = {c["file"] for c in report["couplings"]}
    assert "scripts/score_run.py" in files, (
        "scanner must detect score_run.py's repo_root() coupling"
    )
