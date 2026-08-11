"""Safe division helpers."""


def safe_divide(a, b):
    """Divide a by b, returning 0.0 on division by zero."""
    return a / b  # BUG: raises ZeroDivisionError when b == 0
