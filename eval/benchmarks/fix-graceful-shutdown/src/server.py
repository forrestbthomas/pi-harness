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
        # BUG: the listener socket is never closed, so the accept loop stays
        # blocked in accept() forever; and joining handler threads that are
        # blocked in recv() means shutdown() hangs as long as any client keeps
        # its connection open.
        self._running = False
        for thread in self._threads:
            thread.join()

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.shutdown()
