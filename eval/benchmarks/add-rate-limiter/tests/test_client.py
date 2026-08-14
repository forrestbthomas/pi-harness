"""Hidden tests for the rate-limited APIClient in src/client.py.

Covers:
  * throttle: bursts are paced so at most max_rps requests go out per second
  * regression: ordinary requests still work (URL built, body returned)
  * regression: the configured max_rps limit is honored (high limit = fast)
  * regression: network errors still propagate through request()
"""

import sys
import time
import urllib.request

sys.path.insert(0, "src")

from client import APIClient

_ORIGINAL_URLOPEN = urllib.request.urlopen


class FakeResponse:
    """Minimal stand-in for an http.client.HTTPResponse."""

    def __init__(self, body):
        self._body = body.encode("utf-8")

    def read(self):
        return self._body

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False


class RecordingTransport:
    """Replaces urllib.request.urlopen; records call times and URLs."""

    def __init__(self):
        self.calls = []  # (monotonic timestamp, url)

    def __call__(self, url, **kwargs):
        self.calls.append((time.monotonic(), url))
        if url.endswith("/boom"):
            raise OSError("simulated network error")
        return FakeResponse(url)


def _client(transport, **kwargs):
    client = APIClient("https://api.example.com", **kwargs)
    urllib.request.urlopen = transport
    return client


def _restore():
    urllib.request.urlopen = _ORIGINAL_URLOPEN


def test_throttle_paces_bursts():
    transport = RecordingTransport()
    client = _client(transport, max_rps=2)
    try:
        start = time.monotonic()
        for i in range(5):
            client.request("/item/%d" % i)
        elapsed = time.monotonic() - start
    finally:
        _restore()
    assert len(transport.calls) == 5
    # Correct pacing (0.5s between calls) takes ~2.0s; require at least 75%.
    assert elapsed >= 1.5, "5 requests at max_rps=2 should take ~2.0s, got %.3fs" % elapsed


def test_ordinary_requests_unchanged():
    transport = RecordingTransport()
    client = _client(transport, max_rps=2)
    try:
        assert client.request("/users") == "https://api.example.com/users"
        assert client.request("/posts/1") == "https://api.example.com/posts/1"
    finally:
        _restore()
    urls = [url for _, url in transport.calls]
    assert urls == [
        "https://api.example.com/users",
        "https://api.example.com/posts/1",
    ]


def test_high_limit_is_not_slowed_down():
    transport = RecordingTransport()
    client = _client(transport, max_rps=100)
    try:
        start = time.monotonic()
        for i in range(3):
            client.request("/ping/%d" % i)
        elapsed = time.monotonic() - start
    finally:
        _restore()
    assert len(transport.calls) == 3
    assert elapsed < 0.5, "max_rps=100 should not pace 3 requests, got %.3fs" % elapsed


def test_network_errors_still_propagate():
    transport = RecordingTransport()
    client = _client(transport, max_rps=2)
    raised = None
    try:
        try:
            client.request("/boom")
        except OSError as exc:
            raised = exc
    finally:
        _restore()
    assert raised is not None, "request() must not swallow network errors"


if __name__ == "__main__":
    test_throttle_paces_bursts()
    test_ordinary_requests_unchanged()
    test_high_limit_is_not_slowed_down()
    test_network_errors_still_propagate()
    print("test_client: all assertions passed")
