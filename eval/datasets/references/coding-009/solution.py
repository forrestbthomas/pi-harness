def load_config(path):
    """Return a (timeout, retries) tuple from a JSON config file."""
    import json
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    return data.get("timeout"), data.get("retries", 3)
