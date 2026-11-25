"""Offline tests for season-swap.sh: python3 -m unittest discover -s deploy/season -v

A fake `ssh` on PATH logs every call instead of connecting, so no network is used.
"""
import os
import shutil
import subprocess
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPT = os.path.join(HERE, "season-swap.sh")

FAKE_SSH = """#!/bin/sh
printf "%s\\n" "call" >> "$FAKE_SSH_LOG"
case "$*" in *readlink*) echo /opt/twr/versions/v50 ;; esac
exit 0
"""


class SeasonSwapTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.log = os.path.join(self.tmp, "ssh.log")
        fake = os.path.join(self.tmp, "ssh")
        with open(fake, "w", encoding="utf-8") as f:
            f.write(FAKE_SSH)
        os.chmod(fake, 0o755)

    def tearDown(self):
        shutil.rmtree(self.tmp, ignore_errors=True)

    def run_swap(self, now, *args):
        env = dict(os.environ)
        env["PATH"] = self.tmp + os.pathsep + env.get("PATH", "")
        env["FAKE_SSH_LOG"] = self.log
        env.pop("SEASON_NOW", None)
        if now is not None:
            env["SEASON_NOW"] = now
        proc = subprocess.run(
            ["bash", SCRIPT, "v51", *args],
            env=env, capture_output=True, text=True, timeout=60, check=False,
        )
        return proc

    def ssh_calls(self):
        if not os.path.exists(self.log):
            return []
        with open(self.log, encoding="utf-8") as f:
            return f.read().splitlines()

    def test_a_sunday_without_ok_is_refused_before_any_step(self):
        proc = self.run_swap("2026-11-29T14:00")
        self.assertEqual(proc.returncode, 3, proc.stdout + proc.stderr)
        self.assertIn("Season server swaps need Christopher's OK", proc.stderr)
        self.assertIn("--christopher-ok", proc.stdout)
        self.assertNotIn("== 1/6", proc.stdout)
        self.assertEqual(self.ssh_calls(), [], "no ssh call may run after a refused guard")

    def test_a2_saturday_1759_is_refused_and_1800_is_allowed(self):
        self.assertEqual(self.run_swap("2026-11-28T17:59", "--dry-run").returncode, 3)
        self.assertEqual(self.run_swap("2026-11-28T18:00", "--dry-run").returncode, 0)

    def test_b_saturday_evening_dry_run_prints_steps_and_command(self):
        proc = self.run_swap("2026-11-28T19:00", "--dry-run")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        for n in range(1, 7):
            self.assertIn(f"== {n}/6", proc.stdout)
        self.assertIn("--christopher-ok", proc.stdout)
        self.assertIn("Rollback command", proc.stdout)
        self.assertEqual(self.ssh_calls(), [], "dry run must not contact the server")

    def test_c_tuesday_with_ok_dry_run_passes(self):
        proc = self.run_swap("2026-11-24T10:00", "--christopher-ok", "--dry-run")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertEqual(self.ssh_calls(), [], "dry run must not contact the server")

    def test_d0_saturday_real_run_without_ok_is_refused(self):
        proc = self.run_swap("2026-11-28T19:00")
        self.assertEqual(proc.returncode, 3, proc.stdout + proc.stderr)
        self.assertEqual(self.ssh_calls(), [], "a refused run must not contact the server")

    def test_d_real_run_sends_six_steps_and_prints_rollback_to_previous(self):
        proc = self.run_swap("2026-11-28T19:00", "--christopher-ok")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        calls = self.ssh_calls()
        self.assertEqual(len(calls), 1, calls)
        self.assertIn("--rollback --christopher-ok", proc.stdout)

    def test_e_bad_version_name_is_refused_with_no_ssh(self):
        env_now = "2026-11-28T19:00"
        env = dict(os.environ)
        env["PATH"] = self.tmp + os.pathsep + env.get("PATH", "")
        env["FAKE_SSH_LOG"] = self.log
        env["SEASON_NOW"] = env_now
        proc = subprocess.run(
            ["bash", SCRIPT, "../etc", "--dry-run"],
            env=env, capture_output=True, text=True, timeout=60, check=False,
        )
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertEqual(self.ssh_calls(), [])


if __name__ == "__main__":
    unittest.main()
