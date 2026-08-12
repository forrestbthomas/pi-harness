def fibonacci(n):
    """Return the n-th Fibonacci number (fibonacci(0) == 0, fibonacci(1) == 1)."""
    if n < 0:
        raise ValueError("fibonacci requires a non-negative integer")
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a
