def flatten(arr):
    """Return a flat list of every element of arr, recursing into nested lists."""
    flat = []
    for item in arr:
        if isinstance(item, list):
            flat.extend(flatten(item))
        else:
            flat.append(item)
    return flat
