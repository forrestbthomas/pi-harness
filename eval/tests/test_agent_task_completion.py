"""Task-based agent evaluation: did the agent produce the expected output?"""

import json
import os
from pathlib import Path

import pytest
from conftest import has_api_key, load_dataset, run_pi_print


@pytest.mark.skipif(not has_api_key(), reason="No provider API key found.")
def test_agent_produces_expected_factorial():
    """End-to-end check that Pi's output contains a correct factorial implementation."""
    prompt = "Write a Python function that returns the factorial of a non-negative integer."
    actual = run_pi_print(prompt)

    # A lightweight heuristic: the output should mention factorial and use a loop or recursion.
    lowered = actual.lower()
    assert "factorial" in lowered
    assert "def " in lowered
    assert "for " in lowered or "recursion" in lowered or "range" in lowered


def test_dataset_expected_outputs_are_non_empty():
    """Sanity check for dataset integrity."""
    samples = load_dataset()
    for sample in samples:
        assert sample["expected_output"].strip(), f"Sample {sample['id']} has empty expected output"
