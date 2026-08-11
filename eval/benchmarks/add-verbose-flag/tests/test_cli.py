import sys
from contextlib import redirect_stdout
from io import StringIO

sys.path.insert(0, "src")
from cli import main


def run(*args):
    buf = StringIO()
    with redirect_stdout(buf):
        main(list(args))
    return buf.getvalue()


out = run("world")
assert "Hello, world!" in out
assert "verbose" not in out.lower()

out = run("--verbose", "world")
assert "Hello, world!" in out
assert "verbose mode on" in out

print("test_cli: all assertions passed")
