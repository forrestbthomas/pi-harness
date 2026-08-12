"""Contract tests: pi-run project-understand (determinism, hygiene, exit codes).

The generator has no time.Now()/randomness (internal/cli/project_understand.go),
so two runs must produce byte-identical docs. All runs are hermetic: a fixture
project under a tmp HARNESS_ROOT, tmp output dirs.
"""

import subprocess
from pathlib import Path

import pytest

DOCS = ("product.md", "tech.md", "structure.md")


@pytest.fixture
def understand_project(tmp_path):
    """Fixture repo mirroring internal/cli/project_understand_test.go."""
    root = tmp_path / "project"

    def must_write(rel, content):
        path = root / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    must_write("go.mod", "module example.com/demo\n\ngo 1.26\n")
    must_write(
        "README.md",
        "# Demo Project\n\nThe pi-harness demo project understands things with style.\n\nMore details below.\n",
    )
    must_write("main.go", "package main\n\nfunc main() {\n\t// entry point\n\tprintln(\"hello\")\n}\n")
    must_write("util.go", "package main\n\n// util counts things.\nfunc util() int {\n\treturn 42\n}\n")
    must_write("util_test.go", "package main\n\nfunc TestUtil(t *testing.T) {\n\t_ = util()\n}\n")
    must_write(
        "cmd/tool/tool.py",
        "#!/usr/bin/env python3\n\ndef main() -> None:\n    print(\"tool\")\n\n\nif __name__ == \"__main__\":\n    main()\n",
    )
    must_write("node_modules/junk/index.js", "// junk\n" + "const x = 1;\n" * 200)
    must_write(
        ".github/workflows/ci.yml",
        "name: ci\non: [push]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go test ./...\n",
    )
    must_write(".env", "LOCAL_CONFIG=1\n")
    return root


def _run(pi_run_bin, env, args):
    return subprocess.run(
        [pi_run_bin, *args], env=env, capture_output=True, text=True, timeout=30
    )


def test_project_understand_writes_three_docs(
    pi_run_bin, hermetic_env, understand_project, tmp_path
):
    env = hermetic_env(harness_root=understand_project)
    out = tmp_path / "out"
    result = _run(pi_run_bin, env, ["project-understand", "--out", str(out)])
    assert result.returncode == 0, result.stderr
    assert "wrote product.md, tech.md, structure.md" in result.stdout
    product = (out / "product.md").read_text(encoding="utf-8")
    tech = (out / "tech.md").read_text(encoding="utf-8")
    structure = (out / "structure.md").read_text(encoding="utf-8")
    # The same markers the Go test checks (project_understand_test.go).
    assert "The pi-harness demo project understands things with style." in product
    assert "**Primary stack:** Go, Python" in product
    assert "Go: 14 lines (3 file(s))" in tech
    assert "Python: 5 lines" in tech
    assert "GitHub Actions" in tech
    for marker in (".github/", "cmd/", "tool/", "README.md", "go.mod"):
        assert marker in structure
    assert "Go module: example.com/demo" in structure
    assert "node_modules" not in structure


def test_project_understand_deterministic_reruns(
    pi_run_bin, hermetic_env, understand_project, tmp_path
):
    env = hermetic_env(harness_root=understand_project)
    out = tmp_path / "out"
    first = _run(pi_run_bin, env, ["project-understand", "--out", str(out)])
    assert first.returncode == 0, first.stderr
    first_bytes = {name: (out / name).read_bytes() for name in DOCS}
    second = _run(pi_run_bin, env, ["project-understand", "--out", str(out)])
    assert second.returncode == 0, second.stderr
    for name, data in first_bytes.items():
        assert (out / name).read_bytes() == data, f"{name} differs across reruns"


def test_project_understand_output_hygiene(
    pi_run_bin, hermetic_env, understand_project, tmp_path
):
    env = hermetic_env(harness_root=understand_project)
    out = tmp_path / "out"
    result = _run(pi_run_bin, env, ["project-understand", "--out", str(out)])
    assert result.returncode == 0, result.stderr
    all_text = "\n".join((out / name).read_text(encoding="utf-8") for name in DOCS)
    assert str(understand_project) not in all_text  # no absolute tmp path
    assert str(tmp_path) not in all_text
    assert "test" + "-key" not in all_text  # no placeholder secret value


def test_project_understand_usage_error_exit2(pi_run_bin, hermetic_env, understand_project):
    env = hermetic_env(harness_root=understand_project)
    result = _run(pi_run_bin, env, ["project-understand", "--bogus"])
    assert result.returncode == 2
    assert "unknown flag" in result.stderr


def test_project_understand_default_out(pi_run_bin, hermetic_env, understand_project):
    env = hermetic_env(harness_root=understand_project)
    result = _run(pi_run_bin, env, ["project-understand"])
    assert result.returncode == 0, result.stderr
    out = understand_project / "docs" / "understand"
    for name in DOCS:
        assert (out / name).is_file(), f"default out missing {name}"
