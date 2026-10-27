#!/usr/bin/env python3
"""Replay the current engine offline on SQLite backup-API snapshots; render a triage board."""
import argparse
import collections
import contextlib
import datetime as dt
import gzip
import hashlib
import html
import io
import json
import math
import os
from pathlib import Path
import sqlite3
import subprocess
from urllib.parse import parse_qs, urlparse


@contextlib.contextmanager
def readonly(path):
    connection = sqlite3.connect(path.resolve().as_uri() + "?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    try:
        yield connection
    finally:
        connection.close()


def snapshot(source, destination):
    if destination.exists():
        raise ValueError(f"Refusing to replace snapshot: {destination}; choose a new --out directory")
    with readonly(source) as src:
        with destination.open('xb'):
            pass
        dst = sqlite3.connect(destination)
        try:
            src.backup(dst)
        finally:
            dst.close()
    destination.chmod(0o600)


def archived_export(connection, kind, season, required=True):
    rows = connection.execute("""SELECT f.url, f.fetched_at, r.size, r.body, r.sha256
        FROM fetch_log f JOIN raw_archive r USING (sha256)
        WHERE f.status = 200 AND f.error = '' ORDER BY f.fetched_at DESC, f.fetch_id DESC""")
    for row in rows:
        url = urlparse(row["url"])
        query = parse_qs(url.query)
        if query.get("TYPE") != [kind] or url.path.split("/")[1:2] != [str(season)]:
            continue
        if not 0 <= row['size'] <= 256 * 1024 * 1024:
            raise ValueError(f"Unreasonable archived {kind} size")
        with gzip.GzipFile(fileobj=io.BytesIO(row['body'])) as compressed:
            body = compressed.read(row['size'] + 1)
        if len(body) != row["size"] or hashlib.sha256(body).hexdigest() != row["sha256"]:
            raise ValueError(f"Corrupt archived {kind} body")
        return body, row["fetched_at"]
    if required:
        raise ValueError(f"No held {season} {kind} export; no network fallback is permitted")
    return None, None


def prepare(profile, out):
    out.mkdir(parents=True, exist_ok=True, mode=0o700)
    for name in ("history.db", "thewarroom.db"):
        snapshot(profile / name, out / name)
    with readonly(out / "thewarroom.db") as main:
        mirror = main.execute("SELECT season, snapshot, refreshed_at FROM league_mirror WHERE id = 1").fetchone()
        if mirror is None:
            raise ValueError("No held league mirror")
        roster = json.loads(mirror["snapshot"])
        season = mirror["season"]
        if roster.get("Season") != season or not roster.get("Players"):
            raise ValueError("League mirror has an invalid season or empty roster")
        config = json.loads(main.execute("""SELECT v.payload FROM rulebook_versions v
            JOIN rulebook_active a ON a.version = v.version""").fetchone()[0])
        overrides = {r["param_key"] + ("@" + r["position"] if r["position"] else ""): float(r["value"])
                     for r in main.execute("SELECT * FROM param_overrides")}
    with readonly(out / "history.db") as history:
        players, players_at = archived_export(history, "players", season)
        free_body, free_at = archived_export(history, "freeAgents", season, required=False)
        as_of = history.execute("SELECT MAX(as_of) FROM observations").fetchone()[0]
        points_dates = [dict(r) for r in history.execute("""SELECT season, MAX(as_of) AS as_of
            FROM observations WHERE measure = ? AND season IN (?, ?) GROUP BY season""",
            ("outcome.fantasy_points", season - 1, season))]
        runs = [dict(r) for r in history.execute(
            "SELECT * FROM model_runs WHERE season = ? ORDER BY run_id DESC LIMIT 2", (season,))]
    ids = [p["MFLID"] for p in roster["Players"]]
    if len(ids) != len(set(ids)) or any(not isinstance(i, str) or not i.isdigit() for i in ids):
        raise ValueError("Invalid or duplicate roster player IDs")
    if not as_of or any(not math.isfinite(v) for v in overrides.values()):
        raise ValueError("Missing observation cutoff or non-finite override")
    request = dict(Season=season, LastWeek=int(config["EndWeek"]), AsOf=as_of,
                   Roster=roster["Players"], Overrides=overrides)
    (out / "players.json").write_bytes(players)
    if free_body is not None:
        (out / "free-agents.json").write_bytes(free_body)
    (out / "request.json").write_text(json.dumps(request))
    provenance = dict(profile=str(profile), snapshot_date=dt.datetime.now(dt.timezone.utc).isoformat(),
                      as_of=as_of, season=season, roster_at=mirror["refreshed_at"],
                      players_at=players_at, free_agents_at=free_at or 'unavailable (not valued)',
                      free_agents_available=free_body is not None, points_dates=points_dates,
                      held_runs=runs)
    provenance['source_commit'] = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'], text=True).strip()
    provenance['snapshot_sha256'] = {}
    for name in ('history.db', 'thewarroom.db'):
        with (out / name).open('rb') as snapshot_file:
            provenance['snapshot_sha256'][name] = hashlib.file_digest(snapshot_file, 'sha256').hexdigest()
    (out / "provenance.json").write_text(json.dumps(provenance, indent=2))
    return request, provenance


def export_features(out, request):
    scope = json.loads((out / 'measures.json').read_text())
    measures = scope['Measures']
    placeholders = ','.join('?' for _ in measures)
    query = f"""WITH latest AS (
        SELECT *, ROW_NUMBER() OVER (
            PARTITION BY player_id, season, week, measure, source ORDER BY as_of DESC) AS newest
        FROM observations WHERE as_of <= ? AND (season = 0 OR season BETWEEN ? AND ?)
            AND measure IN ({placeholders})
    ), picked AS (
        SELECT l.*, ROW_NUMBER() OVER (
            PARTITION BY l.player_id, l.season, l.week, l.measure ORDER BY f.priority) AS pick
        FROM latest l JOIN source_fields f ON f.source = l.source AND f.measure = l.measure
        WHERE l.newest = 1
    ) SELECT * FROM picked WHERE pick = 1 ORDER BY season, player_id, week, measure"""
    with readonly(out / 'history.db') as history, (out / 'features.jsonl').open('w') as target:
        for row in history.execute(query, [request['AsOf'], request['Season'] - scope['Lookback'], request['Season'], *measures]):
            feature = dict(PlayerID=row['player_id'], Season=row['season'], Week=row['week'],
                           Measure=row['measure'], Source=row['source'], AsOf=row['as_of'],
                           Value=row['value'] or 0, Text=row['text'] or '')
            target.write(json.dumps(feature, separators=(',', ':')) + '\n')


def ranks(rows, key):
    """Competition ranks: equal numbers share a rank; absent outcomes are not zero."""
    result, previous, rank = {}, None, 0
    ordered = sorted((r for r in rows if r.get(key) is not None), key=lambda r: (-r[key], r["id"]))
    for index, row in enumerate(ordered, 1):
        if row[key] != previous:
            rank = index
        result[row["id"]] = rank
        previous = row[key]
    return result


def assess(players, roster):
    owned = {p["MFLID"]: p for p in roster}
    rows = []
    for player in players:
        score = player["Score"]
        row = dict(player, id=score["MFLID"], position=score["Position"], value=score["DynastyPPG"],
                   now=score["NowPPG"], points=player["ThisPoints"], last=player["LastPoints"],
                   ppg=player["CurrentPPG"], name=player["Facts"]["Name"], flags=[], checks=[],
                   contract=owned.get(score["MFLID"]))
        if any(not math.isfinite(row[key]) for key in ("value", "now")):
            raise ValueError("Non-finite engine output")
        rows.append(row)
    counts = collections.Counter()
    for position in sorted({r["position"] for r in rows}):
        group = [r for r in rows if r["position"] == position]
        value_ranks, points_ranks, ppg_ranks = (ranks(group, key) for key in ("value", "points", "ppg"))
        for row in group:
            row["vr"] = value_ranks[row["id"]]
            row["pr"] = points_ranks.get(row["id"])
            row["ppr"] = ppg_ranks.get(row["id"])
            gap = row["vr"] - row["pr"] if row["pr"] else 0
            row["gap"] = gap
            if row["points"] is not None and abs(gap) >= 20 and min(row["vr"], row["pr"]) <= 20:
                row["checks"].append("Rank gap")
                direction = "below" if gap > 0 else "above"
                row["flags"].append(f"Value rank sits {abs(gap)} places {direction} current points rank.")
            if row["points"] is not None and row["points"] > 0 and row["pr"] <= 20 and row["value"] <= 0:
                row["checks"].append("Zero-valued producer")
                row["flags"].append("Top-20 current scorer has no dynasty value.")
            age = row["Age"]
            old_at = {"QB": 34, "RB": 29, "WR": 31, "TE": 32, "K": 36,
                      "DT": 31, "DE": 31, "LB": 30, "CB": 30, "S": 31}[position]
            if age is not None and age >= old_at and row["vr"] <= 30 and row["points"] is not None and row["last"] is not None:
                younger = [y for y in group if y["Age"] is not None and age - y["Age"] >= 4
                           and y["value"] < row["value"] and y["points"] is not None and y["last"] is not None
                           and y["points"] > max(0, row["points"]) * 1.25
                           and y["last"] > max(0, row["last"]) * 1.25]
                if younger:
                    peer = max(younger, key=lambda y: (y["points"], y["id"]))
                    row["checks"].append("Older over producer")
                    row["flags"].append(f"Above {peer['name']} ({peer['Age']:.1f}), ≥4 years younger with >25% more points in both seasons.")
            counts.update(row["checks"])
    return rows, counts


def top_reads(rows):
    return sorted((r for r in rows if r["flags"]),
                  key=lambda r: (-len(r["checks"]), -abs(r["gap"]), r["position"], r["id"]))


def summary(rows, counts, provenance):
    lines = [f"Data: {provenance['profile']}; held observations through {provenance['as_of']}.",
             f"Current source replay: {len(rows)} players; {sum(bool(r['flags']) for r in rows)} unique candidates."]
    for check in ("Rank gap", "Zero-valued producer", "Older over producer"):
        lines.append(f"{check}: {counts[check]}")
    lines.append("\n| Player | Pos | Value rank / points rank | Candidate reason |")
    lines.append("|---|---|---|---|")
    for row in top_reads(rows)[:20]:
        lines.append(f"| {row['name']} | {row['position']} | {row['vr']} / {row['pr'] or '—'} | {' '.join(row['flags'])} |")
    return "\n".join(lines)


def render(rows, counts, provenance, template):
    escape = lambda value: html.escape(str(value), quote=True)
    sections = []
    for pos in ("QB", "RB", "WR", "TE", "K", "DT", "DE", "LB", "CB", "S"):
        group = sorted((r for r in rows if r["position"] == pos), key=lambda r: (-r["value"], r["id"]))
        rendered = []
        for row in group:
            sc, facts = row["Score"], row["Facts"]
            contract = row["contract"]
            contract_text = (f"${contract['Salary']/100_000_000:.2f}M · {contract['ContractYear']} · "
                             f"{contract['ContractStatus']} · {contract['RosterStatus']} · franchise {contract['FranchiseID']}"
                             if contract else "Held free agent · no league contract")
            if contract and contract.get("ContractInfo"):
                contract_text += " · " + contract["ContractInfo"]
            number = lambda n: "—" if n is None else f"{n:.1f}"
            pull = f" Age pull {row['AgePull']:.3f}." if row.get('AgePull') is not None else ''
            details = (f"{row['Rule']}.{pull} Prior percentile {sc['Prior']:.3f}; earlier effective games {sc['PastGames']:.1f}; "
                       f"current games {sc['SeasonGames']:.0f}; earlier production weight {sc['ZPast']:.1%}; "
                       f"current production weight {sc['ZNow']:.1%}; next-season on-field chance {sc['OnField']:.1%}. "
                       f"Inputs: {', '.join(sc['Inputs'])}. "
                       "Diagnostics remain model diagnostics even when the last-season rule supplies the value.")
            cells = [row["vr"], row["name"], facts["Team"] or "—", number(row["Age"]),
                     f"{row['value']:.2f}", f"{row['now']:.2f}", number(row["last"]), number(row["points"]),
                     row["pr"] or "—", number(row["ppg"]), row["ppr"] or "—"]
            row_class = ' class="flagged"' if row["flags"] else ""
            rendered.append(f'<tr{row_class} data-search="{escape(row["name"].lower())}" data-flag="{int(bool(row["flags"]))}">' +
                            "".join(f"<td>{escape(c)}</td>" for c in cells) +
                            f'<td class="reading">{escape(" ".join(row["flags"]) or "No check triggered.")}' +
                            f'<details><summary>Contract & value inputs</summary><p>{escape(contract_text)}</p>' +
                            f'<p>{escape(details)}</p><small>MFL ID {escape(row["id"])}</small></details></td></tr>')
        sections.append(f'<section id="{pos}"><h2>{pos} <small>{len(group)} players · '
                        f'{sum(bool(r["flags"]) for r in group)} candidates</small></h2>'
                        '<div class="table-wrap" tabindex="0" aria-label="Scrollable player table"><table>'
                        f'<caption>{pos} — dynasty points per game, highest first</caption><thead><tr>' +
                        ''.join(f'<th scope="col">{h}</th>' for h in
                                ('Value rank', 'Player', 'NFL', 'Age', 'Dyn PPG', 'Now PPG', 'Last pts', 'This pts',
                                 'Pts rank', 'Actual PPG', 'PPG rank', 'Review reading')) +
                        '</tr></thead><tbody>' + ''.join(rendered) + '</tbody></table></div></section>')
    info = escape(f"Held {provenance['season']} league data through {provenance['as_of']}; "
                  f"rosters {provenance['roster_at']}; players {provenance['players_at']}; free agents {provenance['free_agents_at']}.")
    checks = ' · '.join(f'{escape(k)}: <strong>{counts[k]}</strong>' for k in
                        ('Rank gap', 'Zero-valued producer', 'Older over producer'))
    coverage = []
    for pos in ('QB', 'RB', 'WR', 'TE', 'K', 'DT', 'DE', 'LB', 'CB', 'S'):
        group = [r for r in rows if r['position'] == pos]
        coverage.append(f"{pos}: {sum(r['points'] is None for r in group)}/{len(group)} current totals missing")
    coverage_text = escape('; '.join(coverage))
    period = escape(f"{provenance['season']} season · observations held through {provenance['as_of'][:10]} · "
                    f"{len(rows):,} valued players")
    pool = ('Held MFL free agents are included alongside rostered players.' if provenance.get('free_agents_available')
            else 'No held free-agent feed: this board covers rostered players only.')
    return (template.replace('@@PROVENANCE@@', info).replace('@@CHECKS@@', checks)
            .replace('@@POOL@@', pool).replace('@@PERIOD@@', period).replace('@@COVERAGE@@', coverage_text)
            .replace('@@TABLES@@', ''.join(sections)))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--profile', type=Path, default=Path.home() / '.config/TheWarRoom')
    parser.add_argument('--out', type=Path, default=Path('.scratch/valuation-review'))
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    out = args.out.resolve()
    if not out.is_relative_to(root) or out == root:
        parser.error('--out must be a subdirectory of this worktree')
    request, provenance = prepare(args.profile.resolve(), out)
    env = dict(os.environ, TWR_VALUE_BOARD_DIR=str(out))
    test = ['go', 'test', '-race', '-tags=valueboard', './internal/modelrun', '-count=1', '-timeout=5m', '-run']
    subprocess.run([*test, '^TestValueBoardMeasures$'], cwd=root, env=env, check=True)
    export_features(out, request)
    subprocess.run([*test, '^TestValueBoardReplay$'], cwd=root, env=env, check=True)
    replay = json.loads((out / 'replay.json').read_text())
    rows, counts = assess(replay['Players'], request['Roster'])
    board = render(rows, counts, provenance, (root / 'scripts/value_board.html').read_text())
    (out / 'warroom-value-board.html').write_text(board)
    (out / 'report.md').write_text(summary(rows, counts, provenance) + '\n')
    print(summary(rows, counts, provenance))
    print(f"\nBoard: {out / 'warroom-value-board.html'}")
    print('Replay report:', json.dumps(replay['Report']))


if __name__ == '__main__':
    main()
