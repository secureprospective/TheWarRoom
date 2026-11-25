"""Tests for schedule.py: run the real script and check the JSON files it writes (python3 -m unittest)."""
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('schedule.py')
KEYS = {'league', 'command', 'args', 'reason', 'request', 'effective', 'visibility'}


def run(league, teams):
    out = tempfile.TemporaryDirectory()
    result = subprocess.run([sys.executable, '-I', str(SCRIPT), '--league', league, '--teams', str(teams),
                             '--out', out.name], capture_output=True, text=True)
    return result, out


def written(teams):
    result, out = run(f'native.rehtest-{teams}', teams)
    assert result.returncode == 0, result.stderr
    with out:
        gen = json.loads((Path(out.name) / 'schedule-generate.json').read_text())
        pub = json.loads((Path(out.name) / 'schedule-publish.json').read_text())
    return gen, pub


class ScheduleTests(unittest.TestCase):
    def test_weeks_1_to_12_empty_and_13_to_18_played(self):
        for n in (10, 12, 16, 32):
            with self.subTest(teams=n):
                weeks = written(n)[0]['args']['schedule']['fixtures']
                self.assertEqual([w['week'] for w in weeks], list(range(1, 19)))
                self.assertTrue(all(not w['games'] for w in weeks[:12]))
                self.assertTrue(all(len(w['games']) == n // 2 for w in weeks[12:]))

    def test_every_franchise_plays_once_a_week_never_itself(self):
        for n in (10, 12, 16, 32):
            ids = [f'franchise.{i:02d}' for i in range(1, n + 1)]
            for week in written(n)[0]['args']['schedule']['fixtures'][12:]:
                with self.subTest(teams=n, week=week['week']):
                    teams = [t for g in week['games'] for t in (g['home'], g['away'])]
                    self.assertCountEqual(teams, ids)
                    self.assertTrue(all(g['home'] != g['away'] for g in week['games']))

    def test_no_pairing_repeats_in_the_six_played_weeks(self):
        for n in (10, 12, 16, 32):
            with self.subTest(teams=n):
                pairs = [frozenset((g['home'], g['away']))
                         for w in written(n)[0]['args']['schedule']['fixtures'] for g in w['games']]
                self.assertEqual(len(pairs), len(set(pairs)))

    def test_command_files_have_exactly_the_wire_keys(self):
        gen, pub = written(12)
        for doc, name in ((gen, 'schedule.generate'), (pub, 'schedule.publish')):
            with self.subTest(command=name):
                self.assertEqual(set(doc), KEYS)
                self.assertEqual(doc['command'], name)
                self.assertEqual(doc['league'], 'native.rehtest-12')
                self.assertIn('rehtest-12', doc['request'])
        self.assertEqual(gen['args']['schedule']['weeks'], 18)

    def test_refuses_bad_input_with_exit_2_and_writes_nothing(self):
        for league, n in (('native.x', 3), ('native.x', 5), ('native.x', 2), ('native.x', 34), ('mfl.14432', 10)):
            with self.subTest(league=league, teams=n):
                result, out = run(league, n)
                with out:
                    self.assertEqual(result.returncode, 2)
                    self.assertEqual(list(Path(out.name).iterdir()), [])


if __name__ == '__main__':
    unittest.main()
