"""One currency for the Lab: wins above replacement, in this league's own scoring and lineups.

Every asset (player, rookie pick) is valued as the wins it is expected to add over the next
five seasons, measured against what a team could get for free. Everything here is computed
"as known at the time" unless a name says hindsight.

Method (see docs/league-history/League_Lab_Method_Research_2026-10-05.md):
- Replacement level: for each position and season, the average points per game of the players
  ranked just below the number the league actually starts at that position.
- Points to wins: a week's points above replacement become the change in the chance of
  beating an average opponent, using that season's spread of weekly team scores.
- Forecast: what comparable players did next (same position group, age, production, the season
  before), with players who dropped out counted as zero so survivors don't flatter the curve.
  A player's own future never trains the model that values him (grouped out-of-fold fits).
- Picks: what players taken around that overall slot went on to produce, season by season.
"""
import collections
import math
import statistics
import zlib
from statistics import NormalDist

NORMAL = NormalDist()
HORIZON = 5                      # seasons valued: the current/upcoming one and four more
GROUP = {'QB': 'QB', 'RB': 'RB', 'WR': 'WR', 'TE': 'TE', 'PK': 'PK',
         'DT': 'DL', 'DE': 'DL', 'LB': 'LB', 'CB': 'DB', 'S': 'DB'}
GROUPS = ['QB', 'RB', 'WR', 'TE', 'PK', 'DL', 'LB', 'DB']
FOLDS = 5
PRIOR_WEEKS = 6                  # in-season blend: last forecast counts as this many weeks
FIRST_VALUED = 2014              # 2013 is a partial season (the league joined MFL mid-year)


def solve(X, Y, ridge=0.0, free=1, W=None):
    """Least squares with an optional ridge on every column after the first `free` ones.
    Small dense normal equations; the Lab's compiler is standard-library only."""
    k = len(X[0])
    A = [[0.0] * k for _ in range(k)]
    b = [0.0] * k
    for n, (x, y) in enumerate(zip(X, Y)):
        w = W[n] if W else 1.0
        for i in range(k):
            xi = x[i] * w
            if xi == 0.0:
                continue
            b[i] += xi * y
            row = A[i]
            for j in range(k):
                row[j] += xi * x[j]
    for i in range(free, k):
        A[i][i] += ridge
    return gauss(A, b), A


def gauss(A, b):
    k = len(b)
    M = [list(r) + [v] for r, v in zip(A, b)]
    for c in range(k):
        p = max(range(c, k), key=lambda r: abs(M[r][c]))
        M[c], M[p] = M[p], M[c]
        if abs(M[c][c]) < 1e-12:
            continue
        for r in range(k):
            if r != c and M[r][c]:
                f = M[r][c] / M[c][c]
                for j in range(c, k + 1):
                    M[r][j] -= f * M[c][j]
    return [M[i][k] / M[i][i] if abs(M[i][i]) > 1e-12 else 0.0 for i in range(k)]


def inverse(A):
    k = len(A)
    cols = []
    for j in range(k):
        e = [0.0] * k
        e[j] = 1.0
        cols.append(gauss(A, e))
    return [[cols[j][i] for j in range(k)] for i in range(k)]


AGE_BANDS = [(0, 24.5, 'young'), (24.5, 27.5, 'prime'), (27.5, 99, 'older')]
AGE_NAMES = {'young': '24 or younger', 'prime': '25 to 27', 'older': '28 or older'}


def age_band(age):
    if age is None:
        return None
    return next(name for lo, hi, name in AGE_BANDS if lo <= age < hi)


def fold_of(pid):
    return zlib.crc32(pid.encode()) % FOLDS


class Wins:
    """Wins above replacement for every player-week the archive scored."""

    def __init__(self, lab):
        self.lab = lab
        self.years = [y for y in lab.years if y >= FIRST_VALUED]
        self.played = {y: sorted(w for w in lab.done_weeks[y] if w <= lab.regular_weeks[y]) for y in lab.years}
        self.spread = {}
        self.replacement = {}
        for y in lab.years:
            scores = [r['pf'] for (yy, w, f), r in lab.results.items() if yy == y and w in self.played[y]]
            if len(scores) > 30:
                self.spread[y] = statistics.pstdev(scores)
            by_pos = collections.defaultdict(list)
            for (pid, yy), weeks in lab.weekly_pts.items():
                if yy != y:
                    continue
                games = [p for w, p in weeks.items() if p and w in self.played[y]]
                pos = lab.pos(pid, y)
                if pos and len(games) >= 4:
                    by_pos[pos].append(sum(games) / len(games))
            for pos, vals in by_pos.items():
                vals.sort(reverse=True)
                line = lab.starter_line[y].get(pos) or 0
                band = vals[line:line + max(3, line // 3)]
                if band:
                    self.replacement[(pos, y)] = statistics.mean(band)
        self.week = {}                                   # (pid, year) -> {week: wins}
        for (pid, y), weeks in lab.weekly_pts.items():
            pos = lab.pos(pid, y)
            if not pos or (pos, y) not in self.replacement or y not in self.spread:
                continue
            self.week[(pid, y)] = {w: self.week_wins(p, pos, y) for w, p in weeks.items() if w in self.played[y]}

    def week_wins(self, pts, pos, year):
        """Change in the chance of beating an average team from starting this player instead of
        a replacement. A week without points is a week the replacement plays: zero."""
        if not pts:
            return 0.0
        return NORMAL.cdf((pts - self.replacement[(pos, year)]) / (self.spread[year] * math.sqrt(2))) - 0.5

    def season(self, pid, year):
        return sum(self.week.get((pid, year), {}).values())

    def rate(self, pid, year, weeks=None):
        """Wins per regular-season week; missed weeks count, so availability is part of value."""
        played = self.played.get(year) or []
        if weeks is not None:
            played = [w for w in played if w in weeks]
        if not played:
            return 0.0
        mine = self.week.get((pid, year), {})
        return sum(mine.get(w, 0.0) for w in played) / len(played)

    def games(self, pid, year):
        return sum(1 for w, p in self.lab.weekly_pts.get((pid, year), {}).items() if p and w in self.played.get(year, ()))

    def season_weeks(self, year):
        return len(self.lab.week_start.get(year, {})) and self.lab.regular_weeks.get(year) or 13

    def currency_check(self):
        """Does the currency describe winning? Correlate each team-season's wins above replacement
        from the players it started with its all-play win rate."""
        lab, ap = self.lab, self.all_play()
        xs, ys = [], []
        for (y, fid), rec in ap.items():
            if y > max(yy for yy in self.years if len(self.played[yy]) == lab.regular_weeks[yy]) or rec['allPlay'] is None:
                continue
            total = 0.0
            for w in self.played[y]:
                for pid in lab.starts.get((y, w, fid), ()):
                    total += self.week.get((pid, y), {}).get(w, 0.0)
            xs.append(total)
            ys.append(rec['allPlay'])
        return {'teamSeasons': len(xs), 'correlation': round(statistics.correlation(xs, ys), 3)}

    def all_play(self):
        """Per team-season: all-play wins and games, actual wins, and schedule luck."""
        lab, out = self.lab, {}
        for y in self.years:
            tally = collections.defaultdict(lambda: [0.0, 0, 0.0, 0])
            for w in self.played[y]:
                week = {f: r for (yy, ww, f), r in lab.results.items() if yy == y and ww == w}
                scores = sorted(r['pf'] for r in week.values())
                for f, r in week.items():
                    below = sum(1 for s in scores if s < r['pf'])
                    tied = sum(1 for s in scores if s == r['pf']) - 1
                    t = tally[f]
                    t[0] += below + 0.5 * tied
                    t[1] += len(scores) - 1
                    t[2] += {'W': 1.0, 'T': 0.5}.get(r['res'], 0.0)
                    t[3] += 1
            for f, (ap, n, won, g) in tally.items():
                expected = ap / n * g if n else 0.0
                out[(y, f)] = {'allPlay': round(ap / n, 3) if n else None, 'wins': won, 'games': g,
                               'expectedWins': round(expected, 2), 'luck': round(won - expected, 2)}
        return out


class Forecast:
    """Expected wins per week in each of the next five seasons, from what comparable players did next."""

    def __init__(self, wins, through=None):
        """through: the last season the model may learn from (a backtest's "known so far")."""
        self.wins = wins
        lab = wins.lab
        self.lab = lab
        self.last = max(y for y in wins.years if wins.played[y] and len(wins.played[y]) == lab.regular_weeks[y])
        if through is not None:
            self.last = min(self.last, through)
        self.point_in_time = through is not None
        self.rows = []
        pids = {pid for (pid, y) in wins.week}
        for pid in pids:
            for y in wins.years:
                if y > self.last:
                    continue
                row = self.features(pid, y)
                if row is None:
                    continue
                row['fut'] = [wins.rate(pid, y + k) if y + k <= self.last else None for k in range(1, HORIZON + 1)]
                self.rows.append(row)
        self.coef = {}
        for g in GROUPS:
            mine = [r for r in self.rows if r['g'] == g]
            for k in range(HORIZON):
                usable = [r for r in mine if r['fut'][k] is not None]
                for fold in ([FOLDS] if self.point_in_time else range(FOLDS + 1)):   # FOLDS = all players
                    train = [r for r in usable if fold == FOLDS or r['fold'] != fold]
                    if len(train) < 30:
                        # Early backtest seasons have not yet seen players this many seasons on:
                        # carry the furthest horizon learned so far.
                        self.coef[(g, k, fold)] = self.coef.get((g, k - 1, fold), [0.0] * 7)
                        continue
                    beta, _ = solve([self.x(r) for r in train], [r['fut'][k] for r in train], ridge=1.0)
                    self.coef[(g, k, fold)] = beta
        self.rookie = self.rookie_classes()
        self.calibration = self.calibrate()

    def predict(self, r, k, fold):
        return max(0.0, sum(b * v for b, v in zip(self.coef[(r['g'], k, fold)], self.x(r))))

    def calibrate(self):
        """A straight-line model misses how sharply older players fall away (retirements count as
        zero). Out-of-fold, compare what each position group and age band actually produced with
        what was forecast, and scale forecasts by that ratio."""
        acc = collections.defaultdict(lambda: [0.0, 0.0])
        for r in self.rows:
            band = age_band(r['age'] + 1)
            for k in range(HORIZON):
                if r['fut'][k] is not None:
                    a = acc[(r['g'], band, k)]
                    a[0] += r['fut'][k]
                    a[1] += self.predict(r, k, FOLDS if self.point_in_time else r['fold'])
        return {key: real / pred for key, (real, pred) in acc.items() if pred > 0}

    def check(self):
        """Out-of-fold accuracy, after calibration: share of variation explained and real/forecast."""
        out = {}
        for g in GROUPS:
            mine = [r for r in self.rows if r['g'] == g]
            for k in range(HORIZON):
                pairs = [(r['fut'][k], self.predict(r, k, r['fold']) * self.calibration.get((g, age_band(r['age'] + 1), k), 1.0))
                         for r in mine if r['fut'][k] is not None]
                if len(pairs) < 30:
                    continue
                mean = statistics.mean(t for t, _ in pairs)
                sse = sum((t - p) ** 2 for t, p in pairs)
                sst = sum((t - mean) ** 2 for t, _ in pairs)
                out[f'{g}+{k + 1}'] = {'n': len(pairs), 'explained': round(1 - sse / sst, 3)}
        return out

    def features(self, pid, year):
        """Inputs known at the end of `year`: this season's and last season's wins per week, age, games."""
        lab, wins = self.lab, self.wins
        pos = lab.pos(pid, year)
        if not pos:
            return None
        g0, g1 = wins.games(pid, year), wins.games(pid, year - 1)
        if g0 < 1 and g1 < 1:
            return None
        birth = lab.birth.get(pid)
        if not birth:
            return None
        from build_lab import epoch, dt
        age = (epoch(dt.date(year, 9, 1)) - birth) / (365.25 * 86400)
        weeks = max(1, len(wins.played[year]))
        return {'pid': pid, 'y': year, 'g': GROUP[pos], 'age': age, 'r0': wins.rate(pid, year),
                'r1': wins.rate(pid, year - 1) if year - 1 in wins.played else 0.0,
                'gm': g0 / weeks, 'fold': fold_of(pid)}

    @staticmethod
    def x(r):
        a = r['age'] - 26
        return [1.0, r['r0'], r['r1'], a, a * a, r['r0'] * a, r['gm']]

    def rookie_classes(self):
        """Players with no NFL season yet: what rookies of that position and NFL draft round did,
        season by season (undrafted = round 8). Zeros for players who never scored."""
        lab, wins = self.lab, self.wins
        acc = collections.defaultdict(lambda: [[0.0, 0] for _ in range(HORIZON)])
        for pid, (year, rnd, _) in lab.nfl_draft.items():
            if year < FIRST_VALUED or year > self.last:
                continue
            pos = lab.pos(pid, year)
            if not pos:
                continue
            key = (GROUP[pos], min(rnd or 8, 8))
            for k in range(HORIZON):
                if year + k <= self.last:
                    acc[key][k][0] += wins.rate(pid, year + k)
                    acc[key][k][1] += 1
        return {key: Picks.carry([s / n if n else None for s, n in seasons]) for key, seasons in acc.items()}

    def rates(self, pid, base_year, rookie_year=None):
        """Wins per week expected in base_year+1 .. base_year+5, out of fold for this player.
        Returns None when nothing is known about him."""
        row = self.features(pid, base_year)
        if row is not None:
            fold = FOLDS if self.point_in_time or pid not in self._by_pid() else row['fold']
            band = age_band(row['age'] + 1)
            return [self.predict(row, k, fold) * self.calibration.get((row['g'], band, k), 1.0) for k in range(HORIZON)]
        nfl = self.lab.nfl_draft.get(pid)
        pos = self.lab.pos(pid, base_year + 1)
        if nfl and pos and nfl[0] >= base_year:
            seasons = self.rookie.get((GROUP[pos], min(nfl[1] or 8, 8)))
            if seasons:
                lead = nfl[0] - (base_year + 1)          # drafted a year later than the first valued season
                return [seasons[k - lead] if 0 <= k - lead < HORIZON else 0.0 for k in range(HORIZON)]
        return None

    def _by_pid(self):
        if not hasattr(self, '_index'):
            self._index = collections.defaultdict(list)
            for r in self.rows:
                self._index[r['pid']].append(r)
        return self._index


def local_linear(points, at, bandwidth):
    """Weighted straight-line fit around `at` with Gaussian weights; returns its value at `at`
    (never below zero). None when there is nothing to fit."""
    sw = swx = swy = swxx = swxy = 0.0
    for x, y in points:
        w = math.exp(-0.5 * ((x - at) / bandwidth) ** 2)
        if w < 1e-6:
            continue
        dx = x - at
        sw += w; swx += w * dx; swy += w * y; swxx += w * dx * dx; swxy += w * dx * y
    if sw == 0:
        return None
    det = sw * swxx - swx * swx
    if det <= 1e-9:
        return swy / sw
    return max(0.0, (swxx * swy - swx * swxy) / det)


class Picks:
    """What a rookie-draft pick returns, by overall slot and season since the draft.

    Realized wins per week of every player taken, season by season (0 = rookie season), from the
    classes that have finished that season. Smoothed across nearby slots with a Gaussian kernel,
    because one class has one player per slot."""

    BANDWIDTH = 8.0

    def __init__(self, wins, draft, last):
        self.wins, self.last = wins, last
        self.picks, self.unseen = [], []          # unseen: classes drafted after the knowledge cut-off
        order = collections.Counter()
        for d in sorted(draft, key=lambda d: (d['year'], d['round'], d['slot'])):
            order[d['year']] += 1
            seasons = [wins.rate(d['pid'], d['year'] + j) if d['year'] + j <= last else None for j in range(HORIZON)]
            (self.unseen if d['year'] > last + 1 else self.picks).append({**d, 'overall': order[d['year']], 'seasons': seasons})
        self.size = {y: n for y, n in order.items()}
        self.all_picks = list(self.picks) + list(self.unseen)
        self.rounds = collections.defaultdict(list)            # round -> overall slots seen
        for p in self.picks:
            self.rounds[p['round']].append(p['overall'])
        self.max_overall = max(order.values())
        self.curve = [self.carry([self.smooth(o, j) for j in range(HORIZON)]) for o in range(1, self.max_overall + 1)]

    @staticmethod
    def carry(seasons):
        """Seasons no class has reached yet take the furthest season seen (early backtests)."""
        out = []
        for v in seasons:
            out.append(v if v is not None else (out[-1] if out else 0.0))
        return out

    def smooth(self, overall, season, exclude_year=None):
        """Local-linear kernel fit at one overall slot: a plain weighted average would pull the
        first few slots down toward the picks after them, because nothing comes before pick 1."""
        return local_linear([(p['overall'], p['seasons'][season]) for p in self.picks
                             if p['seasons'][season] is not None and p['year'] != exclude_year], overall, self.BANDWIDTH)

    def expected_without_class(self):
        """Each pick's slot expectation from every other class, season by season: the fair
        benchmark for judging a pick, since its own outcome never sets its bar."""
        out = {}
        for p in self.picks:
            out[(p['year'], p['round'], p['slot'])] = [self.smooth(p['overall'], j, exclude_year=p['year']) for j in range(HORIZON)]
        return out

    def rates(self, round_, overall=None):
        """Expected wins per week by season since the draft, for an exact slot or a whole round."""
        if overall:
            return self.curve[min(overall, self.max_overall) - 1]
        slots = self.rounds.get(round_) or self.rounds[max(self.rounds)]
        n = len(slots)
        return [sum(self.curve[o - 1][j] for o in slots) / n for j in range(HORIZON)]


class Valuer:
    """Values any asset at any moment, in wins over the frame's five seasons."""

    def __init__(self, wins, forecast, picks, known_slots, estimated_slots, played_through=None):
        """played_through: last finished season, for hindsight (a backtest's forecast knows less)."""
        self.wins, self.forecast, self.picks = wins, forecast, picks
        self.played_through = played_through or forecast.last
        self.lab = wins.lab
        self.known, self.estimated = known_slots, estimated_slots
        self.overall = {(p['year'], p['round'], p['slot']): p for p in picks.all_picks}

    def frame(self, year, ts):
        """The season a move is made for, and how many of its regular weeks were already played."""
        lab = self.lab
        from build_lab import day_of, DAY
        end = lab.marks[year].get('regularEnd')
        if end and day_of(ts) > end:
            return year + 1, 0
        ends = lab.week_end.get(year, {})
        done = sum(1 for w in self.wins.played.get(year, ()) if ends.get(w) and ends[w] + DAY <= ts)
        return year, done

    def weeks(self, season):
        return self.lab.regular_weeks.get(season) if season <= self.forecast.last else 13

    def player(self, pid, season, done):
        """Expected wins in season .. season+4 (rest of season only when weeks are already played)."""
        rates = self.forecast.rates(pid, season - 1)
        if rates is None:
            return None
        rates = list(rates)
        if done:
            seen = set(self.wins.played[season][:done])
            rates[0] = (done * self.wins.rate(pid, season, seen) + PRIOR_WEEKS * rates[0]) / (done + PRIOR_WEEKS)
        out = []
        for k, r in enumerate(rates):
            weeks = self.weeks(season + k) - (done if k == 0 else 0)
            out.append(r * max(0, weeks))
        return out

    def pick_slot(self, asset):
        slot = asset.get('slot') or self.known.get((asset['year'], asset['orig']))
        return slot, bool(slot)

    def pick(self, asset, season):
        """Expected wins from a rookie pick: its slot when known, else the average of its round."""
        slot, exact = self.pick_slot(asset)
        made = self.overall.get((asset['year'], asset['round'], slot)) if slot else None
        rates = self.picks.rates(asset['round'], made['overall'] if made else None)
        lead = asset['year'] - season
        return [rates[k - lead] * self.weeks(season + k) if 0 <= k - lead < HORIZON else 0.0 for k in range(HORIZON)]

    def realized_player(self, pid, season, done):
        """Hindsight: wins the player actually produced over the same seasons (None once unplayed)."""
        out = []
        for k in range(HORIZON):
            y = season + k
            if y > self.played_through:
                out.append(None)
                continue
            weeks = self.wins.played[y][done:] if k == 0 else self.wins.played[y]
            out.append(sum(self.wins.week.get((pid, y), {}).get(w, 0.0) for w in weeks))
        return out

    def realized_pick(self, asset, season):
        slot, _ = self.pick_slot(asset)
        made = self.overall.get((asset['year'], asset['round'], slot)) if slot else None
        if not made:
            return None
        return self.realized_player(made['pid'], season, 0)


MARKET_FROM = 2017               # picks are recorded in trades from 2017; earlier trades hide assets


def next_draft(lab, ts):
    """Year of the next rookie draft after a moment."""
    from build_lab import day_of, dt
    d = day_of(ts)
    end = (lab.marks.get(d.year) or {}).get('draftEnd') or dt.date(d.year, 6, 1)
    return d.year if d <= end else d.year + 1


def asset_class(asset, val, valuer, season, ts):
    if asset['kind'] == 'pick':
        r = asset['round']
        return f"pick{r if r <= 2 else '3'}:{'next' if asset['year'] == next_draft(valuer.lab, ts) else 'later'}"
    pos = valuer.lab.pos(asset['id'], season) or valuer.lab.pos(asset['id'], season - 1)
    from build_lab import epoch, dt
    birth = valuer.lab.birth.get(asset['id'])
    age = (epoch(dt.date(season, 9, 1)) - birth) / (365.25 * 86400) if birth else None
    if not pos or age is None:
        return 'player:unknown'
    return f"{GROUP[pos]}:{age_band(age)}"


ROOKIE_TERM = {1: 5}            # rookie contracts: five seasons for 1st-rounders, three otherwise


def rookie_salaries(lab, draft):
    """Median first contract by draft year and round, from the drafting team's first roster."""
    seen = collections.defaultdict(list)
    for d in draft:
        for w in range(1, 18):
            roster = lab.rosters.get((d['year'], w, d['fid']))
            if roster and d['pid'] in roster:
                if roster[d['pid']]['salary'] is not None:
                    seen[(d['year'], d['round'])].append(roster[d['pid']]['salary'])
                break
    return {k: statistics.median(v) for k, v in seen.items() if v}


def contract_cost(valuer, asset, season, done, giver, receiver, ts, year, rookie_pay):
    """Cap dollars the asset commits in each season of the frame (None where unknown)."""
    out = [0.0] * HORIZON
    if asset['kind'] == 'pick':
        pay = rookie_pay.get((asset['year'], asset['round'])) or rookie_pay.get((max(y for y, _ in rookie_pay), asset['round']))
        if pay is None:
            return None
        for k in range(HORIZON):
            if 0 <= season + k - asset['year'] < ROOKIE_TERM.get(asset['round'], 3):
                out[k] = pay
        return out
    lab = valuer.lab
    c = lab.contract(asset['id'], year, ts, giver) or lab.contract_after(asset['id'], year, ts, receiver)
    if not c or c['salary'] is None or not c['until']:
        return None
    for k in range(HORIZON):
        if season + k <= c['until']:
            share = 1.0 - (done / valuer.weeks(season) if k == 0 and done else 0.0)
            out[k] = c['salary'] * share
    return out


def value_trades(lab, trades, valuer, rookie_pay):
    """Every trade in scope, each asset valued at the moment of the trade, plus hindsight."""
    out = []
    for t in sorted(trades, key=lambda t: t['ts']):
        if t['year'] < MARKET_FROM or t['oneSided']:
            continue
        if any(a['kind'] not in ('player', 'pick') for a in t['aGave'] + t['bGave']):
            continue
        season, done = valuer.frame(t['year'], t['ts'])
        sides = []
        for gave, giver, receiver in ((t['aGave'], t['a'], t['b']), (t['bGave'], t['b'], t['a'])):
            items = []
            for a in gave:
                if a['kind'] == 'player':
                    v = valuer.player(a['id'], season, done)
                    real = valuer.realized_player(a['id'], season, done)
                else:
                    v = valuer.pick(a, season)
                    real = valuer.realized_pick(a, season)
                items.append({'asset': a, 'value': v or [0.0] * HORIZON, 'valued': v is not None,
                              'class': asset_class(a, v, valuer, season, t['ts']), 'real': real,
                              'cost': contract_cost(valuer, a, season, done, giver, receiver, t['ts'], t['year'], rookie_pay)})
            sides.append(items)
        out.append({'t': t, 'season': season, 'done': done, 'sides': sides})
    return out


def discounted(vec, d):
    return sum(v * d ** k for k, v in enumerate(vec))


DISCOUNT = 0.75                  # next season counts 1/0.75 = 1.33x the one after; fitted, see analyse()
ANCHOR = 'pick1:next'            # prices are per expected win, relative to a next-draft 1st
RIDGE = 0.25
WINDOWS = {'opening': 'spring', 'between': 'spring', 'winter': 'spring', 'closed': 'spring',
           'rookieDraft': 'summer', 'summer': 'summer',
           'early': 'season', 'midseason': 'season', 'deadline': 'season'}


def fit_prices(valued, classes, with_cost=False):
    """What the league pays per expected win for each kind of asset.

    Each trade says the two sides were worth the same in the GMs' eyes that day. Model an asset's
    market price as its expected wins times (1 + m) for its class, plus a fixed amount per asset
    (the roster spot and the hassle of a bigger package), optionally plus a price on cap dollars.
    The anchor class has m = 0, so every price reads "per expected win, against a next-draft 1st".
    Ridge-shrunk toward the anchor; standard errors are robust to trades of different size."""
    cls = [c for c in classes if c != ANCHOR]
    idx = {c: i for i, c in enumerate(cls)}
    extra = len(cls) + (2 if with_cost else 1)
    X, Y = [], []
    for v in valued:
        x = [0.0] * extra
        for sign, side in ((-1, v['sides'][0]), (1, v['sides'][1])):
            for it in side:
                val = discounted(it['value'], DISCOUNT)
                if it['class'] in idx:
                    x[idx[it['class']]] += sign * val
                x[len(cls)] += sign
                if with_cost:
                    x[len(cls) + 1] += sign * discounted(it['cost'], DISCOUNT)
        X.append(x)
        Y.append(sum(discounted(it['value'], DISCOUNT) for it in v['sides'][0]) -
                 sum(discounted(it['value'], DISCOUNT) for it in v['sides'][1]))
    beta, A = solve(X, Y, ridge=RIDGE, free=0)
    resid = [y - sum(b * xi for b, xi in zip(beta, x)) for x, y in zip(X, Y)]
    inv = inverse(A)
    k = extra
    meat = [[0.0] * k for _ in range(k)]
    for x, e in zip(X, resid):
        nz = [(i, xi) for i, xi in enumerate(x) if xi]
        for i, xi in nz:
            for j, xj in nz:
                meat[i][j] += xi * xj * e * e
    tmp = [[sum(inv[i][a] * meat[a][b] for a in range(k)) for b in range(k)] for i in range(k)]
    se = [math.sqrt(max(0.0, sum(tmp[i][b] * inv[b][i] for b in range(k)))) for i in range(k)]
    out = {c: {'price': 1 + beta[i], 'se': se[i]} for c, i in idx.items()}
    out[ANCHOR] = {'price': 1.0, 'se': 0.0}
    meta = {'perAsset': beta[len(cls)], 'perAssetSe': se[len(cls)], 'trades': len(valued)}
    if with_cost:
        meta['perCapMillion'], meta['perCapMillionSe'] = beta[-1], se[-1]
    return out, meta


def class_label(c):
    kind, band = c.split(':')
    if kind.startswith('pick'):
        r = kind[4:]
        rnd = {'1': '1st', '2': '2nd', '3': '3rd or later'}[r]
        return f"{rnd}-round pick, {'next draft' if band == 'next' else 'a later draft'}"
    if kind == 'player':
        return 'Player with no record'
    names = {'QB': 'Quarterback', 'RB': 'Running back', 'WR': 'Wide receiver', 'TE': 'Tight end', 'PK': 'Kicker',
             'DL': 'Defensive lineman', 'LB': 'Linebacker', 'DB': 'Defensive back'}
    return f"{names[kind]}, {AGE_NAMES[band]}"


def market(valued, classes, lab, delivered):
    """Prices per class with stability splits, time-of-year windows and the hindsight check."""
    full, meta = fit_prices(valued, classes)
    splits = {
        'odd': [v for v in valued if v['season'] % 2], 'even': [v for v in valued if not v['season'] % 2],
        'early': [v for v in valued if v['season'] <= 2021], 'late': [v for v in valued if v['season'] > 2021],
    }
    for w in ('spring', 'summer', 'season'):
        splits[w] = [v for v in valued if v['window'] == w]
    fitted = {name: fit_prices(vs, classes)[0] for name, vs in splits.items()}
    costed = [v for v in valued if all(it['cost'] is not None for s in v['sides'] for it in s)]
    _, cost_meta = fit_prices(costed, classes, with_cost=True)
    count = collections.Counter(it['class'] for v in valued for s in v['sides'] for it in s)
    rows = []
    for c in classes:
        f = full[c]
        row = {'id': c, 'label': class_label(c), 'n': count[c], 'price': round(f['price'], 3),
               'lo': round(f['price'] - 1.96 * f['se'], 3), 'hi': round(f['price'] + 1.96 * f['se'], 3),
               'avgValue': round(statistics.mean(discounted(it['value'], DISCOUNT) for v in valued for s in v['sides']
                                                 for it in s if it['class'] == c), 3)}
        for name in splits:
            g = fitted[name].get(c)
            n = sum(1 for v in splits[name] for s in v['sides'] for it in s if it['class'] == c)
            row[name] = {'price': round(g['price'], 3), 'se': round(g['se'], 3), 'n': n} if g and n else None
        d = delivered.get(c)
        row['delivered'] = d
        if d and d['ratio'] > 0:
            # What one delivered win cost: the price per expected win over the share delivered.
            # Relative errors combine in quadrature.
            per = row['price'] / d['ratio']
            rel = math.sqrt((f['se'] / max(row['price'], 1e-6)) ** 2 + (d['se'] / d['ratio']) ** 2)
            row['perDelivered'] = {'value': round(per, 3), 'lo': round(per * math.exp(-1.96 * rel), 3),
                                   'hi': round(per * math.exp(1.96 * rel), 3)}
        else:
            row['perDelivered'] = None
        rows.append(row)
    return {'classes': rows, 'anchor': ANCHOR, 'discount': DISCOUNT,
            'perAsset': round(meta['perAsset'], 4), 'perAssetSe': round(meta['perAssetSe'], 4), 'trades': meta['trades'],
            'perCapMillion': round(cost_meta['perCapMillion'], 4), 'perCapMillionSe': round(cost_meta['perCapMillionSe'], 4),
            'capTrades': cost_meta['trades']}


def backtest(lab, trades, draft, known, estimated, wins):
    """What valuing trades this way would have told you at the time, against what happened.

    For each season, the forecast and pick values are rebuilt from earlier seasons only, that
    season's trades are valued, and the assets are followed over every season played since.
    Two results:
    - per trade: the side favoured by expected wins, and what the two sides actually produced;
    - per kind of asset: wins delivered against wins forecast at the time."""
    pay = rookie_salaries(lab, draft)
    last = max(y for y in wins.years if len(wins.played[y]) == lab.regular_weeks[y])
    rows, records = [], []
    kinds = collections.defaultdict(list)          # class -> [(expected, delivered, season)]
    for season in range(MARKET_FROM + 1, last):
        forecast = Forecast(wins, through=season - 1)
        picks = Picks(wins, draft, season - 1)
        valuer = Valuer(wins, forecast, picks, known, estimated, played_through=last)
        for v in value_trades(lab, [t for t in trades if valuer.frame(t['year'], t['ts'])[0] == season], valuer, pay):
            items = [it for side in v['sides'] for it in side]
            common = [k for k in range(HORIZON) if all(it['real'] is not None and it['real'][k] is not None for it in items)]
            if len(common) < 2:
                continue
            for it in items:
                e = sum(it['value'][k] * DISCOUNT ** k for k in common)
                r = sum(it['real'][k] * DISCOUNT ** k for k in common)
                kinds[it['class']].append((e, r, season))
                records.append({'t': v['t'], 'class': it['class'], 'expected': e, 'delivered': r, 'seasons': len(common)})
            exp = [sum(discounted(it['value'], DISCOUNT) for it in side) for side in v['sides']]
            real = [sum(it['real'][k] * DISCOUNT ** k for it in side for k in common) for side in v['sides']]
            gain, got = exp[1] - exp[0], real[1] - real[0]
            rows.append((abs(gain), got if gain >= 0 else -got, season))
    rows.sort()
    out = {'trades': len(rows), 'seasons': [MARKET_FROM + 1, last - 1], 'bands': []}
    for q in range(5):
        part = rows[q * len(rows) // 5:(q + 1) * len(rows) // 5]
        if part:
            out['bands'].append({'expected': round(statistics.mean(p[0] for p in part), 3),
                                 'realized': round(statistics.mean(p[1] for p in part), 3),
                                 'won': round(sum(1 for p in part if p[1] > 0) / len(part), 3), 'n': len(part)})
    xs, ys = [p[0] for p in rows], [p[1] for p in rows]
    mx, my = statistics.mean(xs), statistics.mean(ys)
    out['slope'] = round(sum((a - mx) * (b - my) for a, b in zip(xs, ys)) / sum((a - mx) ** 2 for a in xs), 3)
    out['won'] = round(sum(1 for y in ys if y > 0) / len(ys), 3)
    split = (MARKET_FROM + 1 + last - 1) // 2
    delivered = {}
    for c, obs in kinds.items():
        ratio = delivered_ratio(obs)
        if ratio is None:
            continue
        delivered[c] = {**ratio, 'early': delivered_ratio([o for o in obs if o[2] <= split]),
                        'late': delivered_ratio([o for o in obs if o[2] > split]), 'split': split}
    return out, delivered, records


def delivered_ratio(obs):
    """Wins delivered per win forecast, with a standard error from the spread across assets."""
    e = sum(o[0] for o in obs)
    if len(obs) < 8 or e < 0.5:
        return None
    r = sum(o[1] for o in obs)
    ratio = r / e
    se = math.sqrt(sum((o[1] - ratio * o[0]) ** 2 for o in obs)) / e
    return {'ratio': round(ratio, 3), 'se': round(se, 3), 'n': len(obs)}


def analyse(lab, trades, draft, known, estimated):
    wins = Wins(lab)
    forecast = Forecast(wins)
    picks = Picks(wins, draft, forecast.last)
    valuer = Valuer(wins, forecast, picks, known, estimated)
    valued = value_trades(lab, trades, valuer, rookie_salaries(lab, draft))
    for v in valued:
        v['window'] = WINDOWS[lab.phase(v['t']['year'], v['t']['ts'])]
    classes = sorted({it['class'] for v in valued for s in v['sides'] for it in s})
    trade_check, delivered, records = backtest(lab, trades, draft, known, estimated, wins)
    prices = market(valued, classes, lab, delivered)
    price_of = {c['id']: c['price'] for c in prices['classes']}
    legs, deals = {}, {}
    for v in valued:
        items = [it for side in v['sides'] for it in side]
        # Hindsight compares both sides over the same seasons: those every asset has played out.
        common = [k for k in range(HORIZON) if all(it['real'] is not None and it['real'][k] is not None for it in items)]
        sums = []
        for side_i, side in enumerate(v['sides']):
            val = real = 0.0
            for asset_i, it in enumerate(side):
                x = discounted(it['value'], DISCOUNT)
                r = sum(it['real'][k] * DISCOUNT ** k for k in common) if common else None
                legs[(id(v['t']), side_i, asset_i)] = {
                    'value': round(x, 3), 'class': it['class'], 'price': round(x * price_of[it['class']] + prices['perAsset'], 3),
                    'real': None if r is None else round(r, 3), 'realSeasons': len(common)}
                val += x
                real += r or 0.0
            sums.append((val, real))
        (ga, ra), (gb, rb) = sums
        deals[id(v['t'])] = {'aNet': round(gb - ga, 3), 'aRealNet': round(rb - ra, 3) if common else None,
                             'realSeasons': len(common), 'window': v['window'], 'season': v['season']}
    weeks = {y: len(wins.played[y]) for y in wins.years}
    return {'market': prices,
            'checks2': {'forecast': forecast.check(), 'valuedTrades': len(valued), 'currency': wins.currency_check(),
                        'replacement': {f'{p}:{y}': round(v, 2) for (p, y), v in sorted(wins.replacement.items())},
                        'backtest': trade_check},
            '_legs': legs, '_delivered': records, '_deals': deals, '_allPlay': wins.all_play(),
            '_pickSlots': picks.expected_without_class(), '_pickSeasons': {(p['year'], p['round'], p['slot']): p['seasons'] for p in picks.picks},
            '_weeks': weeks}
