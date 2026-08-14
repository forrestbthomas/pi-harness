"""A minimal HTTP-ish API client.

Requests are sent immediately, with no throttling.
"""

import urllib.request


class APIClient:
    """Small API client that performs one GET-style request per call."""

    def __init__(self, base_url, max_rps=2):
        self.base_url = base_url.rstrip("/")
        self.max_rps = max_rps

    def _send(self, path):
        """Perform the network request and return the response body as str."""
        with urllib.request.urlopen(self.base_url + path, timeout=5) as resp:
            return resp.read().decode("utf-8")

    def request(self, path):
        """Fetch `path` from the API. Returns the response body as a string."""
        return self._send(path)
