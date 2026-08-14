def parse_csv_line(line):
    """Parse a CSV line into a list of fields, honoring double-quoted fields."""
    import csv
    import io
    return next(csv.reader(io.StringIO(line)))
