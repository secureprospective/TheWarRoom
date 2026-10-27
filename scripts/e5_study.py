#!/usr/bin/env python3
"""E5 oracle: count the play-derived edge stats independently of the Go adapter, price them with the
legacy study scorer, and print the five E5 player-seasons. Also cross-checks Sleeper's own red-zone
and tackled-for-loss aggregates and classifies every play the two sides disagree on.

Ruled definitions (see HANDOFF.md). A play counts only if it is live: not play_deleted, not a
two-point try, not nullified by penalty (play_type == no_play).
  carry                   rush_attempt == 1 (kneels and aborted snaps included)
  rushing_tfl             a carry with tackled_for_loss == 1
  red_zone_carry          a carry snapped at yardline_100 <= 20
  red_zone_pass_attempt   yardline_100 <= 20, pass_attempt == 1 and sack != 1
  red_zone_target         the same play, credited to the named receiver
"""
import csv
import gzip
import json
import os
import sys
from collections import defaultdict
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path

sys.dont_write_bytecode = True
STUDY = Path(os.environ.get("TWR_P6_STUDY", "/home/chris/fleet/runs/config-study-p6-2026-10-27"))
sys.path.insert(0, str(STUDY))
import p6
import legacy_scorer as legacy

CACHE = Path(os.environ.get("TWR_STUDY_CACHE", str(STUDY / "cache")))
SLEEPER = Path(os.environ.get("TWR_SLEEPER_STUDY_CACHE", "/dev/shm/p6cache"))
# Owner-ruled prior full-season rates: 2024 PBP tackles for loss over 2024 official carries.
RATES = {"QB": (78, 2352), "RB": (1106, 11832), "WR": (62, 430), "TE": (7, 62)}
PLAYERS = {"13138": ("Hunt", "00-0033923", "4098"), "13130": ("McCaffrey", "00-0033280", "4034"),
           "14783": ("Hurts", "00-0036389", "6904"), "13589": ("Allen", "00-0034857", "4984"),
           "13604": ("Barkley", "00-0034844", "4866")}
CODES = ("RZA", "RZP", "RZT", "RTL")
NAME = {gsis: name for name, gsis, _ in PLAYERS.values()}


def live(row):
    return row["play_deleted"] != "1" and row["two_point_attempt"] != "1" and row["play_type"] != "no_play"


def pbp_edges(gsis):
    """-> ({(gsis, week): {code: n}}, [disputed-play rows]). Counts straight from held play-by-play."""
    counts = defaultdict(lambda: defaultdict(int))
    plays = []
    with gzip.open(CACHE / "play_by_play_2025.csv.gz", "rt", newline="") as fh:
        for row in csv.DictReader(fh):
            if row["season_type"] != "REG" or not 1 <= int(float(row["week"])) <= 13:
                continue
            if not live(row):
                continue
            week = int(float(row["week"]))
            yard = row["yardline_100"]
            in_zone = yard not in ("", "NA") and Decimal(yard) <= 20
            carry = row["rush_attempt"] == "1"
            attempt = row["pass_attempt"] == "1" and row["sack"] != "1"
            for column, code, counted in (("rusher_player_id", "RZA", carry and in_zone),
                                          ("passer_player_id", "RZP", attempt and in_zone),
                                          ("receiver_player_id", "RZT", attempt and in_zone),
                                          ("rusher_player_id", "RTL", carry and row["tackled_for_loss"] == "1")):
                player = row[column]
                if counted and player in gsis:
                    counts[(player, week)][code] += 1
                    plays.append((player, week, code, row))
            if row["rusher_player_id"] in gsis and carry:
                counts[(row["rusher_player_id"], week)]["RA"] += 1
    return counts, plays


def sleeper_edges():
    """-> {(gsis, week): {code: n}} from Sleeper's own weekly aggregates, the independent vendor."""
    out = defaultdict(lambda: defaultdict(int))
    by_sleeper = {sid: gsis for _, gsis, sid in PLAYERS.values()}
    for week in range(1, 14):
        body = json.loads((SLEEPER / f"sleeper_2025_{week}.json").read_text())
        for sid, gsis in by_sleeper.items():
            row = body.get(sid)
            if not isinstance(row, dict):
                continue
            for key, code in (("rush_rz_att", "RZA"), ("pass_rz_att", "RZP"),
                              ("rec_rz_tgt", "RZT"), ("rush_tkl_loss", "RTL"), ("rush_att", "RA")):
                out[(gsis, week)][code] += int(row.get(key) or 0)
    return out


def excluded_by_sleeper(code, row):
    """Name why Sleeper's aggregate leaves out a play the ruled definition counts, or None."""
    if code in ("RZP", "RZT") and row["sack"] == "1":
        return "sack counted as a pass attempt by nflverse, not by official stats"
    if code in ("RZA", "RZP", "RZT") and row["yardline_100"] == "20":
        return "snapped on the 20: Sleeper uses yardline_100 < 20"
    return None


def vendor_crosscheck(counts, plays):
    """Print the exact plays the two sides disagree on, per player-week, and classify each."""
    vendor = sleeper_edges()
    delta = {}
    for key in sorted(set(list(counts) + list(vendor))):
        for code in CODES + ("RA",):
            d = counts[key][code] - vendor[key][code]
            if d:
                delta[key + (code,)] = d
    classes = defaultdict(int)
    explained = defaultdict(int)
    for player, week, code, row in plays:
        key = (player, week, code)
        if delta.get(key, 0) <= 0:
            continue
        reason = excluded_by_sleeper(code, row)
        if reason is None:
            continue
        explained[key] += 1
        classes[(code, reason)] += 1
        print(f"  {NAME[player]:10s} wk{week:<3d} {code} {row['game_id']} play={row['play_id']:>5s} "
              f"yl={row['yardline_100']:>2s} | {reason}\n{' ' * 14}{row['desc'][:120]}")
    residual = {(NAME[k[0]], k[1], k[2]): d - explained[k]
                for k, d in delta.items() if d > 0 and d != explained[k]}
    print("\n  classes of play the ruled definition counts and Sleeper does not:")
    for key in sorted(classes):
        print(f"    {key[0]} {key[1]}: {classes[key]}")
    print("  plays Sleeper counts and the ruled definition does not:")
    shown = False
    for (player, week, code), d in sorted(delta.items()):
        if d < 0:
            shown = True
            print(f"    {NAME[player]:10s} wk{week:<3d} {code}: {-d} (aborted snap charged a rushing "
                  f"loss by Sleeper; nflverse tackled_for_loss == 0, no tackle happened)")
    if not shown:
        print("    none")
    print("  per-player season delta (ruled definition minus Sleeper):")
    for name, gsis, _ in PLAYERS.values():
        totals = {c: sum(d for k, d in delta.items() if k[0] == gsis and k[2] == c) for c in CODES + ("RA",)}
        print(f"    {name}: {({k: v for k, v in totals.items() if v}) or 'none'}")
    print(f"  disputed plays with no identified class: {residual or 'none'}")
    return delta


def cent(value):
    return value.quantize(Decimal(".01"), rounding=ROUND_HALF_UP)


def main():
    gsis = {g for _, g, _ in PLAYERS.values()}
    counts, plays = pbp_edges(gsis)
    print("=== Sleeper vendor cross-check: plays the ruled definition counts ===")
    vendor_crosscheck(counts, plays)
    package = dict(p6.PACKAGES["A light"], RTL=0)
    rules = p6.rules_for(package)
    context = p6.V.ctx(2025)
    totals = {}
    print("\n=== E5 study totals ===")
    for mfl, (name, player, _) in PLAYERS.items():
        position = context["pos"][mfl]
        losses, carries = RATES[position]
        exact_rate = Decimal("0.3124") * losses / carries
        rate = exact_rate.quantize(Decimal(".0001"), rounding=ROUND_HALF_UP)
        breakdown = defaultdict(Decimal)
        exact_rtl = rate_only = Decimal(0)
        for week, original in context["byp"][mfl].items():
            if not 1 <= week <= 13:
                continue
            event = dict(original)
            for code in CODES:
                event[code] = counts[(player, week)][code]
            for rule in rules:
                breakdown[rule["event"]] += Decimal(str(legacy.score(event, position, [rule])))
            attempts = Decimal(counts[(player, week)]["RA"])
            loss = Decimal("-0.3124") * counts[(player, week)]["RTL"]
            exact_rtl += loss + exact_rate * attempts
            rate_only += loss + rate * attempts
            # The engine prices the weekly loss rule and the weekly carry extra as separate cent-rounded lines.
            breakdown["RTL"] += cent(loss) + cent(rate * attempts)
        total = sum(breakdown.values())
        totals[name] = int(total * 10000)
        print(f"{name} {position} carry_extra={rate} total={total:.6f}")
        print("  " + " ".join(f"{k}={v:.4f}" for k, v in sorted(breakdown.items()) if v))
        print(f"  RTL exact={exact_rtl:.9f} four-decimal-rate={rate_only:.9f} weekly-cent={breakdown['RTL']:.9f}"
              f" rate_delta={rate_only - exact_rtl:+.9f} cent_delta={breakdown['RTL'] - rate_only:+.9f}")
    if len(sys.argv) == 3 and sys.argv[1] == "--golden":
        Path(sys.argv[2]).write_text(json.dumps(totals, indent=2) + "\n")
        print(f"\nwrote {sys.argv[2]}")


if __name__ == "__main__":
    main()
