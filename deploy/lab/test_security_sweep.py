"""No pct, curl or external traffic: exercise security decisions with command fakes."""
import argparse
import contextlib
import importlib.util
import io
import json
import os
import subprocess
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location("security_sweep", os.path.join(os.path.dirname(__file__), "security-sweep.py"))
sweep = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sweep)


class SweepTest(unittest.TestCase):
    def make_sweep(self, state=None):
        args = argparse.Namespace(ct="258", host="twr-lab-258.example.test", with_reboot=False, lan_peer=None, v6_peer=None)
        return sweep.Sweep(args, state or {})

    def test_protected_and_mismatched_targets_before_commands(self):
        with mock.patch.object(sweep, "run") as run, contextlib.redirect_stderr(io.StringIO()):
            for ct in ("110", "256", "257", "999", "258;id"):
                self.assertEqual(sweep.main([ct, f"twr-lab-{ct}.example.test"]), 2)
            self.assertEqual(sweep.main(["258", "twr-lab-259.example.test"]), 2)
            run.assert_not_called()
        sweep.validate_target("258", "twr-lab-258.example.test")

    def test_curl_credentials_are_stdin_only(self):
        response = subprocess.CompletedProcess([], 0, "HTTP/2 200\ncontent-type: application/json\n\n{}\nSWEEP_STATUS:200", "")
        with mock.patch.object(sweep, "run", return_value=response) as run:
            result = self.make_sweep().curl("/v1/auth/login", {"password": "fake-password"}, "fake-bearer", recent="fake-proof")
            self.assertEqual(result, (200, "HTTP/2 200\ncontent-type: application/json", "{}", 0))
            args, config, _ = run.call_args.args
            for secret in ("fake-password", "fake-bearer", "fake-proof"):
                self.assertNotIn(secret, ' '.join(args))
                self.assertIn(secret, config)
            self.assertNotIn("--location", args)

    def test_secret_count_uses_stdin_and_never_matching_lines(self):
        state = {"owners": [{"password": "fake-password", "bearers": ["fake-bearer"]}],
                 "recovery_codes": ["fake-code"], "private_key": "fake-key-line1\nfake-key-line2"}
        self.assertEqual(set(sweep.known_secrets(state)), {"fake-password", "fake-bearer", "fake-code", "fake-key-line1", "fake-key-line2"})
        s = self.make_sweep(state)
        def execute(*argv, **kwargs):
            if argv[0] == "grep":
                self.assertEqual(argv, ("grep", "-c", "-F", "-f", "-", "/var/lib/twr/server.log"))
                self.assertIn("fake-password", kwargs["data"])
                return subprocess.CompletedProcess([], 0, "1\n", "")
            if argv[0] == "stat":
                owners = {"/etc/cloudflared/credentials.json": "root 640", "/var/lib/twr": "twr 700"}
                return subprocess.CompletedProcess([], 0, argv[-1] + " " + owners.get(argv[-1], "twr 600") + '\n', "")
            if argv[0] == "ss":
                return subprocess.CompletedProcess([], 0, "tcp LISTEN 0 128 127.0.0.1:8443 0.0.0.0:*\ntcp LISTEN 0 128 127.0.0.1:20241 0.0.0.0:*\n", "")
            return subprocess.CompletedProcess([], 0, "", "")
        with mock.patch.object(s, "x", side_effect=execute), contextlib.redirect_stdout(io.StringIO()) as out:
            s.console()
        self.assertIn("06 FAIL", out.getvalue())
        self.assertIn("01 PASS", out.getvalue())
        self.assertIn("03 PASS", out.getvalue())
        self.assertIn("05 MANUAL", out.getvalue())
        self.assertNotIn("fake-password", out.getvalue())
        self.assertNotIn("fake-bearer", out.getvalue())

    def test_peer_validation(self):
        self.assertEqual(sweep.peer_address("192.168.1.5:443"), ("192.168.1.5", "443"))
        self.assertEqual(sweep.peer_address("[2001:db8::1]:443"), ("2001:db8::1", "443"))
        for bad in ("--help:80", "host:0", "host:65536", "host;id:80", "[invalid]:443", "[::::]:443"):
            with self.assertRaises(argparse.ArgumentTypeError):
                sweep.peer_address(bad)

    def test_egress_manual_without_lan_peer(self):
        s = self.make_sweep()
        with mock.patch.object(s, "x") as execute, mock.patch.object(sweep, "run") as run, contextlib.redirect_stdout(io.StringIO()) as out:
            s.egress()
        self.assertIn("04 MANUAL", out.getvalue())
        execute.assert_not_called()
        run.assert_not_called()

    def test_egress_controls_and_failures(self):
        # An open container connection, dead control, missing utility, route or failed egress must fail.
        cases = [(False, 0, 1, "", 0, 0, "PASS"), (True, 0, 1, "", 0, 0, "PASS"),
                 (False, 0, 0, "", 0, 0, "FAIL"), (True, 0, 0, "", 0, 0, "FAIL"),
                 (False, 1, 1, "", 0, 0, "FAIL"), (False, 0, 127, "", 0, 0, "FAIL"),
                 (False, 0, 1, "default via 2001:db8::1", 0, 0, "FAIL"),
                 (False, 0, 1, "", 1, 0, "FAIL"), (False, 0, 1, "", 0, 22, "FAIL")]
        for v6, control, blocked, routes, dns, https, verdict in cases:
            with self.subTest(v6=v6, control=control, blocked=blocked, routes=routes, dns=dns, https=https):
                s = self.make_sweep()
                s.args.lan_peer = ("192.168.1.5", "443")
                s.args.v6_peer = ("2001:db8::5", "443") if v6 else None
                def execute(*argv, **kwargs):
                    if argv[0] == "nc":
                        code, text = blocked, ""
                    elif argv[0] == "ip":
                        self.assertEqual(argv, ("ip", "-6", "route", "show", "table", "all", "scope", "global"))
                        code, text = 0, routes
                    else:
                        code, text = (dns if argv[0] == "nslookup" else https), ""
                    return subprocess.CompletedProcess([], code, text, "")
                with mock.patch.object(s, "x", side_effect=execute) as x, mock.patch.object(sweep, "run", return_value=subprocess.CompletedProcess([], control, "", "")), contextlib.redirect_stdout(io.StringIO()) as out:
                    s.egress()
                self.assertIn("04 " + verdict, out.getvalue())
                self.assertEqual(s.failed, verdict == "FAIL")
                self.assertTrue(any(call.args[:3] == ("nslookup", "example.com", "1.1.1.1") for call in x.call_args_list))
                self.assertTrue(any("--head" in call.args and "https://example.com" in call.args for call in x.call_args_list))
                if not v6:
                    self.assertIn("global IPv6 route absent=", out.getvalue())

    def test_egress_route_utility_failure_is_not_route_absence(self):
        s = self.make_sweep()
        s.args.lan_peer = ("192.168.1.5", "443")
        def execute(*argv, **kwargs):
            code = {"nc": 1, "ip": 127}.get(argv[0], 0)
            return subprocess.CompletedProcess([], code, "", "")
        with mock.patch.object(s, "x", side_effect=execute), mock.patch.object(sweep, "run", return_value=subprocess.CompletedProcess([], 0, "", "")), contextlib.redirect_stdout(io.StringIO()) as out:
            s.egress()
        self.assertIn("04 FAIL", out.getvalue())
        self.assertIn("global IPv6 route absent=False", out.getvalue())

    def test_roles_refused_and_unchanged(self):
        owner = {"bearers": ["fake-owner"], "principal": "acct.fake"}
        state = {"league": "native.fake", "owners": [owner], "co_commissioner": {"bearers": ["fake-deputy"], "recent_auth": "fake-proof"}}
        s = self.make_sweep(state)
        def request(path, body, token, **kwargs):
            if path == "/v1/native/people":
                data = {"revision": 7, "yourRole": "Owner" if token == "fake-owner" else "Co-commissioner", "mayGrantRoles": False}
            elif path == "/v1/native/ledger":
                data = {"revision": 7, "entries": []}
            else:
                self.assertEqual(body["command"]["command"], "commish.role.grant")
                self.assertEqual(body["command"]["expected"], 7)
                return (403 if path.endswith("preview") else 409), "", "", 0
            return 200, "", json.dumps(data), 0
        with mock.patch.object(s, "curl", side_effect=request), contextlib.redirect_stdout(io.StringIO()) as out:
            s.roles(owner)
        self.assertIn("12 PASS", out.getvalue())
        self.assertFalse(s.failed)

    def test_roles_changed_is_failure(self):
        owner = {"bearers": ["fake-owner"], "principal": "acct.fake"}
        s = self.make_sweep({"league": "native.fake", "co_commissioner": {"bearers": ["fake-deputy"]}})
        responses = [(200, "", '{"revision":7,"yourRole":"Owner","mayGrantRoles":false}', 0),
                     (200, "", '{"revision":7}', 0), (403, "", "", 0), (409, "", "", 0),
                     (200, "", '{"revision":8}', 0), (200, "", '{"revision":8}', 0),
                     (200, "", '{"revision":8,"yourRole":"Co-commissioner","mayGrantRoles":false}', 0),
                     (200, "", '{"revision":8}', 0), (403, "", "", 0), (409, "", "", 0),
                     (200, "", '{"revision":8,"yourRole":"Co-commissioner","mayGrantRoles":false}', 0), (200, "", '{"revision":8}', 0)]
        with mock.patch.object(s, "curl", side_effect=responses), contextlib.redirect_stdout(io.StringIO()) as out:
            s.roles(owner)
        self.assertIn("12 FAIL", out.getvalue())

    def test_throttle_wait_and_correct_signin(self):
        s = self.make_sweep()
        owner = {"user": "fake-user", "password": "fake-password", "bearers": ["fake-bearer"]}
        replies = [(401, "", "", 0)] * 10 + [(429, "Retry-After: 60", "", 0)] * 2 + [(200, "", "", 0)]
        with mock.patch.object(s, "curl", side_effect=replies) as curl, mock.patch.object(sweep.time, "sleep") as sleep, contextlib.redirect_stdout(io.StringIO()) as out:
            s.throttle(owner)
            sleep.assert_called_once_with(61)
            self.assertEqual(curl.call_count, 13)
        self.assertIn("10 PASS", out.getvalue())
        self.assertNotIn("fake-password", out.getvalue())


if __name__ == "__main__":
    unittest.main()
