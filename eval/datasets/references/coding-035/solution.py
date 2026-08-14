def merge_dicts(a, b):
    """Return a new dict merging b into a (b's keys win); inputs are not modified."""
    merged = dict(a)
    merged.update(b)
    return merged
