import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from types import SimpleNamespace
from serve import HistoryHandler

from behavior_archive import validate_tenures, lineup_legal
from build_archive import atomic_write, compile_archive, source, transaction


class BehaviorArchiveTests(unittest.TestCase):
    def test_manual_movements_have_explicit_direction_without_becoming_gm_decisions(self):
        event = transaction({'type': 'LOAD_ROSTERS', 'franchise': '0001', 'transaction': '0835,|1234,'}, 2025, source('x'))
        self.assertEqual(event['movements'], {'added': ['0835'], 'dropped': ['1234']})
        self.assertEqual(event['type'], 'LOAD_ROSTERS')

    def test_tenures_reject_overlap_and_allow_adjacent_distinct_owners(self):
        a = {'id': 'a', 'name': 'Owner A', 'team': '0001', 'from': '2013-01-01', 'to': '2020-01-01', 'evidence': 'operator map'}
        b = dict(a, id='b', name='Owner B', **{'from': '2020-01-01', 'to': '2027-01-01'})
        self.assertEqual(len(validate_tenures([a, b], {'0001'})), 2)
        with self.assertRaises(ValueError):
            validate_tenures([a, dict(b, **{'from': '2019-12-31'})], {'0001'})
        with self.assertRaises(ValueError):
            validate_tenures([dict(a, team='bad')], {'0001'})
        with self.assertRaises(ValueError):
            validate_tenures([dict(a, email='private')], {'0001'})

    def test_legal_lineup_checks_every_constraint_and_excludes_kicker_from_offense_count(self):
        rules = {'count': '3', 'iop_starters': '2', 'idp_starters': '0',
                 'position': [{'name': p, 'limit': '1'} for p in ('QB', 'RB', 'PK')]}
        positions = {'a': 'QB', 'b': 'RB', 'c': 'PK', 'd': 'RB'}
        self.assertTrue(lineup_legal(['a', 'b', 'c'], positions, rules))
        self.assertFalse(lineup_legal(['a', 'b', 'd'], positions, rules))
        self.assertFalse(lineup_legal(['a', 'a', 'c'], positions, rules))
        self.assertFalse(lineup_legal(['a', 'b', 'missing'], positions, rules))

    def test_server_rejects_rebinding_hosts_without_blocking_local_navigation(self):
        handler = object.__new__(HistoryHandler)
        handler.server = SimpleNamespace(server_address=('127.0.0.1',8765),server_name='localhost')
        for host, expected in [('localhost:8765',True),('127.0.0.1:8765',True),('attacker.example:8765',False),('',False),('[invalid',False)]:
            handler.headers = {'Host':host}
            self.assertEqual(handler.host_allowed(),expected)
        handler.server.server_address = ('0.0.0.0',8765)
        for host, expected in [('192.168.1.191:8765',True),('attacker.example:8765',False)]:
            handler.headers = {'Host':host}
            self.assertEqual(handler.host_allowed(),expected)

    def test_atomic_index_replacement_preserves_last_good_output_on_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'archive.json'
            path.write_text('previous complete index')
            with patch.object(Path, 'replace', side_effect=OSError('simulated interruption')):
                with self.assertRaises(OSError):
                    atomic_write(path, 'next complete index')
            self.assertEqual(path.read_text(), 'previous complete index')
            self.assertEqual(list(path.parent.glob('*.tmp')), [])
            atomic_write(path, 'next complete index')
            self.assertEqual(path.read_text(), 'next complete index')

    def test_compiled_context_is_prior_only_with_real_anchors_and_scoring_gate(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            payloads = {
                'league': {'league': {'startWeek': '1', 'franchises': {'franchise': [{'id': '0001', 'name': 'A'}, {'id': '0002', 'name': 'B'}]},
                                     'starters': {'count': '2', 'iop_starters': '2', 'idp_starters': '0',
                                                  'position': [{'name': 'QB', 'limit': '1'}, {'name': 'RB', 'limit': '1'}]}}},
                'players': {'players': {'player': [{'id': '1234', 'position': 'QB', 'name': 'Player A'}, {'id': '5678', 'position': 'RB', 'name': 'Player B'}]}},
                'draftResults': {'draftResults': {'draftUnit': {'draftPick': [{'player': '1234', 'round': '1', 'pick': '1', 'franchise': '0001', 'timestamp': '1752494431'}]}}},
                'nflSchedule_W1': {'nflSchedule': {'week': '1', 'matchup': [{'kickoff': '1757203200'}]}},
                'nflSchedule_W2': {'nflSchedule': {'week': '2', 'matchup': [{'kickoff': '1757808000'}]}},
                'transactions': {'transactions': {'transaction': [
                    {'type': 'TRADE', 'timestamp': '1757203200', 'franchise': '0001', 'franchise2': '0002', 'franchise1_gave_up': '1234,', 'franchise2_gave_up': 'FP_0002_2026_1,', 'by_commish': '1'},
                    {'type': 'TRADE', 'timestamp': '1757293200', 'franchise': '0001', 'franchise2': '0002', 'franchise1_gave_up': '1234,', 'franchise2_gave_up': 'FP_0002_2026_2,', 'comments': 'salary cap cash: PRIVATE'}]}},
                'weeklyResults_WYTD': {'allWeeklyResults': {'weeklyResults': [{'week': '1', 'matchup': [{'regularSeason': '1', 'franchise': [
                    {'id': '0001', 'result': 'W', 'score': '10', 'opt_pts': '10', 'optimal': '1234,5678,',
                     'player': [{'id': '1234', 'score': '10', 'status': 'starter'}, {'id': '5678', 'status': 'starter'}]},
                    {'id': '0002', 'result': 'L', 'score': '13', 'opt_pts': '13', 'optimal': '1234,5678,', 'isHome': '1',
                     'player': [{'id': '1234', 'score': '10', 'status': 'starter'}, {'id': '5678', 'score': '0', 'status': 'starter'}]},
                    {'id': '0002', 'result': 'T', 'score': '', 'opt_pts': ''}]}]}]}}}
            # A missing optimizer score cannot erase an independently scored result.
            payloads['weeklyResults_WYTD']['allWeeklyResults']['weeklyResults'][0]['matchup'][0]['franchise'][1].pop('opt_pts')
            payloads['weeklyResults_WYTD']['allWeeklyResults']['weeklyResults'].append(
                {'week': '2', 'matchup': [{'regularSeason': '1', 'franchise': [
                    {'id': '0001', 'result': 'T', 'score': '0', 'opt_pts': '0'}]}]})
            manifest = []
            for kind, payload in payloads.items():
                relative = f'raw/2025/{kind}.json'
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(json.dumps(payload))
                manifest.append({'file': relative, 'year': 2025, 'type': kind.split('_')[0], 'status': 'ok', 'bytes': 1, 'at': '2025-09-10T00:00:00'})
            (root / 'manifest.jsonl').write_text('\n'.join(map(json.dumps, manifest)))
            result = compile_archive(root)
            self.assertEqual(result['schema'], 2)
            behavior = result['behavior']
            self.assertTrue(behavior['anchors'][2025]['draftStart'])
            self.assertEqual(len(behavior['weeks']), 2)
            self.assertEqual(behavior['coverage'][0]['unscoredRows'], 2)
            a, b = behavior['weeks']
            self.assertEqual(a['gap'], 0)
            self.assertEqual(a['omittedScores'], 1)
            self.assertIsNone(b['gap'])
            self.assertIn('unreconciled-score', b['flags'])
            self.assertIn('missing-optimal-score', b['flags'])
            self.assertTrue(all(e['pickEncodingObserved'] for e in result['events']))
            trades = [e for e in result['events'] if e['type'] == 'TRADE']
            self.assertTrue(trades[0]['commissioner'])
            self.assertEqual(trades[0]['context'], {})
            self.assertEqual(trades[1]['context']['0001']['lastResult'], 'W')
            self.assertTrue(trades[1]['context']['0001']['recordComplete'])
            self.assertEqual(trades[1]['context']['0002']['lastResult'], 'L')
            self.assertIn('non-token-consideration-note', trades[1]['flags'])
            self.assertNotIn('PRIVATE', json.dumps(result))
            self.assertEqual(trades[0]['sides'][0][0]['position'], 'QB')


if __name__ == '__main__':
    unittest.main()
