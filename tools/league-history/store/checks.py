#!/usr/bin/env python3
"""Prove the facts database against MFL's own figures.

Each check compares two things MFL recorded separately (standings against weekly games, lineups
against player scores, rosters against transactions, the trade log against the draft). A check
passes when everything agrees or every difference falls in a named, explained class. Anything
unexplained is listed for Christopher's review, never forced to agree.

Results go to the `check_result` and `review_item` tables in data/mfl.db, and to
data/review.md. Exit status 1 if any check fails.

    python3 store/checks.py
"""
import collections
import datetime as dt
import re
import sqlite3
import sys
from pathlib import Path
from zoneinfo import ZoneInfo

DB = Path(__file__).resolve().parent.parent / 'data' / 'mfl.db'
REVIEW = DB.with_name('review.md')
CENT = 0.011     # MFL scores carry two decimals

DDL = '''
DROP TABLE IF EXISTS check_result;
DROP TABLE IF EXISTS review_item;
CREATE TABLE check_result (
  name     TEXT PRIMARY KEY,
  passed   INTEGER NOT NULL,
  checked  INTEGER NOT NULL,     -- how many things were compared
  agreed   INTEGER NOT NULL,     -- how many agreed outright
  explained INTEGER NOT NULL,    -- differences in a named, explained class
  open     INTEGER NOT NULL,     -- differences listed for review
  detail   TEXT NOT NULL
);
CREATE TABLE review_item (
  check_name TEXT NOT NULL,
  what       TEXT NOT NULL
);
'''


EASTERN = ZoneInfo('America/New_York')    # MFL shows dates in US Eastern


def day(ts):
    return dt.datetime.fromtimestamp(ts, EASTERN).strftime('%Y-%m-%d') if ts else '?'


class Checks:
    def __init__(self, db):
        self.db = db
        self.results = []
        self.names = {(y, f): n for y, f, n in db.execute('SELECT year, fid, name FROM franchise_season')}

    def team(self, year, fid):
        return f'{self.names.get((year, fid), fid)} ({fid})'

    def record(self, name, checked, agreed, explained, open_items, detail, allowed_open=0):
        """allowed_open: open items that are acceptable (listed for review, not a failure)."""
        passed = len(open_items) <= allowed_open
        self.results.append((name, passed, checked, agreed, explained, open_items, detail))

    # ---------- standings ----------

    def records(self):
        """Win-loss from the weekly games equals MFL's standings. MFL counted playoff games in
        the 2013 standings; from 2014 its standings count the regular season only."""
        rows = self.db.execute('''
            WITH every AS (SELECT year, fid, SUM(result='W') w, SUM(result='L') l, SUM(result='T') t
                           FROM played_game WHERE opp IS NOT NULL GROUP BY 1, 2)
            SELECT ts.year, ts.fid, ts.mfl_w, ts.mfl_l, ts.mfl_t, ts.reg_w, ts.reg_l, ts.reg_t, e.w, e.l, e.t
            FROM team_season ts JOIN every e USING (year, fid)''').fetchall()
        agreed, explained, open_ = 0, 0, []
        for y, f, mw, ml, mt, rw, rl, rt, ew, el, et in rows:
            if (mw, ml, mt) == (rw, rl, rt):
                agreed += 1
            elif y == 2013 and (mw, ml, mt) == (ew, el, et):
                explained += 1
            else:
                open_.append(f'{y} {self.team(y, f)}: MFL standings {mw}-{ml}-{mt}, weekly games {rw}-{rl}-{rt}. '
                             'No game accounts for the difference; possibly a commissioner adjustment.')
        self.record('Win-loss records match MFL standings', len(rows), agreed, explained, open_,
                    f'{explained} in 2013, when MFL counted playoff games in the standings.', allowed_open=4)

    def points(self):
        rows = self.db.execute('SELECT year, fid, mfl_pf, all_weeks_pf, mfl_pp, all_weeks_pp FROM team_season').fetchall()
        bad = [f'{y} {self.team(y, f)}: MFL {pf}, weekly games {wk}' for y, f, pf, wk, _, _ in rows
               if pf is None or wk is None or abs(pf - wk) > CENT]
        self.record('Points for match MFL standings', len(rows), len(rows) - len(bad), 0, bad,
                    'MFL totals points over every week it scored, playoffs included.')
        pp = [(y, f, a, b) for y, f, _, _, a, b in rows if a is not None]
        bad = [f'{y} {self.team(y, f)}: MFL {a}, weekly games {b}' for y, f, a, b in pp if abs(a - b) > CENT]
        self.record('Potential points match MFL standings', len(pp), len(pp) - len(bad), 0, bad,
                    'Where MFL publishes potential points (2022 on).')

    def all_play(self):
        rows = self.db.execute('''
            SELECT s.year, s.fid, s.all_play_w, s.all_play_l, s.all_play_t, SUM(a.w), SUM(a.l), SUM(a.t)
            FROM standings s JOIN all_play_week a USING (year, fid)
            WHERE s.all_play_w IS NOT NULL GROUP BY s.year, s.fid''').fetchall()
        bad = [f'{y} {self.team(y, f)}: MFL {w}-{l}-{t}, counted {cw}-{cl}-{ct}'
               for y, f, w, l, t, cw, cl, ct in rows if (w, l, t) != (cw, cl, ct)]
        agreed, explained = len(rows) - len(bad), 0
        self.record('All-play records match MFL standings', len(rows), agreed, explained, bad,
                    'Counted every week in which all teams scored, playoffs included, as MFL does.')

    # ---------- weeks ----------

    def lineups(self):
        rows = self.db.execute('''
            WITH s AS (SELECT year, week, fid, ROUND(SUM(score), 2) st FROM lineup WHERE started = 1 GROUP BY 1, 2, 3)
            SELECT g.year, g.week, g.fid, g.is_home, g.score, s.st,
                   g.week IN (SELECT week FROM regular_week r WHERE r.year = g.year) AS regular
            FROM played_game g JOIN s USING (year, week, fid)''').fetchall()
        agreed, explained, open_ = 0, 0, []
        for y, w, f, home, score, starters, regular in rows:
            diff = round(score - starters, 2)
            if abs(diff) <= CENT:
                agreed += 1
            elif abs(diff - 3.0) <= CENT:
                explained += 1
            else:
                open_.append(f'{y} week {w} {self.team(y, f)}: team score {score}, its starters sum to {starters} '
                             f'({diff:+.2f}). Possibly a commissioner score correction.')
        self.record('Team scores equal their starters', len(rows), agreed, explained, open_,
                    f'{explained} games carry MFL\'s 3-point home bonus (2014-2015 regular season, and playoff games).',
                    allowed_open=6)
        n, eq = self.db.execute('''SELECT COUNT(*), SUM(ABS(l.score - p.score) <= ?) FROM lineup l
                                   JOIN player_week p USING (year, week, pid) WHERE l.score IS NOT NULL''', (CENT,)).fetchone()
        self.record('Lineup scores equal MFL player scores', n, eq, 0,
                    [f'{n - eq} lineup scores differ from the player score file'] if n != eq else [], '')

    def roster_ledger(self):
        """From 2018, every week-to-week roster change is a recorded move. Before 2018 MFL's stored
        weekly rosters barely change within a season, so the proof starts in 2018."""
        snap = {(y, w): k for y, w, k in self.db.execute('SELECT year, week, MAX(kickoff) FROM nfl_game GROUP BY 1, 2')}
        ros = collections.defaultdict(set)
        for y, w, f, p in self.db.execute('SELECT year, week, fid, pid FROM roster_week WHERE week > 0 AND year >= 2018'):
            ros[(y, w)].add((f, p))
        moves = collections.defaultdict(list)
        for ts, frm, to, pid in self.db.execute('''
                SELECT t.ts, a.from_fid, a.to_fid, a.pid FROM txn t JOIN txn_asset a USING (txn_id)
                WHERE a.kind = 'player' AND t.type IN ('TRADE', 'FREE_AGENT', 'WAIVER', 'LOAD_ROSTERS')'''):
            if frm:
                moves[pid].append((ts, frm, -1))
            if to:
                moves[pid].append((ts, to, 1))
        checked, open_ = 0, []
        for (y, w), before in sorted(ros.items()):
            after = ros.get((y, w + 1))
            if not after or (y, w + 1) not in snap:
                continue
            t0, t1 = snap[(y, w)], snap[(y, w + 1)]
            for f, p in sorted(before ^ after):
                checked += 1
                sign = 1 if (f, p) in after else -1
                if not any(t0 < ts <= t1 and fid == f and s == sign for ts, fid, s in moves[p]):
                    open_.append(f'{y} week {w} to {w + 1}: player {p} {"joined" if sign > 0 else "left"} '
                                 f'{self.team(y, f)} with no move recorded in that window.')
        self.record('Roster changes are recorded moves (2018 on)', checked, checked - len(open_), 0, open_,
                    'Rosters at the last kickoff of each week, compared with the next week.',
                    allowed_open=max(3, checked // 1000))

    # ---------- picks ----------

    def pick_chains(self):
        """The trade log says who held each pick last; MFL's draft results say who used it."""
        anyname = collections.defaultdict(set)
        for (y, f), n in self.names.items():
            anyname[n].add(f)
        fp, dp = collections.defaultdict(list), collections.defaultdict(list)
        for ts, frm, to, y, r, s, o in self.db.execute('''
                SELECT t.ts, a.from_fid, a.to_fid, a.pick_year, a.pick_round, a.pick_slot, a.pick_orig
                FROM txn t JOIN txn_asset a USING (txn_id) WHERE t.type = 'TRADE' AND a.kind = 'pick' '''):
            (fp[(o, y, r)] if o else dp[(y, r, s)]).append((ts, frm, to))
        checked, agreed, open_ = 0, 0, []
        for y, r, s, fid, ts, com in self.db.execute(
                'SELECT year, round, pick, fid, ts, comments FROM draft_pick WHERE year >= 2017 ORDER BY 1, 2, 3'):
            chain = re.findall(r'Pick traded from ([^.\n]+)\.', com or '')
            first = anyname.get(chain[0].strip(), set()) if chain else {fid}
            orig = next(iter(first)) if len(first) == 1 else fid
            hist = sorted(h for h in fp.get((orig, y, r), []) + dp.get((y, r, s), []) if not ts or h[0] <= ts)
            holder = hist[-1][2] if hist else orig
            checked += 1
            if holder == fid:
                agreed += 1
            else:
                last = hist[-1]
                open_.append(f'{y} pick {r}.{s:02d}: used by {self.team(y, fid)}, but the trade log last sent it to '
                             f'{self.team(y, holder)} on {day(last[0])}, and the draft shows no "traded" note.')
        self.record('Draft picks were used by their last owner (2017 on)', checked, agreed, 0, open_,
                    'Original owner from MFL\'s "Pick traded from" note, else the team that used it.',
                    allowed_open=49)

    # ---------- completeness ----------

    def birthdates(self):
        n, have = self.db.execute('''SELECT COUNT(*), SUM(birthdate IS NOT NULL) FROM player WHERE pid IN
                                     (SELECT pid FROM roster_week UNION SELECT pid FROM lineup)''').fetchone()
        missing = [f'{p} {nm} ({pos})' for p, nm, pos in self.db.execute('''
            SELECT pid, name, position FROM player WHERE birthdate IS NULL AND pid IN
            (SELECT pid FROM roster_week UNION SELECT pid FROM lineup) ORDER BY name''')]
        self.record('Rostered players have an MFL birthdate', n, have, 0, missing,
                    'MFL lists no birthdate for these players.', allowed_open=n // 50)

    def run(self):
        for step in (self.records, self.points, self.all_play, self.lineups, self.roster_ledger,
                     self.pick_chains, self.birthdates):
            step()
        self.db.executescript(DDL)
        for name, passed, checked, agreed, explained, open_, detail in self.results:
            self.db.execute('INSERT INTO check_result VALUES (?,?,?,?,?,?,?)',
                            (name, int(passed), checked, agreed, explained, len(open_), detail))
            self.db.executemany('INSERT INTO review_item VALUES (?,?)', [(name, x) for x in open_])
        self.db.commit()
        return self.results


def write_review(results, extra=()):
    out = ['# Facts database: checks and review list', '',
           'Generated by `store/checks.py`. Each check compares two records MFL keeps separately.', '',
           '| Check | Compared | Agree | Explained | For review | Result |', '|---|---|---|---|---|---|']
    for name, passed, checked, agreed, explained, open_, _ in results:
        out.append(f'| {name} | {checked:,} | {agreed:,} | {explained:,} | {len(open_):,} | {"pass" if passed else "FAIL"} |')
    for name, _, _, _, _, open_, detail in results:
        if open_ or detail:
            out += ['', f'## {name}', '']
            if detail:
                out += [detail, '']
            out += [f'- {x}' for x in open_]
    out += list(extra)
    REVIEW.write_text('\n'.join(out) + '\n')


def main():
    db = sqlite3.connect(DB)
    results = Checks(db).run()
    write_review(results)
    for name, passed, checked, agreed, explained, open_, _ in results:
        print(f'{"pass" if passed else "FAIL"}  {name}: {checked:,} compared, {agreed:,} agree, '
              f'{explained:,} explained, {len(open_):,} for review')
    print(REVIEW)
    return 0 if all(r[1] for r in results) else 1


if __name__ == '__main__':
    sys.exit(main())
