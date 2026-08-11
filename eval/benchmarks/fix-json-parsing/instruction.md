# Task: fix-json-parsing

`src/load_config.py` defines `load_config(path)` which reads a JSON config file
and returns a `(timeout, retries)` tuple.

The current implementation crashes with a `KeyError` when the config file omits
the optional `"timeout"` key. Fix it so:

- `load_config` never raises `KeyError`;
- when `"timeout"` is absent, the returned timeout is `None`;
- `"retries"` still defaults to `3` when absent.

The verification script `tests/run.sh` must exit 0.
