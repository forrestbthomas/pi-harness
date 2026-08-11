"""Load a JSON config file, tolerating missing optional keys."""

import json


def load_config(path):
    """Return a (timeout, retries) tuple from a JSON config file."""
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    # BUG: raises KeyError when 'timeout' is absent
    return data["timeout"], data.get("retries", 3)
