"""Packaging and real cross-process platform-lock regressions, without Wails or data."""

import hashlib
import io
from pathlib import Path
import shutil
import subprocess
import tempfile
import tarfile
import time
import unittest

import package


class DesktopTests(unittest.TestCase):
    def test_release_target_names(self):
        self.assertEqual(package.artifact_name("windows-amd64"), "warroom-windows-amd64.exe")
        self.assertEqual(package.artifact_name("darwin-arm64"), "warroom-darwin-arm64.dmg")
        self.assertEqual(package.artifact_name("linux-amd64"), "warroom-linux-amd64.deb")
        with self.assertRaises(KeyError):
            package.artifact_name("linux-arm64")

    @unittest.skipUnless(package.platform.system() == "Linux", "native Debian tooling")
    def test_debian_install_contract_and_digest(self):
        base = package.ROOT / ".tmp/desktop"
        base.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=base) as directory:
            root = Path(directory)
            source = root / "thewarroom"
            source.write_bytes(b"fixture executable")
            artifact = package.package(source, root / "artifacts", "linux-amd64")
            digest = hashlib.sha256(artifact.read_bytes()).hexdigest()
            self.assertEqual(artifact.with_suffix(".deb.sha256").read_text(),
                             f"{digest}  {artifact.name}\n")
            depends = subprocess.check_output(["dpkg-deb", "-f", str(artifact), "Depends"], text=True)
            self.assertIn("libwebkit2gtk-4.1-0", depends)
            self.assertIn("libgtk-3-0t64", depends)
            payload = subprocess.check_output(["dpkg-deb", "--fsys-tarfile", str(artifact)])
            with tarfile.open(fileobj=io.BytesIO(payload)) as stream:
                self.assertEqual(stream.getmember(".").mode, 0o755)
                self.assertEqual(stream.getmember(".").uid, 0)
            unpacked = root / "unpacked"
            subprocess.run(["dpkg-deb", "-x", str(artifact), str(unpacked)], check=True)
            binary = unpacked / "usr/bin/thewarroom"
            self.assertEqual(binary.read_bytes(), source.read_bytes())
            self.assertEqual(binary.stat().st_mode & 0o777, 0o755)
            launcher = (unpacked / "usr/share/applications/thewarroom.desktop").read_text()
            self.assertIn("Exec=/usr/bin/thewarroom", launcher)
            self.assertIn("Terminal=false", launcher)
            self.assertTrue((unpacked / "usr/share/doc/thewarroom/INSTALL.md").is_file())

    def test_real_platform_lock_contention_and_crash_release(self):
        base = package.ROOT / ".tmp/desktop"
        base.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=base) as directory:
            root = Path(directory)
            suffix = "windows" if package.platform.system() == "Windows" else "unix"
            source = package.ROOT / f"internal/app/platform/instance_lock_{suffix}.go"
            # Compile the existing implementation verbatim apart from its package name.
            (root / "lock.go").write_text(source.read_text().replace(
                "package platform", "package main", 1))
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
