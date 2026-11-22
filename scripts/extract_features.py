#!/usr/bin/env python3
"""Extract reach and tier features from workflow histories.

Reads workflows.csv.gz (metadata) and workflows.tar.gz (file contents),
then writes two derived tables:
  histories.csv  one row per history with exposure inside the window
  events.csv     one row per in-window change that altered reach or tier
No author names or emails are written. Only a bot flag is kept.

Usage:
  python3 scripts/extract_features.py --out OUTDIR [--pilot N] [--seed S]
         [--csv PATH] [--tar PATH]
Without --pilot every history is processed.
"""
import argparse
import csv
import gzip
import os
import random
import re
import sys
import tarfile
from datetime import datetime, timezone

import yaml

try:
    Loader = yaml.CSafeLoader
except AttributeError:
    Loader = yaml.SafeLoader

W0 = int(datetime(2025, 3, 26, tzinfo=timezone.utc).timestamp())
W1 = int(datetime(2026, 3, 26, 23, 59, 59, tzinfo=timezone.utc).timestamp())
BOT_PATTERNS = ["dependabot", "renovate", "[bot]", "github-actions",
                "greenkeeper", "snyk-bot"]
OFFICIAL = {"actions", "github"}
PIN_RANK = {"branch": 0, "major": 1, "full": 2, "sha": 3}
LEVEL = {"none": 0, "read": 1, "default": 2, "write": 3}

SHA_RE = re.compile(r"^[0-9a-f]{40}$")
FULL_RE = re.compile(r"^v?\d+\.\d+")
MAJOR_RE = re.compile(r"^v?\d+$")


def pin_type(ref):
    if SHA_RE.match(ref):
        return "sha"
    if FULL_RE.match(ref):
        return "full"
    if MAJOR_RE.match(ref):
        return "major"
    return "branch"


def perm_level(p):
    """Return (level, idtoken_write) for one permissions value."""
    if p is None:
        return "default", False
    if isinstance(p, str):
        s = p.strip().lower()
        if s == "write-all":
            return "write", True
        if s == "read-all":
            return "read", False
        return "default", False
    if isinstance(p, dict):
        vals = {str(k).lower(): str(v).lower() for k, v in p.items()}
        idt = vals.get("id-token") == "write"
        if any(v == "write" for v in vals.values()):
            return "write", idt
        if any(v == "read" for v in vals.values()):
            return "read", idt
        return "none", idt
    return "default", False


def parse_workflow(text):
    """Return (reach: dict key->pintype, tier:int) or None if unusable."""
    try:
        doc = yaml.load(text, Loader=Loader)
    except Exception:
        return None
    if not isinstance(doc, dict):
        return None
    jobs = doc.get("jobs")
    if not isinstance(jobs, dict):
        return None
    reach = {}

    def add(uses):
        if not isinstance(uses, str):
            return
        u = uses.strip()
        if u.startswith("./") or u.startswith("docker://") or "/" not in u:
            return
        name, _, ref = u.partition("@")
        key = sys.intern(name.lower())
        pt = pin_type(ref) if ref else "branch"
        old = reach.get(key)
        if old is None or PIN_RANK[pt] < PIN_RANK[old]:
            reach[key] = pt

    wf_level, wf_idt = perm_level(doc.get("permissions"))
    best = -1
    any_idt = wf_idt
    if not jobs:
        best = LEVEL[wf_level]
    for job in jobs.values():
        if not isinstance(job, dict):
            continue
        add(job.get("uses"))
        steps = job.get("steps")
        if isinstance(steps, list):
            for st in steps:
                if isinstance(st, dict):
                    add(st.get("uses"))
        if "permissions" in job:
            lv, idt = perm_level(job.get("permissions"))
        else:
            lv, idt = wf_level, False
        best = max(best, LEVEL[lv])
        any_idt = any_idt or idt
    if best < 0:
        best = LEVEL[wf_level]
    tier = best * 2 + (1 if any_idt else 0)
    return reach, tier


def is_bot(name, email):
    ident = ((name or "") + " " + (email or "")).lower()
    return any(p in ident for p in BOT_PATTERNS)


def main():
    ap = argparse.ArgumentParser()
    home = os.path.expanduser("~/data/gha-workflows")
    ap.add_argument("--csv", default=os.path.join(home, "workflows.csv.gz"))
    ap.add_argument("--tar", default=os.path.join(home, "workflows.tar.gz"))
    ap.add_argument("--out", required=True)
    ap.add_argument("--pilot", type=int, default=0)
    ap.add_argument("--seed", type=int, default=20261123)
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    csv.field_size_limit(10 ** 9)

    # Pass 1: choose histories.
    selected = None
    if a.pilot:
        uids = set()
        with gzip.open(a.csv, "rt", encoding="utf-8", errors="replace", newline="") as fh:
            for r in csv.DictReader(fh):
                uids.add(r["uid"])
        pool = sorted(uids)
        random.Random(a.seed).shuffle(pool)
        selected = set(pool[:a.pilot])
        print("pilot histories selected:", len(selected), "seed", a.seed)

    # Pass 2: collect rows for selected histories.
    rows = {}
    repo_of = {}
    with gzip.open(a.csv, "rt", encoding="utf-8", errors="replace", newline="") as fh:
        for r in csv.DictReader(fh):
            u = r["uid"]
            if selected is not None and u not in selected:
                continue
            try:
                ep = int((r["committed_date"] or "0").strip())
            except ValueError:
                ep = 0
            rows.setdefault(u, []).append(
                (ep, (r["git_change_type"] or "").strip(),
                 (r["file_hash"] or "").strip(),
                 is_bot(r["author_name"], r["author_email"])))
            repo_of[u] = r["repository"]
    print("histories loaded:", len(rows))

    stats = {"reversed": 0, "tied_histories": 0, "deleted_before_window": 0,
             "no_exposure": 0, "rename_empty_hash": 0}
    plans = {}
    needed = set()
    for u, rs in rows.items():
        if rs[0][0] > rs[-1][0]:
            rs.reverse()
            stats["reversed"] += 1
        rs.sort(key=lambda x: x[0])
        eps = [x[0] for x in rs]
        if len(set(eps)) != len(eps):
            stats["tied_histories"] += 1
        first = eps[0]
        dele = None
        for ep, ct, fh, bot in rs:
            if ct == "D":
                dele = ep
                break
        if dele is not None and dele < W0:
            stats["deleted_before_window"] += 1
            continue
        start = max(W0, first)
        end = min(W1, dele) if dele is not None else W1
        if end <= start:
            stats["no_exposure"] += 1
            continue
        content = []
        for ep, ct, fh, bot in rs:
            if ct == "D":
                break
            if not fh:
                stats["rename_empty_hash"] += 1
                continue
            content.append((ep, ct, fh, bot))
        baseline = None
        inwin = []
        for c in content:
            if c[0] < W0:
                baseline = c
            elif c[0] <= W1:
                inwin.append(c)
        if inwin:
            if baseline:
                needed.add(baseline[2])
            for c in inwin:
                needed.add(c[2])
        plans[u] = (start, end, dele is not None, baseline, inwin)
    print("histories with exposure:", len(plans), "needed files:", len(needed), stats)

    # Stream the tarball once and parse only what is needed.
    feats = {}
    unparsed = 0
    remaining = set(needed)
    seen = 0
    with tarfile.open(a.tar, "r|gz") as tf:
        for m in tf:
            if not m.isfile():
                continue
            h = m.name.rsplit("/", 1)[-1]
            if h not in remaining:
                continue
            data = tf.extractfile(m).read().decode("utf-8", errors="replace")
            f = parse_workflow(data)
            feats[h] = f
            if f is None:
                unparsed += 1
            remaining.discard(h)
            seen += 1
            if seen % 20000 == 0:
                print("parsed", seen, "remaining", len(remaining), file=sys.stderr)
            if not remaining:
                break
    print("parsed:", seen, "unparsable:", unparsed, "missing from archive:", len(remaining))

    hist_f = open(os.path.join(a.out, "histories.csv"), "w", newline="")
    ev_f = open(os.path.join(a.out, "events.csv"), "w", newline="")
    hw = csv.writer(hist_f)
    ew = csv.writer(ev_f)
    hw.writerow(["uid", "repository", "exposure_days", "deleted_in_window",
                 "in_window_versions", "in_window_bot_versions",
                 "baseline_reach_size", "baseline_tier", "unparsed_versions"])
    ew.writerow(["uid", "repository", "epoch", "is_bot", "change_type",
                 "gain", "loss", "gain_official", "gain_sha", "gain_full",
                 "gain_major", "gain_branch", "tier_old", "tier_new",
                 "tier_dir", "reach_size_new"])
    n_ev = n_drift = n_visible = n_bot_ev = 0
    tot_days = 0.0
    for u, (start, end, deleted, baseline, inwin) in plans.items():
        days = (end - start) / 86400.0
        tot_days += days
        prev = None
        if baseline:
            prev = feats.get(baseline[2])
        bad = 0
        base_reach = len(prev[0]) if prev else ""
        base_tier = prev[1] if prev else ""
        nbot = sum(1 for c in inwin if c[3])
        first_is_baseline = baseline is None
        for i, (ep, ct, fh, bot) in enumerate(inwin):
            cur = feats.get(fh)
            if cur is None:
                bad += 1
                continue
            if first_is_baseline and prev is None:
                prev = cur
                base_reach, base_tier = len(cur[0]), cur[1]
                continue
            if prev is None:
                prev = cur
                continue
            pr, pt = prev
            cr, ct_ = cur
            gained = [k for k in cr if k not in pr]
            lost = [k for k in pr if k not in cr]
            tier_dir = "up" if ct_ > pt else ("down" if ct_ < pt else "none")
            if gained or lost or tier_dir != "none":
                g = {"sha": 0, "full": 0, "major": 0, "branch": 0}
                off = 0
                for k in gained:
                    g[cr[k]] += 1
                    if k.split("/")[0] in OFFICIAL:
                        off += 1
                ew.writerow([u, repo_of[u], ep, int(bot), ct, len(gained),
                             len(lost), off, g["sha"], g["full"], g["major"],
                             g["branch"], pt, ct_, tier_dir, len(cr)])
                n_ev += 1
                if bot:
                    n_bot_ev += 1
                elif gained and tier_dir == "none":
                    n_drift += 1
                elif gained and tier_dir == "up":
                    n_visible += 1
            prev = cur
        hw.writerow([u, repo_of[u], round(days, 3), int(deleted), len(inwin),
                     nbot, base_reach, base_tier, bad])
    hist_f.close()
    ev_f.close()
    print("exposure days total:", round(tot_days, 1))
    print("events written:", n_ev, "bot events:", n_bot_ev)
    print("non bot drift events (gain, tier unchanged):", n_drift)
    print("non bot visible growth events (gain, tier up):", n_visible)
    if tot_days:
        print("descriptive drift events per history per 90 days: %.4f" %
              (n_drift / (tot_days / 90.0)))


if __name__ == "__main__":
    main()
