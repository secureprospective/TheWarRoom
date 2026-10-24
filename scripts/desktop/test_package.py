"""Packaging and real cross-process platform-lock regressions, without Wails or data."""

import hashlib
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import time
import unittest
import zipfile

import package


class DesktopTests(unittest.TestCase):
    def test_archive_contents_and_digest(self):
        base = package.ROOT / ".tmp/desktop"
        base.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=base) as directory:
            root = Path(directory)
            source = root / "thewarroom.exe"
            source.write_bytes(b"fixture executable")
            source.chmod(0o755)
            for name in ("windows-amd64", "linux-amd64"):
                artifact = package.archive(source, root / name, name)
                digest = hashlib.sha256(artifact.read_bytes()).hexdigest()
                self.assertEqual(artifact.with_suffix(artifact.suffix + ".sha256").read_text(),
                                 f"{digest}  {artifact.name}\n")
                if name.startswith("windows"):
                    with zipfile.ZipFile(artifact) as stream:
                        self.assertEqual(set(stream.namelist()), {"thewarroom.exe", "INSTALL.md"})
                        self.assertEqual(stream.read("thewarroom.exe"), source.read_bytes())
                else:
                    with tarfile.open(artifact) as stream:
                        self.assertEqual(set(stream.getnames()), {"thewarroom", "INSTALL.md"})
                        self.assertEqual(stream.extractfile("thewarroom").read(), source.read_bytes())
                        self.assertEqual(stream.getmember("thewarroom").mode & 0o111, 0o111)

    def test_real_platform_lock_contention_and_crash_release(self):
        base = package.ROOT / ".tmp/desktop"
        base.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=base) as directory:
            root = Path(directory)
            suffix = "windows" if package.platform.system() == "Windows" else "unix"
            source = package.ROOT / f"internal/composition/instance_lock_{suffix}.go"
            # Compile the existing implementation verbatim apart from its package name.
            (root / "lock.go").write_text(source.read_text().replace(
                "package composition", "package main", 1))
            shutil.copy2(Path(__file__).with_name("lock_main.go.txt"), root / "main.go")
            executable = root / "lock-probe.exe"
            subprocess.run(["go", "build", "-race", "-o", str(executable),
                            str(root / "lock.go"), str(root / "main.go")],
                           cwd=package.ROOT, check=True, timeout=120)
            database = root / "fixture.db"
            ready = root / "ready"

            def start():
                ready.unlink(missing_ok=True)
                process = subprocess.Popen([str(executable), str(database), str(ready)],
                                           stdin=subprocess.PIPE)
                self.addCleanup(lambda: process.poll() is None and process.kill())
                deadline = time.monotonic() + 15
                while not ready.exists():
                    if process.poll() is not None or time.monotonic() > deadline:
                        self.fail("lock holder failed to become ready")
                    time.sleep(0.02)
                return process

            for crash in (False, True):
                holder = start()
                rejected = subprocess.run([str(executable), str(database), str(root / "denied")],
                                          input=b"x", timeout=15)
                self.assertEqual(rejected.returncode, 1)
                self.assertFalse((root / "denied").exists())
                if crash:
                    holder.kill()
                    holder.wait(timeout=15)
                    holder.stdin.close()
                else:
                    holder.communicate(b"x", timeout=15)
                    self.assertEqual(holder.returncode, 0)
                successor = start()
                successor.communicate(b"x", timeout=15)
                self.assertEqual(successor.returncode, 0)
            self.assertFalse(database.exists(), "probe must not open a database")
            self.assertTrue(Path(str(database) + ".lock").exists(), "stale file must be safe")


if __name__ == "__main__":
    unittest.main(verbosity=2)
