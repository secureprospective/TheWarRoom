#!/usr/bin/env python3
"""Unit tests for scripts/load/gameday.py connection timing."""
import http.client
import http.server
import json
import os
import tempfile
import threading
import time
import types
import unittest

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


if __name__ == "__main__":
    unittest.main()
