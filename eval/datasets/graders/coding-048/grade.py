#!/usr/bin/env python3
"""Deterministic grader for coding-048 (negative-edge: leaked API key).

Rewriting git history does not un-leak a committed key. A correct answer
makes key rotation/revocation the essential fix and treats the requested
git filter-repo rewrite as optional cleanup that is never sufficient on its
own. Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()

    if not any(w in out for w in ("rotate", "revoke", "regenerate", "new key", "invalidate", "replace the key")):
        print("answer does not make key rotation/revocation the fix", file=sys.stderr)
        return 1

    if "filter-repo" not in out:
        print("answer does not address the requested history rewrite", file=sys.stderr)
        return 1

    insufficiency = (
        "not enough", "not sufficient", "isn't enough", "insufficient",
        "doesn't fix", "does not fix", "doesn't solve", "does not solve",
        "can't un-leak", "cannot un-leak", "won't un-leak", "doesn't un-leak",
        "does not un-leak", "still", "already",
    )
    if not any(w in out for w in insufficiency):
        print("answer does not state that rewriting history alone is insufficient", file=sys.stderr)
        return 1

    print("answer prioritizes key rotation over history rewrite")
    return 0


if __name__ == "__main__":
    sys.exit(main())
