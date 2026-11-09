#!/usr/bin/env python3
"""Lab only: summarise a stress.py OUTDIR: latency per phase/kind/op vs budget (200 ms screen, 100 ms
confirmed click), status counts, stream outcomes, and the CT sample CSV over time."""
import collections, json, os, sys

out = sys.argv[1]
BUDGET = {"screen": 200, "click": 100, "preview": 200}
phases, t0 = [], None
for line in open(os.path.join(out, "timeline.txt")):
    t, name = line.split(" ", 1)
    phases.append((float(t), name.strip()))
    t0 = t0 or float(t)


def segment(t):
    name = "?"
    for start, n in phases:
        if t >= start:
            name = n
    return name.split(" start")[0].split(" end")[0]


def pct(v, q):
    return v[min(len(v) - 1, int(q * len(v)))]


rows = collections.defaultdict(list)
status = collections.Counter()
errors = collections.Counter()
for line in open(os.path.join(out, "latency.jsonl")):
    r = json.loads(line)
    seg = "cycles" if segment(r["t"]).startswith("cycle") else segment(r["t"])
    status[(r["kind"], r["s"])] += 1
    if r["s"] == 200:
        rows[(seg, r["kind"], "ALL")].append(r["ms"])
        rows[("all", r["kind"], r["op"])].append(r["ms"])
    elif r.get("err"):
        errors[(r["op"], r["s"], r["err"][:60])] += 1
print("| segment | kind | op | n | p50 | p95 | p99 | max | over budget |")
print("|---|---|---|---|---|---|---|---|---|")
for k in sorted(rows):
    v = sorted(rows[k])
    over = sum(1 for x in v if x > BUDGET[k[1]])
    print(f"| {k[0]} | {k[1]} | {k[2]} | {len(v)} | {pct(v,.5):.0f} | {pct(v,.95):.0f} | {pct(v,.99):.0f} | {v[-1]:.0f} | {over} ({100*over/len(v):.0f}%) |")
print("\nstatus counts:", dict(sorted(status.items(), key=str)))
print("\ntop errors:")
for (op, s, e), n in errors.most_common(15):
    print(f"  {n:6d} {op} {s} {e}")
ev = collections.Counter()
for line in open(os.path.join(out, "events.jsonl")):
    r = json.loads(line)
    ev[(r["ev"], r.get("s"))] += 1
print("\nevents:", dict(sorted(ev.items(), key=str)))
sample = os.path.join(out, "sample.csv")
if os.path.exists(sample):
    lines = [l.strip().split(",") for l in open(sample) if l[0].isdigit()]
    print("\n| t+min | rss anon MB | rss file MB | cgroup cur MB | cgroup peak MB | db MB | wal MB | fds | tcp est |")
    print("|---|---|---|---|---|---|---|---|---|")
    step = max(1, len(lines) // 24)
    mb = lambda x, d=1024: f"{int(x)/d:.0f}" if x else "-"
    for l in lines[::step] + [lines[-1]]:
        print(f"| {(int(l[0]) - t0)/60:.0f} | {mb(l[2])} | {mb(l[3])} | {mb(l[4], 1<<20)} | {mb(l[5], 1<<20)} | {mb(l[6], 1<<20)} | {mb(l[7], 1<<20)} | {l[8]} | {l[9]} |")
    peak = max((int(l[2] or 0) for l in lines), default=0)
    print(f"\nmax rss anon {peak/1024:.0f} MB; max wal {max(int(l[7] or 0) for l in lines)/(1<<20):.1f} MB; max cgroup peak {max(int(l[5] or 0) for l in lines)/(1<<20):.0f} MB")
