#!/usr/bin/env python3
"""Compile the Legacy NFL archive into the League Behavior Lab's data file.

Reads only the facts database (data/mfl.db, built by store/build.py from the MFL archive and
checked against MFL's own figures) and writes data/lab.json (and a gzip copy). Birthdates and
NFL draft slots are MFL's; days are US Eastern, as MFL shows them.

Every fact about a decision is stored as it stood at the moment of the decision:
standings through the last finished week, the contract on the last roster snapshot
before the move, production known at the time. What happened afterwards is kept in
separate hindsight fields, and the app always labels which of the two a number is.

The valuation (wins above replacement, forecasts, pick values, the market model and its
backtest) lives in value.py; the proof that the move history is complete in ledger.py.
"""
import argparse
import bisect
import collections
import datetime as dt
import gzip
import hashlib
import json
import re
import sqlite3
import statistics
from pathlib import Path
from zoneinfo import ZoneInfo

import ledger
import value

HERE = Path(__file__).resolve().parent
TOOL = HERE.parent
REPO = TOOL.parent.parent
DB = TOOL / 'data' / 'mfl.db'
OUT = TOOL / 'data' / 'lab.json'

FIRST_YEAR, LAST_YEAR = 2013, 2026
DAY = 86400
EASTERN = ZoneInfo('America/New_York')    # MFL's clock: every day the Lab shows is an Eastern day

# Eras: where the league's own rules or calendar changed. Each entry is what a GM
# would recognise, taken from league settings, scoring rules and the rulebook.
ERAS = [
    {'id': 'founding', 'name': 'Founding years', 'years': [2013, 2014],
     'summary': 'No salary cap or contracts yet. 53-man rosters and waiver claims. 2013 is only partly recorded (the league joined MFL mid-season); 12-week regular season and 12-team playoffs from 2014.'},
    {'id': 'cap', 'name': 'Cap arrives', 'years': [2015, 2016],
     'summary': '$100 cap and contracts begin. Rookie draft held off MFL, so traded picks are not recorded. Cap space could still be traded.'},
    {'id': 'draft', 'name': 'Rookie draft on MFL', 'years': [2017, 2020],
     'summary': 'Rookie draft moves onto MFL and picks are recorded in trades. Free-agent adds replace waivers. Cap trading abolished. Rosters grow to 60, then 75. 12-team playoffs, 14 from 2020.'},
    {'id': 'taxi', 'name': 'Taxi squad, 13 weeks', 'years': [2021, 2023],
     'summary': 'Practice squad of 8 arrives. Regular season becomes 13 weeks with 14-team playoffs. Cap $105, then $110. Defensive linemen score tackles.'},
    {'id': 'current', 'name': 'Current rules', 'years': [2024, 2026],
     'summary': '48-man active roster. Cap rises $5 a year ($115 to $125). New defensive-back scoring. Rookie draft in July. No trading from the deadline until March.'},
]

# NFL Draft first day, a fixed outside marker for the offseason.
NFL_DRAFT = {2013: '04-25', 2014: '05-08', 2015: '04-30', 2016: '04-28', 2017: '04-27',
             2018: '04-26', 2019: '04-25', 2020: '04-23', 2021: '04-29', 2022: '04-28',
             2023: '04-27', 2024: '04-25', 2025: '04-24', 2026: '04-23'}

POSITIONS = ['QB', 'RB', 'WR', 'TE', 'PK', 'DT', 'DE', 'LB', 'CB', 'S']
OFFENSE = {'QB', 'RB', 'WR', 'TE'}
DEFENSE = {'DT', 'DE', 'LB', 'CB', 'S'}


def num(value):
    try:
        return float(str(value).replace('$', '').replace(',', ''))
    except (TypeError, ValueError):
        return None


def connect(path=DB):
    if not path.exists():
        raise SystemExit(f'{path} is missing: run python3 store/build.py first')
    return sqlite3.connect(f'file:{path}?mode=ro', uri=True)


def era_of(year):
    for era in ERAS:
        if era['years'][0] <= year <= era['years'][1]:
            return era['id']
    raise ValueError(year)


def day_of(ts):
    return dt.datetime.fromtimestamp(ts, EASTERN).date()


def epoch(date):
    return int(dt.datetime(date.year, date.month, date.day, tzinfo=EASTERN).timestamp())


def norm_pos(pos):
    return {'Def': None, 'Off': None, 'K': 'PK'}.get(pos, pos) if pos in POSITIONS or pos in ('K',) else None


class Lab:
    def __init__(self, capture_ts, db):
        self.capture_ts = capture_ts
        self.db = db
        self.years = list(range(FIRST_YEAR, LAST_YEAR + 1))
        self.franchise_names = collections.defaultdict(dict)   # fid -> {year: name}
        self.player_name = {}
        self.player_pos = collections.defaultdict(dict)       # pid -> {year: pos}
        self.birth = {}
        self.nfl_draft = {}
        self.week_start = {}        # year -> {week: first kickoff}
        self.week_end = {}          # year -> {week: last kickoff}
        self.regular_weeks = {}     # year -> number of regular-season weeks
        self.done_weeks = {}        # year -> weeks with real results
        self.scores = collections.defaultdict(dict)  # (year, week) -> {pid: pts}
        self.results = {}           # (year, week, fid) -> dict
        self.starts = collections.defaultdict(set)   # (year, week, fid) -> starters
        self.rosters = {}           # (year, week, fid) -> {pid: contract dict}
        self.dead = collections.defaultdict(list)    # (year, fid) -> [(ts, amount)]
        self.cap = {}
        self.settings = {}

    # ---------- reference data ----------
    def read_reference(self):
        db = self.db
        for pid, name, born, nfl_year, nfl_round, nfl_pick in db.execute(
                'SELECT pid, name, birthdate, draft_year, draft_round, draft_pick FROM player'):
            self.player_name[pid] = name
            if born:
                self.birth[pid] = epoch(dt.date.fromisoformat(born))
            if nfl_year:
                self.nfl_draft[pid] = (nfl_year, nfl_round, nfl_pick)    # pick within the round, as MFL gives it
        for year, fid, name in db.execute('SELECT year, fid, name FROM franchise_season'):
            self.franchise_names[fid][year] = name
        for year, cap, raw in db.execute('SELECT year, salary_cap, raw FROM season'):
            self.cap[year] = cap
            self.settings[year] = json.loads(raw)
        for year, pid, pos in db.execute('SELECT year, pid, position FROM player_season'):
            pos = norm_pos(pos)
            if pos:
                self.player_pos[pid][year] = pos

    def pos(self, pid, year):
        by_year = self.player_pos.get(pid)
        if not by_year:
            return None
        if year in by_year:
            return by_year[year]
        near = min(by_year, key=lambda y: abs(y - year))
        return by_year[near]

    # ---------- weeks, scores, results ----------
    def read_weeks(self):
        db = self.db
        for year in self.years:
            self.week_start[year], self.week_end[year] = {}, {}
        for year, week, first, last in db.execute(
                'SELECT year, week, MIN(kickoff), MAX(kickoff) FROM nfl_game WHERE kickoff GROUP BY year, week'):
            self.week_start[year][week], self.week_end[year][week] = first, last
        for year, week, pid, pts in db.execute('SELECT year, week, pid, score FROM player_week WHERE score IS NOT NULL'):
            self.scores[(year, week)][pid] = pts
        for year, week, fid, pid, salary, until, status, note, slot in db.execute(
                'SELECT year, week, fid, pid, salary, contract_year, contract_status, contract_info, status '
                'FROM roster_week WHERE week > 0'):
            self.rosters.setdefault((year, week, fid), {})[pid] = {
                'salary': salary, 'until': int(until) if str(until or '').isdigit() else None,
                'status': status or None, 'note': (note or '').strip(), 'slot': slot}
        # Regular season and finished weeks as the facts database defines them (store/views.sql):
        # the weeks where every team had an opponent, finished once every one of them has a score.
        self.regular_weeks = {y: 0 for y in self.years}
        self.regular_weeks.update(db.execute('SELECT year, MAX(week) FROM regular_week GROUP BY year'))
        self.done_weeks = {y: set() for y in self.years}
        for year, week in db.execute('SELECT g.year, g.week FROM game g JOIN regular_week USING (year, week) '
                                     'GROUP BY g.year, g.week HAVING SUM(g.score IS NULL) = 0'):
            self.done_weeks[year].add(week)
        # Only games MFL has scored: unplayed weeks carry a blank score and a "T" that is not a tie.
        for year, week, fid, res, pf, opp, pa in db.execute(
                'SELECT g.year, g.week, g.fid, g.result, g.score, g.opp, o.score FROM played_game g '
                'LEFT JOIN game o ON o.year = g.year AND o.week = g.week AND o.fid = g.opp '
                "WHERE g.result IN ('W', 'L', 'T')"):
            self.results[(year, week, fid)] = {'res': res, 'pf': pf, 'pa': pa, 'opp': opp}
        for year, week, fid, pid in db.execute('SELECT year, week, fid, pid FROM lineup WHERE started = 1'):
            self.starts[(year, week, fid)].add(pid)

    def week_at(self, year, ts):
        """Last NFL week whose first kickoff is at or before ts (0 = before week 1)."""
        starts = self.week_start.get(year, {})
        week = 0
        for w in sorted(starts):
            if starts[w] <= ts:
                week = w
        return week

    # ---------- standings ----------
    def build_standings(self):
        """Rank all 32 teams after every finished regular-season week, league-wide."""
        self.rank_after = {}   # (year, week) -> {fid: (rank, w, l, t, pf)}
        self.final = {}        # year -> {fid: rank}
        self.last_done = {}    # year -> last finished regular-season week
        for year in self.years:
            tally = collections.defaultdict(lambda: [0, 0, 0, 0.0])
            last = 0
            for week in range(1, self.regular_weeks[year] + 1):
                if week not in self.done_weeks[year]:
                    continue
                last = week
                for fid in self.franchise_names:
                    r = self.results.get((year, week, fid))
                    if not r:
                        continue
                    t = tally[fid]
                    t[{'W': 0, 'L': 1, 'T': 2}[r['res']]] += 1
                    t[3] += r['pf']
                order = sorted(tally, key=lambda f: (-(tally[f][0] + 0.5 * tally[f][2]) / max(1, sum(tally[f][:3])), -tally[f][3]))
                self.rank_after[(year, week)] = {f: (i + 1, *tally[f]) for i, f in enumerate(order)}
            # 2013's last four weeks have no recorded results; its finish is after week 13.
            self.last_done[year] = last
            if last and (last == self.regular_weeks[year] or year < LAST_YEAR):
                self.final[year] = {f: v[0] for f, v in self.rank_after[(year, last)].items()}

    def build_playoffs(self):
        self.playoff_teams = {y: set() for y in self.years}
        self.champion, self.finalist = {}, {}
        games = collections.defaultdict(dict)     # (year, bracket) -> {(week, game): {side: (fid, points)}}
        for year, bracket, week, game, fid, side, pts in self.db.execute(
                'SELECT year, bracket_id, week, game_id, fid, side, points FROM playoff_game WHERE fid IS NOT NULL'):
            self.playoff_teams[year].add(fid)
            games[(year, bracket)].setdefault((week, game), {})[side] = (fid, pts)
        for year, bracket, name in self.db.execute('SELECT year, bracket_id, name FROM playoff_bracket'):
            title = games.get((year, bracket))
            if 'bowl' not in (name or '').lower() or not title:
                continue
            final = title[max(title)]
            (hf, hp), (af, ap) = final.get('home', (None, None)), final.get('away', (None, None))
            if hp is not None and ap is not None and (hp or ap):
                self.champion[year], self.finalist[year] = (hf, af) if hp > ap else (af, hf)

    # ---------- season calendar markers ----------
    def build_calendar(self, trades_by_year, draft_spans):
        self.marks = {}
        for year in self.years:
            starts, ends = self.week_start[year], self.week_end[year]
            kickoff = starts.get(1)
            season_end = ends.get(17) or ends.get(max(ends)) if ends else None
            clean = sorted(t['ts'] for t in trades_by_year.get(year, []) if not t['oneSided'])
            in_season = [ts for ts in clean if kickoff and kickoff <= ts <= (season_end or ts)]
            deadline, expected = None, False
            if in_season:
                last = max(in_season)
                after = [ts for ts in clean if ts > last]
                gap = (after[0] - last) / DAY if after else None
                # A real deadline is followed by a long silence (or by the end of the data
                # for a finished season). 2026 is still before its deadline.
                if (gap is None and year < LAST_YEAR) or (gap is not None and gap > 40):
                    deadline = day_of(last)
            if deadline is None and year == LAST_YEAR:
                weeks = [self.week_at(y, epoch(self.marks[y]['deadline'])) for y in (year - 1, year - 2) if self.marks.get(y, {}).get('deadline')]
                if weeks and starts.get(weeks[0] + 1):
                    deadline, expected = day_of(starts[weeks[0] + 1]) - dt.timedelta(days=4), True
            span = draft_spans.get(year)
            m, d = NFL_DRAFT[year].split('-')
            self.marks[year] = {
                'open': day_of(clean[0]) if clean else None,
                'nflDraft': dt.date(year, int(m), int(d)),
                'draftStart': day_of(span[0]) if span else None,
                'draftEnd': day_of(span[1]) if span else None,
                'kickoff': day_of(kickoff) if kickoff else None,
                'deadline': deadline, 'deadlineExpected': expected,
                'regularEnd': day_of(ends[self.regular_weeks[year]]) if self.regular_weeks[year] in ends else None,
                'seasonEnd': day_of(season_end) if season_end else None,
            }

    def phase(self, year, ts):
        """Plain-English phase of the league year, from that year's own markers."""
        mk, d = self.marks[year], day_of(ts)
        if mk['seasonEnd'] and d > mk['seasonEnd']:
            return 'winter'
        if mk['deadline'] and d > mk['deadline']:
            return 'closed'
        if mk['kickoff'] and d >= mk['kickoff']:
            if mk['deadline'] and (mk['deadline'] - d).days < 14:
                return 'deadline'
            return 'early' if self.week_at(year, ts) <= 4 else 'midseason'
        if d < mk['nflDraft']:
            return 'opening'
        if mk['draftStart'] is None:
            return 'summer'
        if d < mk['draftStart']:
            return 'between'
        if d <= mk['draftEnd']:
            return 'rookieDraft'
        return 'summer'

    # ---------- team situation at a moment ----------
    def situation(self, year, ts, fid):
        """Standing as known at ts: live rank once 3 games are final, else last season's finish."""
        week = self.week_at(year, ts)
        ends = self.week_end[year]
        finished = 0
        for w in range(1, min(week, self.regular_weeks[year]) + 1):
            if ends.get(w) and ends[w] + DAY <= ts and w in self.done_weeks[year]:
                finished = w
        if mk := self.marks[year]['regularEnd']:
            if day_of(ts) > mk and self.last_done[year]:
                finished = self.last_done[year]
        if finished >= 3:
            rank, w, l, t, pf = self.rank_after[(year, finished)][fid] if fid in self.rank_after[(year, finished)] else (None, 0, 0, 0, 0)
            return rank, 'live', finished, f'{w}-{l}' + (f'-{t}' if t else '')
        prev = self.final.get(year - 1, {})
        if fid in prev:
            return prev[fid], 'last', 0, None
        return None, None, 0, None

    def year_before(self, year, basis, fid):
        """Rank one year before the standing used at the time: last season's finish when the
        standing is live, the season before that when the standing is itself last season's finish."""
        back = {'live': 1, 'last': 2}.get(basis)
        return self.final.get(year - back, {}).get(fid) if back else None

    def contract_coverage(self, year):
        """Share of rostered players with a salary on the week-5 roster."""
        players = [c for (y, w, f), roster in self.rosters.items() if y == year and w == 5 for c in roster.values()]
        return round(sum(1 for c in players if c['salary']) / len(players), 3) if players else 0.0

    def cap_room(self, year, ts, fid):
        """Cap room on the last weekly roster before ts, after dead money charged so far.
        Only for seasons whose contracts are fully recorded on MFL."""
        week = self.week_at(year, ts)
        if week < 1 or self.cap.get(year) is None or self.coverage[year] < 0.85:
            return None
        roster = self.rosters.get((year, week, fid))
        if not roster:
            return None
        used = sum(c['salary'] or 0 for c in roster.values())
        used += sum(a for when, a in self.dead[(year, fid)] if when <= ts)
        return round(self.cap[year] - used, 2)

    def contract(self, pid, year, ts, fid):
        """Contract from the latest roster snapshot at or before ts; never a later one."""
        week = self.week_at(year, ts)
        probes = [(year, w) for w in range(week, 0, -1)] + [(year - 1, w) for w in range(17, 0, -1)]
        for y, w in probes:
            roster = self.rosters.get((y, w, fid))
            if roster and pid in roster:
                return roster[pid]
        return None

    def contract_after(self, pid, year, ts, fid):
        """Contract terms from the receiving team's next roster snapshot, for players the
        giving team acquired after its last snapshot (rookies, offseason signings)."""
        week = self.week_at(year, ts)
        for y, w in [(year, w) for w in range(week + 1, 18)] + [(year + 1, w) for w in range(1, 4)]:
            roster = self.rosters.get((y, w, fid))
            if roster and pid in roster:
                return roster[pid]
        return None

    # ---------- production ----------
    def build_production(self):
        """Season totals, games with points, league-wide position ranks, starter lines."""
        self.season_pts = collections.defaultdict(float)
        self.season_games = collections.Counter()
        self.weekly_pts = collections.defaultdict(dict)   # (pid, year) -> {week: pts}
        for (year, week), scores in self.scores.items():
            if week > self.regular_weeks[year] or week not in self.done_weeks[year]:
                continue
            for pid, pts in scores.items():
                self.weekly_pts[(pid, year)][week] = pts
                self.season_pts[(pid, year)] += pts
                if pts:
                    self.season_games[(pid, year)] += 1
        # Starter line: how many players at each position the league actually starts per week.
        self.starter_line = {}
        for year in self.years:
            counts = collections.Counter()
            weeks = [w for w in range(1, self.regular_weeks[year] + 1) if w in self.done_weeks[year]]
            for w in weeks:
                for fid in self.franchise_names:
                    for pid in self.starts.get((year, w, fid), ()):
                        pos = self.pos(pid, year)
                        if pos:
                            counts[pos] += 1
            self.starter_line[year] = {p: round(counts[p] / len(weeks)) if weeks else None for p in POSITIONS}
        self.season_rank = {}
        for year in self.years:
            by_pos = collections.defaultdict(list)
            for (pid, y), pts in self.season_pts.items():
                if y == year and pts > 0:
                    pos = self.pos(pid, year)
                    if pos:
                        by_pos[pos].append((pts, pid))
            for pos, items in by_pos.items():
                for i, (_, pid) in enumerate(sorted(items, reverse=True)):
                    self.season_rank[(pid, year)] = i + 1

    def tier_from_rank(self, rank, pos, year):
        line = self.starter_line.get(year, {}).get(pos)
        if rank is None or not line:
            return None
        if rank <= max(1, line // 4):
            return 'top'
        if rank <= line:
            return 'starter'
        if rank <= line * 2:
            return 'depth'
        return 'fringe'

    def known_tier(self, pid, year, ts):
        """Production tier a GM could see at ts."""
        pos = self.pos(pid, year)
        if not pos:
            return None
        week = self.week_at(year, ts)
        if week >= 5:
            done = [w for w in self.done_weeks[year] if w < week and w <= self.regular_weeks[year]]
            mine = self.weekly_pts.get((pid, year), {})
            pts, games = sum(mine.get(w, 0) for w in done), sum(1 for w in done if mine.get(w))
            if games >= 3:
                rank = self.live_rank(pos, year, tuple(done), pts / games)
                return self.tier_from_rank(rank, pos, year)
        prev = year - 1
        if (pid, prev) in self.season_rank:
            return self.tier_from_rank(self.season_rank[(pid, prev)], pos, prev)
        drafted = self.nfl_draft.get(pid)
        if drafted and drafted[0] >= prev:
            return 'rookie'
        return 'none'

    def live_rank(self, pos, year, done, ppg):
        key = (pos, year, done)
        if key not in self._live:
            vals = []
            for (pid, y), weeks in self.weekly_pts.items():
                if y != year or self.pos(pid, year) != pos:
                    continue
                pts = sum(weeks.get(w, 0) for w in done)
                games = sum(1 for w in done if weeks.get(w))
                if games >= 3:
                    vals.append(pts / games)
            self._live[key] = sorted(vals, reverse=True)
        vals = self._live[key]
        return bisect.bisect_left([-v for v in vals], -ppg) + 1

    def after_use(self, pid, year, ts, fid):
        """Hindsight: starts and starter points for the receiving team over the next 17 played weeks."""
        out_starts, out_pts, seen = 0, 0.0, 0
        y, w = year, self.week_at(year, ts) + 1
        while seen < 17 and y <= LAST_YEAR:
            if w > self.regular_weeks.get(y, 0):
                y, w = y + 1, 1
                continue
            if w not in self.done_weeks.get(y, ()):
                break
            seen += 1
            if pid in self.starts.get((y, w, fid), ()):
                out_starts += 1
                out_pts += self.weekly_pts.get((pid, y), {}).get(w, 0)
            w += 1
        return (out_starts, round(out_pts, 1), seen) if seen else (None, None, 0)


OUTCOME = {'TRADE_ACCEPT': 'accepted', 'TRADE_REJECTION': 'rejected', 'TRADE_REVOKE': 'withdrawn',
           'TRADE_OFFER_EXPIRED': 'expired'}


def offer_outcomes(db, lab):
    """Every trade offer in the archive and how it ended. MFL shows a franchise only its own offers,
    so every one involves the archive owner's franchise. A proposal is matched to the first later
    outcome between the same two teams with the same expiry (and the same message, where one fits);
    one left unmatched ends 'unknown'. The stretch of the year and the season follow the proposal."""
    rows = db.execute('SELECT txn_id, year, ts, type, offered_by, offered_to, comments, expires FROM offer '
                      'WHERE ts ORDER BY ts, txn_id').fetchall()
    outcomes = [r for r in rows if r[3] in OUTCOME]
    used = set()
    O = collections.defaultdict(list)
    for txn_id, year, ts, kind, by, to, comments, expires in rows:
        if kind != 'TRADE_PROPOSAL':
            continue
        later = [o for o in outcomes if o[0] not in used and o[4] == by and o[5] == to and o[7] == expires and o[2] >= ts]
        same = [o for o in later if (o[6] or '') == (comments or '')]
        end = min(same or later, key=lambda o: (o[2], o[0])) if (same or later) else None
        if end:
            used.add(end[0])
        regular_end = lab.marks.get(year, {}).get('regularEnd')
        O['ts'].append(ts); O['year'].append(year)
        O['season'].append(year + 1 if regular_end and day_of(ts) > regular_end else year)
        O['phase'].append(lab.phase(year, ts) if year in lab.marks else None)
        O['by'].append(by); O['to'].append(to)
        O['outcome'].append(OUTCOME[end[3]] if end else 'unknown')
        O['decided'].append(end[2] if end else None)
    return dict(O)


def build(capture_ts, db):
    lab = Lab(capture_ts, db)
    lab._live = {}
    lab.read_reference()
    lab.read_weeks()
    lab.build_standings()
    lab.build_playoffs()
    lab.build_production()
    lab.coverage = {y: lab.contract_coverage(y) for y in lab.years}

    db = lab.db
    # Dead money charged to each franchise, with time, for cap room at the time. Each season's
    # list holds every cut still charging that season's cap (35% of salary per year left).
    for year, fid, amount, ts in db.execute('SELECT year, fid, amount, ts FROM cap_charge WHERE amount AND ts'):
        lab.dead[(year, fid)].append((ts, amount))

    # Draft selections: picks where a player was taken (MFL marks skipped or forfeited picks
    # with no player).
    draft, draft_spans = [], {}
    for year, rnd, slot, fid, pid, ts in db.execute(
            'SELECT year, round, pick, fid, pid, ts FROM draft_pick WHERE pid IS NOT NULL AND ts '
            'ORDER BY year, round, pick'):
        draft.append({'year': year, 'round': rnd, 'slot': slot, 'fid': fid, 'pid': pid, 'ts': ts})
        first, last = draft_spans.get(year, (ts, ts))
        draft_spans[year] = (min(first, ts), max(last, ts))

    # Completed trades, each side's assets in MFL's order. A pick carries its original team
    # (future picks) or its slot (current-year picks), never both: that is all MFL records.
    gave = collections.defaultdict(list)    # (txn_id, from) -> assets
    for txn_id, frm, kind, pid, pick_year, pick_round, pick_slot, orig in db.execute(
            "SELECT a.txn_id, a.from_fid, a.kind, a.pid, a.pick_year, a.pick_round, a.pick_slot, a.pick_orig "
            "FROM txn_asset a JOIN txn t USING (txn_id) WHERE t.type = 'TRADE' ORDER BY a.rowid"):
        gave[(txn_id, frm)].append({'kind': 'player', 'id': pid} if kind == 'player' else
                                   {'kind': 'pick', 'orig': orig, 'year': pick_year, 'round': pick_round, 'slot': pick_slot})
    trades, trades_by_year = [], collections.defaultdict(list)
    for txn_id, year, ts, a, b in db.execute(
            "SELECT txn_id, year, ts, fid, fid2 FROM txn WHERE type = 'TRADE' AND ts ORDER BY txn_id"):
        a_gave, b_gave = gave[(txn_id, a)], gave[(txn_id, b)]
        t = {'year': year, 'ts': ts, 'a': a, 'b': b, 'aGave': a_gave, 'bGave': b_gave, 'oneSided': not a_gave or not b_gave}
        trades.append(t)
        trades_by_year[year].append(t)

    # Every other roster move, and every offer and how it ended, for the ledger check.
    kinds = {'add': 'add', 'load': 'add', 'drop': 'drop', 'taxi_off': 'taxiUp', 'taxi_on': 'taxiDown',
             'ir_on': 'irOn', 'ir_off': 'irOff'}
    moves = [{'year': year, 'ts': ts, 'fid': fid, 'kind': kinds[role], 'pid': pid} for year, ts, fid, role, pid in db.execute(
        "SELECT t.year, t.ts, t.fid, a.role, a.pid FROM txn t JOIN txn_asset a USING (txn_id) "
        "WHERE t.type IN ('FREE_AGENT', 'LOAD_ROSTERS', 'WAIVER', 'TAXI', 'IR') AND t.ts ORDER BY t.txn_id, a.rowid")]
    offers = {'TRADE_PROPOSAL': 'offer', 'TRADE_REJECTION': 'offerDeclined', 'TRADE_REVOKE': 'offerWithdrawn',
              'TRADE_OFFER_EXPIRED': 'offerExpired'}
    moves += [{'year': year, 'ts': ts, 'fid': fid, 'kind': offers[kind], 'pid': None, 'other': other}
              for year, ts, fid, kind, other in db.execute(
                  'SELECT year, ts, fid, type, fid2 FROM txn WHERE type IN (%s) AND ts ORDER BY txn_id'
                  % ','.join(f"'{k}'" for k in offers))]

    lab.build_calendar(trades_by_year, draft_spans)
    ledger_check = ledger.reconcile(lab, trades, moves)
    offers = offer_outcomes(db, lab)

    # Draft order by original owner, used to place a traded future pick in its eventual slot.
    # Order follows the previous regular season, worst first; the champion and finalist pick last.
    slot_of = {}
    for year in lab.years:
        prev = lab.final.get(year - 1)
        if not prev:
            continue
        playoff = lab.playoff_teams.get(year - 1, set())
        champ, runner = lab.champion.get(year - 1), lab.finalist.get(year - 1)
        def key(f):
            return (2 if f == champ else 1 if f == runner else 0, 1 if f in playoff else 0, -prev[f])
        for i, f in enumerate(sorted(prev, key=key)):
            slot_of[(year, f)] = i + 1
    # True original slots: every round uses the same order, so any round where a team
    # used its own untouched pick (never traded, and the only pick it made that round)
    # gives that team's slot for the whole draft.
    traded_fp, moved_dp = set(), set()
    for t in trades:
        for asset in t['aGave'] + t['bGave']:
            if asset['kind'] == 'pick' and asset['orig']:
                traded_fp.add((asset['orig'], asset['year'], asset['round']))
            elif asset['kind'] == 'pick':
                moved_dp.add((asset['year'], asset['round'], asset['slot']))
    per_round = collections.Counter((d['year'], d['round'], d['fid']) for d in draft)
    evidence = collections.defaultdict(collections.Counter)
    for d in draft:
        if (per_round[(d['year'], d['round'], d['fid'])] == 1 and (d['fid'], d['year'], d['round']) not in traded_fp
                and (d['year'], d['round'], d['slot']) not in moved_dp):
            evidence[(d['year'], d['fid'])][d['slot']] += 1
    known = {k: c.most_common(1)[0][0] for k, c in evidence.items()}
    checked = sum(1 for k in known if k in slot_of)
    matched = sum(1 for k, v in known.items() if slot_of.get(k) == v)
    estimated = dict(slot_of)
    # Proof of the zero-based reading: the last holder of each traded current-year pick
    # should be the team that made that selection.
    last_holder = {}
    for t in sorted(trades, key=lambda t: t['ts']):
        for gave, receiver in ((t['aGave'], t['b']), (t['bGave'], t['a'])):
            for asset in gave:
                if asset['kind'] == 'pick' and asset['slot']:
                    last_holder[(asset['year'], asset['round'], asset['slot'])] = receiver
    made_by = {(d['year'], d['round'], d['slot']): d['fid'] for d in draft}
    pick_proof = {'checked': sum(1 for k in last_holder if k in made_by),
                  'matched': sum(1 for k, f in last_holder.items() if made_by.get(k) == f)}
    slot_check = {'teamDraftsWithKnownSlot': len(known), 'standingsEstimateChecked': checked,
                  'standingsEstimateMatched': matched}
    by_slot = {(d['year'], d['round'], d['slot']): d for d in draft}
    analysis = value.analyse(lab, trades, draft, known, estimated)

    franchises = sorted(lab.franchise_names)

    def years_out(pick_year, ts, year):
        """0 = the next rookie draft, 1 = the one after, and so on."""
        d = day_of(ts)
        mk = lab.marks.get(d.year) or {}
        end = mk.get('draftEnd') or dt.date(d.year, 6, 1)
        next_draft = d.year if d <= end else d.year + 1
        return pick_year - next_draft

    def age_at(pid, ts):
        b = lab.birth.get(pid)
        return round((ts - b) / (365.25 * DAY), 1) if b else None

    # ---------- outputs ----------
    T = collections.defaultdict(list)   # trades
    L = collections.defaultdict(list)   # legs: one asset moving one way
    trade_index = {}
    for t in sorted(trades, key=lambda t: t['ts']):
        year, ts = t['year'], t['ts']
        tid = len(T['ts'])
        trade_index[id(t)] = tid
        sides = []
        for fid in (t['a'], t['b']):
            rank, basis, played, record = lab.situation(year, ts, fid)
            sides.append((rank, basis, record, lab.year_before(year, basis, fid)))
        np_a = sum(a['kind'] == 'player' for a in t['aGave']); nk_a = sum(a['kind'] == 'pick' for a in t['aGave'])
        np_b = sum(a['kind'] == 'player' for a in t['bGave']); nk_b = sum(a['kind'] == 'pick' for a in t['bGave'])
        mk = lab.marks[year]
        T['ts'].append(ts); T['year'].append(year); T['a'].append(t['a']); T['b'].append(t['b'])
        T['aRank'].append(sides[0][0]); T['bRank'].append(sides[1][0]); T['rankBasis'].append(sides[0][1])
        T['aRecord'].append(sides[0][2]); T['bRecord'].append(sides[1][2])
        T['aPrev'].append(sides[0][3]); T['bPrev'].append(sides[1][3])
        T['aPlayers'].append(np_a); T['aPicks'].append(nk_a); T['bPlayers'].append(np_b); T['bPicks'].append(nk_b)
        T['aOther'].append(len(t['aGave']) - np_a - nk_a); T['bOther'].append(len(t['bGave']) - np_b - nk_b)
        T['oneSided'].append(t['oneSided'])
        T['phase'].append(lab.phase(year, ts)); T['week'].append(lab.week_at(year, ts))
        T['toDeadline'].append((mk['deadline'] - day_of(ts)).days if mk['deadline'] else None)
        T['toDraft'].append((mk['draftStart'] - day_of(ts)).days if mk['draftStart'] else None)
        T['aCap'].append(lab.cap_room(year, ts, t['a'])); T['bCap'].append(lab.cap_room(year, ts, t['b']))
        T['aName'].append(lab.franchise_names[t['a']].get(year)); T['bName'].append(lab.franchise_names[t['b']].get(year))
        deal = analysis['_deals'].get(id(t), {})
        for k in ('aNet', 'aRealNet', 'realSeasons', 'window', 'season'):
            T[k].append(deal.get(k))
        for giver, receiver, gave, gi, ri in ((t['a'], t['b'], t['aGave'], 0, 1), (t['b'], t['a'], t['bGave'], 1, 0)):
            for asset_i, asset in enumerate(gave):
                lv = analysis['_legs'].get((id(t), gi, asset_i), {})
                for k in ('value', 'class', 'price', 'real', 'realSeasons'):
                    L['x' + k[0].upper() + k[1:]].append(lv.get(k))
                L['trade'].append(tid); L['from'].append(giver); L['to'].append(receiver)
                L['fromRank'].append(sides[gi][0]); L['toRank'].append(sides[ri][0])
                L['fromPrev'].append(sides[gi][3]); L['toPrev'].append(sides[ri][3])
                L['kind'].append(asset['kind'])
                if asset['kind'] == 'player':
                    pid = asset['id']
                    pos = lab.pos(pid, year)
                    c, seen_on = lab.contract(pid, year, ts, giver), 'before'
                    if c is None:
                        c, seen_on = lab.contract_after(pid, year, ts, receiver), 'after'
                    L['contractFrom'].append(seen_on if c else None)
                    starts, pts, seen = lab.after_use(pid, year, ts, receiver)
                    L['pid'].append(pid); L['pos'].append(pos); L['age'].append(age_at(pid, ts))
                    L['salary'].append(c['salary'] if c else None)
                    # Winter trades (Jan-Feb after the season) already count the next season.
                    contract_season = max(year, day_of(ts).year)
                    L['yearsLeft'].append(max(0, c['until'] - contract_season + 1) if c and c['until'] else None)
                    L['contract'].append(c['status'] if c else None)
                    L['tier'].append(lab.known_tier(pid, year, ts))
                    L['afterStarts'].append(starts); L['afterPts'].append(pts); L['afterWeeks'].append(seen)
                    for k in ('pickYear', 'round', 'yearsOut', 'slot', 'third', 'orig', 'origRank', 'pickPid'):
                        L[k].append(None)
                elif asset['kind'] == 'pick':
                    for k in ('pid', 'pos', 'age', 'salary', 'yearsLeft', 'contract', 'contractFrom', 'tier', 'afterStarts', 'afterPts', 'afterWeeks'):
                        L[k].append(None)
                    orig = asset['orig']
                    # Exact slot only: a current-year pick code, or a team whose slot that
                    # year is proven by an untouched pick. Otherwise the player is not named.
                    slot = asset['slot'] or known.get((asset['year'], orig))
                    made = by_slot.get((asset['year'], asset['round'], slot)) if slot else None
                    guess = slot or estimated.get((asset['year'], orig))
                    L['pickYear'].append(asset['year']); L['round'].append(asset['round'])
                    L['yearsOut'].append(years_out(asset['year'], ts, year))
                    L['slot'].append(slot)
                    L['third'].append(None if guess is None else 'early' if guess <= 11 else 'middle' if guess <= 22 else 'late')
                    L['orig'].append(orig)
                    L['origRank'].append(lab.situation(year, ts, orig)[0] if orig else None)
                    L['pickPid'].append(made['pid'] if made else None)
                else:
                    for k in ('pid', 'pos', 'age', 'salary', 'yearsLeft', 'contract', 'contractFrom', 'tier', 'afterStarts', 'afterPts', 'afterWeeks',
                              'pickYear', 'round', 'yearsOut', 'slot', 'third', 'orig', 'origRank', 'pickPid'):
                        L[k].append(None)

    # Rookie draft outcomes: seasons at starter level in this league's scoring.
    def starter_seasons(pid, first_year, span=3):
        out, best = 0, None
        for y in range(first_year, min(first_year + span, LAST_YEAR + 1)):
            if y == LAST_YEAR:
                continue
            rank = lab.season_rank.get((pid, y))
            pos = lab.pos(pid, y)
            tier = lab.tier_from_rank(rank, pos, y) if rank else None
            if tier in ('top', 'starter'):
                out += 1
            if tier and (best is None or ['top', 'starter', 'depth', 'fringe'].index(tier) < ['top', 'starter', 'depth', 'fringe'].index(best)):
                best = tier
        return out, best

    D = collections.defaultdict(list)
    order = collections.Counter()
    for d in sorted(draft, key=lambda d: (d['year'], d['round'], d['slot'])):
        order[d['year']] += 1
        pid, year = d['pid'], d['year']
        n, best = starter_seasons(pid, year)
        seasons_seen = max(0, min(3, LAST_YEAR - year))
        pts3 = sum(lab.season_pts.get((pid, y), 0) for y in range(year, min(year + 3, LAST_YEAR)))
        nfl = lab.nfl_draft.get(pid)
        D['year'].append(year); D['round'].append(d['round']); D['slot'].append(d['slot'])
        D['overall'].append(order[year]); D['fid'].append(d['fid']); D['pid'].append(pid)
        D['pos'].append(lab.pos(pid, year)); D['ts'].append(d['ts'])
        D['group'].append(value.GROUP.get(lab.pos(pid, year)))
        D['starterSeasons'].append(n); D['best'].append(best); D['seasonsSeen'].append(seasons_seen)
        D['pts3'].append(round(pts3, 1))
        D['nflRound'].append(nfl[1] if nfl and nfl[0] == year else None)
        D['nflPick'].append(nfl[2] if nfl and nfl[0] == year else None)     # pick within the NFL round
        D['age'].append(age_at(pid, d['ts']))
        rank, basis, _, _ = lab.situation(year, d['ts'], d['fid'])
        D['rank'].append(rank); D['prev'].append(lab.year_before(year, basis, d['fid']))
        # Still on the drafting team at the next season's week 1?
        kept = (year + 1, 1, d['fid']) in lab.rosters and pid in lab.rosters[(year + 1, 1, d['fid'])]
        D['keptYear2'].append(kept if (year + 1) <= LAST_YEAR else None)
        key = (year, d['round'], d['slot'])
        weeks = analysis['_weeks']
        D['wins'].append([None if r is None else round(r * weeks.get(year + j, 13), 3)
                          for j, r in enumerate(analysis['_pickSeasons'].get(key, [None] * value.HORIZON))])
        D['slotWins'].append([None if r is None else round(r * weeks.get(year + j, 13), 3)
                              for j, r in enumerate(analysis['_pickSlots'].get(key, [None] * value.HORIZON))])

    # Backtest records: each traded asset's expected wins at the time (from earlier seasons only)
    # against the wins it delivered over the seasons played since.
    X = collections.defaultdict(list)
    for r in analysis['_delivered']:
        X['trade'].append(trade_index[id(r['t'])]); X['class'].append(r['class'])
        X['expected'].append(round(r['expected'], 4)); X['delivered'].append(round(r['delivered'], 4))
        X['seasons'].append(r['seasons'])

    # Team seasons: finish, playoffs, title, activity, cap at kickoff, roster age.
    S = collections.defaultdict(list)
    for year in lab.years:
        final = lab.final.get(year, {})
        reg = lab.regular_weeks[year]
        for fid in franchises:
            if year not in lab.franchise_names[fid]:
                continue
            rec = lab.rank_after.get((year, lab.last_done[year]), {}).get(fid)
            r1 = lab.rosters.get((year, 1, fid))
            used = sum(c['salary'] or 0 for c in r1.values()) if r1 else None
            ages = []
            for w in range(1, reg + 1):
                for pid in lab.starts.get((year, w, fid), ()):
                    a = age_at(pid, lab.week_start[year].get(w, 0))
                    if a:
                        ages.append(a)
            S['year'].append(year); S['fid'].append(fid)
            S['rank'].append(final.get(fid) if year < LAST_YEAR else None)
            S['liveRank'].append(rec[0] if rec else None)
            S['w'].append(rec[1] if rec else None); S['l'].append(rec[2] if rec else None); S['t'].append(rec[3] if rec else None)
            S['pf'].append(round(rec[4], 1) if rec else None)
            S['playoffs'].append(fid in lab.playoff_teams.get(year, set()) if lab.playoff_teams.get(year) else None)
            S['champion'].append(lab.champion.get(year) == fid)
            S['finalist'].append(lab.finalist.get(year) == fid)
            S['capUsed'].append(round(used, 2) if used is not None else None)
            notes = [c['note'] for c in r1.values()] if r1 else []
            # Contract notes are free text typed by the commissioner: "EXT1 (2024)...",
            # "Extended in 2023...", "FT1 (2025)", "Franchised 2024...".
            S['extended'].append(sum(1 for n in notes if re.search(r'\bEXT|Extend', n, re.I)) if r1 else None)
            S['tagged'].append(sum(1 for n in notes if re.match(r'\s*(FT|Franchis)', n, re.I)) if r1 else None)
            S['starterAge'].append(round(statistics.mean(ages), 2) if ages else None)
            S['name'].append(lab.franchise_names[fid][year])
            ap = analysis['_allPlay'].get((year, fid), {})
            for k in ('allPlay', 'expectedWins', 'luck'):
                S[k].append(ap.get(k))

    # Player directory (only players who appear in an output).
    seen = set(x for x in L['pid'] if x) | set(D['pid']) | set(x for x in L['pickPid'] if x)
    P = collections.defaultdict(list)
    for pid in sorted(seen):
        P['id'].append(pid); P['name'].append(lab.player_name.get(pid, pid))
        P['pos'].append(lab.pos(pid, LAST_YEAR)); P['birth'].append(lab.birth.get(pid))
        nfl = lab.nfl_draft.get(pid)
        P['nflYear'].append(nfl[0] if nfl else None); P['nflRound'].append(nfl[1] if nfl else None)

    def iso(d):
        return d.isoformat() if d else None

    seasons = []
    for year in lab.years:
        mk = lab.marks[year]
        st = lab.settings[year]
        seasons.append({
            'year': year, 'era': era_of(year), 'cap': lab.cap[year], 'rosterSize': num(st.get('rosterSize')),
            'taxi': num(st.get('taxiSquad')), 'regularWeeks': lab.regular_weeks[year],
            'contractsRecorded': lab.coverage[year], 'lastFinishedWeek': lab.last_done[year],
            'playoffTeams': len(lab.playoff_teams.get(year, ())) or None,
            'weeksDone': sorted(lab.done_weeks[year]),
            'weekStart': {w: lab.week_start[year][w] for w in sorted(lab.week_start[year])},
            'starterLine': lab.starter_line[year],
            'champion': lab.champion.get(year), 'finalist': lab.finalist.get(year),
            'marks': {k: (iso(v) if not isinstance(v, bool) else v) for k, v in mk.items()},
        })

    franchise_rows = []
    for fid in franchises:
        names = lab.franchise_names[fid]
        latest = names[max(names)]
        div = db.execute('SELECT division FROM franchise_season WHERE year = ? AND fid = ?', (max(names), fid)).fetchone()[0]
        franchise_rows.append({'id': fid, 'name': latest, 'short': latest.split()[-1],
                               'formerNames': sorted({n for n in names.values() if n != latest}), 'division': div})

    data = {
        'schema': 'league-lab-2',
        'built': dt.datetime.now(dt.timezone.utc).isoformat(timespec='seconds'),
        'archiveThrough': dt.datetime.fromtimestamp(capture_ts, dt.timezone.utc).isoformat(timespec='seconds'),
        'eras': ERAS, 'seasons': seasons, 'franchises': franchise_rows,
        'checks': {'draftOrderEstimate': slot_check, 'currentPickCodes': pick_proof,
                   'birthdateCoverage': round(sum(1 for p in P['id'] if lab.birth.get(p)) / max(1, len(P['id'])), 3)},
        'trades': dict(T), 'legs': dict(L), 'draft': dict(D), 'delivered': dict(X),
        'teamSeasons': dict(S), 'players': dict(P),
        'offers': offers,
        **{k: v for k, v in analysis.items() if not k.startswith('_')},
        'ledger': ledger_check,
        'source': source_checks(db),
    }
    return data


def source_checks(db):
    """The facts database's own proof: its checks against MFL's figures, and the fact exam
    against MFL's web pages (store/factexam.py), when that has been run."""
    checks = [{'name': n, 'checked': c, 'agreed': a, 'explained': e, 'open': o, 'detail': d}
              for n, c, a, e, o, d in db.execute(
                  'SELECT name, checked, agreed, explained, open, detail FROM check_result ORDER BY rowid')]
    exam = TOOL / 'data' / 'factexam.json'
    answers = json.loads(exam.read_text()) if exam.exists() else []
    return {'checks': checks, 'pages': {'asked': len(answers), 'matched': sum(1 for a in answers if a[3] is True)}}


def capture_time():
    """Latest archive capture that brought data (not an MFL error). Manifest times are Beelink
    local time without a zone."""
    manifest = REPO / 'league-archive' / 'manifest.jsonl'
    latest = 0
    for line in manifest.read_text().splitlines():
        try:
            row = json.loads(line)
            at = row.get('at')
            if at and str(row.get('status', 'ok')).startswith('ok'):
                latest = max(latest, int(dt.datetime.fromisoformat(at).astimezone().timestamp()))
        except ValueError:
            continue
    if not latest:
        raise SystemExit('manifest has no capture times')
    return latest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, default=OUT)
    args = parser.parse_args()
    data = build(capture_time(), connect())
    body = json.dumps(data, separators=(',', ':'), allow_nan=False).encode()
    args.out.parent.mkdir(parents=True, exist_ok=True)
    tmp = args.out.with_suffix('.tmp')
    tmp.write_bytes(body)
    tmp.replace(args.out)
    gz = args.out.with_suffix('.json.gz')
    tmpgz = gz.with_suffix('.tmp')
    tmpgz.write_bytes(gzip.compress(body, 9))
    tmpgz.replace(gz)
    revision = hashlib.sha256(body).hexdigest()[:16]
    (args.out.parent / 'lab-revision.json').write_text(json.dumps({'revision': revision}))
    print(f"wrote {args.out} ({len(body)/1e6:.1f} MB, gzip {gz.stat().st_size/1e6:.1f} MB) revision {revision}")
    print(f"trades {len(data['trades']['ts'])} (valued {data['checks2']['valuedTrades']})  legs {len(data['legs']['kind'])}  "
          f"draft {len(data['draft']['pid'])}  team-seasons {len(data['teamSeasons']['fid'])}  "
          f"ledger {data['ledger']['changeRate']}  currency {data['checks2']['currency']['correlation']}")


if __name__ == '__main__':
    main()
