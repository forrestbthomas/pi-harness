"""DeepEval metrics for coding Q&A correctness."""

import os
from pathlib import Path

import pytest
from conftest import has_api_key, load_dataset, run_pi_print
from deepeval import assert_test
from deepeval.metrics import AnswerRelevancyMetric, FaithfulnessMetric, HallucinationMetric
from deepeval.test_case import LLMTestCase


# These tests use DeepEval's LLM-as-a-judge metrics and need an API key.
pytestmark = pytest.mark.skipif(
    not has_api_key(),
    reason="No provider API key found; set OPENAI_API_KEY or similar.",
)


def test_dataset_loaded():
    """Smoke test: ensure the sample dataset is parseable and non-empty."""
    samples = load_dataset()
    assert len(samples) >= 3
    assert all("input" in s and "expected_output" in s for s in samples)


def test_coding_qa_relevancy():
    """Check that a Pi answer to a coding question is relevant."""
    prompt = "Write a Python function that returns the factorial of a non-negative integer."
    actual = run_pi_print(prompt)

    test_case = LLMTestCase(
        input=prompt,
        actual_output=actual,
    )
    metric = AnswerRelevancyMetric(threshold=0.7)
    assert_test(test_case, [metric])


def test_coding_qa_faithfulness():
    """Check that a Pi answer stays faithful to the requested task."""
    prompt = "What is the time complexity of binary search on a sorted array?"
    actual = run_pi_print(prompt)

    test_case = LLMTestCase(
        input=prompt,
        actual_output=actual,
        retrieval_context=["Binary search halves the search space on each step, giving O(log n) time complexity."],
    )
    metric = FaithfulnessMetric(threshold=0.7)
    assert_test(test_case, [metric])


def test_coding_qa_no_hallucination():
    """Check that a Pi answer does not hallucinate facts."""
    prompt = "Explain the difference between `is` and `==` in Python."
    actual = run_pi_print(prompt)

    test_case = LLMTestCase(
        input=prompt,
        actual_output=actual,
        context=["`==` checks value equality. `is` checks object identity."],
    )
    metric = HallucinationMetric(threshold=0.5)
    assert_test(test_case, [metric])
