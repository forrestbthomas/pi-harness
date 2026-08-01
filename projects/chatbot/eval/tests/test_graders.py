import json
import subprocess
import unittest
from unittest.mock import patch

from graders import grade_case
from run_evals import online_response


class GradeCaseTests(unittest.TestCase):
    def test_requires_all_expected_signals(self) -> None:
        case = {
            "id": "uncertainty",
            "required": ["cannot verify", "support"],
            "forbidden": ["guaranteed"],
        }

        result = grade_case(
            case, "I cannot verify that here, so please contact support."
        )

        self.assertTrue(result["passed"])
        self.assertEqual([], result["missing"])
        self.assertEqual([], result["forbidden_found"])

    def test_online_response_uses_silent_json_adapter(self) -> None:
        completed = subprocess.CompletedProcess(
            args=[], returncode=0, stdout=json.dumps({"output": "safe output"}), stderr=""
        )
        with patch("run_evals.subprocess.run", return_value=completed) as run:
            self.assertEqual("safe output", online_response("hello"))

        command = run.call_args.args[0]
        self.assertEqual(["npm", "run", "--silent", "eval-response"], command)
        self.assertEqual(json.dumps({"input": "hello"}), run.call_args.kwargs["input"])

    def test_reports_missing_and_forbidden_signals(self) -> None:
        case = {
            "id": "safe",
            "required": ["cannot verify", "support"],
            "forbidden": ["guaranteed"],
        }

        result = grade_case(case, "This is guaranteed.")

        self.assertFalse(result["passed"])
        self.assertEqual(["cannot verify", "support"], result["missing"])
        self.assertEqual(["guaranteed"], result["forbidden_found"])


if __name__ == "__main__":
    unittest.main()
