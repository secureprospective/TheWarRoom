"""Offline regression tests: python3 -m unittest discover -s deploy/lab -v."""
import contextlib
import http.client
import http.server
import io
import json
import os
import sys
import tempfile
import threading
import time
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(__file__))
import stress
import stress_storm


class FakeServer(http.server.ThreadingHTTPServer):
    request_queue_size = 256
    daemon_threads = True


class StormTest(unittest.TestCase):
    def test_threshold_and_report(self):
        lock = threading.Lock()
        active = [0]
        tokens = set()

        class Handler(http.server.BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"

            def log_message(self, *_):
                pass

            def assert_observer_request(self, request):
                assert self.path == "/v1/native/workflow"
                assert self.headers["Authorization"] == "Bearer fake-observer"
                assert request["workflow"]["mode"] == "read"
                assert request["workflow"]["arguments"]["league"] == "native.fake"

            def do_POST(self):
                request = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                command = request.get("workflow", {}).get("command")
                data = {"native.directory.read": {"league": "native.fake", "rosterSeason": 2026},
                        "week.read": {"schedule": {"league": "native.fake", "franchise": "a", "season": 2026}, "current": {"week": 10}},
                        "results.read": {"games": [{"away": "a", "home": "b"}],
                                         "scores": [{"franchise": "a", "starters": [{"player": "001"}]}]},
                        "nfl.games.read": {"board": {"games": [{"status": "in", "id": "fake-game"}]}}}.get(command, {})
                reply = json.dumps({"data": data}).encode()
                with lock:
                    if command:
                        self.assert_observer_request(request)

                    tokens.add(self.headers.get("Authorization"))
                    active[0] += 1
                    status = 429 if active[0] > 20 else 200
                try:
                    time.sleep(.025)
                    self.send_response(status)
                    self.send_header("Content-Length", str(len(reply)))
                    self.end_headers()
                    self.wfile.write(reply)
                finally:
                    with lock:
                        active[0] -= 1

        with FakeServer(("127.0.0.1", 0), Handler) as server:
            thread = threading.Thread(target=server.serve_forever)
            thread.start()

            class LocalClient(stress.Client):
                def post(self, *args, **kwargs):
                    if self.conn is None:
                        self.conn = http.client.HTTPConnection("127.0.0.1", server.server_port, timeout=2)
                    return super().post(*args, **kwargs)

            try:
                state = {"league": "native.fake", "owners": [
                    {"bearers": ["fake-observer"], "franchise": "a"}, {"bearers": ["fake-load"], "franchise": "b"}]}
                with tempfile.TemporaryDirectory(dir=os.path.dirname(__file__)) as out, contextlib.redirect_stdout(io.StringIO()) as output:
                    result = stress_storm.storm("https://twr-lab-258.example.test", state, out,
                        max_step=64, step_seconds=.7, hold_seconds=.25, factory=LocalClient,
                        baseline_interval=.001, hold_interval=.05, recovery_seconds=.15, recovery_interval=.05)
                    with open(os.path.join(out, "storm.json")) as f:
                        report = json.load(f)
                    self.assertEqual([r["clients"] for r in report["steps"]], [16, 32])
                    self.assertLess(report["steps"][0]["refusal_rate"], .1)
                    self.assertGreaterEqual(report["steps"][1]["refusal_rate"], .1)
                    self.assertEqual(report["hold"]["clients"], 32)
                    self.assertEqual(report["stop_reason"], "refusals")
                    self.assertEqual(len(report["baseline"]), 10)
                    self.assertEqual(report["pulse_endpoint"], "POST /v1/native/workflow")
                    self.assertEqual(report["pulse_commands"], list(stress_storm.PULSE_COMMANDS))
                    for opened in report["baseline"] + report["recovery_opens"]:
                        self.assertEqual([r["command"] for r in opened["requests"]], list(stress_storm.PULSE_COMMANDS))
                        self.assertGreaterEqual(opened["ms"], sum(r["ms"] for r in opened["requests"]))
                    for field in ("requests_sent", "statuses", "connection_errors", "served_rps", "p50_ms", "p95_ms", "worst_ms", "refusal_rate"):
                        self.assertIn(field, report["steps"][1])
                    self.assertTrue(report["hold_opens"])
                    self.assertTrue(report["recovery_opens"])
                    self.assertEqual(report["verdicts"]["gate11"]["within_120_seconds"], True)
                    self.assertEqual(report["verdicts"]["gate9"]["unexpected_5xx"], 0)
                    self.assertNotIn("fake-load", json.dumps(report) + output.getvalue())
                    self.assertEqual(result["hold"]["clients"], 32)
                self.assertEqual(tokens, {"Bearer fake-observer", "Bearer fake-load"})
            finally:
                server.shutdown()
                thread.join()

    def test_pulse_failure_cannot_count_as_success(self):
        client = mock.Mock(error=None)
        client.post.return_value = (503, b"refused", 4, None)
        opened = stress_storm.pulse(client, {"bearers": ["fake"], "franchise": "a"}, "native.fake", time.monotonic())
        self.assertEqual(opened["status"], 503)
        self.assertEqual(len(opened["requests"]), 1)
        client.post.assert_called_once()
        client.post.return_value = (200, b'{"data":{"rosterSeason":"bad"}}', 4, None)
        opened = stress_storm.pulse(client, {"bearers": ["fake"], "franchise": "a"}, "native.fake", time.monotonic())
        self.assertEqual(opened["status"], 0)
        self.assertEqual(opened["error"], "InvalidPulseResponse")

    def test_pulse_empty_week_and_scoreboard_skip_conditional_reads(self):
        client = mock.Mock(error=None)
        data = [{"league": "native.fake", "rosterSeason": 2026},
                {"schedule": {"league": "native.fake", "franchise": "a", "season": 2026}},
                {}, {}, {}, {"board": {"games": []}}]
        client.post.side_effect = [(200, json.dumps({"data": row}).encode(), 1, None) for row in data]
        opened = stress_storm.pulse(client, {"bearers": ["fake"], "franchise": "a"}, "native.fake", time.monotonic())
        self.assertEqual(opened["status"], 200)
        self.assertEqual([r["command"] for r in opened["requests"]],
                         ["native.directory.read", "week.read", "standings.read", "bracket.read", "odds.read", "nfl.games.read"])
        week_args = client.post.call_args_list[1].args[1]["workflow"]["arguments"]
        self.assertEqual(week_args, {"league": "native.fake", "franchise": "a", "season": 2026, "localZone": "UTC", "at": ""})
        client.post.side_effect = [(200, json.dumps({"data": data[0]}).encode(), 1, None), (429, b"", 1, None)]
        opened = stress_storm.pulse(client, {"bearers": ["fake"], "franchise": "a"}, "native.fake", time.monotonic())
        self.assertEqual(opened["status"], 429)
        self.assertEqual([r["status"] for r in opened["requests"]], [200, 429])

    def test_guard_before_io(self):
        with mock.patch.object(stress_storm, "Client") as client:
            for base in ("https://production.example", "http://twr-lab-258.example", "https://twr-lab-258.example@production.example",
                         "https://twr-lab-258.example:8443", "https://twr-lab-258.example/?code=secret"):
                with self.assertRaises(ValueError):
                    stress_storm.storm(base, {}, "unused")
            client.assert_not_called()

    def test_errors_not_retried_and_measured(self):
        c = stress.Client("https://twr-lab-258.example")
        c.retry = False
        connection = mock.Mock()
        connection.request.side_effect = ConnectionResetError()
        c.conn = connection
        status, body, ms, _ = c.post("/v1/native/ledger", {})
        self.assertEqual(status, 0)
        self.assertEqual(c.error, "ConnectionResetError")
        self.assertGreaterEqual(ms, 0)
        connection.request.assert_called_once()
        stats = stress_storm.Metrics()
        stats.add(status, ms, c.error)
        self.assertEqual(stats.summary(1)["connection_errors"], {"ConnectionResetError": 1})


if __name__ == "__main__":
    unittest.main()
