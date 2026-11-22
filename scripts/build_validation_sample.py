#!/usr/bin/env python3
"""Build the validation sample for protocol section 6.

Draws a seeded random sample of flagged drift events and a stratified sample
of unflagged events from events.csv, finds the before and after workflow
files, and writes:
  key.csv              item_id, stratum, population size, uid, epoch, checks
  rater_sheet_A.csv    blind sheet for rater A (shuffled)
  rater_sheet_B.csv    blind sheet for rater B (shuffled differently)
Raters see the repository, commit hash, actions added and removed and a diff.
They do not see the stratum or the flag. No author names or emails are written.

Usage:
  python3 scripts/build_validation_sample.py DERIVED_DIR --out OUTDIR
         [--flagged 200] [--per-stratum 40] [--seed S] [--csv PATH] [--tar PATH]
"""
import argparse
import csv
import difflib
import gzip
import os
import random
import sys
import tarfile
from datetime import datetime, timezone

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from extract_features import parse_workflow, is_bot, OFFICIAL  # noqa: E402

UNFLAGGED = ["visible_growth", "gain_tier_down", "official_only_gain",
             "loss_only", "tier_only"]


def kind(e):
    gain = int(e["gain"])
    tp = gain - int(e["gain_official"])
    loss = int(e["loss"])
    t = e["tier_dir"]
    if tp > 0 and t == "none":
        return "flagged_drift"
    if tp > 0 and t == "up":
        return "visible_growth"
    if tp > 0:
        return "gain_tier_down"
    if gain > 0:
        return "official_only_gain"
    if t == "none" and loss > 0:
        return "loss_only"
    if t != "none":
        return "tier_only"
    return "other"


def main():
    ap = argparse.ArgumentParser()
    home = os.path.expanduser("~/data/gha-workflows")
    ap.add_argument("derived")
    ap.add_argument("--out", required=True)
    ap.add_argument("--flagged", type=int, default=200)
    ap.add_argument("--per-stratum", type=int, default=40)
    ap.add_argument("--seed", type=int, default=20261129)
    ap.add_argument("--csv", default=os.path.join(home, "workflows.csv.gz"))
    ap.add_argument("--tar", default=os.path.join(home, "workflows.tar.gz"))
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    csv.field_size_limit(10 ** 9)

    frame = {}
    for e in csv.DictReader(open(os.path.join(a.derived, "events.csv"))):
        if e["is_bot"] == "1":
            continue
        frame.setdefault(kind(e), []).append(e)
    print("frame sizes:", {k: len(v) for k, v in sorted(frame.items())})

    picks = []
    plan = [("flagged_drift", a.flagged)] + [(s, a.per_stratum) for s in UNFLAGGED]
    for stratum, n in plan:
        pool = frame.get(stratum, [])
        rng = random.Random("%d:%s" % (a.seed, stratum))
        chosen = rng.sample(pool, min(n, len(pool)))
        for e in chosen:
            picks.append((stratum, len(pool), e))
    print("sampled items:", len(picks))

    wanted = {e["uid"] for _, _, e in picks}
    rows = {}
    with gzip.open(a.csv, "rt", encoding="utf-8", errors="replace", newline="") as fh:
        for r in csv.DictReader(fh):
            u = r["uid"]
            if u not in wanted:
                continue
            try:
                ep = int((r["committed_date"] or "0").strip())
            except ValueError:
                ep = 0
            rows.setdefault(u, []).append(
                (ep, (r["git_change_type"] or "").strip(),
                 (r["file_hash"] or "").strip(),
                 is_bot(r["author_name"], r["author_email"]),
                 r["commit_hash"], r["file_path"]))
    for u, rs in rows.items():
        if rs[0][0] > rs[-1][0]:
            rs.reverse()
        rs.sort(key=lambda x: x[0])

    located = []
    need = set()
    for stratum, size, e in picks:
        rs = rows.get(e["uid"], [])
        ep = int(e["epoch"])
        idx = [i for i, r in enumerate(rs)
               if r[0] == ep and r[1] == e["change_type"] and r[2] and not r[3]]
        if not idx:
            located.append((stratum, size, e, None, [], 0))
            continue
        i = idx[0]
        content = [r for r in rs[:i] if r[1] != "D" and r[2]]
        cands = content[-3:][::-1]
        for c in cands:
            need.add(c[2])
        need.add(rs[i][2])
        located.append((stratum, size, e, rs[i], cands, 1 if len(idx) > 1 else 0))

    texts = {}
    remaining = set(need)
    with tarfile.open(a.tar, "r|gz") as tf:
        for m in tf:
            if not m.isfile():
                continue
            h = m.name.rsplit("/", 1)[-1]
            if h in remaining:
                texts[h] = tf.extractfile(m).read().decode("utf-8", errors="replace")
                remaining.discard(h)
                if not remaining:
                    break
    print("files needed:", len(need), "missing:", len(remaining))

    items = []
    mism = 0
    unlocated = 0
    for stratum, size, e, cur_row, cands, tie in located:
        if cur_row is None:
            unlocated += 1
            continue
        cur_text = texts.get(cur_row[2], "")
        cur = parse_workflow(cur_text)
        prev_text, prev = "", None
        for c in cands:
            t = texts.get(c[2], "")
            p = parse_workflow(t)
            if p is not None:
                prev_text, prev = t, p
                break
        if cur is None or prev is None:
            unlocated += 1
            continue
        gained = sorted(k for k in cur[0] if k not in prev[0])
        lost = sorted(k for k in prev[0] if k not in cur[0])
        owner = e["repository"].split("/")[0].lower()
        same = sum(1 for k in gained if k.split("/")[0] == owner)
        check = int(len(gained) == int(e["gain"]) and len(lost) == int(e["loss"]))
        if not check:
            mism += 1
        diff = list(difflib.unified_diff(prev_text.splitlines(), cur_text.splitlines(),
                                         lineterm="", n=2))
        if len(diff) > 150:
            diff = diff[:150] + ["... diff truncated ..."]
        items.append({
            "stratum": stratum, "size": size, "uid": e["uid"], "epoch": e["epoch"],
            "repository": e["repository"], "file_path": cur_row[5],
            "commit_hash": cur_row[4],
            "commit_utc": datetime.fromtimestamp(int(e["epoch"]), timezone.utc)
                                  .strftime("%Y-%m-%d %H:%M"),
            "added": "; ".join(gained), "removed": "; ".join(lost),
            "diff": "\n".join(diff), "same_owner": same, "check": check, "tie": tie})
    print("items built:", len(items), "unlocated:", unlocated,
          "count mismatches vs events.csv:", mism)

    ids = random.Random(a.seed).sample(range(100000, 1000000), len(items))
    for it, i in zip(items, ids):
        it["item_id"] = str(i)
    with open(os.path.join(a.out, "key.csv"), "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["item_id", "stratum", "stratum_size", "uid", "epoch",
                    "gained_same_owner", "count_check_ok", "tie_ambiguous"])
        for it in sorted(items, key=lambda x: x["item_id"]):
            w.writerow([it["item_id"], it["stratum"], it["size"], it["uid"],
                        it["epoch"], it["same_owner"], it["check"], it["tie"]])
    cols = ["item_id", "repository", "file_path", "commit_hash", "commit_utc",
            "actions_added", "actions_removed", "diff", "label", "fp_reason",
            "reach_type", "notes"]
    for tag, off in (("A", 1), ("B", 2)):
        order = list(items)
        random.Random(a.seed + off).shuffle(order)
        with open(os.path.join(a.out, "rater_sheet_%s.csv" % tag), "w", newline="") as f:
            w = csv.writer(f)
            w.writerow(cols)
            for it in order:
                w.writerow([it["item_id"], it["repository"], it["file_path"],
                            it["commit_hash"], it["commit_utc"], it["added"],
                            it["removed"], it["diff"], "", "", "", ""])
    print("wrote key.csv, rater_sheet_A.csv, rater_sheet_B.csv to", a.out)


if __name__ == "__main__":
    main()
