"""Offline checks for explicit release read exclusions."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("stress", Path(__file__).with_name("stress.py"))
stress = importlib.util.module_from_spec(spec)
spec.loader.exec_module(stress)


class SkipReadsTest(unittest.TestCase):
    def test_default_and_unknown(self):
        self.assertEqual(stress.selected_screens(), stress.SCREENS)
        with self.assertRaisesRegex(ValueError, "unknown skipped reads"):
            stress.selected_screens(("typo.read",))

    def test_probe_reports_skip_and_calls_other_fifteen(self):
        state = {"league": "test", "owners": [{"franchise": "1", "bearers": ["test"]}]}
        calls = []

        def workflow(client, bearer, name, mode, args):
            calls.append(name)
            return 200, b"{}", 1, None

        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            path.write_text(json.dumps(state))
            output = io.StringIO()
            with patch.object(stress, "season_of", return_value=(2026, {})), patch.object(
                stress, "wf", side_effect=workflow
            ), patch.object(stress.Client, "post", return_value=(200, b"{}", 1, None)), contextlib.redirect_stdout(output):
                stress.probe("https://example.invalid", path, ("value.read",))
            self.assertNotIn("value.read", calls)
            self.assertEqual(len(calls), 14)  # plus the ledger POST
            self.assertEqual(output.getvalue().count(" 200 "), 15)
            self.assertIn("value.read               SKIPPED (not in this release)", output.getvalue())

    def test_run_filters_and_hammer_skips_without_request(self):
        with tempfile.TemporaryDirectory() as directory, contextlib.redirect_stdout(io.StringIO()) as output:
            run = stress.Run("https://example.invalid", {"league": "test"}, directory, ("value.read",))
            try:
                self.assertEqual(len(run.screens), 15)
                self.assertNotIn("value.read", dict(run.screens))
            finally:
                for file in (run.lat, run.confirmed, run.events):
                    file.close()
            with patch.object(stress, "Client") as client:
                stress.hammer("https://example.invalid", "not-opened", "value.read", 1, ("value.read",))
                client.assert_not_called()
            self.assertEqual(output.getvalue().count("SKIPPED (not in this release)"), 2)


if __name__ == "__main__":
    unittest.main()
