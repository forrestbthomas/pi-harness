import sys
from datetime import datetime

sys.path.insert(0, "src")
from parser import parse_ts


def expect_value_error(s):
    try:
        parse_ts(s)
    except ValueError:
        return
    raise AssertionError("parse_ts(%r) must raise ValueError" % s)


# --- Fix: offset timestamps must normalize to a naive UTC datetime ---
assert parse_ts("2024-01-01T12:00:00+02:00") == datetime(2024, 1, 1, 10, 0, 0)
assert parse_ts("2024-01-01T12:00:00+02:00").tzinfo is None
assert parse_ts("2024-01-01T08:00:00-05:00") == datetime(2024, 1, 1, 13, 0, 0)
assert parse_ts("2024-01-01T08:00:00-05:00").tzinfo is None
assert parse_ts("2024-01-01T10:00:00Z") == datetime(2024, 1, 1, 10, 0, 0)
assert parse_ts("2024-01-01T10:00:00Z").tzinfo is None
assert parse_ts("2024-12-31T23:30:00+05:30") == datetime(2024, 12, 31, 18, 0, 0)

# --- Sibling behavior: naive timestamps keep working ---
assert parse_ts("2024-01-01T10:00:00") == datetime(2024, 1, 1, 10, 0, 0)
assert parse_ts("2024-01-01T10:00:00").tzinfo is None
assert parse_ts("2024-12-31T23:59:59") == datetime(2024, 12, 31, 23, 59, 59)

# --- Sibling behavior: unparseable input still raises ValueError ---
expect_value_error("not-a-date")
expect_value_error("")
expect_value_error("2024-13-01T10:00:00")
expect_value_error("2024-01-01T25:00:00")
expect_value_error("2024-01-01T10:00:00+02:00 extra")

print("test_parser: all assertions passed")
