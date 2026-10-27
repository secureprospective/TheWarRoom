"""Prints the default32_2024.json golden fixture to stdout: real 2024 player-weeks scored by the
study's reference scorer under the locked default. Run: python3 -I default32_golden.py > ../default32_2024.json
(see README.md)."""
import sys, os, csv, gzip, json, sqlite3
P5 = "/home/chris/fleet/runs/config-study-p5-2026-10-26"
sys.path.insert(0, P5)
import study4c, legacy_scorer as L
V = study4c.V
rules = study4c.VARIANTS["C2 continuous + dyn kicker"](V.BASE)
year = 2024
ev = study4c.V.load(year)
con = sqlite3.connect(f"file:{L.DB}?mode=ro", uri=True)
pos = dict(con.execute("SELECT pid, position FROM player_season WHERE year=?", (year,)))
names = dict(con.execute("SELECT pid, name FROM player_season WHERE year=?", (year,)))
m2g = {m: g for g, m in L.bridge().items()}
scored = []
for (w, pid), e in ev.items():
    p = pos.get(pid)
    if p not in L.POS:
        continue
    scored.append((L.score(e, p, rules), w, pid, p))
pick = []
want_names = ["Barkley, Saquon", "Westbrook-Ikhine, Nick", "Boswell, Chris", "McBride, Trey", "Franklin, Zaire",
              "Hendrickson, Trey", "Chase, Ja'Marr", "Robinson, Alontae"]
for s, w, pid, p in sorted(scored, key=lambda t: -t[0]):
    n = names.get(pid, "")
    if n in want_names and sum(1 for x in pick if x[2] == pid) < 2:
        pick.append((s, w, pid, p))
# the 98-yard play week and a few per-position tops not yet chosen
for p in L.POS:
    top = [t for t in sorted(scored, key=lambda t: -t[0]) if t[3] == p][:1]
    pick += [t for t in top if t not in pick]
wi = [t for t in scored if names.get(t[2]) == "Westbrook-Ikhine, Nick" and t[1] == 11]
pick += [t for t in wi if t not in pick]
keys = {(m2g[pid], w): (s, w, pid, p) for s, w, pid, p in pick if pid in m2g}
plays = {k: {"chunk": [], "pass": []} for k in keys}
with gzip.open(os.path.join(L.CACHE, f"play_by_play_{year}.csv.gz"), "rt", newline="") as fh:
    for r in csv.DictReader(fh):
        if r["season_type"] != "REG" or int(r["week"]) > 13:
            continue
        w = int(r["week"])
        if r["rusher_player_id"] not in ("", "NA") and r["rushing_yards"] not in ("", "NA") and (r["rusher_player_id"], w) in plays:
            y = float(r["rushing_yards"])
            if y >= 20: plays[(r["rusher_player_id"], w)]["chunk"].append(y)
        if r["complete_pass"] == "1" and r["receiver_player_id"] not in ("", "NA") and r["receiving_yards"] not in ("", "NA") and (r["receiver_player_id"], w) in plays:
            y = float(r["receiving_yards"])
            if y >= 20: plays[(r["receiver_player_id"], w)]["chunk"].append(y)
        if r["complete_pass"] == "1" and r["passer_player_id"] not in ("", "NA") and r["passing_yards"] not in ("", "NA") and (r["passer_player_id"], w) in plays:
            y = float(r["passing_yards"])
            if y >= 40: plays[(r["passer_player_id"], w)]["pass"].append(y)
rows = []
for (g, w), (s, _, pid, p) in sorted(keys.items(), key=lambda kv: (kv[1][3], kv[1][1])):
    e = ev[(w, pid)]
    ck = sum(1 + 0.025 * (y - 20) for y in plays[(g, w)]["chunk"])
    if abs(round(ck, 4) - e.get("CK2", 0)) > 1e-6 or len(plays[(g, w)]["pass"]) != e.get("P40", 0):
        print("skip", names[pid], w, ck, e.get("CK2"), len(plays[(g, w)]["pass"]), e.get("P40"), file=sys.stderr)
        continue
    codes = {k: v for k, v in e.items() if not k.startswith("_") and k not in ("CK1", "CK3", "CKMAX", "CK2", "FGD", "MGD", "P40") and v}
    rows.append(dict(source=f"nflverse stats_player_week_{year} + play_by_play_{year}, REG week {w}, gsis {g}, mfl {pid}",
                     name=names[pid], position=p, week=w, codes=codes, chunk_plays=plays[(g, w)]["chunk"],
                     passes_40=plays[(g, w)]["pass"], fg_made=e["_FG"], fg_missed=e["_MG"], expected=s))
json.dump(dict(reference="legacy_scorer.score under study4c.VARIANTS['C2 continuous + dyn kicker'] (config-study p5, 2026-10-26)",
               rows=rows), sys.stdout, indent=1)
