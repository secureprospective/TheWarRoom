#!/usr/bin/env python3
"""Load the MFL archive into the facts database, data/mfl.db.

Reads league-archive/raw/ and nothing else. Values go in as MFL gives them: the only reading
done is decoding MFL's own codes (pick codes, "added|dropped" lists, Unix birthdates). See
schema.sql for the rules every table follows. Rebuilds the database from scratch each run.

    python3 store/load.py            # about a minute
"""
import datetime as dt
import hashlib
import json
import re
import sqlite3
import sys
from pathlib import Path
from zoneinfo import ZoneInfo

TOOL = Path(__file__).resolve().parent.parent
RAW = TOOL.parent.parent / 'league-archive' / 'raw'
DB = TOOL / 'data' / 'mfl.db'
SCHEMA = Path(__file__).with_name('schema.sql')
VIEWS = Path(__file__).with_name('views.sql')
WEEKS = range(1, 18)


def rows(value):
    """MFL sends a lone record as an object and several as a list."""
    if value in (None, ''):
        return []
    return [value] if isinstance(value, dict) else list(value)


def ids(text):
    return [t.strip() for t in (text or '').split(',') if t.strip()]


def num(value):
    try:
        return float(str(value).replace('$', '').replace(',', ''))
    except (TypeError, ValueError):
        return None


def whole(value):
    n = num(value)
    return int(n) if n is not None else None


def text(value):
    """MFL wraps some strings as {"$t": "..."}."""
    if isinstance(value, dict):
        return value.get('$t')
    return value if value not in ('',) else None


def flag(value):
    return 1 if str(value or '') == '1' else 0


EASTERN = ZoneInfo('America/New_York')    # MFL shows every date and time in US Eastern


def iso_date(ts):
    """A Unix time as the calendar date MFL shows (US Eastern). Birthdates are midnight Eastern."""
    if not ts or not str(ts).lstrip('-').isdigit() or int(ts) == 0:
        return None
    return dt.datetime.fromtimestamp(int(ts), EASTERN).date().isoformat()


class Loader:
    def __init__(self, db):
        self.db = db
        self.years = sorted(int(p.name) for p in RAW.iterdir() if p.is_dir() and p.name.isdigit())

    def read(self, rel):
        """A raw file as JSON, recorded in source_file. Missing or broken files read as {}."""
        path = RAW / rel
        if not path.is_file():
            return {}, None
        body = path.read_bytes()
        try:
            data = json.loads(body)
        except ValueError:
            return {}, None
        self.db.execute('INSERT OR IGNORE INTO source_file VALUES (?,?,?)',
                        (rel, len(body), hashlib.sha256(body).hexdigest()))
        if 'error' in data:
            return {}, None
        return data, rel

    def note(self, step, src, note, n=1):
        self.db.execute('INSERT INTO load_note VALUES (?,?,?,?)', (step, src, note, n))

    def insert(self, table, rows_):
        rows_ = list(rows_)
        if not rows_:
            return
        cols = list(rows_[0])
        self.db.executemany(f'INSERT INTO {table} ({",".join(cols)}) VALUES ({",".join("?" * len(cols))})',
                            [tuple(r[c] for c in cols) for r in rows_])

    # ---------- the league ----------

    def league(self):
        for y in self.years:
            d, src = self.read(f'{y}/league.json')
            lg = d.get('league')
            if not lg:
                continue
            self.insert('season', [{
                'year': y, 'league_id': lg.get('id'), 'name': lg.get('name'),
                'start_week': whole(lg.get('startWeek')), 'end_week': whole(lg.get('endWeek')),
                'last_regular_week': whole(lg.get('lastRegularSeasonWeek')),
                'salary_cap': num(lg.get('salaryCapAmount')), 'roster_size': whole(lg.get('rosterSize')),
                'taxi_squad': json.dumps(lg.get('taxiSquad')), 'injured_reserve': json.dumps(lg.get('injuredReserve')),
                'starters': json.dumps(lg.get('starters')), 'raw': json.dumps(lg), 'src': src}])
            divs = {x['id']: x for x in rows((lg.get('divisions') or {}).get('division'))}
            confs = {x['id']: x for x in rows((lg.get('conferences') or {}).get('conference'))}
            out = []
            for f in rows((lg.get('franchises') or {}).get('franchise')):
                div = divs.get(f.get('division'), {})
                conf = confs.get(div.get('conference'), {})
                out.append({'year': y, 'fid': f['id'], 'name': f.get('name'), 'abbrev': f.get('abbrev'),
                            'division': f.get('division'), 'division_name': div.get('name'),
                            'conference': div.get('conference'), 'conference_name': conf.get('name'), 'src': src})
            self.insert('franchise_season', out)

    def standings(self):
        def wlt(s):
            parts = (s or '').split('-')
            return [whole(p) for p in parts] if len(parts) == 3 else [None] * 3
        for y in self.years:
            d, src = self.read(f'{y}/leagueStandings.json')
            out = []
            for f in rows((d.get('leagueStandings') or {}).get('franchise')):
                ap = wlt(f.get('all_play_wlt'))
                out.append({'year': y, 'fid': f['id'], 'h2h_w': whole(f.get('h2hw')), 'h2h_l': whole(f.get('h2hl')),
                            'h2h_t': whole(f.get('h2ht')), 'pf': num(f.get('pf')), 'pa': num(f.get('pa')),
                            'pp': num(f.get('pp')), 'all_play_w': ap[0], 'all_play_l': ap[1], 'all_play_t': ap[2],
                            'eff': num(f.get('eff')), 'raw': json.dumps(f), 'src': src})
            self.insert('standings', out)

    def rules(self):
        for y in self.years:
            d, src = self.read(f'{y}/rules.json')
            out = []
            for group in rows((d.get('rules') or {}).get('positionRules')):
                for r in rows(group.get('rule')):
                    out.append({'year': y, 'positions': group.get('positions'), 'event': text(r.get('event')),
                                'range': text(r.get('range')), 'points': text(r.get('points')), 'src': src})
            self.insert('scoring_rule', out)
        d, src = self.read('allRules.json')
        self.insert('rule_code', [{'code': text(r.get('abbreviation')), 'short': text(r.get('shortDescription')),
                                   'description': text(r.get('detailedDescription')), 'src': src}
                                  for r in rows((d.get('allRules') or {}).get('rule'))])

    def calendar(self):
        for y in self.years:
            d, src = self.read(f'{y}/calendar.json')
            self.insert('calendar_event', [{'year': y, 'event_id': e.get('id'), 'type': e.get('type'),
                                            'title': e.get('title') or None, 'start_ts': whole(e.get('start_time')),
                                            'end_ts': whole(e.get('end_time')), 'src': src}
                                           for e in rows((d.get('calendar') or {}).get('event'))])

    # ---------- players ----------

    FIELDS = ('name', 'position', 'birthdate', 'draft_year', 'draft_team', 'draft_round', 'draft_pick',
              'college', 'height', 'weight')

    def players(self):
        """Identity from the detailed lists, latest season winning; disagreements kept."""
        best, seen, conflicts, seasons = {}, {}, [], []
        for y in self.years:
            d, src = self.read(f'{y}/players_DETAILS1.json')
            if not src:
                d, src = self.read(f'{y}/players.json')
            for p in rows((d.get('players') or {}).get('player')):
                pid = p['id']
                seasons.append({'year': y, 'pid': pid, 'name': p.get('name'), 'position': p.get('position'),
                                'nfl_team': p.get('team'), 'src': src})
                rec = {k: (p.get(k) or None) for k in self.FIELDS}
                if pid in best:
                    old = best[pid]
                    for k in ('name', 'birthdate', 'draft_year', 'draft_team', 'draft_round', 'draft_pick', 'college'):
                        if old[k] and rec[k] and old[k] != rec[k]:
                            conflicts.append({'pid': pid, 'field': k, 'year': seen[pid][1], 'value': old[k], 'kept': rec[k]})
                    for k in self.FIELDS:   # a later list missing a field does not erase it
                        if not rec[k]:
                            rec[k] = old[k]
                best[pid] = rec
                first = seen.get(pid, (y,))[0]
                seen[pid] = (first, y, src)
        out = []
        for pid, r in best.items():
            first, last, src = seen[pid]
            out.append({'pid': pid, 'name': r['name'], 'position': r['position'], 'birthdate': iso_date(r['birthdate']),
                        'birthdate_ts': whole(r['birthdate']), 'draft_year': whole(r['draft_year']) or None,
                        'draft_team': r['draft_team'], 'draft_round': whole(r['draft_round']),
                        'draft_pick': whole(r['draft_pick']), 'college': r['college'], 'height': r['height'],
                        'weight': r['weight'], 'first_year': first, 'last_year': last, 'src': src})
        self.insert('player', out)
        self.insert('player_conflict', conflicts)
        self.insert('player_season', seasons)

    # ---------- weeks ----------

    def results(self):
        for y in self.years:
            d, src = self.read(f'{y}/weeklyResults_WYTD.json')
            games, lineups = [], []
            dupes = byes = 0
            for wk in rows((d.get('allWeeklyResults') or {}).get('weeklyResults')):
                week = whole(wk.get('week'))
                sides = [rows(m.get('franchise')) for m in rows(wk.get('matchup'))]
                in_games = {f['id'] for teams in sides for f in teams}
                for f in rows(wk.get('franchise')):     # teams scored that week without a head-to-head game
                    if f['id'] in in_games:
                        # 2018: MFL repeats one team's game as a standalone copy each week.
                        if num(f.get('score')) != next(num(g.get('score')) for t in sides for g in t if g['id'] == f['id']):
                            raise ValueError(f'{src} week {week}: standalone copy of {f["id"]} differs from its game')
                        dupes += 1
                        continue
                    sides.append([f])
                for teams in sides:
                    for i, f in enumerate(teams):
                        other = teams[1 - i] if len(teams) == 2 else None
                        if f['id'] == 'BYE':     # MFL's stand-in opponent for a team with a bye
                            byes += 1
                            continue
                        if other and other['id'] == 'BYE':
                            other = None
                        games.append({'year': y, 'week': week, 'fid': f['id'], 'opp': other['id'] if other else None,
                                      'is_home': whole(f.get('isHome')), 'score': num(f.get('score')),
                                      'result': f.get('result') or None, 'opt_pts': num(f.get('opt_pts')), 'src': src})
                        started = set(ids(f.get('starters')))
                        listed = {p['id']: p for p in rows(f.get('player'))}
                        for pid in started | set(ids(f.get('nonstarters'))) | set(listed):
                            p = listed.get(pid, {})
                            status = p.get('status')
                            lineups.append({'year': y, 'week': week, 'fid': f['id'], 'pid': pid,
                                            'started': 1 if (status == 'starter' or (not status and pid in started)) else 0,
                                            'should_start': whole(p.get('shouldStart')), 'score': num(p.get('score')),
                                            'src': src})
            self.insert('game', games)
            self.insert('lineup', lineups)
            if dupes:
                self.note('results', src, 'standalone copy of a team already in a game that week: identical score, skipped', dupes)
            if byes:
                self.note('results', src, "MFL's BYE stand-in team: not a franchise, skipped", byes)

    def player_weeks(self):
        for y in self.years:
            for w in WEEKS:
                d, _ = self.read(f'{y}/weekly/playerScores_W{w:02d}.json')
                p, _ = self.read(f'{y}/weekly/projectedScores_W{w:02d}.json')
                scores = {s['id']: num(s.get('score')) for s in rows((d.get('playerScores') or {}).get('playerScore'))}
                proj = {s['id']: num(s.get('score')) for s in rows((p.get('projectedScores') or {}).get('playerScore'))}
                self.insert('player_week', [{'year': y, 'week': w, 'pid': pid, 'score': scores.get(pid),
                                             'projected': proj.get(pid)} for pid in scores.keys() | proj.keys()])
                i, src = self.read(f'{y}/weekly/injuries_W{w:02d}.json')
                inj = {x['id']: x for x in rows((i.get('injuries') or {}).get('injury'))}
                self.insert('injury_week', [{'year': y, 'week': w, 'pid': pid, 'status': x.get('status'),
                                             'details': x.get('details') or None, 'exp_return': x.get('exp_return') or None,
                                             'src': src} for pid, x in inj.items()])
                n, src = self.read(f'{y}/weekly/nflSchedule_W{w:02d}.json')
                games = []
                for m in rows((n.get('nflSchedule') or {}).get('matchup')):
                    teams = rows(m.get('team'))
                    for k, t in enumerate(teams):
                        games.append({'year': y, 'week': w, 'team': t['id'],
                                      'opp': teams[1 - k]['id'] if len(teams) == 2 else None,
                                      'is_home': whole(t.get('isHome')), 'score': whole(t.get('score')),
                                      'spread': num(t.get('spread')), 'kickoff': whole(m.get('kickoff')), 'src': src})
                self.insert('nfl_game', games)

    def roster_rows(self, y, week, d, src):
        out, doubled = {}, 0
        for f in rows((d.get('rosters') or {}).get('franchise')):
            for p in rows(f.get('player')):
                key = (f['id'], p['id'])
                if key in out:
                    # MFL lists a player on IR (or taxi) twice: once as ROSTER, once with the status.
                    # He is one player on the roster; the specific status is the fact.
                    doubled += 1
                    if p.get('status') == 'ROSTER':
                        continue
                out[key] = {'year': y, 'week': week, 'fid': f['id'], 'pid': p['id'], 'status': p.get('status'),
                            'salary': num(p.get('salary')), 'contract_year': (p.get('contractYear') or '').strip() or None,
                            'contract_status': (p.get('contractStatus') or '').strip() or None,
                            'contract_info': (p.get('contractInfo') or '').strip() or None, 'src': src}
        if doubled:
            self.note('rosters', src, 'player listed twice on one roster (ROSTER and IR or taxi): kept once, with the specific status', doubled)
        return list(out.values())

    def rosters(self):
        for y in self.years:
            for w in WEEKS:
                d, src = self.read(f'{y}/weekly/rosters_W{w:02d}.json')
                self.insert('roster_week', self.roster_rows(y, w, d, src))
            d, src = self.read(f'{y}/rosters.json')
            self.insert('roster_week', self.roster_rows(y, 0, d, src))

    def contracts(self):
        for y in self.years:
            d, src = self.read(f'{y}/salaries.json')
            self.insert('contract', [{'year': y, 'pid': p['id'], 'salary': num(p.get('salary')),
                                      'contract_year': p.get('contractYear') or None,
                                      'contract_status': p.get('contractStatus') or None,
                                      'contract_info': (p.get('contractInfo') or '').strip() or None, 'src': src}
                                     for p in rows(((d.get('salaries') or {}).get('leagueUnit') or {}).get('player'))])
            d, src = self.read(f'{y}/salaryAdjustments.json')
            self.insert('cap_charge', [{'year': y, 'adj_id': a.get('id'), 'fid': a.get('franchise_id'),
                                        'amount': num(a.get('amount')), 'description': a.get('description'),
                                        'ts': whole(a.get('timestamp')), 'src': src}
                                       for a in rows((d.get('salaryAdjustments') or {}).get('salaryAdjustment'))])

    def points_allowed(self):
        for y in self.years:
            d, src = self.read(f'{y}/pointsAllowed.json')
            self.insert('points_allowed', [{'year': y, 'nfl_team': t['id'], 'position': p.get('name'),
                                            'points': num(p.get('points')), 'src': src}
                                           for t in rows((d.get('pointsAllowed') or {}).get('team'))
                                           for p in rows(t.get('position'))])

    # ---------- picks ----------

    def draft(self):
        for y in self.years:
            d, src = self.read(f'{y}/draftResults.json')
            out = []
            for unit in rows((d.get('draftResults') or {}).get('draftUnit')):
                for p in rows(unit.get('draftPick')):
                    pid = p.get('player')
                    out.append({'year': y, 'round': whole(p.get('round')), 'pick': whole(p.get('pick')),
                                'fid': p.get('franchise'), 'pid': None if pid in (None, '', '----') else pid,
                                'ts': whole(p.get('timestamp')), 'day': iso_date(p.get('timestamp')),
                                'comments': (p.get('comments') or '').strip() or None,
                                'src': src})
            self.insert('draft_pick', out)

    def picks_owned(self):
        for y in self.years:
            d, src = self.read(f'{y}/futureDraftPicks.json')
            out = []
            for f in rows((d.get('futureDraftPicks') or {}).get('franchise')):
                for p in rows(f.get('futureDraftPick')):
                    out.append({'snapshot_year': y, 'source': 'futureDraftPicks', 'fid': f['id'],
                                'pick_year': whole(p.get('year')), 'round': whole(p.get('round')), 'slot': None,
                                'orig_fid': p.get('originalPickFor'), 'code': None, 'description': None, 'src': src})
            d, src = self.read(f'{y}/assets.json')
            for f in rows((d.get('assets') or {}).get('franchise')):
                for key in ('currentYearDraftPicks', 'futureYearDraftPicks'):
                    for p in rows((f.get(key) or {}).get('draftPick')):
                        a = decode_asset(p.get('pick', ''), y)
                        out.append({'snapshot_year': y, 'source': 'assets', 'fid': f['id'], 'pick_year': a['pick_year'],
                                    'round': a['pick_round'], 'slot': a['pick_slot'], 'orig_fid': a['pick_orig'],
                                    'code': p.get('pick'), 'description': p.get('description'), 'src': src})
            self.insert('pick_owned', out)

    def trade_bait(self):
        for y in self.years:
            d, src = self.read(f'{y}/tradeBait.json')
            self.insert('trade_bait', [{'year': y, 'fid': b.get('franchise_id'), 'ts': whole(b.get('timestamp')),
                                        'will_give_up': b.get('willGiveUp') or None,
                                        'in_exchange_for': (b.get('inExchangeFor') or '').strip() or None, 'src': src}
                                       for b in rows((d.get('tradeBaits') or {}).get('tradeBait'))])

    # ---------- transactions ----------

    OFFERS = ('TRADE', 'TRADE_PROPOSAL', 'TRADE_ACCEPT', 'TRADE_REJECTION', 'TRADE_REVOKE', 'TRADE_OFFER_EXPIRED')

    def transactions(self):
        txn_id = 0
        for y in self.years:
            d, src = self.read(f'{y}/transactions.json')
            txns, assets = [], []
            for i, x in enumerate(rows((d.get('transactions') or {}).get('transaction'))):
                txn_id += 1
                kind, fid = x.get('type'), x.get('franchise') or None
                txns.append({'txn_id': txn_id, 'year': y, 'ts': whole(x.get('timestamp')),
                             'day': iso_date(x.get('timestamp')), 'type': kind, 'fid': fid,
                             'fid2': x.get('franchise2') or None, 'by_commish': flag(x.get('by_commish')),
                             'expires': whole(x.get('expires')), 'comments': (x.get('comments') or '').strip() or None,
                             'raw': json.dumps(x), 'src': src, 'src_idx': i})

                def add(role, code, frm, to):
                    assets.append({'txn_id': txn_id, 'role': role, 'from_fid': frm, 'to_fid': to, **decode_asset(code, y)})

                if kind in self.OFFERS:
                    for code in ids(x.get('franchise1_gave_up')):
                        add('sent', code, fid, x.get('franchise2'))
                    for code in ids(x.get('franchise2_gave_up')):
                        add('sent', code, x.get('franchise2'), fid)
                elif kind in ('FREE_AGENT', 'LOAD_ROSTERS'):
                    added, _, dropped = (x.get('transaction') or '').partition('|')
                    for code in ids(added):
                        add('load' if kind == 'LOAD_ROSTERS' else 'add', code, None, fid)
                    for code in ids(dropped):
                        add('drop', code, fid, None)
                elif kind == 'WAIVER':
                    for code in ids(x.get('added')):
                        add('add', code, None, fid)
                    for code in ids(x.get('dropped')):
                        add('drop', code, fid, None)
                elif kind == 'IR':
                    for code in ids(x.get('deactivated')):
                        add('ir_on', code, fid, fid)
                    for code in ids(x.get('activated')):
                        add('ir_off', code, fid, fid)
                elif kind == 'TAXI':
                    for code in ids(x.get('demoted')):
                        add('taxi_on', code, fid, fid)
                    for code in ids(x.get('promoted')):
                        add('taxi_off', code, fid, fid)
            self.insert('txn', txns)
            self.insert('txn_asset', assets)

    # ---------- playoffs and all of MFL ----------

    def playoffs(self):
        for y in self.years:
            d, src = self.read(f'{y}/playoffBrackets.json')
            brackets = rows((d.get('playoffBrackets') or {}).get('playoffBracket'))
            self.insert('playoff_bracket', [{'year': y, 'bracket_id': b['id'], 'name': b.get('name'),
                                             'winner_title': b.get('bracketWinnerTitle'),
                                             'start_week': whole(b.get('startWeek')), 'teams': whole(b.get('teamsInvolved')),
                                             'src': src} for b in brackets])
            for b in brackets:
                d, src = self.read(f'{y}/playoffBracket_{b["id"]}.json')
                out = []
                for rnd in rows((d.get('playoffBracket') or {}).get('playoffRound')):
                    for g in rows(rnd.get('playoffGame')):
                        for side in ('home', 'away'):
                            t = g.get(side) or {}
                            if t.get('franchise_id'):
                                out.append({'year': y, 'bracket_id': b['id'], 'week': whole(rnd.get('week')),
                                            'game_id': g.get('game_id'), 'fid': t['franchise_id'], 'side': side,
                                            'points': num(t.get('points')), 'seed': t.get('seed'), 'src': src})
                self.insert('playoff_game', out)

    def mfl_wide(self):
        for y in self.years:
            d, src = self.read(f'{y}/adp.json')
            self.insert('mfl_adp', [{'year': y, 'pid': p['id'], 'rank': whole(p.get('rank')),
                                     'avg_pick': num(p.get('averagePick')), 'min_pick': whole(p.get('minPick')),
                                     'max_pick': whole(p.get('maxPick')), 'drafts': whole(p.get('draftsSelectedIn')),
                                     'src': src} for p in rows((d.get('adp') or {}).get('player'))])
            d, src = self.read(f'{y}/aav.json')
            self.insert('mfl_aav', [{'year': y, 'pid': p['id'], 'rank': whole(p.get('rank')),
                                     'avg_value': num(p.get('averageValue')), 'min_value': num(p.get('minValue')),
                                     'max_value': num(p.get('maxValue')), 'src': src}
                                    for p in rows((d.get('aav') or {}).get('player'))])


def decode_asset(code, year):
    """MFL's asset tokens. FP_<orig>_<year>_<round>: a future pick. DP_<r>_<s>: a pick in this
    season's draft, counted from zero (DP_3_20 is round 4, pick 21). Digits: a player."""
    code = code.strip()
    out = {'kind': 'other', 'pid': None, 'pick_year': None, 'pick_round': None, 'pick_slot': None, 'pick_orig': None,
           'code': code}
    fut = re.fullmatch(r'FP_(\d{4})_(\d{4})_(\d+)', code)
    cur = re.fullmatch(r'DP_(\d+)_(\d+)', code)
    if fut:
        out.update(kind='pick', pick_orig=fut.group(1), pick_year=int(fut.group(2)), pick_round=int(fut.group(3)))
    elif cur:
        out.update(kind='pick', pick_year=year, pick_round=int(cur.group(1)) + 1, pick_slot=int(cur.group(2)) + 1)
    elif code.isdigit():
        out.update(kind='player', pid=code)
    return out


STEPS = ['league', 'standings', 'rules', 'calendar', 'players', 'results', 'player_weeks', 'rosters', 'contracts',
         'points_allowed', 'draft', 'picks_owned', 'trade_bait', 'transactions', 'playoffs', 'mfl_wide']


def retire(path):
    """Make way for a new build. The old database must be idle: its write-ahead log is flushed and
    removed, because a new file beside an old log would be read as one damaged database."""
    sidecars = [Path(f'{path}-wal'), Path(f'{path}-shm')]
    if path.exists():
        old = sqlite3.connect(path, timeout=5)
        try:
            old.execute('BEGIN EXCLUSIVE')
            old.rollback()
            busy = old.execute('PRAGMA wal_checkpoint(TRUNCATE)').fetchone()[0]
        except sqlite3.OperationalError as e:
            raise SystemExit(f'{path} is in use ({e}); stop whatever is using it and build again')
        finally:
            old.close()
        if busy:
            raise SystemExit(f'{path} is in use; stop whatever is using it and build again')
    for f in sidecars:
        f.unlink(missing_ok=True)


def build(path=DB):
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix('.tmp')
    tmp.unlink(missing_ok=True)
    db = sqlite3.connect(tmp)
    db.executescript(SCHEMA.read_text())
    loader = Loader(db)
    for step in STEPS:
        getattr(loader, step)()
    db.executescript(VIEWS.read_text())
    db.commit()
    db.close()
    retire(path)
    tmp.replace(path)    # only a complete build replaces the old one
    final = sqlite3.connect(path)
    final.execute('PRAGMA journal_mode = WAL')     # readers keep working while the index is written
    final.close()
    return path


def main():
    path = build()
    db = sqlite3.connect(path)
    tables = [r[0] for r in db.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")]
    views = [r[0] for r in db.execute("SELECT name FROM sqlite_master WHERE type='view' ORDER BY name")]
    for t in tables:
        print(f'{t:18s} {db.execute(f"SELECT COUNT(*) FROM {t}").fetchone()[0]:>9,}')
    print('views:', ', '.join(views))
    print(f'{path} {path.stat().st_size:,} bytes')


if __name__ == '__main__':
    sys.exit(main())
