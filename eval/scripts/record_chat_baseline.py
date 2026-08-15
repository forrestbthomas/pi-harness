#!/usr/bin/env python3
"""Record EVAL-17 chat-case baseline entries from the re-baseline report.

Reads /tmp/chat-rebaseline-report.json (5 runs per chat case), computes the
honest per-case stats (passRate, median cost, median tokens, median latency),
and patches eval/baselines/live-baseline.json. A data release (eval lane).
"""

import json
import statistics
import sys
from pathlib import Path

HARNESS = Path(__file__).resolve().parents[2]
BASELINE = HARNESS / "eval" / "baselines" / "live-baseline.json"
CHAT_IDS = ("coding-056", "coding-057", "coding-058")


def per_case_stats(report_path: Path) -> dict:
    report = json.loads(report_path.read_text(encoding="utf-8"))
    stats = {}
    for test in report.get("tests", []):
        nodeid = test.get("nodeid", "")
        case_id = next((cid for cid in CHAT_IDS if cid in nodeid), None)
        if case_id is None or "metrics" in nodeid:
            continue
        props = test.get("user_properties", [])
        runs = []
        cur = {}
        for p in props:
            cur.update(p)
            if "pass" in p:
                runs.append(cur)
                cur = {}
        if not runs:
            continue
        passes = sum(1 for r in runs if r.get("pass"))
        costs = [r.get("costUsd") for r in runs if r.get("costUsd") is not None]
        tokens = [r.get("tokens") for r in runs if r.get("tokens") is not None]
        lat = [r.get("latencyMs") for r in runs if r.get("latencyMs") is not None]
        stats[case_id] = {
            "nRuns": len(runs),
            "passRate": round(passes / len(runs), 4),
            "costPerTaskUsd": round(statistics.median(costs), 9) if costs else None,
            "tokensPerTask": round(statistics.median(tokens)) if tokens else None,
            "latencyMs": round(statistics.median(lat), 1) if lat else None,
        }
    return stats


def main() -> int:
    report_path = Path(sys.argv[1] if len(sys.argv) > 1 else "/tmp/chat-rebaseline-report.json")
    stats = per_case_stats(report_path)
    missing = [cid for cid in CHAT_IDS if cid not in stats]
    if missing:
        print(f"ERROR: no stats for {missing}", file=sys.stderr)
        return 1
    baseline = json.loads(BASELINE.read_text(encoding="utf-8"))
    for cid, s in stats.items():
        if s["nRuns"] != 5:
            print(f"WARNING: {cid} has {s['nRuns']} runs, expected 5", file=sys.stderr)
        entry = baseline["cases"].setdefault(cid, {})
        entry["passRate"] = s["passRate"]
        entry["costPerTaskUsd"] = s["costPerTaskUsd"]
        entry["tokensPerTask"] = s["tokensPerTask"]
        entry["latencyMs"] = s["latencyMs"]
        print(f"{cid}: passRate={s['passRate']} cost=${s['costPerTaskUsd']} tokens={s['tokensPerTask']} lat={s['latencyMs']}ms")
    BASELINE.write_text(json.dumps(baseline, indent=2) + "\n", encoding="utf-8")
    print("baseline patched:", BASELINE)
    return 0


if __name__ == "__main__":
    sys.exit(main())
