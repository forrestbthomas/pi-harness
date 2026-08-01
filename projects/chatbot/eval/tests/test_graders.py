import unittest

from graders import grade_case


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
