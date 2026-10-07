#!/usr/bin/env python3
"""Write a plain-English card for every trade, offer, draft pick, player season, franchise season
and season, from the facts in data/mfl.db, and index the cards for exact-word search (SQLite FTS5).

Cards are for finding things. Each card carries the keys of the rows it was written from
(`ref`), so an answer always goes back to the facts for its numbers. Cards are written by code,
never by a model, so the same facts always give the same card.

    python3 store/cards.py          # after store/load.py; then store/index.py adds meaning search
"""
import collections
import datetime as dt
import json
import re
import sqlite3
import sys
from pathlib import Path
from zoneinfo import ZoneInfo

DB = Path(__file__).resolve().parent.parent / 'data' / 'mfl.db'
MY_TEAM = '0025'    # the archive was pulled with this franchise's login

DDL = '''
DROP TABLE IF EXISTS card;
DROP TABLE IF EXISTS card_fts;
CREATE TABLE card (
  card_id  TEXT PRIMARY KEY,        -- kind:keys, e.g. trade:2421, player_season:2024:14073
  kind     TEXT NOT NULL,           -- trade, offer, draft_pick, player_season, team_season, season
  year     INTEGER,
  fids     TEXT,                    -- franchises involved, space-separated
  pids     TEXT,                    -- players involved, space-separated
  ref      TEXT NOT NULL,           -- JSON: the fact rows this card was written from
  title    TEXT NOT NULL,
  body     TEXT NOT NULL
);
CREATE VIRTUAL TABLE card_fts USING fts5(title, body, content='card', content_rowid='rowid',
                                         tokenize="unicode61 tokenchars '._'");
'''

ROUND = {1: '1st', 2: '2nd', 3: '3rd'}


def ordinal(n):
    return ROUND.get(n, f'{n}th')


def possessive(name):
    return f"{name}'" if name.endswith('s') else f"{name}'s"


EASTERN = ZoneInfo('America/New_York')    # MFL shows dates in US Eastern


def day(ts):
    return dt.datetime.fromtimestamp(ts, EASTERN).strftime('%Y-%m-%d') if ts else 'date unknown'


def money(x):
    return f'${x:,.2f}M' if x is not None else 'no salary listed'


class Writer:
    def __init__(self, db):
        self.db = db
        q = db.execute
        self.names = {(y, f): n for y, f, n in q('SELECT year, fid, name FROM franchise_season')}
        self.latest_name = {}
        for (y, f), n in sorted(self.names.items()):
            self.latest_name[f] = n
        self.players = {r[0]: r for r in q('''SELECT pid, name, position, birthdate, draft_year, draft_team,
                                              draft_round, draft_pick, college FROM player''')}
        self.nfl_team = {(y, p): t for y, p, t in q('SELECT year, pid, nfl_team FROM player_season')}
        self.season_name = {(y, p): n for y, p, n in q('SELECT year, pid, name FROM player_season')}
        self.aliases = collections.defaultdict(set)    # every name MFL has listed a player under
        for p, n in q('SELECT DISTINCT pid, name FROM player_season WHERE name IS NOT NULL'):
            self.aliases[p].add(n)
        self.kickoffs = collections.defaultdict(dict)    # year -> week -> (first, last)
        for y, w, a, b in q('SELECT year, week, MIN(kickoff), MAX(kickoff) FROM nfl_game GROUP BY 1, 2'):
            if a:
                self.kickoffs[y][w] = (a, b)
        self.draft_span = {y: (a, b) for y, a, b in q('SELECT year, MIN(ts), MAX(ts) FROM draft_pick WHERE ts > 0 GROUP BY 1')}
        self.drafted = {(y, r, s): (f, p) for y, r, s, f, p in q('SELECT year, round, pick, fid, pid FROM draft_pick')}
        self.pick_orig = self.original_owners()
        self.cards = []

    # ---------- words ----------

    def team(self, year, fid):
        return self.names.get((year, fid)) or self.latest_name.get(fid) or f'franchise {fid}'

    def player(self, pid, year=None):
        p = self.players.get(pid)
        if not p:
            return f'player {pid} (not in MFL\'s player lists)'
        name = self.plain_name(pid, year)
        team = self.nfl_team.get((year, pid)) if year else None
        return f'{name} ({p[2]}{", " + team if team and team != "FA" else ""})'

    def plain_name(self, pid, year=None):
        """The name MFL listed that season, else the latest."""
        p = self.players.get(pid)
        if not p:
            return pid
        raw = self.season_name.get((year, pid)) or p[1]
        last, _, first = raw.partition(', ')
        return f'{first} {last}'.strip() if first else raw

    def other_names(self, pid, year=None):
        shown = self.season_name.get((year, pid)) or (self.players.get(pid) or [None, None])[1]
        others = sorted(n for n in self.aliases.get(pid, ()) if n != shown and ', ' in n)
        return ('MFL has also listed him as ' + ' and '.join(f'{n.partition(", ")[2]} {n.partition(", ")[0]}'
                                                             for n in others) + '.') if others else None

    def when(self, year, ts):
        """Where in the league year a moment falls, from MFL's own dates."""
        if not ts:
            return 'time unknown'
        weeks = self.kickoffs.get(year, {})
        span = self.draft_span.get(year)
        if weeks:
            first_kick = weeks[min(weeks)][0]
            last_kick = weeks[max(weeks)][1]
            if ts >= first_kick:
                if ts > last_kick:
                    return 'after the season'
                week = max(w for w, (a, _) in weeks.items() if a <= ts)
                return f'season week {week}'
        if span:
            if ts < span[0]:
                return 'offseason, before the rookie draft'
            if ts <= span[1]:
                return 'during the rookie draft'
            return 'after the rookie draft, before the season'
        return 'offseason'

    def original_owners(self):
        """(year, round, pick) -> original franchise, from MFL's "Pick traded from" notes."""
        anyname = collections.defaultdict(set)
        for (y, f), n in self.names.items():
            anyname[n].add(f)
        out = {}
        for y, r, s, f, com in self.db.execute('SELECT year, round, pick, fid, comments FROM draft_pick'):
            chain = re.findall(r'Pick traded from ([^.\n]+)\.', com or '')
            cand = anyname.get(chain[0].strip(), set()) if chain else {f}
            out[(y, r, s)] = next(iter(cand)) if len(cand) == 1 else None
        return out

    def became(self, year, rnd, slot=None, orig=None):
        """Who a pick became, where MFL's draft results say."""
        if slot is None:
            hits = [k for k, o in self.pick_orig.items() if k[0] == year and k[1] == rnd and o == orig]
            if len(hits) != 1:
                return None
            slot = hits[0][2]
        got = self.drafted.get((year, rnd, slot))
        if not got:
            return None
        return f'pick {rnd}.{slot:02d}, ' + (f'used on {self.player(got[1], year)}' if got[1] else 'no player taken')

    def asset(self, year, code, pid, py, rnd, slot, orig):
        if pid:
            return self.player(pid, year)
        if py:
            if slot:
                text = f'{py} pick {rnd}.{slot:02d} ({ordinal(rnd)} round)'
            else:
                text = f'{py} {ordinal(rnd)}-round pick (originally {possessive(self.team(year, orig))})'
            b = self.became(py, rnd, slot, orig)
            return text + (f', which became {b}' if b and not slot else (f', {b.split(", ", 1)[1]}' if b else ''))
        return f'asset {code}'

    def add(self, card_id, kind, year, fids, pids, ref, title, lines):
        body = '\n'.join(x for x in lines if x)
        self.cards.append((card_id, kind, year, ' '.join(sorted(set(f for f in fids if f))),
                           ' '.join(sorted(set(p for p in pids if p))), json.dumps(ref), title, body))

    # ---------- cards ----------

    def trades(self):
        legs = collections.defaultdict(list)
        for row in self.db.execute('''SELECT txn_id, from_fid, to_fid, code, pid, pick_year, pick_round, pick_slot,
                                      pick_orig FROM txn_asset WHERE role = 'sent' '''):
            legs[row[0]].append(row[1:])
        readings = collections.defaultdict(list)
        has_readings = self.db.execute("SELECT 1 FROM sqlite_master WHERE name = 'note_reading'").fetchone()
        for tid, text, clarity in (self.db.execute('''SELECT txn_id, reading, clarity FROM note_reading
                                                     WHERE kind NOT IN ('none') ORDER BY rowid''') if has_readings else []):
            readings[tid].append(text + ('' if clarity == 'clear' else f' [{clarity}]'))
        for tid, year, ts, a, b, commish, note in self.db.execute(
                "SELECT txn_id, year, ts, fid, fid2, by_commish, comments FROM txn WHERE type = 'TRADE'"):
            sides = []
            for frm, to in ((a, b), (b, a)):
                items = [self.asset(year, c, p, py, r, s, o) for f, t, c, p, py, r, s, o in legs[tid] if f == frm]
                sides.append(f'{self.team(year, frm)} sent: ' + ('; '.join(items) if items else 'nothing recorded in MFL'))
            one_sided = any(not [x for x in legs[tid] if x[0] == f] for f in (a, b))
            pre_picks = year < 2017
            self.add(f'trade:{tid}', 'trade', year, [a, b], [x[3] for x in legs[tid]], {'txn': tid},
                     f'Trade {day(ts)}: {self.team(year, a)} and {self.team(year, b)}',
                     [f'Trade completed {day(ts)} ({year} league year, {self.when(year, ts)}).',
                      *sides,
                      f'Note recorded with the trade: "{note}"' if note else None,
                      'Reading of the note (not yet confirmed by Christopher): ' + '; '.join(readings[tid]) + '.'
                      if readings[tid] else None,
                      'Entered by the commissioner.' if commish else None,
                      'Before 2017 MFL did not record draft picks in trades; any picks are only in the note.' if pre_picks else None,
                      'One side has no assets recorded in MFL.' if one_sided and not pre_picks else None])

    def offers(self):
        words = {'TRADE_PROPOSAL': 'Trade offer made', 'TRADE_ACCEPT': 'Trade offer accepted',
                 'TRADE_REJECTION': 'Trade offer rejected', 'TRADE_REVOKE': 'Trade offer withdrawn',
                 'TRADE_OFFER_EXPIRED': 'Trade offer expired'}
        legs = collections.defaultdict(list)
        for row in self.db.execute('''SELECT a.txn_id, a.from_fid, a.code, a.pid, a.pick_year, a.pick_round, a.pick_slot,
                                      a.pick_orig FROM txn_asset a JOIN txn t USING (txn_id)
                                      WHERE t.type IN ('TRADE_PROPOSAL','TRADE_ACCEPT','TRADE_REJECTION','TRADE_REVOKE',
                                      'TRADE_OFFER_EXPIRED')'''):
            legs[row[0]].append(row[1:])
        for tid, year, ts, kind, a, b, note in self.db.execute(f'''SELECT txn_id, year, ts, type, fid, fid2, comments
                FROM txn WHERE type IN ({",".join("?" * len(words))})''', list(words)):
            sides = []
            for frm in (a, b):
                items = [self.asset(year, c, p, py, r, s, o) for f, c, p, py, r, s, o in legs[tid] if f == frm]
                sides.append(f'{self.team(year, frm)} would send: ' + ('; '.join(items) if items else 'nothing'))
            self.add(f'offer:{tid}', 'offer', year, [a, b], [x[2] for x in legs[tid]], {'txn': tid},
                     f'{words[kind]} {day(ts)}: {self.team(year, a)} and {self.team(year, b)}',
                     [f'{words[kind]} on {day(ts)} ({year} league year, {self.when(year, ts)}). '
                      f'Offer by {self.team(year, a)} to {self.team(year, b)}.',
                      *sides,
                      f'Message with the offer: "{note}"' if note else None,
                      f'MFL shows offers only to the owners involved; these records cover offers involving '
                      f'{self.latest_name.get(MY_TEAM, MY_TEAM)}, whose login pulled the archive.'])

    def draft_picks(self):
        started = self.started_points()
        for y, r, s, fid, pid, ts, com in self.db.execute(
                'SELECT year, round, pick, fid, pid, ts, comments FROM draft_pick ORDER BY 1, 2, 3'):
            orig = self.pick_orig.get((y, r, s))
            later = []
            if pid:
                seasons = sorted((yy, pts, n) for (yy, pp), (pts, n) in started.items() if pp == pid and yy >= y)
                if seasons:
                    later.append('Since the draft: ' + '; '.join(f'{yy}: {n} starts, {pts:.1f} points as a starter'
                                                                 for yy, pts, n in seasons))
                else:
                    later.append('Never started a game for a Legacy NFL team since.')
            what = 'startup draft' if y == 2013 else 'rookie draft'
            self.add(f'draft_pick:{y}:{r}:{s}', 'draft_pick', y, [fid, orig], [pid], {'draft_pick': [y, r, s]},
                     f'{y} {what}, {ordinal(r)} round, pick {r}.{s:02d}: {self.team(y, fid)} '
                     + (f'took {self.plain_name(pid, y)}' if pid else 'took no one'),
                     [f'{y} {what}, a {ordinal(r)}-round pick (round {r}, pick {s}), made {day(ts)} by {self.team(y, fid)}.',
                      f'Selection: {self.player(pid, y)}.' if pid else 'MFL marks this pick as skipped or forfeited.',
                      f'The pick originally belonged to {self.team(y, orig)}.' if orig and orig != fid else None,
                      f'MFL\'s note: {com}' if com else None,
                      self.draft_line(pid), self.other_names(pid, y), *later])

    def draft_line(self, pid):
        p = self.players.get(pid)
        if not p:
            return None
        born = f'born {p[3]}' if p[3] else 'birthdate not listed by MFL'
        by = f' by {p[5]}' if p[5] and p[5] != 'FA' else ''    # MFL lists some drafting teams as FA
        nfl = (f'NFL draft {p[4]}, round {p[6]}, pick {p[7]}{by}' if p[4] and p[6]
               else f'NFL draft class {p[4]}' if p[4] else None)
        return '; '.join(x for x in (born, nfl, f'college: {p[8]}' if p[8] else None) if x) + '.'

    def started_points(self):
        out = {}
        for y, pid, pts, n in self.db.execute('''SELECT year, pid, SUM(score), COUNT(*) FROM lineup
                                                 WHERE started = 1 GROUP BY 1, 2'''):
            out[(y, pid)] = (pts or 0.0, n)
        return out

    def player_seasons(self):
        rows = collections.defaultdict(list)
        for y, pid, fid, week, started, should, score in self.db.execute(
                'SELECT year, pid, fid, week, started, should_start, score FROM lineup ORDER BY year, pid, week'):
            rows[(y, pid)].append((fid, week, started, should, score))
        regular = collections.defaultdict(set)
        for y, w in self.db.execute('SELECT year, week FROM regular_week'):
            regular[y].add(w)
        contracts = {(y, p): (s, cy, st, info) for y, p, s, cy, st, info in self.db.execute(
            'SELECT year, pid, salary, contract_year, contract_status, contract_info FROM contract')}
        moves = collections.defaultdict(list)
        for y, ts, typ, role, frm, to, pid in self.db.execute('''
                SELECT t.year, t.ts, t.type, a.role, a.from_fid, a.to_fid, a.pid FROM txn t JOIN txn_asset a USING (txn_id)
                WHERE a.kind = 'player' AND t.type IN ('TRADE','FREE_AGENT','WAIVER','LOAD_ROSTERS','IR','TAXI')
                ORDER BY t.ts'''):
            what = {'sent': f'traded from {self.team(y, frm)} to {self.team(y, to)}',
                    'add': f'added by {self.team(y, to)} ({"waiver" if typ == "WAIVER" else "free agent"})',
                    'drop': f'dropped by {self.team(y, frm)}', 'load': f'loaded onto {self.team(y, to)}\'s roster',
                    'ir_on': f'put on injured reserve by {self.team(y, frm)}',
                    'ir_off': f'activated from injured reserve by {self.team(y, frm)}',
                    'taxi_on': f'moved to {self.team(y, frm)}\'s taxi squad',
                    'taxi_off': f'promoted from {self.team(y, frm)}\'s taxi squad'}[role]
            moves[(y, pid)].append(f'{day(ts)} {what}')
        for (y, pid), weeks in rows.items():
            p = self.players.get(pid)
            teams = []
            for fid in dict.fromkeys(w[0] for w in weeks):
                ws = [w for w in weeks if w[0] == fid]
                reg = [w for w in ws if w[1] in regular[y]]
                st = [w for w in ws if w[2]]
                pts = sum(w[4] or 0 for w in st)
                should_benched = sum(1 for w in ws if not w[2] and w[3] == 1)
                teams.append(f'{self.team(y, fid)}: on the roster in {len(ws)} weeks (weeks {ws[0][1]}-{ws[-1][1]}), '
                             f'started {len(st)} ({sum(1 for w in reg if w[2])} in the regular season), '
                             f'{pts:.1f} points as a starter'
                             + (f'; benched {should_benched} times when MFL says he should have started' if should_benched else ''))
            all_pts = [w[4] for w in weeks if w[4] is not None]
            age = None
            if p and p[3]:
                age = (dt.date(y, 9, 1) - dt.date.fromisoformat(p[3])).days / 365.25
            c = contracts.get((y, pid))
            contract = (f'Contract on MFL for {y}: {money(c[0])}, contract year {c[1] or "not set"}, '
                        f'status {c[2] or "not set"}' + (f', note "{c[3]}"' if c[3] else '') + '.') if c else None
            self.add(f'player_season:{y}:{pid}', 'player_season', y, [w[0] for w in weeks], [pid],
                     {'player': pid, 'year': y},
                     f'{self.plain_name(pid, y)}, {y} season',
                     [f'{self.player(pid, y)} in the {y} season' + (f', aged {age:.1f} on September 1.' if age else '.'),
                      *teams,
                      f'Weekly scores when rostered: {len(all_pts)} weeks, best {max(all_pts):.1f}, average {sum(all_pts) / len(all_pts):.1f}.'
                      if all_pts else None,
                      contract, self.draft_line(pid), self.other_names(pid, y),
                      'Moves: ' + '; '.join(moves[(y, pid)]) + '.' if moves[(y, pid)] else None])

    def team_seasons(self):
        champs = self.champions()
        trades = collections.Counter()
        for y, a, b in self.db.execute("SELECT year, fid, fid2 FROM txn WHERE type = 'TRADE'"):
            trades[(y, a)] += 1
            trades[(y, b)] += 1
        adds = dict(((y, f), n) for y, f, n in self.db.execute(
            "SELECT year, fid, COUNT(*) FROM roster_move WHERE role = 'add' GROUP BY 1, 2"))
        dead = dict(((y, f), (n, s)) for y, f, n, s in self.db.execute(
            'SELECT year, fid, COUNT(*), SUM(amount) FROM cap_charge GROUP BY 1, 2'))
        top = collections.defaultdict(list)
        for y, f, pid, pts in self.db.execute('''SELECT year, fid, pid, SUM(score) s FROM lineup WHERE started = 1
                                                 GROUP BY 1, 2, 3 ORDER BY s DESC'''):
            if len(top[(y, f)]) < 5:
                top[(y, f)].append(f'{self.plain_name(pid)} {pts:.1f}')
        for r in self.db.execute('''SELECT year, fid, name, division_name, mfl_w, mfl_l, mfl_t, mfl_pf, mfl_pa, mfl_pp,
                                    reg_w, reg_l, reg_t, reg_pf, all_play_w, all_play_l, all_play_t, all_play_source
                                    FROM team_season'''):
            (y, f, name, div, w, l, t, pf, pa, pp, rw, rl, rt, rpf, aw, al, at, src) = r
            n, s = dead.get((y, f), (0, 0.0))
            self.add(f'team_season:{y}:{f}', 'team_season', y, [f], [], {'team_season': [y, f]},
                     f'{name}, {y} season: record and results',
                     [f'{name} ({f}), {div}, {y} season.',
                      f'Record in MFL\'s standings: {w}-{l}-{t}, {pf:,.2f} points for and {pa:,.2f} against (every week MFL scored).'
                      if w is not None else 'MFL published no standings.',
                      f'Regular season from the weekly games: {rw}-{rl}-{rt}, {rpf:,.2f} points.' if rw is not None else None,
                      f'All-play: {aw}-{al}-{at}' + (' (MFL\'s figure).' if src == 'MFL' else ' (counted MFL\'s way from weekly scores).')
                      if aw is not None else None,
                      f'Potential points (best possible lineups): {pp:,.2f}.' if pp else None,
                      champs.get((y, f)),
                      f'Trades: {trades[(y, f)]}. Players added: {adds.get((y, f), 0)}.',
                      f'Cap charges on MFL\'s {y} list: {n}, totalling {money(s)}.' if n else None,
                      'Top starters by points: ' + ', '.join(top[(y, f)]) + '.' if top[(y, f)] else None])

    def champions(self):
        out = {}
        for y, bid, name, title in self.db.execute('SELECT year, bracket_id, name, winner_title FROM playoff_bracket'):
            games = self.db.execute('''SELECT week, game_id, fid, points FROM playoff_game
                                       WHERE year = ? AND bracket_id = ? ORDER BY week''', (y, bid)).fetchall()
            if not games:
                continue
            last = max(g[0] for g in games)
            final = [g for g in games if g[0] == last]
            if len(final) == 2 and final[0][3] is not None and final[1][3] is not None and final[0][3] != final[1][3]:
                win, lose = sorted(final, key=lambda g: -g[3])
                label = title or name
                out[(y, win[2])] = (out.get((y, win[2]), '') + f' Won {name} ({label}), {win[3]:.2f} to {lose[3]:.2f}.').strip()
                out[(y, lose[2])] = (out.get((y, lose[2]), '') + f' Lost {name}, {lose[3]:.2f} to {win[3]:.2f}.').strip()
        return out

    def seasons(self):
        for y, name, cap, size, taxi, ir, starters in self.db.execute(
                'SELECT year, name, salary_cap, roster_size, taxi_squad, injured_reserve, starters FROM season'):
            weeks = [w for (w,) in self.db.execute('SELECT week FROM regular_week WHERE year = ? ORDER BY week', (y,))]
            st = json.loads(starters or 'null') or {}
            lineup = ', '.join(f'{p["name"]} {p["limit"]}' for p in (st.get('position') or []) if isinstance(p, dict))
            rules = self.db.execute('SELECT COUNT(*) FROM scoring_rule WHERE year = ?', (y,)).fetchone()[0]
            span = self.draft_span.get(y)
            self.add(f'season:{y}', 'season', y, [], [], {'season': y}, f'{y} season: how the league was set up',
                     [f'{name}, {y} season.',
                      f'Regular season: weeks {weeks[0]}-{weeks[-1]} ({len(weeks)} weeks).' if weeks else None,
                      f'Salary cap {money(cap)}; roster size {size}; taxi squad {taxi}; injured reserve {ir}.',
                      f'Starting lineup: {st.get("count", "?")} starters ({lineup}).' if lineup else None,
                      f'Scoring rules: {rules}.',
                      f'Rookie draft held {day(span[0])} to {day(span[1])}.' if span else
                      ('MFL kept no rookie draft results for this season.' if y != 2013 else None)])

    def run(self):
        for step in (self.seasons, self.team_seasons, self.trades, self.offers, self.draft_picks, self.player_seasons):
            step()
        self.db.executescript(DDL)
        self.db.executemany('INSERT INTO card (card_id, kind, year, fids, pids, ref, title, body) VALUES (?,?,?,?,?,?,?,?)',
                            self.cards)
        self.db.execute("INSERT INTO card_fts(card_fts) VALUES ('rebuild')")
        self.db.commit()
        return collections.Counter(c[1] for c in self.cards)


def main(path=DB):
    db = sqlite3.connect(path)
    counts = Writer(db).run()
    for k, n in sorted(counts.items()):
        print(f'{k:14s} {n:>7,}')
    print(f'{"total":14s} {sum(counts.values()):>7,}')


if __name__ == '__main__':
    sys.exit(main())
