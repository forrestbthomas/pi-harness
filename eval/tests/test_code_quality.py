"""Custom/customizable code-quality metrics for agent outputs."""

import ast
import re
from pathlib import Path

import pytest
from conftest import load_dataset


class CodeQualityMetric:
    """Simple deterministic metric for Python code outputs.

    Scores:
    - 1.0 if the output is valid Python that defines at least one function/class.
    - 0.5 if it contains a code block but is not standalone valid Python.
    - 0.0 otherwise.
    """

    def __init__(self, threshold: float = 0.5):
        self.threshold = threshold
        self.score = 0.0
        self.reason = ""

    def measure(self, actual_output: str) -> float:
        code = self._extract_code(actual_output)
        if not code:
            self.score = 0.0
            self.reason = "No Python code block found."
            return self.score

        try:
            tree = ast.parse(code)
        except SyntaxError as exc:
            self.score = 0.5
            self.reason = f"Code block present but not valid Python: {exc}"
            return self.score

        definitions = [
            node
            for node in ast.walk(tree)
            if isinstance(node, (ast.FunctionDef, ast.ClassDef, ast.AsyncFunctionDef))
        ]
        if definitions:
            self.score = 1.0
            self.reason = f"Valid Python with {len(definitions)} function/class definition(s)."
        else:
            self.score = 0.5
            self.reason = "Valid Python but no function or class definitions."
        return self.score

    def is_successful(self) -> bool:
        return self.score >= self.threshold

    @staticmethod
    def _extract_code(text: str) -> str | None:
        # Prefer fenced code blocks labeled python
        match = re.search(r"```python\n(.*?)\n```", text, re.DOTALL)
        if match:
            return match.group(1).strip()
        # Fallback to any fenced block
        match = re.search(r"```\n(.*?)\n```", text, re.DOTALL)
        if match:
            return match.group(1).strip()
        # Fallback to treating the whole output as code if it looks like Python
        stripped = text.strip()
        if stripped.startswith("def ") or stripped.startswith("class "):
            return stripped
        return None


def test_sample_quality_metric():
    """Verify the custom metric on a known-good expected output."""
    samples = load_dataset()
    factorial_sample = next(s for s in samples if s["id"] == "coding-001")
    metric = CodeQualityMetric(threshold=0.5)
    score = metric.measure(factorial_sample["expected_output"])
    assert score == 1.0, metric.reason


def test_custom_metric_detects_missing_code():
    metric = CodeQualityMetric(threshold=0.5)
    score = metric.measure("This is just a sentence with no code.")
    assert score == 0.0
    assert not metric.is_successful()
