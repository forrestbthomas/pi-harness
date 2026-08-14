"""Hidden tests for the graceful-shutdown fix in src/server.py.

Covers the fix (shutdown() returns promptly while clients are connected) and
the sibling behavior that must keep working:
  - the server accepts and echoes before shutdown
  - multiple clients are served concurrently
  - after shutdown, the listener refuses new connections
  - in-flight requests started before shutdown still complete afterwards
  - the accept loop actually terminates
"""

import socket
import sys
import threading
import time

sys.path.insert(0, "src")
from server import EchoServer

SHUTDOWN_LIMIT = 2.0  # seconds


def recv_line(conn):
    data = b""
    while not data.endswith(b"\n"):
        chunk = conn.recv(4096)
        if not chunk:
            break
        data += chunk
    return data


def test_echo_before_shutdown(server):
    with socket.create_connection((server.host, server.port), timeout=2) as client:
        client.sendall(b"hello\n")
        assert recv_line(client) == b"hello\n"
        client.sendall(b"again\n")
        assert recv_line(client) == b"again\n"


def test_shutdown_prompt_with_inflight(server):
    with socket.create_connection((server.host, server.port), timeout=2) as client_a:
        with socket.create_connection((server.host, server.port), timeout=2) as client_b:
            client_a.sendall(b"ping\n")
            client_b.sendall(b"pong\n")
            assert recv_line(client_a) == b"ping\n"
            assert recv_line(client_b) == b"pong\n"

            # shutdown() is called while both clients are still connected and idle.
            result = {}

            def call_shutdown():
                start = time.monotonic()
                server.shutdown()
                result["elapsed"] = time.monotonic() - start

            worker = threading.Thread(target=call_shutdown, daemon=True)
            worker.start()
            worker.join(timeout=SHUTDOWN_LIMIT)
            assert not worker.is_alive(), (
                "shutdown() did not return within %.1fs" % SHUTDOWN_LIMIT
            )

            # In-flight connections must still be able to finish their requests.
            client_a.sendall(b"after-a\n")
            client_b.sendall(b"after-b\n")
            assert recv_line(client_a) == b"after-a\n"
            assert recv_line(client_b) == b"after-b\n"

            # New connections must now be refused: the listener is closed.
            try:
                socket.create_connection((server.host, server.port), timeout=1)
            except OSError:
                pass
            else:
                raise AssertionError(
                    "server still accepts connections after shutdown()"
                )


def test_accept_loop_terminates(server):
    server._accept_thread.join(timeout=5)
    assert not server._accept_thread.is_alive(), "accept loop did not exit"


def main():
    server = EchoServer()
    try:
        test_echo_before_shutdown(server)
        test_shutdown_prompt_with_inflight(server)
        test_accept_loop_terminates(server)
    finally:
        # Best-effort cleanup that can never hang the harness.
        cleanup = threading.Thread(target=server.shutdown, daemon=True)
        cleanup.start()
        cleanup.join(timeout=SHUTDOWN_LIMIT)
    print("test_echo_shutdown: all assertions passed")


if __name__ == "__main__":
    main()
