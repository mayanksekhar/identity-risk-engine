#!/usr/bin/env python3
"""Aggregate summary of derived tables. Prints counts only, no repository names."""
import csv
import os
import sys
from collections import Counter


def main():
    d = sys.argv[1] if len(sys.argv) > 1 else os.path.expanduser(
        "~/data/gha-workflows/derived/pilot")
    hist = list(csv.DictReader(open(os.path.join(d, "histories.csv"))))
    ev = list(csv.DictReader(open(os.path.join(d, "events.csv"))))

    def kind(e):
        if e["is_bot"] == "1":
            return "bot"
        gain = int(e["gain"]) > 0
        loss = int(e["loss"]) > 0
        t = e["tier_dir"]
        if gain and t == "none":
            return "drift"
        if gain and t == "up":
            return "visible_growth"
        if gain:
            return "gain_tier_down"
        if loss and t == "none":
            return "loss_only"
        if t == "up":
            return "tier_up_only"
        if t == "down":
            return "tier_down_only"
        return "other"

    for e in ev:
        e["kind"] = kind(e)
    kinds = Counter(e["kind"] for e in ev)
    print("histories with exposure:", len(hist))
    print("repositories with exposure:", len({h["repository"] for h in hist}))
    print("events by kind:", dict(kinds))

    drift = [e for e in ev if e["kind"] == "drift"]
    vis = [e for e in ev if e["kind"] == "visible_growth"]
    print("histories with at least one drift event:", len({e["uid"] for e in drift}))
    print("histories with at least one visible growth event:", len({e["uid"] for e in vis}))
    print("repositories with at least one drift event:", len({e["repository"] for e in drift}))

    per_repo = Counter(e["repository"] for e in drift)
    top = sorted(per_repo.values(), reverse=True)
    print("drift events per repository, top 5 counts:", top[:5])
    if drift:
        print("share of drift events from top 5 repositories: %.1f%%" %
              (100.0 * sum(top[:5]) / len(drift)))
    per_hist = Counter(e["uid"] for e in drift)
    print("drift events per history, top 5 counts:", sorted(per_hist.values(), reverse=True)[:5])

    sizes = Counter()
    for e in drift:
        g = int(e["gain"])
        sizes["1" if g == 1 else "2" if g == 2 else "3 to 5" if g <= 5 else "6 plus"] += 1
    print("actions gained per drift event:", dict(sizes))
    pins = {k: sum(int(e["gain_" + k]) for e in drift) for k in ("sha", "full", "major", "branch")}
    print("pin type of gained actions in drift events:", pins)
    print("gained official owner actions in drift events:",
          sum(int(e["gain_official"]) for e in drift),
          "of", sum(int(e["gain"]) for e in drift))
    exp = sorted(float(h["exposure_days"]) for h in hist)
    if exp:
        print("exposure days p10/p50/p90:", exp[len(exp) // 10], exp[len(exp) // 2],
              exp[(len(exp) * 9) // 10])
    print("histories with unparsed versions:", sum(1 for h in hist if int(h["unparsed_versions"]) > 0))
    print("histories with zero in window versions:", sum(1 for h in hist if int(h["in_window_versions"]) == 0))


if __name__ == "__main__":
    main()
