"""Pure-stdlib secret-backend selection shared with Go contract tests.

This module holds the single source of truth for the PI_SECRET_BACKEND
identifier contract (which values select which backend) plus the resolution
logic. It imports only the standard library so Go tests
(internal/cli/secret_contract_test.go) can execute it with a bare ``python3``
without the deepeval venv installed. conftest.get_secret delegates here so the
evaluation suite and the Go CLI cannot drift apart.
"""

import os
import subprocess
from pathlib import Path


def canonical_backend(value=None):
    """Return the canonical backend name for a PI_SECRET_BACKEND value.

    Empty string and None mean bitwarden (backward compatible with the
    pre-pluggable behavior). Unknown values return "unknown"; callers must
    treat that as unavailable, never as a fallback to bitwarden.
    """
    raw = value if value is not None else os.environ.get("PI_SECRET_BACKEND", "")
    if raw in ("", "bitwarden", "bw"):
        return "bitwarden"
    if raw in ("1password", "op"):
        return "1password"
    if raw in ("env-only", "env"):
        return "env-only"
    return "unknown"


def resolve_secret(name: str):
    """Return the secret for ``name``: env var first, then the configured backend.

    Backend is selected by PI_SECRET_BACKEND (default bitwarden). Bitwarden
    uses ``bw_get`` (BW_GET override, default ~/bin/bw_get); 1password uses
    ``op read "op://<vault>/<name>/credential"``; env-only never falls back.
    Never logs the value. Returns None when unavailable.
    """
    value = os.environ.get(name)
    if value:
        return value

    backend = canonical_backend()
    if backend == "env-only":
        return None
    if backend == "1password":
        vault = os.environ.get("OP_VAULT", "Personal")
        try:
            result = subprocess.run(
                ["op", "read", f"op://{vault}/{name}/credential"],
                capture_output=True,
                text=True,
                timeout=30,
            )
        except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
            return None
        if result.returncode != 0:
            return None
        return result.stdout.strip() or None
    if backend == "unknown":
        return None

    # bitwarden (default)
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
    return result.stdout.strip() or None


if __name__ == "__main__":
    import sys

    if len(sys.argv) > 1 and sys.argv[1] == "canonical":
        # Print the canonical backend name for a PI_SECRET_BACKEND value.
        print(canonical_backend(sys.argv[2] if len(sys.argv) > 2 else None))
    else:
        # Print the resolved secret (empty when unavailable) for a key name.
        value = resolve_secret(sys.argv[2]) if len(sys.argv) > 2 else None
        print(value if value is not None else "")
