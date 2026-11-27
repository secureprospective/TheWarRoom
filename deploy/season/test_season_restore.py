"""Execute the real remote payload offline; no network or real service calls."""
import hashlib
import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest

HERE = Path(__file__).resolve().parent


def _setup_env(root, versions, initial_version, initial_data, curl_body='exit 0'):
    app, data, backups = [root / n for n in ('app', 'data', 'backups')]
    for version in versions:
        (app / 'versions' / version).mkdir(parents=True)
    (app / 'current').symlink_to(app / 'versions' / initial_version)
    data.mkdir()
    (data / 'db').write_bytes(initial_data)
    for name, body in {
        'ssh': 'for arg do cmd="$arg"; done\nexec sh -c "$cmd"',
        'rc-service': 'echo "$*" >> "$SERVICE_LOG"',
        'curl': curl_body,
    }.items():
        stub = root / name
        stub.write_text('#!/bin/sh\n' + body + '\n')
        stub.chmod(0o755)
    env = dict(os.environ, PATH=str(root) + ':' + os.environ['PATH'],
               SEASON_APP_ROOT=str(app), SEASON_DATA_DIR=str(data),
               SEASON_BACKUP_DIR=str(backups), SERVICE_LOG=str(root / 'services'))
    return app, data, backups, env


class RestoreTest(unittest.TestCase):
    def test_swap_restore_and_refusals(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            original = bytes(range(256))
            app, data, backups, env = _setup_env(root, ('old', 'new'), 'old', original)

            def run(*args):
                return subprocess.run(['bash', str(HERE / 'season-swap.sh'), *args,
                                       '--christopher-ok'], env=env, capture_output=True,
                                      text=True, timeout=30, check=False)

            proc = run('new')
            self.assertEqual(proc.returncode, 0, proc.stderr)
            archive = next(backups.glob('*.tgz'))
            stamp = archive.name[len('twr-data-'):-4]
            record = Path(str(archive) + '.record')
            saved = archive.read_bytes()
            (data / 'db').write_bytes(b'changed by new build')

            def refused(message):
                before = (root / 'services').read_text()
                proc = run(stamp, '--rollback')
                self.assertNotEqual(proc.returncode, 0)
                self.assertIn(message, proc.stderr)
                self.assertEqual((data / 'db').read_bytes(), b'changed by new build')
                self.assertEqual((root / 'services').read_text(), before)
                self.assertEqual(os.readlink(app / 'current'), str(app / 'versions/new'))

            archive.unlink()
            refused('backup is missing or unreadable')
            archive.write_bytes(b'')
            refused('backup is missing or unreadable')
            archive.write_bytes(b'corrupt')
            refused('does not match the recorded swap')
            bad_record = record.read_text().splitlines()
            bad_record[2] = hashlib.sha256(b'corrupt').hexdigest()
            original_record = record.read_text()
            record.write_text('\n'.join(bad_record) + '\n')
            refused('backup is unreadable')
            record.write_text(original_record)
            archive.write_bytes(saved)
            saved_record = record.read_text()
            record.unlink()
            refused('no recorded swap')
            record.write_text(saved_record)
            (app / 'versions/old').rmdir()
            refused('previous version folder is gone')
            (app / 'versions/old').mkdir()
            (app / 'current').unlink()
            (app / 'current').symlink_to(app / 'versions/old')
            proc = run(stamp, '--rollback')
            self.assertIn('does not match the current swap', proc.stderr)
            (app / 'current').unlink()
            (app / 'current').symlink_to(app / 'versions/new')
            proc = run(stamp, '--rollback', '--dry-run')
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual((data / 'db').read_bytes(), b'changed by new build')
            proc = run(stamp, '--rollback')
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual((data / 'db').read_bytes(), original)
            self.assertEqual(os.readlink(app / 'current'), str(app / 'versions/old'))
            self.assertEqual((Path(str(data) + '.before-rollback-' + stamp) / 'db').read_bytes(),
                             b'changed by new build')
            self.assertEqual(archive.read_bytes(), saved)
            self.assertEqual((root / 'services').read_text().splitlines(),
                             ['twr stop', 'twr start', 'twr stop', 'twr start'])

    def test_ab_ba_ab_refuses_older_stamp_and_clears_on_rollback(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            app, data, _, env = _setup_env(root, ('A', 'B'), 'A', b'A1')

            def run(*args):
                return subprocess.run(['bash', str(HERE / 'season-swap.sh'), *args,
                                       '--christopher-ok'], env=env, capture_output=True,
                                      text=True, timeout=30, check=False)

            self.assertEqual(run('B').returncode, 0)
            stamp1 = (app / 'current-swap').read_text().strip()
            (data / 'db').write_bytes(b'B1')
            self.assertEqual(run('A').returncode, 0)
            (data / 'db').write_bytes(b'A2')
            self.assertEqual(run('B').returncode, 0)
            stamp2 = (app / 'current-swap').read_text().strip()
            self.assertNotEqual(stamp1, stamp2)
            (data / 'db').write_bytes(b'B2')

            before_services = (root / 'services').read_text()
            proc = run(stamp1, '--rollback')
            self.assertNotEqual(proc.returncode, 0)
            self.assertIn('does not match the active swap', proc.stderr)
            self.assertEqual((root / 'services').read_text(), before_services)
            self.assertEqual((data / 'db').read_bytes(), b'B2')
            self.assertEqual(os.readlink(app / 'current'), str(app / 'versions/B'))
            self.assertEqual((app / 'current-swap').read_text().strip(), stamp2)

            proc = run(stamp2, '--rollback')
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual((data / 'db').read_bytes(), b'A2')
            self.assertEqual(os.readlink(app / 'current'), str(app / 'versions/A'))
            self.assertFalse((app / 'current-swap').exists())

            before_services = (root / 'services').read_text()
            proc = run(stamp2, '--rollback')
            self.assertNotEqual(proc.returncode, 0)
            self.assertEqual((root / 'services').read_text(), before_services)

    def test_held_lock_refuses_second_run_through_health_check(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            waiting = root / 'curl_waiting'
            release = root / 'curl_release'
            curl_body = (
                f'touch "{waiting}"\n'
                f'i=0\n'
                f'while [ ! -f "{release}" ] && [ "$i" -lt 100 ]; do sleep 0.05; i=$((i+1)); done\n'
                f'exit 0'
            )
            app, data, _, env = _setup_env(root, ('A', 'B'), 'A', b'A1', curl_body=curl_body)
            first = subprocess.Popen(
                ['bash', str(HERE / 'season-swap.sh'), 'B', '--christopher-ok'],
                env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
            )
            try:
                deadline = time.monotonic() + 10
                while not waiting.exists():
                    self.assertLess(time.monotonic(), deadline, 'first run did not reach health check')
                    time.sleep(0.02)
                before_services = (root / 'services').read_text()
                second = subprocess.run(
                    ['bash', str(HERE / 'season-swap.sh'), 'A', '--christopher-ok'],
                    env=env, capture_output=True, text=True, timeout=10, check=False,
                )
                self.assertNotEqual(second.returncode, 0)
                self.assertIn('already running', second.stderr)
                self.assertEqual((root / 'services').read_text(), before_services)
                self.assertEqual((data / 'db').read_bytes(), b'A1')
                self.assertEqual(os.readlink(app / 'current'), str(app / 'versions/B'))
            finally:
                release.touch()
                out, err = first.communicate(timeout=15)
            self.assertEqual(first.returncode, 0, err + out)
            after = subprocess.run(
                ['bash', str(HERE / 'season-swap.sh'), 'A', '--christopher-ok'],
                env=env, capture_output=True, text=True, timeout=10, check=False,
            )
            self.assertEqual(after.returncode, 0, after.stderr)
