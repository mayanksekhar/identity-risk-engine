#!/usr/bin/env python3
"""Frozen estimands from protocol section 12.10.

Reads histories.csv and events.csv written by extract_features.py.
Computes, for each variant: drift rate per history per 90 days, prevalence,
visible growth rate, and the drift to visible growth ratio, each with a 95
percent percentile interval from a cluster bootstrap by repository.
Prints aggregates only and writes results.csv next to the inputs.

Variants:
  primary        third party reach (official owners actions and github removed)
  all_external   every external owner counts as reach
  no_ties        primary, dropping histories with same second commit ties
                 (needs --csv, the original workflows.csv.gz)

Usage:
  python3 scripts/analyze_drift.py DIR [--csv PATH] [--boot 10000] [--seed S]
"""
import argparse
import csv
import gzip
import os
import sys

import numpy as np


def tied_uids(path, wanted):
    csv.field_size_limit(10 ** 9)
    seen = {}
    tied = set()
    with gzip.open(path, "rt", encoding="utf-8", errors="replace", newline="") as fh:
        for r in csv.DictReader(fh):
            u = r["uid"]
            if u not in wanted:
                continue
            ep = r["committed_date"]
            s = seen.setdefault(u, set())
            if ep in s:
                tied.add(u)
            else:
                s.add(ep)
    return tied


def build(hist, ev, scope, drop):
    """Return per repository arrays: histories, exposure, drift events,
    histories with a drift event, visible events."""
    repos = {}
    for h in hist:
        if h["uid"] in drop:
            continue
        r = repos.setdefault(h["repository"], [0, 0.0, 0, set(), 0])
        r[0] += 1
        r[1] += float(h["exposure_days"])
    for e in ev:
        if e["uid"] in drop or e["is_bot"] == "1":
            continue
        r = repos.get(e["repository"])
        if r is None:
            continue
        gain = int(e["gain"])
        if scope == "third_party":
            gain -= int(e["gain_official"])
        if gain <= 0:
            continue
        if e["tier_dir"] == "none":
            r[2] += 1
            r[3].add(e["uid"])
        elif e["tier_dir"] == "up":
            r[4] += 1
    keys = sorted(repos)
    a = np.array([[repos[k][0], repos[k][1], repos[k][2], len(repos[k][3]),
                   repos[k][4]] for k in keys], dtype=float)
    return a


def estimates(a):
    n, days, drift, drift_h, vis = a.sum(axis=0)
    out = {
        "drift_rate_per_90d": drift / (days / 90.0) if days else float("nan"),
        "prevalence": drift_h / n if n else float("nan"),
        "visible_rate_per_90d": vis / (days / 90.0) if days else float("nan"),
        "drift_to_visible": drift / vis if vis else float("nan"),
    }
    return out


def bootstrap(a, B, seed):
    rng = np.random.default_rng(seed)
    n = len(a)
    res = {k: [] for k in ("drift_rate_per_90d", "prevalence",
                           "visible_rate_per_90d", "drift_to_visible")}
    for _ in range(B):
        idx = rng.integers(0, n, size=n)
        e = estimates(a[idx])
        for k, v in e.items():
            res[k].append(v)
    ci = {}
    for k, v in res.items():
        arr = np.array(v, dtype=float)
        arr = arr[~np.isnan(arr)]
        if len(arr) < 0.9 * B:
            ci[k] = (float("nan"), float("nan"))
        else:
            ci[k] = (float(np.percentile(arr, 2.5)), float(np.percentile(arr, 97.5)))
    return ci


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("dir")
    ap.add_argument("--csv", default="")
    ap.add_argument("--boot", type=int, default=10000)
    ap.add_argument("--seed", type=int, default=20261123)
    a = ap.parse_args()
    hist = list(csv.DictReader(open(os.path.join(a.dir, "histories.csv"))))
    ev = list(csv.DictReader(open(os.path.join(a.dir, "events.csv"))))
    variants = [("primary", "third_party", set()),
                ("all_external", "all", set())]
    if a.csv:
        tied = tied_uids(a.csv, {h["uid"] for h in hist})
        print("histories with timestamp ties among exposed:", len(tied))
        variants.append(("no_ties", "third_party", tied))

    rows = []
    for name, scope, drop in variants:
        arr = build(hist, ev, scope, drop)
        est = estimates(arr)
        ci = bootstrap(arr, a.boot, a.seed)
        n, days, drift, drift_h, vis = arr.sum(axis=0)
        print("\nvariant:", name)
        print("  repositories: %d  histories: %d  exposure history years: %.1f" %
              (len(arr), n, days / 365.25))
        print("  drift events: %d  visible growth events: %d" % (drift, vis))
        for k, v in est.items():
            lo, hi = ci[k]
            print("  %-22s %.4f   95%% CI %.4f to %.4f" % (k, v, lo, hi))
            rows.append([name, k, v, lo, hi])
    with open(os.path.join(a.dir, "results.csv"), "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["variant", "metric", "estimate", "ci_low", "ci_high"])
        w.writerows(rows)


if __name__ == "__main__":
    main()
