"""Minimal threaded TCP echo server with a shutdown() method."""

import socket
import threading


class EchoServer:
    """Accept TCP connections and echo each received line back."""

    def __init__(self, host="127.0.0.1", port=0):
        self._listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self._listener.bind((host, port))
        self._listener.listen(5)
        self.host, self.port = self._listener.getsockname()
        self._threads = []
        self._running = True
        self._accept_thread = threading.Thread(target=self._accept_loop, daemon=True)
        self._accept_thread.start()

    def _accept_loop(self):
        while self._running:
            try:
                conn, _ = self._listener.accept()
            except OSError:
                break
            handler = threading.Thread(target=self._handle, args=(conn,), daemon=True)
            handler.start()
            self._threads.append(handler)

    def _handle(self, conn):
        with conn:
            while True:
                data = conn.recv(4096)
                if not data:
                    break
                conn.sendall(data)

    def shutdown(self):
        """Stop accepting new connections and return promptly.

        Closes the listener socket so the accept loop unblocks and exits, then
        joins it. Connected clients are left untouched: their handler threads
        are daemons that finish the in-flight request on their own, so
        shutdown() never waits on a client that stays connected.
        """
        self._running = False
        self._listener.close()
        self._accept_thread.join(timeout=5)

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.shutdown()
