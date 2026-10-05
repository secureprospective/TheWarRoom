import json
from pathlib import Path
import tempfile
import unittest

from build_archive import assets, compile_archive, date_of, inventory, rows, source, transaction
from serve import HistoryHandler


class ArchiveTests(unittest.TestCase):
    def test_asset_dialects_and_leading_zero_ids(self):
        parsed = assets('0835,FP_0025_2027_1,DP_4_27,odd-token,', 2026)
        self.assertEqual(parsed[0], {'kind': 'player', 'id': '0835'})
        self.assertEqual(parsed[1]['original'], '0025')
        self.assertEqual(parsed[1]['year'], 2027)
        self.assertEqual(parsed[2]['round'], 4)
        self.assertEqual(parsed[2]['pick'], 27)
        self.assertEqual(parsed[2]['year'], 2026)
        self.assertEqual(parsed[3]['kind'], 'unknown')

    def test_completed_trade_and_comment_uncertainty(self):
        raw = {'type': 'TRADE', 'franchise': '0001', 'franchise2': '0025',
               'timestamp': '1761879278', 'franchise1_gave_up': '0835,FP_0025_2027_1,',
               'franchise2_gave_up': 'DP_2_3,', 'comments': 'Also a 2027 1st pick'}
        event = transaction(raw, 2026, source('raw/2026/transactions.json', 7))
        self.assertEqual(event['picks'], 2)
        self.assertEqual(event['players'], ['0835'])
        self.assertTrue(event['pickNote'])
        self.assertNotIn('comments', event)
        self.assertNotIn('Also a 2027 1st pick', json.dumps(event))
        raw['type'] = 'TRADE_PROPOSAL'
        self.assertFalse(transaction(raw, 2026, source('x'))['pickNote'])

    def test_free_agent_add_drop_direction(self):
        event = transaction({'type': 'FREE_AGENT', 'transaction': '0835,1234,|5678,', 'franchise': '0025'}, 2026, source('x'))
        self.assertEqual(event['movements'], {'added': ['0835', '1234'], 'dropped': ['5678']})
        self.assertEqual(event['players'], ['0835', '1234', '5678'])
        self.assertIsNone(event['date'])
        with self.assertRaises(ValueError):
            transaction({'type': 'FREE_AGENT', 'transaction': 'not-a-player|'}, 2026, source('x'))

    def test_dates_are_utc_not_archive_season(self):
        self.assertEqual(date_of('0'), '1970-01-01T00:00:00+00:00')
        self.assertIsNone(date_of(''))
        self.assertTrue(date_of('1395377590').startswith('2014-'))

    def test_object_array_boundary(self):
        self.assertEqual(rows({'id': '1'}), [{'id': '1'}])
        self.assertEqual(rows(None), [])
        with self.assertRaises(ValueError):
            rows(['bad'])

    def test_latest_manifest_status_wins_and_path_is_checked(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root / 'manifest.jsonl'
            raw = {'file': 'raw/2026/calendar.json', 'year': 2026, 'type': 'calendar', 'bytes': 10, 'status': 'empty'}
            path.write_text(json.dumps(raw)+'\n'+json.dumps(dict(raw, status='ok:calendar')))
            self.assertEqual(len(inventory(root)), 1)
            self.assertEqual(inventory(root)[0]['health'], 'ok')
            raw['file'] = '../private.json'
            path.write_text(json.dumps(raw))
            with self.assertRaises(ValueError):
                inventory(root)

    def test_server_denies_code_raw_and_traversal(self):
        handler = object.__new__(HistoryHandler)
        for path in ['/', '/data/archive.json?version=1', '/model.mjs']:
            handler.path = path
            self.assertTrue(handler.allowed())
        for path in ['/build_archive.py', '/README.md', '/raw/2026/league.json', '/../league-archive/raw/2026/league.json', '/%2e%2e/private', '/.git/config']:
            handler.path = path
            self.assertFalse(handler.allowed())

    def test_compiler_deduplicates_and_preserves_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = []
            trade = {'type': 'TRADE', 'timestamp': '1761879278', 'franchise': '0001', 'franchise2': '0025', 'franchise1_gave_up': '0835,FP_0025_2027_1,'}
            for year in [2025, 2026]:
                base = root / 'raw' / str(year)
                base.mkdir(parents=True)
                payloads = {'transactions': {'transactions': {'transaction': [trade]}},
                            'league': {'league': {'franchises': {'franchise': {'id': '0001', 'name': 'Bills'}}}},
                            'players': {'players': {'player': {'id': '0835', 'name': 'Test, Player', 'position': 'QB'}}}}
                for kind, payload in payloads.items():
                    relative = f'raw/{year}/{kind}.json'
                    (root / relative).write_text(json.dumps(payload))
                    manifest.append({'file': relative, 'year': year, 'type': kind, 'status': 'ok', 'bytes': 1})
            (root / 'manifest.jsonl').write_text('\n'.join(map(json.dumps, manifest)))
            compiled = compile_archive(root)
            self.assertEqual(len(compiled['events']), 1)
            event = compiled['events'][0]
            self.assertEqual(event['picks'], 1)
            self.assertEqual(len(event['sources']), 2)
            self.assertEqual(event['players'], ['0835'])
            self.assertIsNone(compiled['seasons'][0]['champion'])


if __name__ == '__main__':
    unittest.main()
