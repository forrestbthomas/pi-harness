def merge_dicts(a, b):
    """Return a new dict with keys from a and b; b wins on conflicts."""
    merged = dict(a)
    merged.update(b)
    return merged
