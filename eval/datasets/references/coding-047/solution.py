def process_file(path):
    """Return the stripped lines of a text file; exit 1 with a clear error
    message if the file is missing or the path is empty/None."""
    import sys

    if not path:
        print(f"error: no such file: {path}", file=sys.stderr)
        sys.exit(1)
    try:
        with open(path, encoding="utf-8") as f:
            return [line.rstrip("\n") for line in f]
    except FileNotFoundError:
        print(f"error: no such file: {path}", file=sys.stderr)
        sys.exit(1)
