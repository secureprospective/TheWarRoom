"""Build native installable candidates; never sign or publish them."""

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tempfile

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


def artifact_name(name):
    extensions = {"windows-amd64": "exe", "darwin-arm64": "dmg", "linux-amd64": "deb"}
    return f"warroom-{name}.{extensions[name]}"


def linux_package(source, output, guide):
    with tempfile.TemporaryDirectory(dir=output.parent) as directory:
        stage = Path(directory)
        stage.chmod(0o755)
        version = json.loads((ROOT / "wails.json").read_text(encoding="utf-8"))["info"]["productVersion"]
        binary = stage / "usr/bin/thewarroom"
        binary.parent.mkdir(parents=True)
        shutil.copy2(source, binary)
        binary.chmod(0o755)
        applications = stage / "usr/share/applications"
        applications.mkdir(parents=True)
        (applications / "thewarroom.desktop").write_text(
            "[Desktop Entry]\nType=Application\nName=The War Room\n"
            "Exec=/usr/bin/thewarroom\nTerminal=false\nCategories=Game;\n", encoding="utf-8")
        docs = stage / "usr/share/doc/thewarroom"
        docs.mkdir(parents=True)
        shutil.copy2(guide, docs / "INSTALL.md")
        control = stage / "DEBIAN"
        control.mkdir()
        (control / "control").write_text(
            f"Package: thewarroom\nVersion: {version}\nArchitecture: amd64\n"
            "Maintainer: Christopher Campbell\nSection: games\nPriority: optional\n"
            "Depends: libgtk-3-0t64, libwebkit2gtk-4.1-0\n"
            "Description: The War Room desktop client\n", encoding="utf-8")
        subprocess.run(["dpkg-deb", "--root-owner-group", "--build", str(stage), str(output)],
                       check=True)


def package(source, destination, name):
    destination.mkdir(parents=True, exist_ok=True)
    output = (destination / artifact_name(name)).resolve()
    guide = ROOT / "docs/delivery/Desktop_Install.md"
    if name == "linux-amd64":
        linux_package(source, output, guide)
    elif name == "darwin-arm64":
        with tempfile.TemporaryDirectory(dir=destination) as directory:
            stage = Path(directory)
            shutil.copytree(source, stage / source.name, symlinks=True)
            shutil.copy2(guide, stage / "INSTALL.md")
            (stage / "Applications").symlink_to("/Applications", target_is_directory=True)
            subprocess.run(["hdiutil", "create", "-volname", "The War Room", "-srcfolder",
                            str(stage), "-ov", "-format", "UDZO", str(output)], check=True)
    else:
        bootstrapper = ROOT / ".tmp/desktop/MicrosoftEdgeWebview2Setup.exe"
        if not bootstrapper.is_file():
            raise FileNotFoundError("stage the authenticated Microsoft WebView2 bootstrapper first")
        makensis = shutil.which("makensis") or str(
            Path(os.environ.get("ProgramFiles(x86)", "C:/Program Files (x86)")) / "NSIS/makensis.exe")
        subprocess.run([makensis, f"/DOUTPUT={output}", f"/DBINARY={source.resolve()}",
                        f"/DBOOTSTRAPPER={bootstrapper.resolve()}", f"/DGUIDE={guide}",
                        str(ROOT / "scripts/desktop/installer.nsi")], check=True)
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
    return package(source, ROOT / ".tmp/desktop/artifacts", name)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()
    os.chdir(ROOT)
    print(build())
