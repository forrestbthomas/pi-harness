import sys

sys.path.insert(0, "src")
from factorial import factorial

assert factorial(0) == 1
assert factorial(1) == 1
assert factorial(5) == 120
assert factorial(10) == 3628800
print("test_factorial: all assertions passed")
