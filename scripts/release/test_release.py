"""Prove published metadata with the reference TUF client, not a local verifier."""

from datetime import datetime, timedelta, timezone
import json
import shutil
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from urllib.parse import urlsplit

from cryptography.hazmat.primitives import serialization
from securesystemslib.signer import CryptoSigner
from tuf.api import exceptions
from tuf.api.metadata import Metadata, Root, Snapshot, Timestamp, MetaFile
from tuf.ngclient import Updater
from tuf.ngclient.fetcher import FetcherInterface

import release


class RepositoryFetcher(FetcherInterface):
    def __init__(self):
        self.files = {}

    def _fetch(self, url):
        path = urlsplit(url).path.lstrip("/")
        if path not in self.files:
            raise exceptions.DownloadHTTPError("missing", 404)
        return iter([self.files[path]])

    def serve(self, repository):
        self.files = {str(p.relative_to(repository)): p.read_bytes()
                      for p in repository.rglob("*") if p.is_file()}


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.keys = {role: [CryptoSigner.generate_ed25519()
                            for _ in range(3 if role == "root" else 1)]
                     for role in release.ROLES}
        self.root = release.create_root(self.manifest(self.keys))
        for key in self.keys["root"][:2]:
            self.root.sign(key, append=True)
        self.config = self.directory / "config.json"
        self.config.write_text(json.dumps({
            "server_url": "https://league.example.org",
            "metadata_url": "https://updates.example.org/metadata/",
            "targets_url": "https://updates.example.org/targets/",
        }))
        self.artifact = self.directory / "warroom-linux-amd64.tar.gz"
        self.artifact.write_bytes(b"throwaway application package")
        self.fetcher = RepositoryFetcher()
        self.cache = self.directory / "cache"
        self.downloads = self.directory / "downloads"
        self.cache.mkdir()
        self.downloads.mkdir()

    @staticmethod
    def manifest(keys):
        return {role: [{"keyid": key.public_key.keyid, "key": key.public_key.to_dict()}
                       for key in signers] for role, signers in keys.items()}

    def repository(self, name, version=1, previous=None, expired=None):
        repo = self.directory / name
        targets, payloads = release.prepare_targets(
            self.root, self.config, [self.artifact], version, previous)
        if expired == "targets":
            targets.signed.expires = datetime.now(timezone.utc) - timedelta(days=1)
        targets.sign(self.keys["targets"][0])
        release.write_targets(repo, targets, payloads)
        # Build even intentionally invalid/expired repositories for client rejection tests.
        snapshot = Metadata(Snapshot(version=version, expires=release.future(7), meta={
            "targets.json": MetaFile.from_data(version, targets.to_bytes(), ["sha256"])}))
        if expired == "snapshot":
            snapshot.signed.expires = datetime.now(timezone.utc) - timedelta(days=1)
        snapshot.sign(self.keys["snapshot"][0])
        timestamp = Metadata(Timestamp(version=version, expires=release.future(1),
                                      snapshot_meta=MetaFile.from_data(
                                          version, snapshot.to_bytes(), ["sha256"])))
        if expired == "timestamp":
            timestamp.signed.expires = datetime.now(timezone.utc) - timedelta(days=1)
        timestamp.sign(self.keys["timestamp"][0])
        for role, metadata in (("targets", targets), ("snapshot", snapshot),
                               ("timestamp", timestamp)):
            (repo / "metadata" / f"{role}.json").write_bytes(metadata.to_bytes())
            if role != "timestamp":
                (repo / "metadata" / f"{version}.{role}.json").write_bytes(metadata.to_bytes())
        (repo / "metadata" / "1.root.json").write_bytes(self.root.to_bytes())
        return repo

    def reset_cache(self):
        shutil.rmtree(self.cache)
        self.cache.mkdir()

    def client(self):
        return Updater(str(self.cache), "https://updates.example.org/metadata/",
                       str(self.downloads), "https://updates.example.org/targets/",
                       fetcher=self.fetcher, bootstrap=self.root.to_bytes())

    def test_signed_artifact_and_public_config_download(self):
        repo = self.repository("valid")
        # Exercise actual publisher linking and finalization as well.
        for name in ("snapshot.json", "timestamp.json"):
            (repo / "metadata" / name).unlink()
        release.link_metadata(self.root, repo, "snapshot")
        snapshot = repo / "metadata" / "snapshot.json"
        item = Metadata.from_file(str(snapshot))
        item.sign(self.keys["snapshot"][0])
        snapshot.write_bytes(item.to_bytes())
        release.link_metadata(self.root, repo, "timestamp")
        timestamp = repo / "metadata" / "timestamp.json"
        item = Metadata.from_file(str(timestamp))
        item.sign(self.keys["timestamp"][0])
        timestamp.write_bytes(item.to_bytes())
        release.finalize(self.root, repo)
        self.fetcher.serve(repo)
        client = self.client()
        client.refresh()
        for name, expected in ((self.artifact.name, self.artifact.read_bytes()),
                               ("client-config.json", self.config.read_bytes())):
            info = client.get_targetinfo(name)
            result = client.download_target(info)
            self.assertEqual(Path(result).read_bytes(), expected)
        public = b"".join(self.fetcher.files.values())
        self.assertNotIn(b"PRIVATE KEY", public)
        self.assertNotIn(b"tunnel", public)
        self.assertNotIn(b"pin", public)

    def test_tampered_metadata_rejected_without_targets_or_downloads(self):
        for role in ("targets", "snapshot", "timestamp"):
            with self.subTest(role=role):
                self.reset_cache()
                repo = self.repository("tamper-" + role)
                self.fetcher.serve(repo)
                name = f"metadata/{'' if role == 'timestamp' else '1.'}{role}.json"
                raw = json.loads(self.fetcher.files[name])
                signature = raw["signatures"][0]["sig"]
                damaged = ("0" if signature[0] != "0" else "1") + signature[1:]
                self.fetcher.files[name] = self.fetcher.files[name].replace(
                    signature.encode(), damaged.encode())
                with self.assertRaises(exceptions.RepositoryError):
                    self.client().refresh()
                self.assertFalse((self.cache / "targets.json").exists())
                self.assertEqual(list(self.downloads.iterdir()), [])

    def test_expiry_rejected(self):
        for role in ("targets", "snapshot", "timestamp"):
            with self.subTest(role=role):
                self.reset_cache()
                repo = self.repository("expired-" + role, expired=role)
                self.fetcher.serve(repo)
                with self.assertRaises(exceptions.ExpiredMetadataError):
                    self.client().refresh()
                self.assertFalse((self.cache / "targets.json").exists())
                self.assertEqual(list(self.downloads.iterdir()), [])

    def test_corrupt_artifact_and_config_leave_no_download(self):
        repo = self.repository("corrupt")
        self.fetcher.serve(repo)
        client = self.client()
        client.refresh()
        for name in (self.artifact.name, "client-config.json"):
            info = client.get_targetinfo(name)
            self.fetcher.files[f"targets/{info.hashes['sha256']}.{name}"] = b"corrupt"
            with self.assertRaises(exceptions.LengthOrHashMismatchError):
                client.download_target(info)
            self.assertEqual(list(self.downloads.iterdir()), [])

    def test_rollback_rejected_across_client_restart(self):
        first = self.repository("first")
        old_targets = Metadata.from_file(str(first / "metadata" / "targets.json"))
        second = self.repository("second", 2, old_targets)
        self.fetcher.serve(second)
        self.client().refresh()
        trusted = (self.cache / "targets.json").read_bytes()
        self.fetcher.serve(first)
        with self.assertRaises(exceptions.BadVersionNumberError):
            self.client().refresh()
        self.assertEqual((self.cache / "targets.json").read_bytes(), trusted)
        self.assertEqual(list(self.downloads.iterdir()), [])
        with self.assertRaises(ValueError):
            release.prepare_targets(self.root, self.config, [self.artifact], 1,
                                    Metadata.from_bytes(trusted))

    def test_expired_root_rejected_by_publisher_and_reference_client(self):
        repo = self.repository("expired-root")
        expired = Metadata.from_bytes(self.root.to_bytes())
        expired.signed.expires = datetime.now(timezone.utc) - timedelta(days=1)
        expired.signatures.clear()
        for key in self.keys["root"][:2]:
            expired.sign(key, append=True)
        with self.assertRaises(ValueError):
            release.verify_root(expired)
        self.fetcher.serve(repo)
        client = Updater(str(self.cache), "https://updates.example.org/metadata/",
                         fetcher=self.fetcher, bootstrap=expired.to_bytes())
        with self.assertRaises(exceptions.ExpiredMetadataError):
            client.refresh()
        self.assertEqual(list(self.downloads.iterdir()), [])

    def test_snapshot_and_targets_rollback_with_fresh_timestamp_rejected(self):
        first = self.repository("rollback-old")
        old = Metadata.from_file(str(first / "metadata" / "targets.json"))
        second = self.repository("rollback-current", 2, old)
        for role in ("snapshot", "targets"):
            with self.subTest(role=role):
                self.reset_cache()
                self.fetcher.serve(second)
                self.client().refresh()
                trusted = (self.cache / "targets.json").read_bytes()
                old_snapshot = Metadata.from_file(str(first / "metadata" / "snapshot.json"))
                if role == "targets":
                    old_snapshot.signed.version = 3
                    old_snapshot.sign(self.keys["snapshot"][0])
                snapshot_data = old_snapshot.to_bytes()
                stamp = Metadata(Timestamp(version=3, expires=release.future(1),
                    snapshot_meta=MetaFile.from_data(old_snapshot.signed.version,
                                                     snapshot_data, ["sha256"])))
                stamp.sign(self.keys["timestamp"][0])
                self.fetcher.files["metadata/timestamp.json"] = stamp.to_bytes()
                self.fetcher.files[f"metadata/{old_snapshot.signed.version}.snapshot.json"] = snapshot_data
                with self.assertRaises(exceptions.BadVersionNumberError):
                    self.client().refresh()
                self.assertEqual((self.cache / "targets.json").read_bytes(), trusted)
                self.assertEqual(list(self.downloads.iterdir()), [])

    def test_threshold_rotation_requires_old_and_new_quorums(self):
        repo = self.repository("rotation")
        new_keys = {**self.keys, "root": [CryptoSigner.generate_ed25519() for _ in range(3)]}
        for old_count, new_count, accepted in ((1, 2, False), (2, 1, False), (2, 2, True)):
            with self.subTest(old=old_count, new=new_count):
                rotated = release.create_root(self.manifest(new_keys), self.root)
                for key in self.keys["root"][:old_count] + new_keys["root"][:new_count]:
                    rotated.sign(key, append=True)
                if accepted:
                    release.verify_root(rotated, self.root)
                else:
                    with self.assertRaises(exceptions.UnsignedMetadataError):
                        release.verify_root(rotated, self.root)
                # Independent reference client enforces both thresholds too.
                cache = self.directory / f"rotation-{old_count}-{new_count}"
                cache.mkdir()
                self.fetcher.serve(repo)
                self.fetcher.files["metadata/2.root.json"] = rotated.to_bytes()
                client = Updater(str(cache), "https://updates.example.org/metadata/",
                                 fetcher=self.fetcher, bootstrap=self.root.to_bytes())
                if accepted:
                    client.refresh()
                    self.assertEqual(Metadata.from_file(str(cache / "root.json")).signed.version, 2)
                else:
                    with self.assertRaises(exceptions.UnsignedMetadataError):
                        client.refresh()
                    self.assertEqual(Metadata.from_file(str(cache / "root.json")).signed.version, 1)

    def test_secret_or_pin_config_rejected_before_output(self):
        valid = json.loads(self.config.read_bytes())
        for extra in ("tunnel_token", "leaf_pin", "password"):
            with self.subTest(extra=extra):
                self.config.write_text(json.dumps({**valid, extra: "not-a-secret"}))
                with self.assertRaises(ValueError):
                    release.prepare_targets(self.root, self.config, [self.artifact], 1)
        for value in ("http://league.example.org", "https://127.0.0.1",
                      "https://user:pass@league.example.org", "https://league.example.org/?token=x",
                      "https://machine.local", "https://league.example.org/secret-token"):
            with self.subTest(value=value):
                self.config.write_text(json.dumps({**valid, "server_url": value}))
                with self.assertRaises(ValueError):
                    release.prepare_targets(self.root, self.config, [self.artifact], 1)

    def test_encrypted_key_signing_and_permissions(self):
        key = self.keys["root"][2]
        private_key = serialization.load_pem_private_key(key.private_bytes, None)
        pem = private_key.private_bytes(serialization.Encoding.PEM,
                                           serialization.PrivateFormat.PKCS8,
                                           serialization.BestAvailableEncryption(b"throwaway"))
        private = self.directory / "test-key.pem"
        private.write_bytes(pem)
        private.chmod(0o600)
        path = self.directory / "root.json"
        path.write_bytes(self.root.to_bytes())
        with patch("getpass.getpass", return_value="throwaway"):
            release.sign_file(path, private)
        self.assertIn(key.public_key.keyid, Metadata.from_file(str(path)).signatures)
        private.chmod(0o644)
        before = path.read_bytes()
        with self.assertRaises(ValueError):
            release.sign_file(path, private)
        self.assertEqual(path.read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
