from __future__ import annotations

import argparse
import json
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from graders import grade_case

ROOT = Path(__file__).resolve().parent.parent
CASES_PATH = ROOT / "eval" / "cases.json"
RESULTS_DIR = ROOT / "eval" / "results"
REQUIRED_CASE_KEYS = {"id", "input", "required", "forbidden", "offline_response"}


def load_cases(path: Path) -> list[dict[str, Any]]:
    loaded = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(loaded, list):
        raise ValueError("Evaluation cases must be a JSON array.")

    cases: list[dict[str, Any]] = []
    for case in loaded:
        if not isinstance(case, dict) or not REQUIRED_CASE_KEYS.issubset(case):
            raise ValueError(
                "Each evaluation case must include id, input, required, forbidden, and offline_response."
            )
        cases.append(case)
    return cases


def online_response(case_input: str) -> str:
    completed = subprocess.run(
        ["npm", "run", "--silent", "eval-response"],
        cwd=ROOT,
        input=json.dumps({"input": case_input}),
        text=True,
        capture_output=True,
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(
            "Chatbot evaluation request failed. Verify Bitwarden is unlocked and model access is available."
        )

    try:
        payload = json.loads(completed.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError(
            "Chatbot evaluation response was not valid JSON."
        ) from error

    output = payload.get("output") if isinstance(payload, dict) else None
    if not isinstance(output, str):
        raise RuntimeError(
            "Chatbot evaluation response did not contain a string output."
        )
    return output


def write_report(report: dict[str, Any]) -> Path:
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
    path = RESULTS_DIR / f"report-{timestamp}.json"
    path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--offline",
        action="store_true",
        help="Grade fixture responses without Bitwarden or OpenAI.",
    )
    args = parser.parse_args()

    cases = load_cases(CASES_PATH)
    results: list[dict[str, Any]] = []
    for case in cases:
        response = (
            str(case["offline_response"])
            if args.offline
            else online_response(str(case["input"]))
        )
        results.append(grade_case(case, response))

    passed = sum(result["passed"] for result in results)
    report = {
        "mode": "offline" if args.offline else "online",
        "passed": passed,
        "total": len(results),
        "results": results,
    }
    report_path = write_report(report)

    print(f"{report['mode']} eval: {passed}/{len(results)} passed")
    for result in results:
        state = "PASS" if result["passed"] else "FAIL"
        print(f"{state} {result['id']}")
    print(f"Report: {report_path.relative_to(ROOT)}")
    return 0 if passed == len(results) else 1


if __name__ == "__main__":
    sys.exit(main())
