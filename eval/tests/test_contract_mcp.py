"""Contract tests: pi-run mcp-server JSON-RPC 2.0 over stdio (hermetic).

A minimal stdlib raw JSON-RPC client drives the real binary via
subprocess.Popen with select-bounded reads. stdout is a protocol surface:
every line must parse as JSON-RPC 2.0 and stderr stays empty on the happy
path (the stdout-is-sacred rule).
"""

import json
import os
import re
import select
import subprocess
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[2]

INITIALIZE_PARAMS = {
    "protocolVersion": "2025-03-26",
    "capabilities": {},
}


class MCPClient:
    """Line-delimited JSON-RPC 2.0 client over `pi-run mcp-server` stdio."""

    def __init__(self, binary, env):
        self._buf = b""
        self.proc = subprocess.Popen(
            [binary, "mcp-server"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env,
        )

    def send(self, payload: str) -> None:
        data = payload.encode("utf-8")
        if not data.endswith(b"\n"):
            data += b"\n"
        self.proc.stdin.write(data)
        self.proc.stdin.flush()

    def read_line(self, timeout: float = 5.0):
        """Read one complete line, or None on timeout/EOF (select-bounded).

        Reads from the raw pipe fd into an internal buffer so partial reads
        never get stranded in a BufferedReader while select says not-ready.
        """
        buf = self._buf
        while b"\n" not in buf:
            ready, _, _ = select.select([self.proc.stdout], [], [], timeout)
            if not ready:
                self._buf = buf
                return None
            chunk = os.read(self.proc.stdout.fileno(), 4096)
            if not chunk:  # EOF
                self._buf = buf
                return None
            buf += chunk
        line, rest = buf.split(b"\n", 1)
        self._buf = rest
        return line.decode("utf-8", "replace").strip()

    def request(self, method: str, msg_id=None, params=None) -> dict:
        message = {"jsonrpc": "2.0"}
        if msg_id is not None:
            message["id"] = msg_id
        message["method"] = method
        if params is not None:
            message["params"] = params
        self.send(json.dumps(message))
        line = self.read_line()
        assert line is not None, f"no response line for {method}"
        return json.loads(line)

    def drain(self, timeout: float = 5.0) -> list[str]:
        lines = []
        while True:
            line = self.read_line(timeout)
            if line is None:
                break
            lines.append(line)
        return lines

    def close(self, timeout: float = 10.0) -> int:
        if self.proc.stdin:
            try:
                self.proc.stdin.close()
            except BrokenPipeError:
                pass
        try:
            return self.proc.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            self.proc.kill()
            return self.proc.wait(timeout=5)


@pytest.fixture
def mcp_client(pi_run_bin, hermetic_env):
    client = MCPClient(pi_run_bin, hermetic_env())
    yield client
    client.close()


def _conftest_provider_keys() -> set[str]:
    """SUPPORTED_PROVIDER_KEYS tuple from eval/conftest.py (regex, no import).

    Mirrors the Go drift guard TestMCPProvidersKeyEnvMatchesPythonMirror so
    the two sides of the mirror are checked independently.
    """
    text = (REPO_ROOT / "eval" / "conftest.py").read_text(encoding="utf-8")
    match = re.search(r"SUPPORTED_PROVIDER_KEYS\s*=\s*\((.*?)\)", text, re.S)
    assert match, "SUPPORTED_PROVIDER_KEYS tuple not found in eval/conftest.py"
    return set(re.findall(r'"([A-Z0-9_]+)"', match.group(1)))


def test_mcp_initialize_handshake(mcp_client):
    resp = mcp_client.request("initialize", msg_id=1, params=INITIALIZE_PARAMS)
    assert resp["jsonrpc"] == "2.0"
    assert resp["id"] == 1  # echoed verbatim
    result = resp["result"]
    assert result["protocolVersion"] == "2025-03-26"
    assert result["serverInfo"]["name"] == "pi-run"
    assert result["capabilities"]["tools"]["listChanged"] is False


def test_mcp_initialized_notification_silent(mcp_client):
    mcp_client.send('{"jsonrpc":"2.0","method":"notifications/initialized"}')
    resp = mcp_client.request("tools/list", msg_id=2)
    assert resp["id"] == 2
    assert "error" not in resp
    # The notification produced no response line: nothing may follow the
    # tools/list response within a short silence window.
    assert mcp_client.read_line(timeout=0.5) is None


def test_mcp_tools_list(mcp_client):
    resp = mcp_client.request("tools/list", msg_id=3)
    tools = resp["result"]["tools"]
    assert [tool["name"] for tool in tools] == ["providers", "cost", "benchmark_dry_run"]
    for tool in tools:
        assert tool.get("description")


def test_mcp_call_providers(mcp_client):
    resp = mcp_client.request(
        "tools/call", msg_id=4, params={"name": "providers", "arguments": {}}
    )
    assert resp["result"].get("isError", False) is False
    text = resp["result"]["content"][0]["text"]
    entries = json.loads(text)
    assert isinstance(entries, list)
    for entry in entries:
        assert entry.get("name")
        assert entry.get("defaultModel")
        assert entry.get("keyEnv")
    by_name = {entry["name"]: entry for entry in entries}
    assert by_name["openai"]["keyEnv"] == "OPENAI_API_KEY"


def test_mcp_providers_keyenv_matches_conftest(mcp_client):
    resp = mcp_client.request(
        "tools/call", msg_id=5, params={"name": "providers", "arguments": {}}
    )
    entries = json.loads(resp["result"]["content"][0]["text"])
    tool_keys = {entry["keyEnv"] for entry in entries}
    assert tool_keys == _conftest_provider_keys()


def test_mcp_unknown_tool_is_tool_error(mcp_client):
    resp = mcp_client.request("tools/call", msg_id=6, params={"name": "bogus_tool"})
    assert "error" not in resp  # tool-level failure, never a JSON-RPC error
    assert resp["result"]["isError"] is True
    assert "unknown tool" in resp["result"]["content"][0]["text"]
    # The server keeps serving: a ping still answers.
    ping = mcp_client.request("ping", msg_id=7)
    assert "error" not in ping


def test_mcp_unknown_method_jsonrpc_error(mcp_client):
    resp = mcp_client.request("bogus/method", msg_id=8)
    assert resp["error"]["code"] == -32601
    assert "result" not in resp
    ping = mcp_client.request("ping", msg_id=9)
    assert "error" not in ping


def test_mcp_malformed_line_jsonrpc_error(mcp_client):
    mcp_client.send("this is not json at all")
    line = mcp_client.read_line()
    assert line is not None, "no response for the malformed line"
    resp = json.loads(line)
    assert resp["error"]["code"] == -32700
    assert resp["id"] is None
    ping = mcp_client.request("ping", msg_id=10)
    assert "error" not in ping


def test_mcp_eof_exits_zero(mcp_client):
    resp = mcp_client.request("initialize", msg_id=1, params=INITIALIZE_PARAMS)
    assert resp["result"]["protocolVersion"] == "2025-03-26"
    # EOF on stdin must end the server with exit 0 (the runMCPServer contract).
    assert mcp_client.close() == 0


def test_mcp_stdout_is_sacred(mcp_client):
    sequence = [
        ('{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{}}}', True),
        ('{"jsonrpc":"2.0","method":"notifications/initialized"}', False),
        ('{"jsonrpc":"2.0","id":2,"method":"tools/list"}', True),
        ('{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"providers","arguments":{}}}', True),
        ('{"jsonrpc":"2.0","id":4,"method":"ping"}', True),
        ("this is not json at all", True),
        ('{"jsonrpc":"2.0","id":5,"method":"bogus/method"}', True),
        ('{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"bogus_tool"}}', True),
    ]
    responses = []
    for payload, expects_response in sequence:
        mcp_client.send(payload)
        if expects_response:
            line = mcp_client.read_line()
            assert line is not None, f"no response for {payload}"
            responses.append(json.loads(line))
    assert mcp_client.close() == 0
    responses.extend(json.loads(line) for line in mcp_client.drain())
    # stdout is a protocol surface: every line is JSON-RPC 2.0.
    for resp in responses:
        assert resp["jsonrpc"] == "2.0"
    # stderr stays empty on the happy path.
    stderr = mcp_client.proc.stderr.read().decode("utf-8", "replace")
    assert stderr == ""
