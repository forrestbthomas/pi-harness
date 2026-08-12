def safe_divide(a, b):
    """Divide a by b, returning 0.0 on division by zero."""
    if b == 0:
        return 0.0
    return a / b
