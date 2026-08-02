"""Shared fixtures and helpers for the DeepEval evaluation suite."""

import json
import os
import subprocess
from pathlib import Path
from typing import Any

import pytest
from deepeval.test_case import LLMTestCase


DATASET_PATH = Path(__file__).parent / "datasets" / "coding_samples.jsonl"
PI_DIR = Path(__file__).parent.parent / ".pi"


def load_dataset(path: Path = DATASET_PATH) -> list[dict[str, Any]]:
    """Load a JSONL dataset of coding evaluation samples."""
    samples = []
    if not path.exists():
        return samples
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            samples.append(json.loads(line))
    return samples


@pytest.fixture
def dataset() -> list[dict[str, Any]]:
    return load_dataset()


@pytest.fixture
def sample_cases(dataset) -> list[LLMTestCase]:
    """Convert dataset samples into DeepEval LLMTestCase objects."""
    cases = []
    for sample in dataset:
        cases.append(
            LLMTestCase(
                input=sample["input"],
                actual_output="",  # populated by the test
                expected_output=sample.get("expected_output", ""),
                context=sample.get("context") or [],
                retrieval_context=[],
                id=sample.get("id"),
            )
        )
    return cases


def run_pi_print(prompt: str, cwd: Path | None = None, extra_args: list[str] | None = None) -> str:
    """Run Pi in print mode and return the text output.

    Requires a provider API key to be set in the environment. The harness
    directory is used as the working directory so AGENTS.md and .pi/SYSTEM.md
    are loaded.
    """
    cwd = cwd or Path(__file__).parent.parent
    cmd = ["pi-run", "print"]
    if extra_args:
        cmd.extend(extra_args)
    cmd.append(prompt)

    result = subprocess.run(
        cmd,
        cwd=cwd,
        text=True,
        capture_output=True,
        env=os.environ.copy(),
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"pi print failed (exit {result.returncode}):\n{result.stderr}"
        )
    return result.stdout


SUPPORTED_PROVIDER_KEYS = (
    "OPENROUTER_API_KEY",
    "OPENAI_API_KEY",
    "ANTHROPIC_API_KEY",
    "GEMINI_API_KEY",
    "GROQ_API_KEY",
    "DEEPSEEK_API_KEY",
)


def get_secret(name: str) -> str | None:
    """Return the secret for ``name``: env var first, then Bitwarden via bw_get.

    Never logs the value. Returns None when unavailable (env unset and the
    vault is locked, the bw binary is missing, or the item does not exist) so
    callers can fall back or skip.
    """
    value = os.environ.get(name)
    if value:
        return value
    bw_get = os.environ.get("BW_GET", str(Path.home() / "bin" / "bw_get"))
    try:
        result = subprocess.run(
            [bw_get, name],
            capture_output=True,
            text=True,
            timeout=30,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
        return None
    if result.returncode != 0:
        return None
    value = result.stdout.strip()
    return value or None


def has_api_key() -> bool:
    """Return True if any supported provider API key is present (env or Bitwarden)."""
    return any(get_secret(key) for key in SUPPORTED_PROVIDER_KEYS)


def judge_provider() -> str:
    """Return the configured DeepEval judge provider (defaults to openai)."""
    model = os.environ.get("DEEPEVAL_MODEL", "").strip()
    if model:
        return model.split("/", 1)[0]
    return "openai"


@pytest.fixture
def pi_available() -> bool:
    """Skip tests that require Pi if it is not runnable."""
    try:
        subprocess.run(["pi", "--version"], check=True, capture_output=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        pytest.skip("Pi CLI is not available")
    return True


# Informational: when a provider key is present, report which provider DeepEval
# will use as the judge (set DEEPEVAL_MODEL to override the OpenAI default).
if has_api_key():
    print(f"  [eval] judge provider: {judge_provider()} (set DEEPEVAL_MODEL to override)")
