"""Writes source/testdata/line_plays.csv and line_rushing.csv: two real 2024 team-games from held
play-by-play and PFR advanced rushing. Run: python3 -I oline_line_fixtures.py (see README.md)."""
import csv, gzip, os, sqlite3, sys
C = "/home/chris/fleet/runs/sol61-sol61-warroom-config-study-p3-2026-10-26/cache/"
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "source", "testdata") + os.sep
comp = [r for r in csv.DictReader(open(OUT + "../../testdata/oline_setB_2023_2025.csv")) if r["year"] == "2024"]
nos = next(r["game_id"] for r in comp if r["nosack"] == "1.0")
games = {"2024_01_PIT_ATL", nos}
cols = ["game_id", "play_id", "season", "week", "season_type", "posteam", "play_type", "rush_attempt", "qb_kneel",
        "two_point_attempt", "play_deleted", "rushing_yards", "lateral_rushing_yards", "rush_touchdown", "td_team",
        "down", "ydstogo", "third_down_converted", "fourth_down_converted", "tackled_for_loss", "sack",
        "qb_hit_1_player_id", "qb_hit_2_player_id"]
with gzip.open(C + "play_by_play_2024.csv.gz", "rt", newline="") as f, open(OUT + "line_plays.csv", "w", newline="") as o:
    w = csv.writer(o); w.writerow(cols); n = 0
    for r in csv.DictReader(f):
        if r["game_id"] in games:
            w.writerow([r[c] for c in cols]); n += 1
con = sqlite3.connect("file:/home/chris/work/TheWarRoom/tools/league-history/data/mfl.db?mode=ro", uri=True)
rb = {pid for (pid,) in con.execute("SELECT pid FROM player_season WHERE year=2024 AND position='RB'")}
pfr_rb = {r["pfr_id"] for r in csv.DictReader(open(C + "db_playerids.csv")) if r["mfl_id"] in rb and r["pfr_id"] not in ("", "NA")}
with open(C + "advstats_week_rush_2024.csv", newline="") as f, open(OUT + "line_rushing.csv", "w", newline="") as o:
    w = csv.writer(o); w.writerow(["game_id", "season", "week", "game_type", "team", "pfr_player_id", "rushing_yards_before_contact", "league_rb"]); m = 0
    for r in csv.DictReader(f):
        if r["game_id"] in games:
            w.writerow([r["game_id"], r["season"], r["week"], r["game_type"], r["team"], r["pfr_player_id"], r["rushing_yards_before_contact"], "1" if r["pfr_player_id"] in pfr_rb else "0"]); m += 1
print(games, n, m)
for r in comp:
    if r["game_id"] in games: print(r)
