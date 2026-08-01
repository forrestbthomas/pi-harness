from __future__ import annotations

from typing import Any


def grade_case(case: dict[str, Any], response: str) -> dict[str, Any]:
    """Score one response using transparent, deterministic string signals."""
    normalized_response = response.casefold()
    required = [str(signal) for signal in case["required"]]
    forbidden = [str(signal) for signal in case["forbidden"]]
    missing = [
        signal for signal in required if signal.casefold() not in normalized_response
    ]
    forbidden_found = [
        signal
        for signal in forbidden
        if signal.casefold() in normalized_response
    ]

    return {
        "id": str(case["id"]),
        "passed": not missing and not forbidden_found,
        "missing": missing,
        "forbidden_found": forbidden_found,
    }
