"""Soak coverage -> review list. Usage: soak-report.py RUN_DIR. Writes RUN_DIR/report.md.
Never-ran is NOT unused: season events (draft, rollover, trade accept, playoffs, commish fixes)
don't fire in week 9. This is a review list against the 5 mocks + rulings, not a delete list."""
import collections, os, re, sys

run = sys.argv[1]
MOD = "github.com/secureprospective/TheWarRoom/"
RULED_OUT = {"internal/lab": "Lab, ruled out 10-26", "internal/labcompile": "Lab, ruled out 10-26",
             "internal/app/research": "Research, split out 10-28",
             "internal/app/mflref": "MFL ref #1, check overlap", "internal/mflref": "MFL ref #2, check overlap"}

pkg = collections.defaultdict(lambda: [0, 0])          # pkg -> [stmts, ran]
seen = {}
with open(os.path.join(run, "cover.out")) as f:
    next(f)
    for line in f:
        m = re.match(r"(.+):(\d+\.\d+,\d+\.\d+) (\d+) (\d+)", line.strip())
        if not m:
            continue
        key = (m[1], m[2]); n, c = int(m[3]), int(m[4])
        if key in seen:                                   # same block from several counter files
            if c and not seen[key]:
                pkg[os.path.dirname(m[1])][1] += n; seen[key] = 1
            continue
        seen[key] = 1 if c else 0
        p = pkg[os.path.dirname(m[1])]; p[0] += n; p[1] += n if c else 0

zero = collections.defaultdict(list)
with open(os.path.join(run, "func.txt")) as f:
    for line in f:
        parts = line.split()
        if len(parts) >= 3 and parts[-1] == "0.0%" and parts[0].startswith(MOD):
            zero[os.path.dirname(parts[0].split(":")[0]).replace(MOD, "")].append(parts[1])

rows = sorted(((k.replace(MOD, ""), s, r) for k, (s, r) in pkg.items()), key=lambda t: -(t[1] - t[2]))
ts, tr = sum(r[1] for r in rows), sum(r[2] for r in rows)
loops = sum(1 for l in open(os.path.join(run, "walk.log")) if "step=hq-home" in l)
out = [f"# Coverage soak report — {os.path.basename(run)}", "",
       f"Walk loops completed: {loops}. Statements: {ts:,} total, {tr:,} ran ({100*tr/max(ts,1):.1f}%), "
       f"{ts-tr:,} never ran.", "",
       "Never-ran ≠ unused. Review each package: in a mock / on no screen / ruled out.", "",
       "| Package | Stmts | Ran | Never ran | % ran | Note |", "|---|---:|---:|---:|---:|---|"]
for name, s, r in rows:
    out.append(f"| {name} | {s:,} | {r:,} | {s-r:,} | {100*r/max(s,1):.0f}% | {RULED_OUT.get(name, '')} |")
out += ["", "## Functions that never ran, by package (top 40 packages by never-ran statements)", ""]
for name, s, r in rows[:40]:
    fs = zero.get(name, [])
    if fs:
        out.append(f"- **{name}** ({len(fs)}): " + ", ".join(sorted(fs)[:60]) + (" …" if len(fs) > 60 else ""))
open(os.path.join(run, "report.md"), "w").write("\n".join(out) + "\n")
print(f"packages={len(rows)} stmts={ts} ran={tr}")
