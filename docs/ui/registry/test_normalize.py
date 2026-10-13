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

    def test_current_registry_is_preserved_byte_for_byte(self):
        with tempfile.TemporaryDirectory(dir=Path(__file__).parent) as directory:
            path = Path(directory) / 'registry.csv'
            original = REGISTRY.read_bytes()
            path.write_bytes(original)
            with contextlib.redirect_stdout(io.StringIO()):
                normalize(path)
            self.assertEqual(path.read_bytes(), original)

    def test_raw_rows_normalize_owner_and_commissioner_metadata(self):
        added = {'disposition_note', 'merged_into', 'merged_place', 'gravity_reason',
                 'ring_note', 'gravity_note'}
        columns = [key for key in self.rows[0] if key not in added]
        selected = [r for r in self.rows if r['id'] == 'M-001' or r['id'].startswith('CM-')]
        raw = [{key: row[key] for key in columns} for row in selected]
        raw[0]['gravity'] = 'G2 — active lineup calls lock at kickoff'
        for row in raw[1:]:
            if row['id'] not in ('CM-039', 'CM-056'):
                row['governs'] = 'M-001'
        with tempfile.TemporaryDirectory(dir=Path(__file__).parent) as directory:
            path = Path(directory) / 'raw.csv'
            with path.open('w', newline='', encoding='utf-8') as file:
                writer = csv.DictWriter(file, columns)
                writer.writeheader()
                writer.writerows(raw)
            with contextlib.redirect_stdout(io.StringIO()):
                normalize(path)
            with path.open(newline='', encoding='utf-8') as file:
                normalized = list(csv.DictReader(file))
            validate(normalized)
            self.assertEqual(normalized[0]['gravity'], 'G2')
            self.assertEqual(normalized[0]['gravity_reason'], 'active lineup calls lock at kickoff')
            self.assertEqual(normalized[0]['gravity_note'], raw[0]['gravity'])
            self.assertEqual(normalized[0]['disposition_note'], 'kept')
            deferred = next(r for r in normalized if r['id'] == 'CM-055')
            self.assertEqual(deferred['ring'], '6')
            self.assertEqual(deferred['governs'], 'M-001')

    def test_each_validation_rejects_a_mutant(self):
        merged = next(i for i, r in enumerate(self.rows) if r['merged_into'])
        retired = next(i for i, r in enumerate(self.rows) if r['disposition'] == 'retired')
        mutations = (
            (0, 'id', self.rows[1]['id'], 'duplicate'),
            (0, 'ring', '10', 'invalid ring'),
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

    def test_commissioner_validation_rejects_mutants(self):
        indexes = {r['id']: i for i, r in enumerate(self.rows)}
        mutations = (
            ('CM-001', 'governs', 'M-999', 'not an owner id'),
            ('CM-001', 'governs', 'CM-002', 'not an owner id'),
            ('CM-001', 'governs', '', 'empty governs'),
            ('CM-001', 'governs', 'M-091 M-091', 'duplicate governs'),
            ('CM-001', 'governs', 'ledger-wide', 'invalid ledger-wide'),
            ('CM-039', 'governs', 'ledger-wide M-091', 'invalid ledger-wide'),
            ('CM-056', 'governs', 'M-091', 'requires ledger-wide'),
            ('M-001', 'governs', 'M-002', 'owner has governs'),
            ('CM-028', 'id', 'CM-057', 'contiguous'),
            ('CM-028', 'id', 'CM-027', 'duplicate endpoint'),
        )
        for ident, field, value, error in mutations:
            with self.subTest(ident=ident, field=field, value=value):
                rows = [dict(r) for r in self.rows]
                rows[indexes[ident]][field] = value
                with self.assertRaisesRegex(ValueError, error):
                    validate(rows)

    def test_original_disposition_and_conditional_chrome_are_preserved(self):
        by_id = {r['id']: r for r in self.rows}
        self.assertEqual(by_id['M-145']['merged_place'], 'Inspector')
        self.assertEqual(by_id['M-145']['merged_into'], '')
        self.assertEqual(by_id['L-34']['placement'], 'Control Room › Data and sources')
        self.assertEqual(by_id['M-057']['gravity_reason'], '→G2 gameday')
        self.assertIn('is ring 4', by_id['X-03']['ring_note'])


if __name__ == '__main__':
    unittest.main()
