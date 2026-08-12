#!/usr/bin/env python3
"""Deterministic grader for coding-020 (harness-routing: MCP version).

When the server announces an older MCP protocol version than the client
prefers, the client must fall back to the server's version.
Exit 0 = pass, exit 1 = fail.
"""

import sys


def main():
    out = sys.stdin.read().lower()
    if "version" not in out:
        print("answer does not discuss the protocol version", file=sys.stderr)
        return 1
    if "server" not in out:
        print("answer does not reference the server's version", file=sys.stderr)
        return 1
    proceed = (
        "fall back",
        "fallback",
        "accept",
        "negotiat",
        "use the server",
        "server's version",
    )
    if not any(word in out for word in proceed):
        print("answer does not say the client proceeds with the server's version", file=sys.stderr)
        return 1
    print("answer falls back to the server's announced protocol version")
    return 0


if __name__ == "__main__":
    sys.exit(main())
