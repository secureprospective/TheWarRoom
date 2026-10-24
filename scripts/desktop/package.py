"""Build native desktop candidates; never sign or publish them."""

import argparse
from datetime import datetime, timezone
import hashlib
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tarfile
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]


def target():
    system = platform.system()
    machine = platform.machine().lower()
    targets = {("Windows", "amd64"): "windows-amd64",
               ("Darwin", "arm64"): "darwin-arm64",
               ("Linux", "x86_64"): "linux-amd64"}
    if (system, machine) not in targets:
        raise ValueError(f"unsupported native target: {system}/{machine}")
    return targets[system, machine]


def archive(source, destination, name):
    destination.mkdir(parents=True, exist_ok=True)
    if name.startswith("linux-"):
        output = destination / f"thewarroom-{name}.tar.gz"
        with tarfile.open(output, "w:gz") as bundle:
            bundle.add(source, arcname="thewarroom")
            bundle.add(ROOT / "docs/delivery/Desktop_Install.md", arcname="INSTALL.md")
    else:
        output = destination / f"thewarroom-{name}.zip"
        with tempfile.TemporaryDirectory(dir=destination) as directory:
            stage = Path(directory)
            if source.is_dir():
                shutil.copytree(source, stage / source.name, symlinks=True)
            else:
                shutil.copy2(source, stage / source.name)
            shutil.copy2(ROOT / "docs/delivery/Desktop_Install.md", stage / "INSTALL.md")
            if name.startswith("darwin-"):
                # ditto retains app bundle symlinks and executable modes.
                subprocess.run(["ditto", "-c", "-k", "--sequesterRsrc", str(stage),
                                str(output.resolve())], check=True)
            else:
                with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED) as bundle:
                    for path in stage.rglob("*"):
                        if path.is_file():
                            bundle.write(path, path.relative_to(stage))
    digest = hashlib.sha256(output.read_bytes()).hexdigest()
    output.with_suffix(output.suffix + ".sha256").write_text(
        f"{digest}  {output.name}\n", encoding="utf-8")
    return output


def build():
    name = target()
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    pkg = "github.com/secureprospective/TheWarRoom/internal/composition"
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    flags = f"-X {pkg}.version=cs45-candidate -X {pkg}.commit={commit} -X {pkg}.buildDate={stamp}"
    command = ["wails", "build", "-m", "-nosyncgomod", "-skipbindings", "-s",
               "-trimpath", "-ldflags", flags]
    if name.startswith("linux-"):
        command += ["-tags", "webkit2_41"]
    if name.startswith("windows-"):
        command += ["-webview2", "error"]
    subprocess.run(command, cwd=ROOT, check=True)
    source = ROOT / "build/bin/thewarroom"
    if name.startswith("windows-"):
        source = source.with_suffix(".exe")
    elif name.startswith("darwin-"):
        source = source.with_suffix(".app")
    if not source.exists():
        raise FileNotFoundError(source)
    return archive(source, ROOT / ".tmp/desktop/artifacts", name)


def smoke(bundle):
    """Exercise the extracted executable's loader and CLI, without opening any data."""
    base = ROOT / ".tmp/desktop"
    base.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=base) as directory:
        unpacked = Path(directory)
        if bundle.name.endswith(".tar.gz"):
            with tarfile.open(bundle) as stream:
                stream.extractall(unpacked, filter="data")
        elif platform.system() == "Darwin":
            subprocess.run(["ditto", "-x", "-k", str(bundle), str(unpacked)], check=True)
        else:
            with zipfile.ZipFile(bundle) as stream:
                stream.extractall(unpacked)
        executable = unpacked / "thewarroom"
        if platform.system() == "Windows":
            executable = executable.with_suffix(".exe")
        elif platform.system() == "Darwin":
            executable = unpacked / "thewarroom.app/Contents/MacOS/thewarroom"
        result = subprocess.run([str(executable), "-version"], check=True,
                                capture_output=True, text=True, timeout=30)
        if "cs45-candidate" not in result.stdout:
            raise ValueError(f"missing candidate build stamp: {result.stdout}")
        print(result.stdout.strip())


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()
    os.chdir(ROOT)
    artifact = build()
    smoke(artifact)
    print(artifact)
