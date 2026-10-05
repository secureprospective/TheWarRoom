#!/usr/bin/env python3
"""Compile a read-only, privacy-limited browser index from the local MFL archive."""
import argparse
import collections
import datetime as dt
import hashlib
import json
from pathlib import Path
import re
import tempfile

from behavior_archive import enrich_archive


def rows(value):
    if value is None or value == "":
        return []
    if isinstance(value, dict):
        return [value]
    if isinstance(value, list) and all(isinstance(row, dict) for row in value):
        return value
    raise ValueError("expected an MFL object or object list")


def number(value):
    if value in (None, ""):
        return None
    result = float(str(value).replace("$", "").replace(",", ""))
    if not (-1e12 < result < 1e12):
        raise ValueError(f"invalid number: {value}")
    return result


def date_of(value):
    if not value:
        return None
    return dt.datetime.fromtimestamp(int(value), dt.timezone.utc).isoformat()


def assets(value, year):
    result = []
    for token in (value or "").split(","):
        token = token.strip()
        if not token:
            continue
        future = re.fullmatch(r"FP_(\d+)_(\d{4})_(\d+)", token)
        current = re.fullmatch(r"DP_(\d+)_(\d+)", token)
        if future:
            original, target_year, round_num = future.groups()
            result.append({"kind": "pick", "token": token, "year": int(target_year),
                           "round": int(round_num), "original": original})
        elif current:
            round_num, pick = current.groups()
            result.append({"kind": "pick", "token": token, "year": year,
                           "round": int(round_num), "pick": int(pick)})
        elif token.isdigit():
            result.append({"kind": "player", "id": token})
        else:
            result.append({"kind": "unknown", "token": token})
    return result


def source(path, index=None):
    result = {"file": str(path)}
    if index is not None:
        result["record"] = index
    return result


def read_json(root, path):
    file = root / path
    if not file.is_file():
        return {}
    try:
        result = json.loads(file.read_text())
    except (ValueError, OSError) as exc:
        raise ValueError(f"{path}: {exc}") from exc
    if not isinstance(result, dict):
        raise ValueError(f"{path}: expected JSON object")
    return result


def inventory(root):
    latest = {}
    for line in (root / "manifest.jsonl").read_text().splitlines():
        raw = json.loads(line)
        path = Path(raw["file"])
        if path.is_absolute() or ".." in path.parts or path.parts[0] != "raw":
            raise ValueError(f"unsafe manifest path: {path}")
        if not isinstance(raw["year"], int) or not isinstance(raw["status"], str):
            raise ValueError("invalid manifest record")
        latest[str(path)] = {key: raw[key] for key in ("file", "year", "type", "status", "bytes", "at") if key in raw}
    files = sorted(latest.values(), key=lambda row: row["file"])
    for row in files:
        row["health"] = "error" if row["status"].startswith("mfl-error") else "empty" if row["status"] == "empty" else "ok"
        row["exists"] = (root / row["file"]).is_file()
    return files


def transaction(raw, year, provenance):
    kind = raw.get("type", "UNKNOWN")
    teams = [raw[key] for key in ("franchise", "franchise2") if raw.get(key)]
    sides = []
    for key in ("franchise1_gave_up", "franchise2_gave_up"):
        sides.append(assets(raw.get(key), year))
    players = {asset["id"] for side in sides for asset in side if asset["kind"] == "player"}
    movements = {}
    for key in ("added", "dropped", "activated", "deactivated", "promoted", "demoted"):
        ids = [item for item in raw.get(key, "").split(",") if item]
        if ids:
            if not all(item.isdigit() for item in ids):
                raise ValueError(f"{provenance}: invalid movement player IDs")
            movements[key] = ids
            players.update(ids)
    if kind in ("FREE_AGENT", "LOAD_ROSTERS"):
        added, _, dropped = raw.get("transaction", "").partition("|")
        for key, value in (("added", added), ("dropped", dropped)):
            ids = [item for item in value.split(",") if item]
            if not all(item.isdigit() for item in ids):
                raise ValueError(f"{provenance}: invalid free-agent player IDs")
            if ids:
                movements[key] = ids
                players.update(ids)
    picks = [asset for side in sides for asset in side if asset["kind"] == "pick"]
    note = raw.get("comments", "")
    comment_pick = kind == "TRADE" and bool(re.search(r"\bpicks?\b|\b20\d\d\s+(?:[1-9](?:st|nd|rd|th)|round)", note, re.I))
    return {"season": year, "date": date_of(raw.get("timestamp")), "type": kind,
            "teams": teams, "players": sorted(players), "sides": sides, "movements": movements,
            "picks": len(picks), "pickNote": comment_pick, "sources": [provenance],
            "commissioner": raw.get("by_commish") == "1",
            "considerationNote": bool(re.search(r"\$|salary|cash|cap\\s+space", note, re.I))}


def champion(root, year):
    path = Path("raw") / str(year) / "playoffBrackets.json"
    brackets = rows(read_json(root, path).get("playoffBrackets", {}).get("playoffBracket"))
    for bracket in brackets:
        if not re.search(r"super\s*bowl", bracket.get("name", ""), re.I):
            continue
        detail = path.with_name(f"playoffBracket_{bracket['id']}.json")
        rounds = rows(read_json(root, detail).get("playoffBracket", {}).get("playoffRound"))
        if not rounds:
            continue
        games = rows(rounds[-1].get("playoffGame"))
        if len(games) != 1:
            continue
        home, away = games[0].get("home", {}), games[0].get("away", {})
        h, a = number(home.get("points")), number(away.get("points"))
        if h is None or a is None or h == a:
            continue
        winner = home if h > a else away
        if winner.get("franchise_id"):
            return {"team": winner["franchise_id"], "score": max(h, a), "opponentScore": min(h, a), "source": source(detail)}
    return None


def compile_archive(root):
    files = inventory(root)
    years = sorted({row["year"] for row in files})
    teams, players, seasons, events, snapshots, scores = {}, {}, [], {}, collections.defaultdict(list), collections.defaultdict(list)
    for year in years:
        base = Path("raw") / str(year)
        league = read_json(root, base / "league.json").get("league", {})
        for raw in rows(league.get("franchises", {}).get("franchise")):
            team = teams.setdefault(raw["id"], {"id": raw["id"], "names": {}})
            team.update(name=raw["name"], abbrev=raw.get("abbrev", raw["id"]))
            team["names"][str(year)] = raw["name"]
        for raw in rows(read_json(root, base / "players.json").get("players", {}).get("player")):
            if not isinstance(raw.get("id"), str) or not raw["id"].isdigit():
                raise ValueError(f"{year}: invalid player ID")
            players[raw["id"]] = {key: raw.get(key, "") for key in ("id", "name", "position", "team")}
        standings_path = base / "leagueStandings.json"
        standings = []
        for index, raw in enumerate(rows(read_json(root, standings_path).get("leagueStandings", {}).get("franchise"))):
            standings.append({"team": raw["id"], "wins": number(raw.get("h2hw")), "losses": number(raw.get("h2hl")),
                              "ties": number(raw.get("h2ht")), "pf": number(raw.get("pf")), "pa": number(raw.get("pa")),
                              "allPlay": number(raw.get("all_play_pct")), "source": source(standings_path, index)})
        transactions_path = base / "transactions.json"
        txs = rows(read_json(root, transactions_path).get("transactions", {}).get("transaction"))
        for index, raw in enumerate(txs):
            event = transaction(raw, year, source(transactions_path, index))
            fingerprint = json.dumps({key: event[key] for key in ("date", "type", "teams", "sides", "movements")}, sort_keys=True)
            # Undated records cannot be deduplicated safely across seasons.
            if event["date"] is None:
                fingerprint += f"{year}:{index}"
            key = hashlib.sha256(fingerprint.encode()).hexdigest()[:20]
            if key in events:
                events[key]["sources"].extend(event["sources"])
                events[key]["pickNote"] |= event["pickNote"]
                events[key]["considerationNote"] |= event["considerationNote"]
                events[key]["commissioner"] |= event["commissioner"]
            else:
                events[key] = dict(event, id=key)
        draft_path = base / "draftResults.json"
        drafts = []
        for unit in rows(read_json(root, draft_path).get("draftResults", {}).get("draftUnit")):
            for index, raw in enumerate(rows(unit.get("draftPick"))):
                if not raw.get("player"):
                    continue
                event = {"id": f"draft-{year}-{len(drafts)}", "season": year, "date": date_of(raw.get("timestamp")),
                         "type": "DRAFT", "teams": [raw["franchise"]], "players": [raw["player"]],
                         "round": int(raw["round"]), "pick": int(raw["pick"]), "picks": 0, "pickNote": False,
                         "sides": [], "movements": {}, "sources": [source(draft_path, index)],
                         "lineage": raw.get("comments", "")}
                drafts.append(event)
                events[event["id"]] = event
        score_path = base / "playerScores_WYTD.json"
        for raw in rows(read_json(root, score_path).get("playerScores", {}).get("playerScore")):
            scores[raw["id"]].append({"season": year, "score": number(raw.get("score")), "source": source(score_path)})
        roster_paths = [base / "rosters.json"] + [Path(row["file"]) for row in files if row["year"] == year and row["type"] == "rosters" and "/weekly/" in row["file"] and row["health"] == "ok"]
        grouped = collections.defaultdict(lambda: {"weeks": [], "sources": []})
        for roster_path in roster_paths:
            for franchise in rows(read_json(root, roster_path).get("rosters", {}).get("franchise")):
                week = franchise.get("week") or re.search(r"_W(\d+)", str(roster_path))
                if isinstance(week, re.Match):
                    week = week.group(1)
                week = int(week) if week else None
                for raw in rows(franchise.get("player")):
                    key = (raw["id"], franchise["id"], raw.get("status", ""), number(raw.get("salary")), raw.get("contractYear", ""))
                    observed = grouped[key]
                    if week not in observed["weeks"]:
                        observed["weeks"].append(week)
                    observed["sources"].append(str(roster_path))
        for (player, team, status, salary, contract), observation in grouped.items():
            snapshots[player].append({"season": year, "team": team, "status": status, "salary": salary,
                                      "contract": contract, "weeks": sorted(observation["weeks"], key=lambda w: w or 0),
                                      "sources": sorted(set(observation["sources"]))})
        seasons.append({"year": year, "name": league.get("name", ""), "standings": standings, "champion": champion(root, year),
                        "drafts": len(drafts), "transactions": len(txs), "teams": len(rows(league.get("franchises", {}).get("franchise")))})
    event_list = sorted(events.values(), key=lambda event: (event["date"] or "", event["id"]))
    archive_at = max((row.get("at", "") for row in files), default="")
    file_ids = {row["file"]: index for index, row in enumerate(files)}
    for event in event_list:
        for provenance in event["sources"]:
            provenance["file"] = file_ids[provenance["file"]]
    for season in seasons:
        for standing in season["standings"]:
            standing["source"]["file"] = file_ids[standing["source"]["file"]]
        if season["champion"]:
            provenance = season["champion"]["source"]
            provenance["file"] = file_ids[provenance["file"]]
    for observations in snapshots.values():
        for observation in observations:
            observation["sources"] = [file_ids[path] for path in observation["sources"]]
    for observations in scores.values():
        for observation in observations:
            observation["source"]["file"] = file_ids[observation["source"]["file"]]
    result = {"schema": 2, "years": years, "archiveAt": archive_at, "files": files, "teams": list(teams.values()),
            "players": players, "seasons": seasons, "events": event_list, "snapshots": snapshots, "scores": scores,
            "counts": dict(collections.Counter(row["health"] for row in files)),
            "notes": {"undated": sum(event["date"] is None for event in event_list),
                      "commentOnlyPickTrades": sum(event["type"] == "TRADE" and event["pickNote"] and not event["picks"] for event in event_list)}}
    return enrich_archive(root, result)


def atomic_write(path, content):
    """Readers see the previous complete index or the next one, never a partial JSON."""
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", dir=path.parent,
                                         prefix=path.name + ".", suffix=".tmp", delete=False) as handle:
            temporary = Path(handle.name)
            handle.write(content)
        temporary.replace(path)
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--archive", type=Path, default=Path(__file__).resolve().parents[2] / "league-archive")
    parser.add_argument("--output", type=Path, default=Path(__file__).resolve().parent / "data/archive.json")
    args = parser.parse_args()
    data = compile_archive(args.archive)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    content = json.dumps(data, ensure_ascii=False, separators=(",", ":"), allow_nan=False)
    atomic_write(args.output, content)
    revision = hashlib.sha256(content.encode()).hexdigest()[:12]
    atomic_write(args.output.with_name("revision.json"), json.dumps({"revision": revision}))
    print(f"Built {len(data['events']):,} events, {len(data['players']):,} players, {len(data['files']):,} files; {len(content.encode()) / 1e6:.1f} MB; revision {revision}")


if __name__ == "__main__":
    main()
