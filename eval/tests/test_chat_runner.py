"""Hermetic tests for the EVAL-17 chat-session runner (run_pi_session).

These never launch pi — they monkeypatch subprocess.run and assert the
command construction, transcript assembly, failure honesty, and session
isolation, so the runner is verifiable keyless in CI.
"""

import subprocess

import pytest

import conftest


class FakeResult:
    def __init__(self, returncode: int = 0, stdout: str = "", stderr: str = ""):
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


def _capture(monkeypatch, outputs):
    """Monkeypatch subprocess.run to return outputs in order and capture cmds/env."""
    captured = []

    def fake_run(cmd, cwd, text, capture_output, env, timeout):
        captured.append({"cmd": cmd, "cwd": str(cwd), "env": env, "timeout": timeout})
        return outputs.pop(0)

    monkeypatch.setattr(subprocess, "run", fake_run)
    return captured


def test_turn1_uses_print_with_session_id(monkeypatch):
    captured = _capture(monkeypatch, [FakeResult(stdout="first")])
    conftest.run_pi_session(["hello"], session_id="eval-coding-056")
    cmd = captured[0]["cmd"]
    assert cmd[0] == "pi-run" and cmd[1] == "print"
    assert "--session-id" in cmd and "eval-coding-056" in cmd
    assert "--cost-mode" in cmd and "live-eval" in cmd
    assert cmd[-1] == "hello"


def test_turns2_plus_use_resume_with_same_session_id(monkeypatch):
    captured = _capture(
        monkeypatch,
        [FakeResult(stdout="one"), FakeResult(stdout="two"), FakeResult(stdout="three")],
    )
    conftest.run_pi_session(["t1", "t2", "t3"], session_id="eval-coding-056")
    assert captured[0]["cmd"][1] == "print"
    for i in (1, 2):
        assert captured[i]["cmd"][1] == "resume"
        assert "--session" in captured[i]["cmd"]
        assert "eval-coding-056" in captured[i]["cmd"]
        assert captured[i]["cmd"][-1] == f"t{i + 1}"


def test_transcript_joins_turn_outputs(monkeypatch):
    _capture(
        monkeypatch,
        [FakeResult(stdout="first answer"), FakeResult(stdout="second answer")],
    )
    transcript = conftest.run_pi_session(["t1", "t2"], session_id="eval-coding-056")
    assert transcript == "first answer\n\nsecond answer"


def test_agentic_extra_args_apply_to_launch_only(monkeypatch):
    """Launch-only args (--model-tier, --permission-mode) apply to turn 1
    only — resume rejects --model-tier (the session continues as launched)."""
    captured = _capture(
        monkeypatch,
        [FakeResult(stdout="a"), FakeResult(stdout="b")],
    )
    conftest.run_pi_session(
        ["t1", "t2"],
        session_id="eval-coding-056",
        extra_args=["--model-tier", "cheap", "--permission-mode", "plan"],
    )
    launch, resume = captured[0]["cmd"], captured[1]["cmd"]
    assert launch[1] == "print"
    assert "--model-tier" in launch and "cheap" in launch
    assert "--permission-mode" in launch and "plan" in launch
    assert resume[1] == "resume"
    assert "--model-tier" not in resume
    assert "--permission-mode" not in resume
    assert "--session" in resume and "eval-coding-056" in resume


def test_extra_env_flows_to_child(monkeypatch):
    captured = _capture(monkeypatch, [FakeResult(stdout="a")])
    conftest.run_pi_session(["t1"], session_id="eval-coding-056", extra_env={"PI_SELF_HEAL": "1"})
    assert captured[0]["env"].get("PI_SELF_HEAL") == "1"


def test_session_id_isolation_between_cases(monkeypatch):
    captured = _capture(
        monkeypatch,
        [FakeResult(stdout="a"), FakeResult(stdout="b"), FakeResult(stdout="c"), FakeResult(stdout="d")],
    )
    conftest.run_pi_session(["t1", "t2"], session_id="eval-coding-056")
    conftest.run_pi_session(["t1", "t2"], session_id="eval-coding-057")
    ids = [c["cmd"][c["cmd"].index("--session") + 1] if "--session" in c["cmd"] else c["cmd"][c["cmd"].index("--session-id") + 1] for c in captured]
    assert ids == ["eval-coding-056", "eval-coding-056", "eval-coding-057", "eval-coding-057"]


def test_nonzero_exit_fails_honestly(monkeypatch):
    _capture(monkeypatch, [FakeResult(returncode=9, stdout="", stderr="watchdog terminated")])
    with pytest.raises(RuntimeError, match="exit 9"):
        conftest.run_pi_session(["t1"], session_id="eval-coding-056")


def test_empty_stdout_fails_honestly(monkeypatch):
    _capture(monkeypatch, [FakeResult(returncode=0, stdout="   ")])
    with pytest.raises(RuntimeError, match="empty stdout"):
        conftest.run_pi_session(["t1"], session_id="eval-coding-056")


def test_timeout_fails_honestly(monkeypatch):
    def boom(cmd, cwd, text, capture_output, env, timeout):
        raise subprocess.TimeoutExpired(cmd=cmd, timeout=timeout)

    monkeypatch.setattr(subprocess, "run", boom)
    with pytest.raises(RuntimeError, match="timed out"):
        conftest.run_pi_session(["t1"], session_id="eval-coding-056")


def test_run_pi_print_unchanged(monkeypatch):
    """The shared _run_pi refactor must preserve print-mode behavior."""
    captured = _capture(monkeypatch, [FakeResult(stdout="ok")])
    out = conftest.run_pi_print("hi")
    assert out == "ok"
    assert captured[0]["cmd"][0] == "pi-run" and captured[0]["cmd"][1] == "print"
    assert captured[0]["cmd"][-1] == "hi"
