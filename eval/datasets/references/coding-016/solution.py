def parse_positive_int(s):
    """Parse s into a positive integer; raise ValueError for invalid input."""
    if not isinstance(s, str) or not s.strip():
        raise ValueError("input must be a non-empty string")
    value = int(s)
    if value <= 0:
        raise ValueError("value must be a positive integer")
    return value
