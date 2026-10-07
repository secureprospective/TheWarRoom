import contextlib
import csv
import io
import tempfile
import unittest
from pathlib import Path
from normalize import normalize, validate

REGISTRY = Path(__file__).resolve().parents[1] / 'endpoint-registry.csv'


class NormalizeTests(unittest.TestCase):
    def setUp(self):
        with REGISTRY.open(newline='', encoding='utf-8') as file:
            self.rows = list(csv.DictReader(file))

    def test_idempotent_and_bad_copy_is_not_rewritten(self):
        with tempfile.TemporaryDirectory(dir=Path(__file__).parent) as directory:
            path = Path(directory) / 'registry.csv'
            path.write_bytes(REGISTRY.read_bytes())
            with contextlib.redirect_stdout(io.StringIO()):
                normalize(path)
                first = path.read_bytes()
                normalize(path)
            self.assertEqual(first, path.read_bytes())
            self.rows[0]['placement'] = 'Home'
            with path.open('w', newline='', encoding='utf-8') as file:
                writer = csv.DictWriter(file, list(self.rows[0]))
                writer.writeheader()
                writer.writerows(self.rows)
            broken = path.read_bytes()
            with self.assertRaisesRegex(ValueError, 'kept row needs valid placement'):
                normalize(path)
            self.assertEqual(broken, path.read_bytes())

    def test_raw_input_normalizes_without_losing_original_notes(self):
        added = {'disposition_note', 'merged_into', 'merged_place', 'gravity_reason',
                 'ring_note', 'gravity_note'}
        columns = [key for key in self.rows[0] if key not in added]
        raw = [{key: row[key] for key in columns} for row in self.rows]
        for row, original in zip(raw, self.rows):
            row['disposition'] = original['disposition_note']
            row['gravity'] = original['gravity_note']
            row['ring'] = original['ring_note']
        raw[0]['placement'] = 'Home'
        next(r for r in raw if r['id'] == 'L-34')['placement'] = 'Control Room › Data & sources'
        with tempfile.TemporaryDirectory(dir=Path(__file__).parent) as directory:
            path = Path(directory) / 'raw.csv'
            with path.open('w', newline='', encoding='utf-8') as file:
                writer = csv.DictWriter(file, columns)
                writer.writeheader()
                writer.writerows(raw)
            with contextlib.redirect_stdout(io.StringIO()):
                normalize(path)
            self.assertEqual(path.read_bytes(), REGISTRY.read_bytes())

    def test_each_validation_rejects_a_mutant(self):
        merged = next(i for i, r in enumerate(self.rows) if r['merged_into'])
        retired = next(i for i, r in enumerate(self.rows) if r['disposition'] == 'retired')
        mutations = (
            (0, 'id', self.rows[1]['id'], 'duplicate'),
            (0, 'ring', '5', 'invalid ring'),
            (0, 'gravity', 'G4', 'invalid gravity'),
            (retired, 'placement', 'Inspector', 'retired row has placement'),
            (merged, 'merged_into', 'M-999', 'not kept'),
            (merged, 'merged_into', '', 'exactly one'),
            (merged, 'merged_place', 'Inspector', 'exactly one'),
        )
        for index, field, value, error in mutations:
            with self.subTest(field=field, value=value):
                rows = [dict(r) for r in self.rows]
                rows[index][field] = value
                with self.assertRaisesRegex(ValueError, error):
                    validate(rows)

    def test_original_disposition_and_conditional_chrome_are_preserved(self):
        by_id = {r['id']: r for r in self.rows}
        self.assertEqual(by_id['M-145']['merged_place'], 'Inspector')
        self.assertEqual(by_id['M-145']['merged_into'], '')
        self.assertEqual(by_id['L-34']['placement'], 'Control Room › Data and sources')
        self.assertEqual(by_id['M-057']['gravity_reason'], '→G2 gameday')
        self.assertEqual(by_id['X-03']['ring_note'], '1 (Classic) / 4 (Default)')


if __name__ == '__main__':
    unittest.main()
