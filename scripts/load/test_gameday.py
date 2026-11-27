#!/usr/bin/env python3
"""Offline tests for game-day load coverage and connection timing."""
import http.client
import http.server
import io
import json
import os
import sys
import tempfile
import threading
import time
import types
import unittest
from contextlib import redirect_stdout
from unittest.mock import Mock, patch

import gameday


class _KeepAliveHandler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        if length:
            self.rfile.read(length)
        payload = json.dumps({"data": {"season": 2026, "current": {"week": 7}}}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, format, *args):
        return


class TestConnectTiming(unittest.TestCase):
    def test_parse_server_timing(self):
        self.assertEqual(gameday.parse_server_timing("app;dur=12.3"), 12.3)
        self.assertIsNone(gameday.parse_server_timing(None))
        self.assertIsNone(gameday.parse_server_timing("invalid"))
        self.assertEqual(gameday.parse_server_timing('cfL4;desc="?proto=TCP", app;dur=7.5'), 7.5)
        self.assertIsNone(gameday.parse_server_timing("myapp;dur=9"))

    def setUp(self):
        self.server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), _KeepAliveHandler)
        self.port = self.server.server_address[1]
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.tmpdir = tempfile.TemporaryDirectory()

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)
        self.tmpdir.cleanup()

    def _make_runner(self):
        args = types.SimpleNamespace(
            target=f"http://127.0.0.1:{self.port}",
            cadence_sec=15.0,
            burst_sec=0.0,
            gzip=False,
            server_pid=0,
            log_dir=self.tmpdir.name,
        )
        state = {"league": "test-league", "owners": []}
        return gameday.LoadRunner(args, state, ssl_ctx=None)

    def test_first_request_excludes_connect_and_records_connect_ms(self):
        runner = self._make_runner()
        client = gameday.Client(runner.base, ssl_ctx=None, use_gzip=False)

        orig_new_conn = client._new_conn

        def slow_connect_new_conn(timeout):
            conn = orig_new_conn(timeout)
            orig_connect = conn.connect

            def delayed_connect():
                time.sleep(0.12)
                return orig_connect()

            conn.connect = delayed_connect
            return conn

        client._new_conn = slow_connect_new_conn
        secret_bearer = "secret-bearer-token-do-not-log"
        try:
            ctx = runner.discover_context(client, secret_bearer, "0001", 1)
        finally:
            client.close()
            if runner.log_fp is not None:
                runner.log_fp.close()

        self.assertEqual(ctx, {"season": 2026, "week": 7})
        self.assertEqual(len(runner.latencies), 2)
        self.assertLess(runner.latencies[0], 100.0)

        log_path = os.path.join(self.tmpdir.name, "latency.jsonl")
        with open(log_path, "r") as f:
            raw_log = f.read()
        self.assertNotIn(secret_bearer, raw_log)

        records = [json.loads(line) for line in raw_log.splitlines() if line.strip()]
        self.assertEqual(len(records), 2)
        first, second = records
        self.assertEqual(first["op"], "native.directory.read")
        self.assertIn("connect_ms", first)
        self.assertGreaterEqual(first["connect_ms"], 100.0)
        self.assertLess(first["ms"], 100.0)

        self.assertEqual(second["op"], "week.read")
        self.assertNotIn("connect_ms", second)

    def test_real_http_keepalive_records_connect_ms_once(self):
        runner = self._make_runner()
        client = gameday.Client(runner.base, ssl_ctx=None, use_gzip=False)
        try:
            runner.read_op(client, "b", "0001", 1, "native.directory.read", {"league": "L"})
            runner.read_op(client, "b", "0001", 1, "nfl.games.read", {"league": "L"})
        finally:
            client.close()
            if runner.log_fp is not None:
                runner.log_fp.close()

        log_path = os.path.join(self.tmpdir.name, "latency.jsonl")
        with open(log_path, "r") as f:
            records = [json.loads(line) for line in f if line.strip()]
        self.assertEqual(len(records), 2)
        self.assertIn("connect_ms", records[0])
        self.assertGreater(records[0]["connect_ms"], 0.0)
        self.assertNotIn("connect_ms", records[1])


class TestLoadCoverage(unittest.TestCase):
    def _run_main(self, teams, devices_per_team, state, summary_line="PASS: load completed"):
        runner = Mock()
        runner.summary_line.return_value = summary_line
        output = io.StringIO()
        args = [
            "gameday.py",
            "--join-file",
            "unused",
            "--teams",
            str(teams),
            "--devices-per-team",
            str(devices_per_team),
            "--state-file",
            "/dev/shm/twr-gameday-test.json",
        ]
        with (
            patch.object(sys, "argv", args),
            patch.object(gameday, "make_ssl_context", return_value=None),
            patch.object(gameday, "onboard", return_value=state),
            patch.object(gameday, "LoadRunner", return_value=runner) as load_runner,
            patch.object(gameday.os.path, "exists", return_value=True),
            patch.object(gameday.os, "remove") as remove,
            redirect_stdout(output),
        ):
            with self.assertRaises(SystemExit) as exit_error:
                gameday.main()
        return exit_error.exception.code, output.getvalue(), runner, load_runner, remove

    def test_main_rejects_fewer_owners_before_starting_workers(self):
        state = {
            "league": "L",
            "owners": [{"franchise": "f1", "bearers": ["secret-bearer"]}],
        }
        code, output, runner, load_runner, remove = self._run_main(2, 1, state)

        self.assertEqual(code, 1)
        self.assertIn("FAIL: owners actual=1 expected=2", output)
        self.assertIn("keeping state file /dev/shm/twr-gameday-test.json", output)
        self.assertIn("seats were claimed", output)
        self.assertNotIn("secret-bearer", output)
        remove.assert_not_called()
        load_runner.assert_not_called()
        runner.run.assert_not_called()

    def test_main_rejects_too_few_devices_per_team_before_workers(self):
        state = {
            "league": "L",
            "owners": [
                {"franchise": "f1", "bearers": ["secret-bearer-1", "secret-bearer-2"]},
                {"franchise": "f2", "bearers": ["secret-bearer-3"]},
            ],
        }
        code, output, runner, load_runner, remove = self._run_main(2, 2, state)

        self.assertEqual(code, 1)
        self.assertIn("FAIL: team=2 devices actual=1 expected=2", output)
        self.assertIn("keeping state file /dev/shm/twr-gameday-test.json", output)
        self.assertIn("seats were claimed", output)
        self.assertNotIn("secret-bearer", output)
        remove.assert_not_called()
        load_runner.assert_not_called()
        runner.run.assert_not_called()

    def test_main_runs_when_explicit_smaller_load_has_complete_coverage(self):
        state = {
            "league": "L",
            "owners": [{"franchise": "f1", "bearers": ["secret-bearer"]}],
        }
        code, output, runner, load_runner, remove = self._run_main(1, 1, state)

        self.assertEqual(code, 0)
        self.assertIn("PASS: load completed", output)
        self.assertNotIn("keeping state file", output)
        remove.assert_called_once_with("/dev/shm/twr-gameday-test.json")
        load_runner.assert_called_once()
        runner.run.assert_called_once()

    def test_main_keeps_state_file_on_load_fail(self):
        state = {
            "league": "L",
            "owners": [{"franchise": "f1", "bearers": ["secret-bearer"]}],
        }
        code, output, runner, load_runner, remove = self._run_main(
            1, 1, state, summary_line="FAIL: 1 x 502, 0 x 5xx, p95 10 ms, max RSS n/a MiB"
        )

        self.assertEqual(code, 1)
        self.assertIn("FAIL: 1 x 502", output)
        self.assertIn("keeping state file /dev/shm/twr-gameday-test.json", output)
        remove.assert_not_called()
        load_runner.assert_called_once()
        runner.run.assert_called_once()

    def test_main_deletes_state_file_when_zero_seats_claimed(self):
        state = {"league": "L", "owners": []}
        code, output, runner, load_runner, remove = self._run_main(1, 1, state)

        self.assertEqual(code, 1)
        self.assertIn("FAIL: owners actual=0 expected=1", output)
        self.assertNotIn("keeping state file", output)
        remove.assert_called_once_with("/dev/shm/twr-gameday-test.json")
        load_runner.assert_not_called()
        runner.run.assert_not_called()

    def _new_shm_state_file(self):
        fd, path = tempfile.mkstemp(dir="/dev/shm", prefix="twr-gameday-test-", suffix=".json")
        os.close(fd)
        os.remove(path)
        self.addCleanup(lambda: os.path.exists(path) and os.remove(path))
        return path

    def test_partial_onboard_coverage_error_keeps_real_state_file(self):
        state_path = self._new_shm_state_file()
        franchises = {"franchises": [{"id": "f1", "taken": False}, {"id": "f2", "taken": False}]}
        responses = iter([
            (200, json.dumps(franchises).encode(), 1.0),
            (201, b"{}", 1.0),
            (500, b"{}", 1.0),
            (200, json.dumps({"bearer": "secret-bearer-token"}).encode(), 1.0),
        ])
        client = Mock()
        output = io.StringIO()
        args = [
            "gameday.py",
            "--join-file",
            "unused",
            "--teams",
            "2",
            "--devices-per-team",
            "1",
            "--state-file",
            state_path,
        ]
        with (
            patch.object(sys, "argv", args),
            patch.object(gameday, "make_ssl_context", return_value=None),
            patch.object(gameday, "parse_join_file", return_value=("L", "code")),
            patch.object(gameday, "Client", return_value=client),
            patch.object(gameday, "auth_post", side_effect=lambda *_: next(responses)),
            redirect_stdout(output),
        ):
            with self.assertRaises(SystemExit) as exit_error:
                gameday.main()

        self.assertEqual(exit_error.exception.code, 1)
        out = output.getvalue()
        self.assertIn("FAIL: owners actual=1 expected=2", out)
        self.assertIn(f"keeping state file {state_path}", out)
        self.assertNotIn("secret-bearer-token", out)
        self.assertTrue(os.path.exists(state_path))
        with open(state_path, "r") as f:
            saved = json.load(f)
        self.assertEqual(len(saved["owners"]), 1)
        self.assertEqual(saved["owners"][0]["franchise"], "f1")
        self.assertEqual(saved["owners"][0]["bearers"], ["secret-bearer-token"])
        self.assertNotIn(saved["owners"][0]["password"], out)

    def test_partial_onboard_exception_keeps_real_state_file(self):
        state_path = self._new_shm_state_file()
        franchises = {"franchises": [{"id": "f1", "taken": False}, {"id": "f2", "taken": False}]}

        def fake_auth_post(_client, _pacer, path, _body):
            if path == "/v1/auth/join/franchises":
                return (200, json.dumps(franchises).encode(), 1.0)
            if path == "/v1/auth/join/claim":
                if not hasattr(fake_auth_post, "claimed"):
                    fake_auth_post.claimed = True
                    return (201, b"{}", 1.0)
                raise RuntimeError("boom during second claim")
            raise AssertionError(f"unexpected path {path}")

        client = Mock()
        output = io.StringIO()
        args = [
            "gameday.py",
            "--join-file",
            "unused",
            "--teams",
            "2",
            "--devices-per-team",
            "1",
            "--state-file",
            state_path,
        ]
        with (
            patch.object(sys, "argv", args),
            patch.object(gameday, "make_ssl_context", return_value=None),
            patch.object(gameday, "parse_join_file", return_value=("L", "code")),
            patch.object(gameday, "Client", return_value=client),
            patch.object(gameday, "auth_post", side_effect=fake_auth_post),
            redirect_stdout(output),
        ):
            with self.assertRaisesRegex(RuntimeError, "boom during second claim"):
                gameday.main()

        out = output.getvalue()
        self.assertIn(f"keeping state file {state_path}", out)
        self.assertTrue(os.path.exists(state_path))
        with open(state_path, "r") as f:
            saved = json.load(f)
        self.assertEqual(len(saved["owners"]), 1)
        self.assertEqual(saved["owners"][0]["franchise"], "f1")

    def test_onboard_rejects_short_seat_list_before_claims(self):
        client = Mock()
        franchises = {"franchises": [{"id": "f1", "taken": False}]}
        with (
            patch.object(gameday, "parse_join_file", return_value=("L", "code")),
            patch.object(gameday, "Client", return_value=client),
            patch.object(
                gameday,
                "auth_post",
                return_value=(200, json.dumps(franchises).encode(), 1.0),
            ) as auth_post,
        ):
            with self.assertRaisesRegex(
                gameday.CoverageError, "available owners actual=1 expected=2"
            ):
                gameday.onboard("https://example.invalid", None, "unused", "unused", 2, 1, False)

        auth_post.assert_called_once()
        client.close.assert_called_once()


if __name__ == "__main__":
    unittest.main()
