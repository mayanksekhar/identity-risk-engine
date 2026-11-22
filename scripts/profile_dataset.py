#!/usr/bin/env python3
"""Aggregate profile of workflows.csv.gz.

Prints counts only. It never prints author names, author emails or any
other row level value, so the output is safe to paste into an issue.
Standard library only.

Usage: python3 scripts/profile_dataset.py [path/to/workflows.csv.gz]
"""
import csv
import gzip
import os
import re
import sys
from collections import Counter

WINDOW_START = "2025-03-26"
WINDOW_END = "2026-03-26"
BOT_PATTERNS = ["dependabot", "renovate", "[bot]", "github-actions",
                "greenkeeper", "snyk-bot"]


def shape(value):
    """Reduce a date string to its format shape, e.g. 9999-99-99T99:99:99+99:99."""
    return re.sub(r"\d", "9", value)


def pct(sorted_vals, p):
    if not sorted_vals:
        return 0
    return sorted_vals[min(len(sorted_vals) - 1, int(len(sorted_vals) * p))]


def main():
    default = os.path.join(os.path.expanduser("~"), "data", "gha-workflows",
                           "workflows.csv.gz")
    path = sys.argv[1] if len(sys.argv) > 1 else default
    csv.field_size_limit(10 ** 9)

    rows = 0
    change = Counter()
    valid_yaml = Counter()
    valid_wf = Counter()
    prob_wf = Counter()
    empty_file_hash = Counter()
    empty_prev_hash = Counter()
    date_shapes = Counter()
    in_window = 0
    out_window = 0
    unparsed_dates = 0
    min_date = None
    max_date = None
    versions = Counter()
    del_uids = set()
    invalid_uids = set()
    bot_rows = Counter()
    bot_any = 0
    repos = set()

    with gzip.open(path, "rt", encoding="utf-8", errors="replace", newline="") as fh:
        reader = csv.DictReader(fh)
        cols = reader.fieldnames or []
        for r in reader:
            rows += 1
            ct = (r.get("git_change_type") or "").strip() or "(empty)"
            change[ct] += 1
            valid_yaml[(r.get("valid_yaml") or "").strip() or "(empty)"] += 1
            valid_wf[(r.get("valid_workflow") or "").strip() or "(empty)"] += 1
            prob_wf[(r.get("probably_workflow") or "").strip() or "(empty)"] += 1
            if not (r.get("file_hash") or "").strip():
                empty_file_hash[ct] += 1
            if not (r.get("previous_file_hash") or "").strip():
                empty_prev_hash[ct] += 1
            uid = r.get("uid") or ""
            versions[uid] += 1
            if ct == "D":
                del_uids.add(uid)
            if (r.get("valid_yaml") or "").strip().lower() not in ("true", "1"):
                invalid_uids.add(uid)
            repos.add(r.get("repository") or "")

            d = (r.get("committed_date") or "").strip()
            date_shapes[shape(d)] += 1
            if re.match(r"\d{4}-\d{2}-\d{2}", d):
                day = d[:10]
                if min_date is None or day < min_date:
                    min_date = day
                if max_date is None or day > max_date:
                    max_date = day
                if WINDOW_START <= day <= WINDOW_END:
                    in_window += 1
                else:
                    out_window += 1
            else:
                unparsed_dates += 1

            ident = ((r.get("author_name") or "") + " " +
                     (r.get("author_email") or "")).lower()
            hit = False
            for p in BOT_PATTERNS:
                if p in ident:
                    bot_rows[p] += 1
                    hit = True
            if hit:
                bot_any += 1

    per = sorted(versions.values())
    print("file:", path)
    print("columns:", ",".join(cols))
    print("rows:", rows)
    print("distinct repositories:", len(repos))
    print("distinct histories (uid):", len(versions))
    print("change types:", dict(change))
    print("valid_yaml:", dict(valid_yaml))
    print("valid_workflow:", dict(valid_wf))
    print("probably_workflow:", dict(prob_wf))
    print("empty file_hash by change type:", dict(empty_file_hash))
    print("empty previous_file_hash by change type:", dict(empty_prev_hash))
    print("committed_date shapes (top 5):", date_shapes.most_common(5))
    print("committed_date min day:", min_date, "max day:", max_date)
    print("rows in window %s to %s: %d" % (WINDOW_START, WINDOW_END, in_window))
    print("rows outside window:", out_window, "unparsed dates:", unparsed_dates)
    print("versions per history p50/p90/p99/max:",
          pct(per, 0.50), pct(per, 0.90), pct(per, 0.99), per[-1] if per else 0)
    print("histories with at least one delete:", len(del_uids))
    print("histories with at least one invalid yaml version:", len(invalid_uids))
    print("rows matching any bot pattern:", bot_any)
    print("bot pattern hits by pattern:", dict(bot_rows))


if __name__ == "__main__":
    main()
