"""Install the downloaded candidate on a fresh native runner, then exercise its WebView and lock."""

import argparse
from pathlib import Path
import platform
import subprocess
import tempfile
import time


def install(artifact, directory):
    system = platform.system()
    if system == "Windows":
        destination = directory / "installed"
        subprocess.run([str(artifact), "/S", f"/D={destination}"], check=True, timeout=300)
        return destination / "thewarroom.exe"
    if system == "Darwin":
        mount = directory / "mounted"
        mount.mkdir()
        subprocess.run(["hdiutil", "attach", str(artifact), "-mountpoint", str(mount),
                        "-nobrowse", "-readonly"], check=True, timeout=60)
        destination = Path.home() / "Applications/thewarroom.app"
        destination.parent.mkdir(exist_ok=True)
        try:
            subprocess.run(["ditto", str(mount / "thewarroom.app"), str(destination)], check=True)
        finally:
            subprocess.run(["hdiutil", "detach", str(mount)], check=True, timeout=60)
        return destination / "Contents/MacOS/thewarroom"
    subprocess.run(["sudo", "apt-get", "update"], check=True, timeout=180)
    subprocess.run(["sudo", "apt-get", "install", "-y", str(artifact)], check=True, timeout=300)
    subprocess.run(["gtk-launch", "--version"], check=True, timeout=15)
    return Path("/usr/bin/thewarroom")


def smoke(executable, directory):
    ready = directory / "ready"
    log = directory / "smoke.log"
    command = [str(executable), "-package-smoke", str(directory)]
    with log.open("wb") as output:
        for attempt in range(2):
            ready.unlink(missing_ok=True)
            holder = subprocess.Popen(command, stdout=output, stderr=output)
            try:
                deadline = time.monotonic() + 90
                while not ready.exists():
                    if holder.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError(f"installed WebView failed to become ready: {log.read_text(errors='replace')}")
                    time.sleep(0.1)
                if holder.poll() is not None:
                    raise RuntimeError("installed app exited after readiness")
                rejected = subprocess.run(command, stdout=output, stderr=output, timeout=30)
                if rejected.returncode != 1:
                    raise RuntimeError(f"second installed copy was not refused: {rejected.returncode}")
                print(f"PASS installed WebView ready, second copy refused (attempt {attempt + 1})", flush=True)
            finally:
                if holder.poll() is None:
                    holder.kill()
                holder.wait(timeout=30)
        if not (directory / "smoke.db.lock").exists():
            raise RuntimeError("missing lock file: test did not exercise the platform guard")
        if (directory / "smoke.db").exists():
            raise RuntimeError("installation smoke must never create a league database")
        print("PASS killed-process lock release with stale lock file; no league database", flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("artifact", type=Path)
    args = parser.parse_args()
    # Caller controls all paths; artifacts are CI-built candidates, not arbitrary uploads.
    with tempfile.TemporaryDirectory(prefix="warroom-install-") as directory:
        directory = Path(directory)
        executable = install(args.artifact.resolve(), directory)
        smoke(executable, directory)
