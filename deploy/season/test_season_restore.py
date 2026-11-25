"""Execute the real remote payload offline; no network or real service calls."""
import hashlib
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

HERE = Path(__file__).resolve().parent


class RestoreTest(unittest.TestCase):
    def test_swap_restore_and_refusals(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            app, data, backups = [root / n for n in ('app', 'data', 'backups')]
            for version in ('old', 'new'):
                (app / 'versions' / version).mkdir(parents=True)
            (app / 'current').symlink_to(app / 'versions/old')
            data.mkdir()
            original = bytes(range(256))
            (data / 'db').write_bytes(original)
            for name, body in {
                'ssh': 'for arg do cmd="$arg"; done\nexec sh -c "$cmd"',
                'rc-service': 'echo "$*" >> "$SERVICE_LOG"',
                'curl': 'exit 0',
            }.items():
                stub = root / name
                stub.write_text('#!/bin/sh\n' + body + '\n')
                stub.chmod(0o755)
            env = dict(os.environ, PATH=str(root) + ':' + os.environ['PATH'],
                       SEASON_APP_ROOT=str(app), SEASON_DATA_DIR=str(data),
                       SEASON_BACKUP_DIR=str(backups), SERVICE_LOG=str(root / 'services'))

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
