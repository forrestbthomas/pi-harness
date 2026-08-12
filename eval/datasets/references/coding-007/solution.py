def is_palindrome(s):
    """Return True if s is a palindrome, ignoring case and non-alphanumerics."""
    cleaned = "".join(ch for ch in s.lower() if ch.isalnum())
    return cleaned == cleaned[::-1]
