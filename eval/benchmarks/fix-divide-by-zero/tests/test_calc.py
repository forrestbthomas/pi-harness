import sys

sys.path.insert(0, "src")
from calc import safe_divide

assert safe_divide(10, 2) == 5.0
assert safe_divide(1, 0) == 0.0
assert safe_divide(0, 0) == 0.0
print("test_calc: all assertions passed")
