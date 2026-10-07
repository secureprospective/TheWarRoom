"""Tests for the facts database: nothing lost on the way in, the checks hold, the cards trace back.

    python3 -m unittest store/test_store.py        # builds a private copy; about 30 seconds
"""
import hashlib
import json
import sqlite3
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import cards     # noqa: E402
import checks    # noqa: E402
import load      # noqa: E402
import notes     # noqa: E402

TMP = Path(tempfile.mkdtemp(prefix='mfl-store-test-'))
DB = TMP / 'mfl.db'


def setUpModule():
    load.build(DB)
    db = sqlite3.connect(DB)
    checks.Checks(db).run()
    db.close()
    notes.main(DB)
    cards.main(DB)


def raw(year, name):
    path = load.RAW / str(year) / name
    try:
        return json.loads(path.read_bytes())
    except (OSError, ValueError):
        return {}


class NothingLost(unittest.TestCase):
    """Every record in MFL's files is a row, or is accounted for in load_note."""

    @classmethod
    def setUpClass(cls):
        cls.db = sqlite3.connect(DB)

    def count(self, sql, *args):
        return self.db.execute(sql, args).fetchone()[0]

    def test_transactions(self):
        for y in load.Loader(self.db).years:
            n = len(load.rows((raw(y, 'transactions.json').get('transactions') or {}).get('transaction')))
            self.assertEqual(self.count('SELECT COUNT(*) FROM txn WHERE year = ?', y), n, y)

    def test_draft_contracts_cap(self):
        for y in load.Loader(self.db).years:
            units = load.rows((raw(y, 'draftResults.json').get('draftResults') or {}).get('draftUnit'))
            self.assertEqual(self.count('SELECT COUNT(*) FROM draft_pick WHERE year = ?', y),
                             sum(len(load.rows(u.get('draftPick'))) for u in units), y)
            sal = load.rows(((raw(y, 'salaries.json').get('salaries') or {}).get('leagueUnit') or {}).get('player'))
            self.assertEqual(self.count('SELECT COUNT(*) FROM contract WHERE year = ?', y), len(sal), y)
            adj = load.rows((raw(y, 'salaryAdjustments.json').get('salaryAdjustments') or {}).get('salaryAdjustment'))
            self.assertEqual(self.count('SELECT COUNT(*) FROM cap_charge WHERE year = ?', y), len(adj), y)

    def test_rosters(self):
        """Rows plus the doubled IR and taxi listings equal MFL's roster entries."""
        for y in load.Loader(self.db).years:
            entries = 0
            for w in [*load.WEEKS, None]:
                d = raw(y, f'weekly/rosters_W{w:02d}.json' if w else 'rosters.json')
                entries += sum(len(load.rows(f.get('player'))) for f in load.rows((d.get('rosters') or {}).get('franchise')))
            rows = self.count('SELECT COUNT(*) FROM roster_week WHERE year = ?', y)
            doubled = self.count("SELECT COALESCE(SUM(n), 0) FROM load_note WHERE step = 'rosters' AND src LIKE ?", f'{y}/%')
            self.assertEqual(rows + doubled, entries, y)

    def test_players_are_mfl_only(self):
        srcs = {s for (s,) in self.db.execute('SELECT DISTINCT src FROM player')}
        self.assertTrue(srcs and all(s.endswith('players_DETAILS1.json') or s.endswith('players.json') for s in srcs), srcs)
        for f in (p for p in Path(__file__).parent.glob('*.py') if p.name != 'test_store.py'):
            self.assertNotIn('db_playerids', f.read_text(), f'{f.name} reads outside player data')

    def test_every_row_has_a_source(self):
        tables = [t for (t,) in self.db.execute("SELECT name FROM sqlite_master WHERE type = 'table'")]
        for t in tables:
            cols = [c[1] for c in self.db.execute(f'PRAGMA table_info({t})')]
            if 'src' in cols and t != 'source_file':
                missing = self.count(f'SELECT COUNT(*) FROM {t} WHERE src IS NOT NULL AND src NOT IN (SELECT src FROM source_file)')
                self.assertEqual(missing, 0, t)


class SafeRebuild(unittest.TestCase):
    def test_rebuild_refuses_while_the_database_is_in_use(self):
        path = TMP / 'busy.db'
        load.build(path)
        writer = sqlite3.connect(path)
        writer.execute('BEGIN IMMEDIATE')          # something else is writing
        try:
            with self.assertRaises(SystemExit):
                load.retire(path)
        finally:
            writer.rollback()
            writer.close()
        load.retire(path)                          # idle again: fine
        self.assertFalse(Path(f'{path}-wal').exists())


class NoteReadings(unittest.TestCase):
    def test_every_trade_note_has_a_reading(self):
        db = sqlite3.connect(DB)
        unread = db.execute('''SELECT COUNT(*) FROM txn WHERE type = 'TRADE' AND comments IS NOT NULL
                               AND txn_id NOT IN (SELECT txn_id FROM note_reading)''').fetchone()[0]
        self.assertEqual(unread, 0)

    def test_readings_name_the_two_teams_in_the_trade(self):
        db = sqlite3.connect(DB)
        wrong = db.execute('''SELECT n.txn_id FROM note_reading n JOIN txn t USING (txn_id)
                              WHERE n.from_fid IS NOT NULL AND NOT (
                                (n.from_fid = t.fid AND n.to_fid = t.fid2) OR (n.from_fid = t.fid2 AND n.to_fid = t.fid))''').fetchall()
        self.assertEqual(wrong, [])

    def test_readings_are_never_marked_confirmed_by_the_build(self):
        db = sqlite3.connect(DB)
        self.assertEqual(db.execute('SELECT SUM(confirmed) FROM note_reading').fetchone()[0], 0)


class Decoding(unittest.TestCase):
    def test_current_year_picks_count_from_zero(self):
        a = load.decode_asset('DP_3_20', 2024)
        self.assertEqual((a['kind'], a['pick_year'], a['pick_round'], a['pick_slot']), ('pick', 2024, 4, 21))

    def test_future_pick(self):
        a = load.decode_asset('FP_0025_2026_1', 2024)
        self.assertEqual((a['pick_orig'], a['pick_year'], a['pick_round'], a['pick_slot']), ('0025', 2026, 1, None))

    def test_player_ids_keep_leading_zeros(self):
        self.assertEqual(load.decode_asset('0151', 2024)['pid'], '0151')


class ChecksHold(unittest.TestCase):
    def test_all_checks_pass(self):
        db = sqlite3.connect(DB)
        failed = db.execute('SELECT name, open FROM check_result WHERE passed = 0').fetchall()
        self.assertEqual(failed, [])
        self.assertGreaterEqual(db.execute('SELECT COUNT(*) FROM check_result').fetchone()[0], 9)

    def test_points_for_match_mfl_everywhere(self):
        db = sqlite3.connect(DB)
        self.assertEqual(db.execute("SELECT checked, agreed FROM check_result WHERE name LIKE 'Points for%'").fetchone(),
                         (448, 448))


class CardsTraceBack(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.db = sqlite3.connect(DB)

    def test_trade_and_offer_cards_point_at_real_transactions(self):
        for cid, ref in self.db.execute("SELECT card_id, ref FROM card WHERE kind IN ('trade', 'offer')"):
            tid = json.loads(ref)['txn']
            self.assertIsNotNone(self.db.execute('SELECT 1 FROM txn WHERE txn_id = ?', (tid,)).fetchone(), cid)

    def test_every_trade_has_a_card(self):
        trades = self.db.execute("SELECT COUNT(*) FROM txn WHERE type = 'TRADE'").fetchone()[0]
        self.assertEqual(self.db.execute("SELECT COUNT(*) FROM card WHERE kind = 'trade'").fetchone()[0], trades)

    def test_cards_are_the_same_every_build(self):
        def digest(db):
            h = hashlib.sha256()
            for row in db.execute('SELECT card_id, title, body FROM card ORDER BY card_id'):
                h.update(repr(row).encode())
            return h.hexdigest()
        before = digest(self.db)
        path = load.build(TMP / 'again.db')    # a second build from the raw files
        notes.main(path)
        cards.main(path)
        again = sqlite3.connect(path)
        self.assertEqual(digest(again), before)

    def test_words_search_finds_a_trade_note(self):
        hit = self.db.execute('''SELECT c.card_id FROM card_fts f JOIN card c ON c.rowid = f.rowid
                                 WHERE card_fts MATCH '"KC"' AND c.kind = 'trade' LIMIT 1''').fetchone()
        self.assertIsNotNone(hit)


if __name__ == '__main__':
    unittest.main()
