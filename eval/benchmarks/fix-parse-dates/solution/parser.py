"""Timestamp parsing helpers."""

from datetime import datetime, timezone


def parse_ts(s):
    """Parse an ISO-8601 timestamp into a naive UTC datetime.

    Naive timestamps like "2024-01-01T10:00:00" are returned unchanged;
    offset timestamps like "2024-01-01T12:00:00+02:00" are normalized to
    UTC. Raises ValueError on unparseable input.
    """
    dt = datetime.fromisoformat(s)
    if dt.tzinfo is None:
        return dt
    return dt.astimezone(timezone.utc).replace(tzinfo=None)
