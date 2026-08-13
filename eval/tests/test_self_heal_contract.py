"""Contract tests: pi-run self-heal command surface (W1).

self-heal does not spawn pi — it is a pure git-state recovery command — so the
hermetic contract is exercised against real fixture git repos in tmp dirs
(no network, no keys). The process-level behavior (stall detector, group-kill,
escalation packet, exit 9) is covered by Go unit tests; this file pins the
CLI contract: command existence, exit codes, and the abort flag surface.
"""

import subprocess
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]


def _run(pi_run_bin, env, args, cwd):
    return subprocess.run(
        [pi_run_bin, *args], env=env, capture_output=True, text=True, timeout=60, cwd=cwd
    )


def _git(cwd, env, *args):
    subprocess.run(
        ["git", *args], cwd=cwd, env=env, capture_output=True, text=True, timeout=60, check=True
    )


def _git_env(env):
    e = dict(env)
    e.update(
        {
            "GIT_AUTHOR_NAME": "testvalue",
            "GIT_AUTHOR_EMAIL": "t@testvalue",
            "GIT_COMMITTER_NAME": "testvalue",
            "GIT_COMMITTER_EMAIL": "t@testvalue",
            "GIT_EDITOR": "true",
        }
    )
    return e


def _repo_with_conflict(env, tmp_path):
    """Create a repo mid-rebase with a real conflict (feature vs main on f.txt)."""
    _git(tmp_path, env, "init", "-q", ".")
    _git(tmp_path, env, "config", "user.name", "testvalue")
    _git(tmp_path, env, "config", "user.email", "t@testvalue")
    (tmp_path / "f.txt").write_text("base\n")
    _git(tmp_path, env, "add", "f.txt")
    _git(tmp_path, env, "commit", "-qm", "base")
    _git(tmp_path, env, "checkout", "-qb", "feature")
    (tmp_path / "f.txt").write_text("feature\n")
    _git(tmp_path, env, "add", "f.txt")
    _git(tmp_path, env, "commit", "-qm", "feature change")
    _git(tmp_path, env, "checkout", "-q", "main")
    (tmp_path / "f.txt").write_text("mainline\n")
    _git(tmp_path, env, "add", "f.txt")
    _git(tmp_path, env, "commit", "-qm", "main change")
    _git(tmp_path, env, "checkout", "-q", "feature")
    # Rebase feature onto main: f.txt conflicts -> rebase stays in progress.
    subprocess.run(
        ["git", "rebase", "main"],
        cwd=tmp_path,
        env=env,
        capture_output=True,
        text=True,
        timeout=60,
    )  # expected non-zero (conflict)


def test_self_heal_usage_lists_command(pi_run_bin, hermetic_env):
    result = _run(pi_run_bin, hermetic_env(), ["--exit-codes"], cwd=REPO_ROOT)
    assert result.returncode == 0, result.stderr
    # self-heal is a command; exit 9 is the watchdog-terminated code.
    assert "self-heal" in result.stdout or "self-heal" in subprocess.run(
        [pi_run_bin, "help"], env=hermetic_env(), capture_output=True, text=True, timeout=30, cwd=REPO_ROOT
    ).stdout


def test_self_heal_no_git_state_exits_zero(pi_run_bin, hermetic_env, tmp_path):
    result = _run(pi_run_bin, hermetic_env(), ["self-heal"], cwd=tmp_path)
    assert result.returncode == 0, result.stderr
    assert "ok" in result.stdout.lower() or "no git state" in result.stdout.lower()


def test_self_heal_conflict_reports_needs_attention(pi_run_bin, hermetic_env, tmp_path):
    env = _git_env(hermetic_env())
    _repo_with_conflict(env, tmp_path)
    result = _run(pi_run_bin, hermetic_env(), ["self-heal"], cwd=tmp_path)
    assert result.returncode == 1, f"expected needs-attention exit 1, got {result.returncode}: {result.stdout}{result.stderr}"
    assert "needs-attention" in result.stdout.lower()
    assert "f.txt" in result.stdout  # conflict path is reported


def test_self_heal_abort_explicit_only(pi_run_bin, hermetic_env, tmp_path):
    env = _git_env(hermetic_env())
    _repo_with_conflict(env, tmp_path)
    result = _run(pi_run_bin, hermetic_env(), ["self-heal", "--abort"], cwd=tmp_path)
    assert result.returncode == 0, result.stderr
    assert "abort" in result.stdout.lower()
