"""Checks on the compiled Lab data against the facts database. Run: python3 -m unittest compile/test_build_lab.py"""
import collections
import datetime as dt
import json
import re
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import build_lab as B  # noqa: E402
import value as V  # noqa: E402

DATA = None
WINS = None
DB = None


def setUpModule():
    global DATA, WINS, DB
    DB = B.connect()
    DATA = B.build(B.capture_time(), DB)
    lab = B.Lab(0, DB)
    lab._live = {}
    lab.read_reference()
    lab.read_weeks()
    lab.build_standings()
    lab.build_production()
    WINS = V.Wins(lab)


def table(name):
    cols = DATA[name]
    n = len(next(iter(cols.values())))
    return [{k: v[i] for k, v in cols.items()} for i in range(n)]


class ArchiveTotals(unittest.TestCase):
    def test_every_completed_trade_is_kept(self):
        stored = DB.execute("SELECT COUNT(*) FROM txn WHERE type = 'TRADE'").fetchone()[0]
        self.assertEqual(len(DATA['trades']['ts']), stored)
        self.assertEqual(stored, 2634)

    def test_pick_moves_match_raw_tokens(self):
        picks = sum(1 for leg in table('legs') if leg['kind'] == 'pick')
        self.assertEqual(picks, 3780)

    def test_legs_belong_to_their_trade(self):
        trades = table('trades')
        for leg in table('legs'):
            t = trades[leg['trade']]
            self.assertEqual({leg['from'], leg['to']}, {t['a'], t['b']})


class PickCodes(unittest.TestCase):
    def test_last_holder_of_every_traded_current_pick_made_that_selection(self):
        trades = sorted(table('trades'), key=lambda t: t['ts'])
        legs = table('legs')
        drafted = {(d['year'], d['round'], d['slot']): d['fid'] for d in table('draft')}
        last = {}
        for leg in sorted(legs, key=lambda l: DATA['trades']['ts'][l['trade']]):
            if leg['kind'] == 'pick' and leg['orig'] is None and leg['slot']:
                last[(leg['pickYear'], leg['round'], leg['slot'])] = leg['to']
        checked = [(k, f) for k, f in last.items() if k in drafted]
        self.assertGreater(len(checked), 800)
        self.assertEqual(sum(drafted[k] == f for k, f in checked), len(checked))

    def test_arizona_march_2024_first_round_picks(self):
        # Bee's explorer reported 2 received; both were second-round picks read one round low.
        rec = sent = 0
        trades = table('trades')
        for leg in table('legs'):
            t = trades[leg['trade']]
            when = B.day_of(t['ts'])
            if leg['kind'] == 'pick' and leg['round'] == 1 and not t['oneSided'] and (when.year, when.month) == (2024, 3):
                rec += leg['to'] == '0025'
                sent += leg['from'] == '0025'
        self.assertEqual((rec, sent), (0, 1))


class NoHindsightLeaks(unittest.TestCase):
    def test_live_standings_only_use_finished_weeks(self):
        # A live standing needs three finished games: the trade must come after week 3's
        # Monday game plus a day, i.e. at least four days after week 3's Thursday kickoff.
        seasons = {s['year']: s for s in DATA['seasons']}
        for t in table('trades'):
            if t['rankBasis'] == 'live':
                week3 = seasons[t['year']]['weekStart'][3]
                self.assertGreater(t['ts'], week3 + 4 * 86400)

    def test_standings_are_complete_rankings(self):
        by = collections.defaultdict(list)
        for s in table('teamSeasons'):
            if s['rank'] is not None:
                by[s['year']].append(s['rank'])
        for year, ranks in by.items():
            self.assertEqual(sorted(ranks), list(range(1, 33)), year)

    def test_contract_read_from_later_roster_is_flagged(self):
        for leg in table('legs'):
            if leg['kind'] == 'player' and leg['salary'] is not None:
                self.assertIn(leg['contractFrom'], ('before', 'after'))

    def test_year_earlier_rank_is_a_finished_season(self):
        # Live standing in season Y: a year earlier is Y-1's finish. Last season's finish
        # used as the standing: a year earlier is Y-2's finish.
        final = {(s['year'], s['fid']): s['rank'] for s in table('teamSeasons')}
        checked = 0
        for t in table('trades'):
            back = {'live': 1, 'last': 2}.get(t['rankBasis'])
            if not back:
                continue
            self.assertEqual(t['aPrev'], final.get((t['year'] - back, t['a'])))
            checked += 1
        self.assertGreater(checked, 2000)

    def test_cap_room_only_where_contracts_are_recorded(self):
        cover = {s['year']: s['contractsRecorded'] for s in DATA['seasons']}
        for t in table('trades'):
            if t['aCap'] is not None:
                self.assertGreaterEqual(cover[t['year']], 0.85)


class Days(unittest.TestCase):
    def test_days_are_mfl_eastern_days(self):
        # Week 1 of 2024 kicked off on Thursday 5 September at 8:20 pm Eastern (00:20 UTC on the 6th).
        seasons = {s['year']: s for s in DATA['seasons']}
        self.assertEqual(seasons[2024]['marks']['kickoff'], '2024-09-05')

    def test_draft_days_match_mfl(self):
        made = {(d['year'], d['round'], d['slot']): d['ts'] for d in table('draft')}
        shown = DB.execute('SELECT year, round, pick, day FROM draft_pick WHERE pid IS NOT NULL AND ts').fetchall()
        self.assertEqual([k for *k, day in shown if B.day_of(made[tuple(k)]).isoformat() != day], [])


class Calendar(unittest.TestCase):
    def test_trading_stops_after_each_finished_deadline(self):
        trades = table('trades')
        for s in DATA['seasons']:
            mk = s['marks']
            if not mk['deadline'] or mk['deadlineExpected']:
                continue
            deadline = dt.date.fromisoformat(mk['deadline'])
            end = dt.date.fromisoformat(mk['seasonEnd'])
            late = [t for t in trades if t['year'] == s['year'] and not t['oneSided']
                    and deadline < B.day_of(t['ts']) <= end]
            self.assertEqual(late, [], s['year'])

    def test_eras_cover_every_season_once(self):
        years = [y for e in DATA['eras'] for y in range(e['years'][0], e['years'][1] + 1)]
        self.assertEqual(years, list(range(B.FIRST_YEAR, B.LAST_YEAR + 1)))


class Valuation(unittest.TestCase):
    def test_currency_describes_winning(self):
        check = DATA['checks2']['currency']
        self.assertGreater(check['teamSeasons'], 350)
        self.assertGreater(check['correlation'], 0.9)

    def test_only_trades_with_every_asset_recorded_are_valued(self):
        for t in table('trades'):
            if t['aNet'] is not None:
                self.assertGreaterEqual(t['year'], V.MARKET_FROM)
                self.assertFalse(t['oneSided'])
        valued = {i for i, t in enumerate(table('trades')) if t['aNet'] is not None}
        for leg in table('legs'):
            self.assertEqual(leg['xValue'] is not None, leg['trade'] in valued and leg['kind'] in ('player', 'pick'))

    def test_prices_are_read_against_a_next_draft_first(self):
        prices = {c['id']: c for c in DATA['market']['classes']}
        self.assertEqual(prices[V.ANCHOR]['price'], 1.0)
        self.assertEqual(DATA['market']['trades'], DATA['checks2']['valuedTrades'])

    def test_hindsight_only_counts_seasons_already_played(self):
        last = max(s['year'] for s in DATA['seasons'] if len(s['weeksDone']) >= s['regularWeeks'])
        for t in table('trades'):
            if t['realSeasons']:
                self.assertLessEqual(t['season'] + t['realSeasons'] - 1, last)
        for r in table('delivered'):
            t = table('trades')[r['trade']]
            self.assertLessEqual(t['season'] + r['seasons'] - 1, last)

    def test_backtest_forecast_knows_only_earlier_seasons(self):
        f = V.Forecast(WINS, through=2019)
        self.assertEqual(f.last, 2019)
        self.assertTrue(all(r['y'] <= 2019 for r in f.rows))
        for r in f.rows:
            for k, target in enumerate(r['fut']):
                if target is not None:
                    self.assertLessEqual(r['y'] + k + 1, 2019)
        picks = V.Picks(WINS, [{'year': 2022, 'round': 1, 'slot': 1, 'fid': '0001', 'pid': 'x'}], 2019)
        self.assertEqual(picks.picks, [])

    def test_a_pick_is_never_judged_by_its_own_class(self):
        class Fake:
            def __init__(self, rates):
                self.rates = rates

            def rate(self, pid, year):
                return self.rates.get((pid, year), 0.0)
        # One class wildly better than the others: its own bar must come from the others only.
        rates = {(f'{y}-{o}', y + j): (5.0 if y == 2020 else 0.1) for y in (2018, 2019, 2020) for o in range(1, 21) for j in range(5)}
        draft = [{'year': y, 'round': 1, 'slot': o, 'fid': '0001', 'pid': f'{y}-{o}'} for y in (2018, 2019, 2020) for o in range(1, 21)]
        bars = V.Picks(Fake(rates), draft, 2025).expected_without_class()
        for o in range(1, 21):
            self.assertAlmostEqual(bars[(2020, 1, o)][0], 0.1, places=6)

    def test_slot_curve_has_no_edge_dip(self):
        # Points on a straight line: the fit at the first slot must sit on the line, not below it.
        points = [(x, 2.0 - 0.01 * x) for x in range(1, 60)]
        self.assertAlmostEqual(V.local_linear(points, 1, 8.0), 1.99, places=6)

    def test_ledger_explains_every_move_from_2018(self):
        for year, row in DATA['ledger']['byYear'].items():
            if 2018 <= int(year) <= 2025:
                self.assertGreaterEqual(row['changeRate'], 0.99, year)


if __name__ == '__main__':
    unittest.main()
