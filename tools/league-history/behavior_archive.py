"""Evidence-preserving behavioral inputs. No database writes or inferred identities."""
from bisect import bisect_right
from collections import Counter, defaultdict
from datetime import datetime, timezone
import json
import math
from pathlib import Path


def rows(value):
    if value is None:
        return []
    return value if isinstance(value, list) else [value]


def number(value):
    try:
        result = float(value)
        return result if math.isfinite(result) else None
    except (ValueError, TypeError):
        return None


def stamp(value):
    try:
        return datetime.fromisoformat(value).timestamp()
    except (ValueError, TypeError):
        return None


def iso(value):
    return datetime.fromtimestamp(value, timezone.utc).isoformat() if value is not None else None


def validate_tenures(value, team_ids):
    """Explicit, non-overlapping half-open tenures; never derive an owner from a name."""
    if not isinstance(value, list):
        raise ValueError("Tenures must be an array")
    seen = set()
    by_team = defaultdict(list)
    for item in value:
        if not isinstance(item, dict) or set(item) != {"id", "name", "team", "from", "to", "evidence"}:
            raise ValueError("Each tenure needs id, name, team, from, to and evidence only")
        if not all(isinstance(v, str) and v.strip() for v in item.values()):
            raise ValueError("Tenure fields must be non-empty strings")
        if item["id"] in seen or item["team"] not in team_ids:
            raise ValueError("Duplicate tenure ID or unknown franchise")
        seen.add(item["id"])
        for key in ("from", "to"):
            if len(item[key]) != 10 or datetime.strptime(item[key], "%Y-%m-%d").strftime("%Y-%m-%d") != item[key]:
                raise ValueError("Tenure bounds must be YYYY-MM-DD UTC dates")
        if item["from"] >= item["to"]:
            raise ValueError("Tenure end must follow start; end is exclusive")
        by_team[item["team"]].append((item["from"], item["to"]))
    for periods in by_team.values():
        ordered = sorted(periods)
        if any(right[0] < left[1] for left, right in zip(ordered, ordered[1:])):
            raise ValueError("Overlapping tenures need adjudication, not automatic attribution")
    return value


def limits(value):
    parts = str(value).split("-")
    try:
        return int(parts[0]), int(parts[-1])
    except ValueError:
        return None


def lineup_legal(ids, positions, rules):
    if not ids or len(ids) != len(set(ids)) or not rules:
        return False
    total = limits(rules.get("count", ""))
    if not total or not total[0] <= len(ids) <= total[1]:
        return False
    counts = Counter(positions.get(player) for player in ids)
    allowed = {r["name"]: limits(r.get("limit", "")) for r in rows(rules.get("position"))}
    if any(position not in allowed for position in counts):
        return False
    for position, bounds in allowed.items():
        if not bounds or not bounds[0] <= counts[position] <= bounds[1]:
            return False
    offense = sum(counts[p] for p in ("QB", "RB", "WR", "TE"))
    defense = len(ids) - offense - counts["PK"]
    for key, amount in (("iop_starters", offense), ("idp_starters", defense)):
        if key in rules:
            bound = limits(rules[key])
            if not bound or not bound[0] <= amount <= bound[1]:
                return False
    return True


def enrich_archive(root, result):
    cache = {}
    for index, file in enumerate(result["files"]):
        if file["health"] != "ok":
            continue
        with (root / file["file"]).open(encoding="utf-8") as handle:
            payload = json.load(handle)
        cache[(file["year"], Path(file["file"]).stem)] = (payload, index)

    def get(year, key, default=None):
        return cache.get((year, key), (default or {}, None))

    capture_midnight = stamp(result["archiveAt"][:10] + "T00:00:00+00:00") if result["archiveAt"] else None
    capture_cutoff = capture_midnight - 86400 if capture_midnight is not None else None
    positions = {}
    rules = {}
    anchors = {}
    coverage = []
    weeks = []
    for year in result["years"]:
        players, player_file = get(year, "players")
        positions[year] = {str(p["id"]): p.get("position") for p in rows(players.get("players", {}).get("player"))}
        league, league_file = get(year, "league")
        rules[year] = league.get("league", {}).get("starters", {})
        draft, draft_file = get(year, "draftResults")
        draft_times = [number(p.get("timestamp")) for unit in rows(draft.get("draftResults", {}).get("draftUnit"))
                       for p in rows(unit.get("draftPick")) if p.get("player")]
        draft_times = [t for t in draft_times if t and datetime.fromtimestamp(t, timezone.utc).year == year]
        starts, ends, nfl_sources = {}, {}, {}
        for (source_year, key), (payload, index) in cache.items():
            if source_year != year or not key.startswith("nflSchedule_W"):
                continue
            schedule = payload.get("nflSchedule", {})
            week = number(schedule.get("week"))
            if week is None:
                continue
            times = [number(m.get("kickoff")) for m in rows(schedule.get("matchup"))]
            times = [t for t in times if t and datetime.fromtimestamp(t, timezone.utc).year in (year, year + 1)]
            if times:
                starts[int(week)] = min(times)
                ends[int(week)] = max(times) + 86400
                nfl_sources[int(week)] = index
        weekly, weekly_file = get(year, "weeklyResults_WYTD")
        blocks = rows(weekly.get("allWeeklyResults", {}).get("weeklyResults"))
        playoff_weeks = [int(b["week"]) for b in blocks if any(m.get("regularSeason") == "0" for m in rows(b.get("matchup")))]
        first_playoff = min(playoff_weeks) if playoff_weeks else None
        anchors[year] = {"draftStart": iso(min(draft_times)) if draft_times else None,
                         "draftEnd": iso(max(draft_times)) if draft_times else None,
                         "kickoff": iso(starts.get(1)),
                         "playoffs": iso(starts.get(first_playoff)), "firstPlayoffWeek": first_playoff,
                         "startWeek": number(league.get("league", {}).get("startWeek")),
                         "end": iso(max(ends.values())) if ends else None,
                         "sources": [x for x in (draft_file, weekly_file) if x is not None] + list(nfl_sources.values())}
        incomplete = 0
        for block_index, block in enumerate(blocks):
            week = int(block.get("week", 0))
            for match_index, match in enumerate(rows(block.get("matchup"))):
                for raw in rows(match.get("franchise")):
                    score, optimum = number(raw.get("score")), number(raw.get("opt_pts"))
                    future = capture_cutoff is not None and (ends.get(week) is None or ends[week] > capture_cutoff)
                    if score is None or raw.get("result") not in ("W", "L", "T") or future:
                        incomplete += 1
                        continue
                    entries = rows(raw.get("player"))
                    actual = [p["id"] for p in entries if p.get("status") == "starter"]
                    optimal = [p for p in str(raw.get("optimal", "")).split(",") if p]
                    # MFL omits scores on some explicitly listed roster players. Zero is
                    # admitted only through exact team AND optimal-score reconciliation.
                    points = {p["id"]: number(p.get("score", "0")) for p in entries}
                    adjustment = number(raw.get("adj_score", "0"))
                    errors = []
                    if optimum is None:
                        errors.append("missing-optimal-score")
                    if adjustment is None:
                        errors.append("unknown-adjustment")
                    if not actual or not optimal or any(points.get(p) is None for p in actual + optimal):
                        errors.append("missing-player-scores")
                    if len(points) != len(entries):
                        errors.append("duplicate-player")
                    actual_sum = sum(points[p] for p in actual) if all(points.get(p) is not None for p in actual) else None
                    optimal_sum = sum(points[p] for p in optimal) if all(points.get(p) is not None for p in optimal) else None
                    residual = round(score - actual_sum - adjustment, 4) if actual_sum is not None and adjustment is not None else None
                    optimal_residual = round(optimum - optimal_sum - adjustment, 4) if optimum is not None and optimal_sum is not None and adjustment is not None else None
                    if residual is None or abs(residual) > 0.02 or optimal_residual is None or abs(optimal_residual) > 0.02:
                        errors.append("unreconciled-score")
                    if not lineup_legal(actual, positions[year], rules[year]) or not lineup_legal(optimal, positions[year], rules[year]):
                        errors.append("unverified-lineup-constraints")
                    if optimum is not None and optimum + .02 < score:
                        errors.append("optimal-below-actual")
                    weeks.append({"season": year, "week": week, "team": str(raw["id"]), "score": score,
                                  "optimal": optimum, "adjustment": adjustment, "residual": residual,
                                  "optimalResidual": optimal_residual, "flags": errors,
                                  "omittedScores": sum("score" not in p for p in entries),
                                  "gap": round(optimum - score, 4) if not errors else None,
                                  "result": raw["result"], "regular": match.get("regularSeason") == "1",
                                  "kickoff": iso(starts.get(week)), "safeAt": iso(ends.get(week)),
                                  "starters": actual, "optimalPlayers": optimal,
                                  "bench": [p["id"] for p in entries if p.get("status") != "starter"],
                                  "points": [[p, value] for p, value in points.items()],
                                  "sources": [{"file": weekly_file, "week": week, "block": block_index + 1,
                                               "matchup": match_index + 1, "team": str(raw["id"])},
                                              *([{"file": nfl_sources[week]}] if week in nfl_sources else []),
                                              *([{"file": league_file}, {"file": player_file}] if league_file is not None and player_file is not None else [])]})
        declared_end = number(league.get("league", {}).get("endWeek")) or 0
        observed_end = max((w["week"] for w in weeks if w["season"] == year), default=0)
        extent = max(declared_end, observed_end)
        anchors[year]["weekDates"] = [{"season": year, "week": week, "kickoff": iso(starts[week]),
                                       "safeAt": iso(ends[week]), "sources": [{"file": nfl_sources[week]}]}
                                      for week in sorted(starts) if week <= extent]
        season_events = [e for e in result["events"] if e["season"] == year and e["type"] == "TRADE"]
        coverage.append({"season": year, "trades": len(season_events), "pickDeals": sum(bool(e["picks"]) for e in season_events),
                         "pickTokensObserved": any(e["picks"] for e in season_events),
                         "draftAnchor": bool(draft_times), "weeklyRows": sum(w["season"] == year for w in weeks),
                         "unscoredRows": incomplete,
                         "usableLineupRows": sum(w["season"] == year and w["gap"] is not None for w in weeks),
                         "positionsSource": player_file, "rulesSource": league_file,
                         "transactionsSource": get(year, "transactions")[1]})

    # Result chronology uses a conservative, disclosed proxy, never commissioner entry time.
    histories = defaultdict(list)
    for row in sorted(weeks, key=lambda w: w["safeAt"] or "9999"):
        if row["safeAt"]:
            histories[(row["season"], row["team"])].append(row)
    history_stamps = {key: [stamp(w["safeAt"]) for w in values] for key, values in histories.items()}
    totals = {}
    for key, values in histories.items():
        wins = losses = ties = streak = 0
        previous = None
        regular_seen = set()
        anchor = anchors[key[0]]
        first_playoff = anchor["firstPlayoffWeek"]
        start_week = anchor["startWeek"]
        for row in values:
            streak = streak + 1 if previous == row["result"] else 1
            previous = row["result"]
            # Some consolation games are labelled regularSeason=1. Do not fold
            # those into the pre-entry regular-season record after playoffs begin.
            if row["regular"] and (first_playoff is None or row["week"] < first_playoff):
                regular_seen.add(row["week"])
                wins += row["result"] == "W"
                losses += row["result"] == "L"
                ties += row["result"] == "T"
            through = min(row["week"], first_playoff - 1) if first_playoff else row["week"]
            complete = start_week is not None and all(w in regular_seen for w in range(int(start_week), through + 1))
            row["record"] = {"wins": wins, "losses": losses, "ties": ties, "complete": complete}
            totals[(key, row["week"])] = {"wins": wins, "losses": losses, "ties": ties, "streak": streak,
                                                  "recordComplete": complete,
                                                  "lastResult": previous, "week": row["week"], "safeAt": row["safeAt"],
                                                  "sources": row["sources"]}
    pick_encoding = {c["season"]: c["pickTokensObserved"] for c in coverage}
    for event in result["events"]:
        event["pickEncodingObserved"] = pick_encoding.get(event["season"], False)
        flags = []
        if event["type"] == "TRADE":
            if len(event["teams"]) != 2 or len(set(event["teams"])) != 2:
                flags.append("unresolved-participants")
            if any(not side for side in event["sides"]):
                flags.append("empty-encoded-side")
            if any(a["kind"] == "unknown" for side in event["sides"] for a in side):
                flags.append("unknown-asset")
            if event["pickNote"]:
                flags.append("comment-pick-reference")
            if event.get("considerationNote"):
                flags.append("non-token-consideration-note")
        if event["type"] == "LOAD_ROSTERS":
            flags.append("manual-roster-adjustment")
        event["flags"] = flags
        event["context"] = {}
        event["phase"] = "unknown"
        event["cycle"] = None
        event["analysisSeason"] = None
        for side in event["sides"]:
            for asset in side:
                if asset["kind"] == "player":
                    asset["position"] = positions.get(event["season"], {}).get(asset["id"])
        if not event["date"]:
            continue
        when = stamp(event["date"])
        year = int(event["date"][:4])
        prev = anchors.get(year - 1, {})
        if prev.get("end") and event["date"] <= prev["end"] and event["date"][5:7] == "01":
            year -= 1
        anchor = anchors.get(year, {})
        event["analysisSeason"] = year
        draft_start, draft_end, kickoff, playoffs, end = (stamp(anchor.get(k)) for k in ("draftStart", "draftEnd", "kickoff", "playoffs", "end"))
        if kickoff and when >= kickoff and (not end or when <= end):
            event["phase"] = "postseason" if playoffs and when >= playoffs else "in-season"
        elif draft_start and when < draft_start:
            event["phase"] = "pre-draft"
        elif draft_start and draft_end and draft_start <= when <= draft_end:
            event["phase"] = "draft"
        elif draft_end and kickoff and draft_end < when < kickoff:
            event["phase"] = "post-draft"
        if draft_end:
            event["cycle"] = year + (when > draft_end)
        for team in event["teams"]:
            key = (year, team)
            candidates = histories.get(key, [])
            index = bisect_right(history_stamps.get(key, []), when) - 1
            if index >= 0 and kickoff and when >= kickoff and event["phase"] in ("in-season", "postseason"):
                latest = candidates[index]
                if when - stamp(latest["safeAt"]) < 7 * 86400:
                    event["context"][team] = totals[(key, latest["week"])]

    tenures_file = root.parent / "tools/league-history/data/tenures.json"
    tenures = validate_tenures(json.loads(tenures_file.read_text()) if tenures_file.is_file() else [], {t["id"] for t in result["teams"]})
    result["behavior"] = {"version": 1, "completedThrough": iso(capture_cutoff), "anchors": anchors, "coverage": coverage,
                           "weeks": weeks, "tenures": tenures,
                           "resultAvailability": "Proxy: last listed NFL kickoff + 24 hours; true result finalization is not archived. Subsequent seven-day windows only.",
                           "recordBasis": "Retrospective regular-season results before the first archived playoff week; later corrections may exist. Incomplete prior regular weeks cannot establish a record band.",
                           "positionBasis": "Position listed in the source-season player directory, not a verified pre-decision attribute.",
                           "coverageBasis": "Observed fields and rows, not proof of complete historical opportunity or permitted trading windows."}
    return result
