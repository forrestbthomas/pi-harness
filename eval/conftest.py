"""Shared fixtures and helpers for the DeepEval evaluation suite."""

import json
import os
import shutil
import subprocess
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

import pytest
from deepeval.test_case import LLMTestCase
from secret_backend import resolve_secret as _resolve_secret


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


def run_pi_print(prompt: str, cwd: Path | None = None, extra_args: list[str] | None = None, timeout: float = 180.0) -> str:
    """Run Pi in print mode and return the text output.

    Requires a provider API key to be set in the environment. The harness
    directory is used as the working directory so AGENTS.md and .pi/SYSTEM.md
    are loaded.

    Failure detection: pi exits 0 even when the model call fails (API/model
    errors go to stderr and stdout is empty), so a run is treated as failed
    when the exit code is non-zero OR stdout is empty. This keeps eval runs
    honest — a silent model failure must never look like a pass.
    """
    cwd = cwd or Path(__file__).parent.parent
    cmd = ["pi-run", "print"]
    if extra_args:
        cmd.extend(extra_args)
    cmd.append(prompt)

    try:
        result = subprocess.run(
            cmd,
            cwd=cwd,
            text=True,
            capture_output=True,
            env=os.environ.copy(),
            timeout=timeout,
        )
    except subprocess.TimeoutExpired as exc:
        raise RuntimeError(
            f"pi print timed out after {timeout:.0f}s: {' '.join(cmd)}"
        ) from exc
    if result.returncode != 0 or not result.stdout.strip():
        raise RuntimeError(
            f"pi print failed (exit {result.returncode}, empty stdout? "
            f"{not result.stdout.strip()}):\n{result.stderr}"
        )
    return result.stdout


# Mirrors internal/cli/eval.go's supportedProviderKeyEnvs and must cover every
# keyEnv in the provider catalog (providers.json / embedded defaultProviders).
# Keep both lists in sync when adding providers.
SUPPORTED_PROVIDER_KEYS = (
    "OPENROUTER_API_KEY",
    "OPENAI_API_KEY",
    "ANTHROPIC_API_KEY",
    "GEMINI_API_KEY",
    "GROQ_API_KEY",
    "DEEPSEEK_API_KEY",
    "LOCAL_API_KEY",
    "AZURE_OPENAI_API_KEY",
    "OLLAMA_API_KEY",
    "MISTRAL_API_KEY",
    "COHERE_API_KEY",
    "TOGETHER_API_KEY",
    "PERPLEXITY_API_KEY",
    "FIREWORKS_API_KEY",
    "MOONSHOT_API_KEY",
    "XAI_API_KEY",
    "BEDROCK_API_KEY",
)


def any_provider_key_env() -> bool:
    """True if any supported provider key is set in the environment (presence only)."""
    return any(os.environ.get(k) for k in SUPPORTED_PROVIDER_KEYS)


def get_secret(name: str) -> str | None:
    """Return the secret for ``name``: env var first, then the configured backend.

    Delegates to secret_backend.resolve_secret so the Go CLI
    (internal/cli/secret.go) and the eval suite share one identifier contract.
    Never logs the value. Returns None when unavailable.
    """
    return _resolve_secret(name)


def has_api_key() -> bool:
    """True if any supported provider key is present in the environment (presence only)."""
    return any_provider_key_env()


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


# ---------------------------------------------------------------------------
# Deterministic contract-test fixtures (spec §4.5): pi_run_bin / hermetic_env /
# fake_launch_env / fake_collector. These exercise the REAL pi-run binary over
# subprocess stdio, hermetically (fake keys, fake node/pi, fake collector, tmp
# roots). Stdlib-only — eval/requirements.txt is unchanged by this suite.
# ---------------------------------------------------------------------------

REPO_ROOT = Path(__file__).resolve().parents[1]

# Env vars beyond SUPPORTED_PROVIDER_KEYS that hermetic_env clears so child
# pi-run processes never see ambient harness configuration. The extra
# PI_PROVIDER / PI_PERMISSION_MODE / PI_MAX_BUDGET_USD clears are defensive:
# ambient values would otherwise change provider/permission/budget resolution.
_HERMETIC_ENV_VARS = (
    "PI_SECRET_BACKEND",
    "BW_GET",
    "PI_OTLP_ENDPOINT",
    "PI_MODEL_TIER",
    "PI_RUN_PROVIDERS_FILE",
    "PI_NODE_VERSION",
    "PI_PROVIDER",
    "PI_PERMISSION_MODE",
    "PI_MAX_BUDGET_USD",
)

_UNSET = object()


@pytest.fixture(scope="session")
def pi_run_bin(tmp_path_factory):
    """Resolve the real pi-run binary under test and probe it loudly.

    Resolution order: PI_RUN_BIN env override, then pi-run on PATH, then a
    one-time `go build` into the session tmp dir. After resolution the binary
    is PROBED: `--exit-codes` must exit 0 and print the '8  scorecard gate
    failed' row, and usage must document mcp-server. A stale/mismatched binary
    FAILS here (never skips, unlike test_benchmark_format.py's skip probe): a
    stale binary is a broken contract, and the CI python-contract job always
    builds fresh.
    """
    override = os.environ.get("PI_RUN_BIN")
    if override:
        binary = os.path.abspath(override)
    else:
        found = shutil.which("pi-run")
        if found:
            binary = os.path.abspath(found)
        else:
            build_dir = tmp_path_factory.mktemp("pi-run-build")
            binary = os.path.join(str(build_dir), "pi-run")
            subprocess.run(
                ["go", "build", "-o", binary, "./cmd/pi-run"],
                cwd=REPO_ROOT,
                check=True,
                timeout=600,
            )

    rebuild_hint = (
        "the pi-run binary under test is stale or does not match the shipped "
        "contract surface. Rebuild it from the repo root with "
        "`go build -o bin/pi-run ./cmd/pi-run` and set PI_RUN_BIN to that "
        "binary (CI builds it fresh in the python-contract job)."
    )
    codes = subprocess.run([binary, "--exit-codes"], capture_output=True, text=True, timeout=60)
    assert codes.returncode == 0, (
        f"probe `pi-run --exit-codes` exited {codes.returncode} — {rebuild_hint}\n"
        f"stdout:\n{codes.stdout}\nstderr:\n{codes.stderr}"
    )
    assert "8  scorecard gate failed" in codes.stdout, (
        f"--exit-codes output is missing the '8  scorecard gate failed' row — "
        f"{rebuild_hint}\n{codes.stdout}"
    )
    usage = subprocess.run([binary, "help"], capture_output=True, text=True, timeout=60)
    assert usage.returncode == 0 and "mcp-server" in usage.stdout, (
        f"usage text does not document mcp-server — {rebuild_hint}\n{usage.stdout}"
    )
    return binary


@pytest.fixture
def hermetic_env(tmp_path):
    """Build hermetic env dicts for contract subprocess tests.

    Returns make_env(harness_root=..., **extra): every provider key env is
    cleared, PI_SECRET_BACKEND=env-only, HOME=<tmp>. HARNESS_ROOT defaults to a
    fresh tmp dir; pass harness_root=None (plus cwd=<repo>) to exercise the
    real checkout, e.g. the providers.json data-driven test.
    """

    def make_env(harness_root=_UNSET, home_dir=None, **extra):
        env = dict(os.environ)
        for key in SUPPORTED_PROVIDER_KEYS:
            env[key] = ""
        for key in _HERMETIC_ENV_VARS:
            env[key] = ""
        env["PI_SECRET_BACKEND"] = "env-only"
        env["HOME"] = str(home_dir) if home_dir is not None else str(tmp_path)
        if harness_root is _UNSET:
            harness_root = tmp_path / "root"
            harness_root.mkdir(exist_ok=True)
        env["HARNESS_ROOT"] = "" if harness_root is None else str(harness_root)
        for key, value in extra.items():
            env[key] = "" if value is None else str(value)
        return env

    return make_env


# Fake pi executed by `pi-run print/chat`: logs its argv and the key it
# received via the ENVIRONMENT (never argv), then exits with FAKE_PI_EXIT.
FAKE_PI_SCRIPT = """#!/bin/sh
echo "ARGS:$*" >> "$FAKE_PI_LOG"
echo "KEY_OPENAI=${OPENAI_API_KEY:-}" >> "$FAKE_PI_LOG"
exit "${FAKE_PI_EXIT:-0}"
"""


class FakeLaunchEnv:
    """Fake nvm node + fake pi install under a tmp HOME.

    Mirrors internal/cli/pi.go exactly: nodeBinDir only stats
    <HOME>/.nvm/versions/node/v22.19.0/bin/node (never executes it) and
    execPiDir runs the absolute <binDir>/pi path.
    """

    NODE_VERSION = "v22.19.0"

    def __init__(self, home: Path, log: Path):
        self.home = home
        self.log = log

    @property
    def node_bin_dir(self) -> Path:
        return self.home / ".nvm" / "versions" / "node" / self.NODE_VERSION / "bin"

    @property
    def log_text(self) -> str:
        return self.log.read_text(encoding="utf-8") if self.log.exists() else ""

    def env(self, base, *, fake_pi_exit=None, **extra):
        """Return base env pointed at this fake toolchain (HOME + PI_NODE_VERSION)."""
        env = dict(base)
        env["HOME"] = str(self.home)
        env["PI_NODE_VERSION"] = self.NODE_VERSION
        env["FAKE_PI_LOG"] = str(self.log)
        if fake_pi_exit is not None:
            env["FAKE_PI_EXIT"] = str(fake_pi_exit)
        for key, value in extra.items():
            env[key] = "" if value is None else str(value)
        return env


@pytest.fixture
def fake_launch_env(tmp_path):
    """tmp HOME with .nvm/versions/node/v22.19.0/bin/{node,pi} (pi.go layout)."""
    node_dir = tmp_path / ".nvm" / "versions" / "node" / FakeLaunchEnv.NODE_VERSION / "bin"
    node_dir.mkdir(parents=True)
    node = node_dir / "node"
    node.write_bytes(b"")
    node.chmod(0o755)
    pi = node_dir / "pi"
    pi.write_text(FAKE_PI_SCRIPT, encoding="utf-8")
    pi.chmod(0o755)
    return FakeLaunchEnv(home=tmp_path, log=tmp_path / "fake-pi.log")


class _CollectorServer(ThreadingHTTPServer):
    """ThreadingHTTPServer that carries the FakeCollector so the handler can
    reach it per-instance (avoids a shared class attribute, which would
    cross-route requests if two collectors were ever alive concurrently)."""

    def __init__(self, addr, handler, collector):
        super().__init__(addr, handler)
        self.collector = collector


class _CollectorHandler(BaseHTTPRequestHandler):
    """Records every request into the owning FakeCollector; configurable status."""

    def _record(self):
        collector = self.server.collector
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else b""
        collector.requests.append(
            {
                "method": self.command,
                "path": self.path,
                "headers": {key: value for key, value in self.headers.items()},
                "body": body,
            }
        )
        self.send_response(collector.status)
        self.send_header("Content-Length", "0")
        self.end_headers()

    do_GET = _record
    do_POST = _record

    def log_message(self, *args):
        pass  # keep the fake collector off stderr


class FakeCollector:
    """ThreadingHTTPServer on an ephemeral 127.0.0.1 port, recording requests."""

    def __init__(self):
        self.requests = []
        self.status = 200
        self._httpd = _CollectorServer(("127.0.0.1", 0), _CollectorHandler, self)
        self._thread = threading.Thread(target=self._httpd.serve_forever, daemon=True)
        self._thread.start()

    @property
    def url(self) -> str:
        host, port = self._httpd.server_address[:2]
        return f"http://{host}:{port}"

    def close(self):
        self._httpd.shutdown()
        self._httpd.server_close()
        self._thread.join(timeout=5)


@pytest.fixture
def fake_collector():
    collector = FakeCollector()
    yield collector
    collector.close()
