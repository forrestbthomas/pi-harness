"""A minimal HTTP-ish API client with per-client rate limiting.

APIClient never issues more than `max_rps` requests per second.
"""

import time
import urllib.request


class APIClient:
    """Small API client that performs one GET-style request per call."""

    def __init__(self, base_url, max_rps=2):
        self.base_url = base_url.rstrip("/")
        self.max_rps = max_rps
        self._min_interval = 1.0 / max_rps
        self._last_request_at = None

    def _send(self, path):
        """Perform the network request and return the response body as str."""
        with urllib.request.urlopen(self.base_url + path, timeout=5) as resp:
            return resp.read().decode("utf-8")

    def request(self, path):
        """Fetch `path`, rate-limited so at most max_rps requests go out per
        second. Returns the response body as a string.
        """
        now = time.monotonic()
        if self._last_request_at is not None:
            wait = self._min_interval - (now - self._last_request_at)
            if wait > 0:
                time.sleep(wait)
                now = time.monotonic()
        self._last_request_at = now
        return self._send(path)
