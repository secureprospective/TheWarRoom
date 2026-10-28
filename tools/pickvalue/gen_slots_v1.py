#!/usr/bin/env python3
"""Regenerate internal/pickvalue/slots_v1.csv: Pick Value v1's league draft history.

Each row is one Legacy NFL rookie-draft selection (2017-2026, from the frozen Lab facts in
internal/lab/testdata/lab.json.gz) with the NFL draft capital of the player taken: his overall
NFL pick, or 0 when he went undrafted. The overall pick is nflverse's where the held features
know it; otherwise it is derived from MFL's round and pick by that round's first overall pick
(the most common start among the players nflverse does know), exactly as the player record does.

Usage: gen_slots_v1.py <lab.json.gz> <players.json> <features.jsonl> > internal/pickvalue/slots_v1.csv
"""
import collections, gzip, json, sys

lab_path, players_path, features_path = sys.argv[1:4]
draft = json.load(gzip.open(lab_path))["draft"]
players = {p["id"]: p for p in json.load(open(players_path))["players"]["player"]}
overall = {}
for line in open(features_path):
    row = json.loads(line)
    if row["Measure"] == "prior.draft_pick":
        overall[row["PlayerID"]] = int(row["Value"])

starts = collections.defaultdict(collections.Counter)
for pid, p in players.items():
    if pid in overall and p.get("draft_round") and p.get("draft_pick") and p.get("draft_year"):
        starts[(int(p["draft_year"]), int(p["draft_round"]))][overall[pid] - int(p["draft_pick"]) + 1] += 1

def nfl_overall(i):
    pid, rnd, pick, year = draft["pid"][i], draft["nflRound"][i], draft["nflPick"][i], draft["year"][i]
    if rnd is None or pick is None:
        return 0
    if pid in overall:
        return overall[pid]
    p = players.get(pid, {})
    nfl_year = int(p.get("draft_year") or year)
    votes = starts.get((nfl_year, rnd))
    if not votes:
        sys.exit(f"no round start for {nfl_year} round {rnd} (player {pid})")
    start = max(sorted(votes), key=lambda s: (votes[s], -s))
    return start + pick - 1

print("year,round,slot,selection,nfl_overall")
rows = sorted(range(len(draft["year"])), key=lambda i: (draft["year"][i], draft["overall"][i]))
for i in rows:
    print(f'{draft["year"][i]},{draft["round"][i]},{draft["slot"][i]},{draft["overall"][i]},{nfl_overall(i)}')
