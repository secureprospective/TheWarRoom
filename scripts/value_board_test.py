"""Behavior tests for the offline valuation review, using only disposable data."""
import contextlib
import copy
import gzip
import hashlib
import io
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest

import value_board as board


class ValueBoardTests(unittest.TestCase):
    def player(self, ident, points=10, value=10, age=25, last=100, pos='WR'):
        return dict(Score=dict(MFLID=ident, Position=pos, DynastyPPG=value, NowPPG=value,
                               Prior=.5, PastGames=17, SeasonGames=3, ZPast=.5, ZNow=.5,
                               OnField=.9, Inputs=['age', 'season.2026']),
                    Facts=dict(Name='Player ' + ident, Team='BUF'), Age=age,
                    LastPoints=last, ThisPoints=points, LastGames=17,
                    CurrentPPG=points / 3 if points is not None else None,
                    Rule='Fitted model', AgePull=None)

    def test_ranks_share_ties_and_do_not_impute_missing(self):
        ranks = board.ranks([dict(id='a', value=0), dict(id='b', value=0),
                             dict(id='c', value=None), dict(id='d', value=2)], 'value')
        self.assertEqual(ranks, dict(d=1, a=2, b=2))

    def test_top_scorer_with_zero_value_triggers_two_checks(self):
        players = [self.player('producer', points=1000, value=0)]
        players += [self.player(str(i), points=i, value=100-i) for i in range(25)]
        rows, counts = board.assess(players, [])
        producer = rows[0]
        self.assertEqual((producer['vr'], producer['pr']), (26, 1))
        self.assertEqual(producer['checks'], ['Rank gap', 'Zero-valued producer'])
        self.assertEqual(counts['Zero-valued producer'], 1)
        self.assertGreater(counts['Rank gap'], 0)

    def test_rank_gap_boundary_and_missing_outcome(self):
        players = [self.player(str(i), points=i+1, value=22-i) for i in range(22)]
        players.append(self.player('missing', points=None, value=100))
        rows, _ = board.assess(players, [])
        missing = rows[-1]
        self.assertIsNone(missing['pr'])
        self.assertEqual(missing['checks'], [])
        for row in rows:
            if row['pr'] is not None and abs(row['gap']) < 20:
                self.assertNotIn('Rank gap', row['checks'])

    def test_old_inversion_requires_both_seasons_and_position_threshold(self):
        players = [self.player('old', age=32, points=20, value=10, last=40),
                   self.player('young', age=24, points=30, value=5, last=60)]
        rows, counts = board.assess(players, [])
        self.assertEqual(counts['Older over producer'], 1)
        self.assertIn('Player young', rows[0]['flags'][0])
        for change in [dict(LastPoints=50), dict(ThisPoints=25), dict(Age=30), dict(LastPoints=None)]:
            changed = copy.deepcopy(players)
            if 'Age' in change:
                changed[0].update(change)
            else:
                changed[1].update(change)
            _, counts = board.assess(changed, [])
            self.assertEqual(counts['Older over producer'], 0)

    def test_no_cross_position_comparison(self):
        _, counts = board.assess([self.player('old', age=35, points=1, last=1, value=20),
                                 self.player('young', pos='RB', age=20, points=100, last=100, value=10)], [])
        self.assertEqual(counts['Older over producer'], 0)

    def test_renderer_escapes_all_external_text(self):
        player = self.player('0001')
        player['Facts']['Name'] = '<script>alert(1)</script>'
        player['Facts']['Team'] = '" onmouseover="bad'
        rows, counts = board.assess([player], [])
        provenance = dict(season=2026, as_of='<bad>', roster_at='held', players_at='held', free_agents_at='none')
        template = '@@PROVENANCE@@@@CHECKS@@@@TABLES@@'
        result = board.render(rows, counts, provenance, template)
        self.assertNotIn('<script>', result)
        self.assertIn('&lt;script&gt;', result)
        self.assertIn('&quot; onmouseover=', result)
        self.assertNotIn('@@', result)

    def test_nonfinite_output_rejected(self):
        player = self.player('0001', value=float('nan'))
        with self.assertRaisesRegex(ValueError, 'Non-finite'):
            board.assess([player], [])

    def test_backup_preserves_source_and_refuses_replacement(self):
        with tempfile.TemporaryDirectory() as folder:
            src, dest = Path(folder) / 'src.db', Path(folder) / 'copy.db'
            with sqlite3.connect(src) as db:
                db.execute('CREATE TABLE held (value TEXT)')
                db.execute('INSERT INTO held VALUES (?)', ('untouched',))
            before = src.read_bytes()
            board.snapshot(src, dest)
            with board.readonly(dest) as db:
                self.assertEqual(db.execute('SELECT value FROM held').fetchone()[0], 'untouched')
                with self.assertRaises(sqlite3.OperationalError):
                    db.execute('DELETE FROM held')
            with self.assertRaisesRegex(ValueError, 'Refusing'):
                board.snapshot(src, dest)
            self.assertEqual(src.read_bytes(), before)

    def test_archive_matches_exact_export_not_playerscores(self):
        with sqlite3.connect(':memory:') as db:
            db.row_factory = sqlite3.Row
            db.executescript('CREATE TABLE fetch_log (url,status,error,fetched_at,fetch_id,sha256);'
                             'CREATE TABLE raw_archive (sha256,size,body);')
            body = b'{"players":{}}'
            digest = hashlib.sha256(body).hexdigest()
            db.execute('INSERT INTO raw_archive VALUES (?,?,?)', (digest, len(body), gzip.compress(body)))
            for i, kind in enumerate(['players', 'playerScores']):
                db.execute('INSERT INTO fetch_log VALUES (?,200,?,?,?,?)',
                           (f'https://example.invalid/2026/export?TYPE={kind}', '', str(i), i, digest))
            result, at = board.archived_export(db, 'players', 2026)
            self.assertEqual((result, at), (body, '0'))
            self.assertEqual(board.archived_export(db, 'freeAgents', 2026, required=False), (None, None))
            with self.assertRaisesRegex(ValueError, 'No held'):
                board.archived_export(db, 'players', 2025)
            db.execute('UPDATE raw_archive SET size=100')
            with self.assertRaisesRegex(ValueError, 'Corrupt'):
                board.archived_export(db, 'players', 2026)

    def test_feature_export_uses_latest_per_source_then_priority_as_of(self):
        with tempfile.TemporaryDirectory() as folder:
            out = Path(folder)
            (out / 'measures.json').write_text(json.dumps(['points']))
            with sqlite3.connect(out / 'history.db') as db:
                db.executescript('CREATE TABLE observations (player_id,season,week,measure,source,as_of,value,text);'
                                 'CREATE TABLE source_fields (source,measure,priority);')
                db.executemany('INSERT INTO source_fields VALUES (?,?,?)', [('a', 'points', 1), ('b', 'points', 2)])
                for source, at, value in [('a', '2026-10-01', 10), ('a', '2026-10-02', 20),
                                          ('a', '2026-10-04', 40), ('b', '2026-10-03', 30)]:
                    db.execute('INSERT INTO observations VALUES (?,?,?,?,?,?,?,NULL)',
                               ('0001', 2026, 0, 'points', source, at, value))
            board.export_features(out, dict(Season=2026, AsOf='2026-10-03'))
            rows = [json.loads(line) for line in (out / 'features.jsonl').read_text().splitlines()]
            self.assertEqual(len(rows), 1)
            self.assertEqual((rows[0]['Source'], rows[0]['Value']), ('a', 20))


if __name__ == '__main__':
    unittest.main()
