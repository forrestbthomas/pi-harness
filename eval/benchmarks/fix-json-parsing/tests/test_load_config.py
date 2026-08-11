import json
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, "src")
from load_config import load_config


def write_config(data):
    tmp = tempfile.TemporaryDirectory()
    path = Path(tmp.name) / "cfg.json"
    path.write_text(json.dumps(data), encoding="utf-8")
    return tmp, str(path)


tmp, path = write_config({"timeout": 30})
timeout, retries = load_config(path)
assert timeout == 30
assert retries == 3

tmp2, path2 = write_config({})
timeout, retries = load_config(path2)
assert timeout is None, f"expected None, got {timeout!r}"
assert retries == 3

tmp3, path3 = write_config({"retries": 7})
timeout, retries = load_config(path3)
assert timeout is None
assert retries == 7

print("test_load_config: all assertions passed")
