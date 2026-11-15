"""Local throwaway-key deliveries for the real Alpine upgrade wrapper. No network."""
import io
from pathlib import Path
import sys
import tarfile
from datetime import timedelta

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "scripts/release"))
import release
from securesystemslib.signer import CryptoSigner
from tuf.api.metadata import Metadata, Targets, Snapshot, Timestamp, MetaFile, TargetFile


def fixtures(source, destination):
    destination.mkdir(parents=True, exist_ok=False)
    keys = {role: [CryptoSigner.generate_ed25519()
                   for _ in range(3 if role == "root" else 1)] for role in release.ROLES}
    manifest = {role: [{"keyid": k.public_key.keyid, "key": k.public_key.to_dict()}
                       for k in signers] for role, signers in keys.items()}
    root = release.create_root(manifest)
    for key in keys["root"][:2]:
        root.sign(key, append=True)
    (destination / "root.json").write_bytes(root.to_bytes())
    data = source.read_bytes()
    for case in ("good", "byte-flipped", "truncated", "script", "expired", "rollback", "lower-release", "sandbox", "newer-file"):
        directory = destination / case
        directory.mkdir()
        version = {"good": 2, "byte-flipped": 3, "truncated": 4, "script": 5,
                   "expired": 6, "rollback": 1, "lower-release": 6, "sandbox": 7,
                   "newer-file": 8}[case]
        payload = data
        if case == "sandbox":
            script = b'#!/bin/sh\n[ "$(id -u)" != 0 ] || exit 91\n[ ! -e /root/twr ] || exit 92\n[ ! -w /usr ] || exit 93\necho sandbox-ok > /var/lib/twr/candidate-marker\nexit 1\n'
            stream = io.BytesIO()
            with tarfile.open(fileobj=stream, mode="w:gz") as archive:
                entry = tarfile.TarInfo("./twr-server")
                entry.size, entry.mode = len(script), 0o755
                archive.addfile(entry, io.BytesIO(script))
            payload = stream.getvalue()
        target = TargetFile.from_data("server.tar.gz", payload, ["sha256"])
        target.unrecognized_fields["custom"] = {"twr_server_release":
            release.server_release() - (case == "lower-release")}
        targets = Metadata(Targets(version=version, expires=release.future(1),
                                   targets={"server.tar.gz": target}))
        targets.sign(keys["targets"][0])
        snapshot = Metadata(Snapshot(version=version, expires=release.future(1), meta={
            "targets.json": MetaFile.from_data(version, targets.to_bytes(), ["sha256"])}))
        snapshot.sign(keys["snapshot"][0])
        timestamp = Metadata(Timestamp(version=version, expires=(release.future(1) if case != "expired"
            else release.future(0) - timedelta(days=1)), snapshot_meta=MetaFile.from_data(
                version, snapshot.to_bytes(), ["sha256"])))
        timestamp.sign(keys["timestamp"][0])
        for role, metadata in (("targets", targets), ("snapshot", snapshot), ("timestamp", timestamp)):
            name = f"{version}.{role}.json" if role != "timestamp" else "timestamp.json"
            (directory / name).write_bytes(metadata.to_bytes())
        if case == "byte-flipped":
            payload = bytes([payload[0] ^ 1]) + payload[1:]
        elif case == "truncated":
            payload = payload[:len(payload)//2]
        elif case == "script":
            payload = b'#!/bin/sh\ntouch /var/lib/twr/candidate-marker\n'
        (directory / "server.tar.gz").write_bytes(payload)


if __name__ == "__main__":
    fixtures(Path(sys.argv[1]), Path(sys.argv[2]))
