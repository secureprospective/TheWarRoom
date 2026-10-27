#!/usr/bin/env python3
"""Pin the held inputs the E5 golden needs: 2025 weeks 1-13 for five players, the play-by-play rows
that credit them, and their Sleeper weekly rows. Reads only public held sources, never engine output.

Plays are filtered to the rows the adapter can attribute to one of the five (it reads only
passer/receiver/rusher/td_player_id), plus one anchor row per game so AttachPlays' coverage check
still sees every game. Writes to internal/fantasyscore/source/testdata and prints each digest.
"""
import csv
import gzip
import hashlib
import io
import json
import os
import sys
from pathlib import Path

sys.dont_write_bytecode = True
CACHE = Path(os.environ.get("TWR_STUDY_CACHE", "/home/chris/fleet/runs/config-study-p6-2026-10-27/cache"))
SLEEPER = Path(os.environ.get("TWR_SLEEPER_STUDY_CACHE", "/dev/shm/p6cache"))
OUT = Path(__file__).resolve().parents[1] / "internal/fantasyscore/source/testdata"
PLAYERS = {"00-0033923": "4098", "00-0033280": "4034", "00-0036389": "6904",
           "00-0034857": "4984", "00-0034844": "4866"}
# Every column rawPlay reads, and nothing else: no desc, no modelling columns.
FIELDS = ("game_id play_id season week season_type passer_player_id receiver_player_id rusher_player_id "
          "passing_yards receiving_yards rushing_yards touchdown td_player_id td_team return_team "
          "interception punt_attempt kickoff_attempt play_deleted two_point_attempt play_type sack "
          "posteam punt_blocked field_goal_attempt field_goal_result third_down_converted "
          "fourth_down_converted rush_attempt pass_attempt complete_pass yardline_100 "
          "tackled_for_loss").split()
CREDITS = ("passer_player_id", "receiver_player_id", "rusher_player_id", "td_player_id")


def read(name):
    with gzip.open(CACHE / name, "rt", newline="") as fh:
        return list(csv.DictReader(fh))


def write_csv(name, rows, fields):
    stream = io.StringIO(newline="")
    writer = csv.DictWriter(stream, fieldnames=fields, extrasaction="ignore")
    writer.writeheader()
    writer.writerows(rows)
    body = stream.getvalue().encode()
    (OUT / name).write_bytes(gzip.compress(body, mtime=0))
    print(f"{name}: {len(rows)} rows; uncompressed sha256={hashlib.sha256(body).hexdigest()}")


def main():
    weeks = [row for row in read("stats_player_week_2025.csv.gz")
             if row["player_id"] in PLAYERS and row["season_type"] == "REG" and int(row["week"]) <= 13]
    games = {row["game_id"] for row in weeks}
    kept, anchored = [], set()
    for row in read("play_by_play_2025.csv.gz"):
        if row["game_id"] not in games:
            continue
        if any(row[column] in PLAYERS for column in CREDITS):
            kept.append(row)
        elif row["game_id"] not in anchored:
            anchored.add(row["game_id"])
            kept.append(row)
    print(f"{len(games)} games, {len(anchored)} anchor-only rows")
    write_csv("e5_weeks.csv.gz", weeks, list(weeks[0]))
    write_csv("e5_plays.csv.gz", kept, FIELDS)
    rows = {}
    for week in range(1, 14):
        body = json.loads((SLEEPER / f"sleeper_2025_{week}.json").read_text())
        ids = {PLAYERS[row["player_id"]] for row in weeks if int(row["week"]) == week}
        rows[str(week)] = {sid: body[sid] for sid in sorted(ids)}
    body = (json.dumps(rows, sort_keys=True, separators=(",", ":")) + "\n").encode()
    (OUT / "e5_sleeper.json.gz").write_bytes(gzip.compress(body, mtime=0))
    print(f"e5_sleeper.json.gz: uncompressed sha256={hashlib.sha256(body).hexdigest()}")


if __name__ == "__main__":
    main()
