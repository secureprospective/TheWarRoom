"""Build signed TUF repositories; private key custody stays with the operator."""

import argparse
from datetime import datetime, timedelta, timezone
import getpass
import ipaddress
import json
import os
from pathlib import Path
import re
from urllib.parse import urlsplit

from cryptography.hazmat.primitives.serialization import load_pem_private_key, load_pem_public_key
from securesystemslib.signer import CryptoSigner, Key, SSlibKey
from tuf.api.metadata import Metadata, Root, Targets, Snapshot, Timestamp, MetaFile, TargetFile


ROLES = ("root", "targets", "snapshot", "timestamp")


def future(days):
    return datetime.now(timezone.utc) + timedelta(days=days)


def validate_config(data):
    """The entire distributable configuration is public, not server configuration."""
    raw = json.loads(data)
    if not isinstance(raw, dict) or set(raw) != {"server_url", "metadata_url", "targets_url"}:
        raise ValueError("config requires only server_url, metadata_url, targets_url")
    for value in raw.values():
        if not isinstance(value, str):
            raise ValueError("config URLs must be strings")
        url = urlsplit(value)
        host = url.hostname or ""
        if (url.scheme != "https" or url.username or url.password or url.query
                or url.fragment or url.port not in (None, 443)
                or not re.fullmatch(r"[a-z0-9.-]+", host) or "." not in host
                or host.endswith((".local", ".localhost", ".internal", ".lan"))
                or url.path not in ("", "/", "/metadata/", "/targets/")):
            raise ValueError("config requires public HTTPS hostnames and repository paths only")
        try:
            ipaddress.ip_address(host)
        except ValueError:
            pass
        else:
            raise ValueError("config requires a hostname, not an IP or certificate pin")
    return raw


def create_root(keys, previous=None):
    if set(keys) != set(ROLES):
        raise ValueError("public key manifest must define all four roles")
    root = Root(version=1 if previous is None else previous.signed.version + 1,
                expires=future(365), consistent_snapshot=True)
    for role in ROLES:
        public = [Key.from_dict(item["keyid"], dict(item["key"])) for item in keys[role]]
        if len({key.keyid for key in public}) != len(public):
            raise ValueError("duplicate role key")
        if len(public) != (3 if role == "root" else 1):
            raise ValueError("root needs three keys; other roles need one each")
        for key in public:
            if key.keytype != "ed25519" or key.scheme != "ed25519":
                raise ValueError("only Ed25519 release keys are supported")
            root.add_key(key, role)
        root.roles[role].threshold = 2 if role == "root" else 1
    if len(root.keys) != 6:
        raise ValueError("each role must have separate keys")
    return Metadata(root)


def sign_file(metadata_path, key_path):
    """Prompt for an encrypted PEM passphrase; never accept it on the command line."""
    info = key_path.lstat()
    if not key_path.is_file() or key_path.is_symlink() or info.st_mode & 0o077:
        raise ValueError("private key must be a regular, owner-only file")
    password = getpass.getpass("Private key passphrase: ").encode()
    if not password:
        raise ValueError("encrypted PEM required")
    signer = CryptoSigner(load_pem_private_key(key_path.read_bytes(), password))
    metadata = Metadata.from_file(str(metadata_path))
    metadata.sign(signer, append=True)
    temporary = metadata_path.with_suffix(".signed.tmp")
    with temporary.open("xb") as stream:
        stream.write(metadata.to_bytes())
    temporary.replace(metadata_path)


def verify_root(root, previous=None):
    if not isinstance(root.signed, Root) or not root.signed.consistent_snapshot:
        raise ValueError("consistent-snapshot root required")
    if (len(root.signed.roles["root"].keyids) != 3
            or root.signed.roles["root"].threshold != 2):
        raise ValueError("root requires a two-of-three threshold")
    if root.signed.is_expired(datetime.now(timezone.utc)):
        raise ValueError("root expired")
    root.signed.verify_delegate("root", root.signed_bytes, root.signatures)
    if previous is not None:
        if root.signed.version != previous.signed.version + 1:
            raise ValueError("root rotation must increment version by one")
        previous.signed.verify_delegate("root", root.signed_bytes, root.signatures)


def prepare_targets(root, config_path, artifacts, version, previous=None):
    verify_root(root)
    if version < 1 or (previous is None and version != 1):
        raise ValueError("initial targets version must be one")
    if previous is not None:
        root.signed.verify_delegate("targets", previous.signed_bytes, previous.signatures)
        if version != previous.signed.version + 1:
            raise ValueError("targets version must increment by one")
    config = config_path.read_bytes()
    validate_config(config)
    payloads = {"client-config.json": config}
    for path in artifacts:
        name = path.name
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,159}", name) or name in payloads:
            raise ValueError("artifact names must be unique safe basenames")
        if path.is_symlink() or not path.is_file():
            raise ValueError("artifact must be a regular file")
        payloads[name] = path.read_bytes()
    if len(payloads) == 1:
        raise ValueError("at least one release artifact required")
    targets = Targets(version=version, expires=future(30))
    for name, data in payloads.items():
        targets.targets[name] = TargetFile.from_data(name, data, ["sha256"])
    return Metadata(targets), payloads


def write_targets(destination, metadata, payloads):
    destination.mkdir(parents=True, exist_ok=False)
    (destination / "metadata").mkdir()
    (destination / "targets").mkdir()
    (destination / "metadata" / "targets.json").write_bytes(metadata.to_bytes())
    for name, data in payloads.items():
        digest = metadata.signed.targets[name].hashes["sha256"]
        (destination / "targets" / f"{digest}.{name}").write_bytes(data)


def link_metadata(root, repository, role):
    """Only link a child after its signatures pass the root's threshold."""
    verify_root(root)
    directory = repository / "metadata"
    child_role = "targets" if role == "snapshot" else "snapshot"
    child_data = (directory / f"{child_role}.json").read_bytes()
    child = Metadata.from_bytes(child_data)
    if child.signed.type != child_role or child.signed.is_expired(datetime.now(timezone.utc)):
        raise ValueError("wrong or expired child metadata")
    root.signed.verify_delegate(child_role, child.signed_bytes, child.signatures)
    version = child.signed.version
    link = MetaFile.from_data(version, child_data, ["sha256"])
    if role == "snapshot":
        parent = Metadata(Snapshot(version=version, expires=future(7),
                                   meta={"targets.json": link}))
    else:
        parent = Metadata(Timestamp(version=version, expires=future(1), snapshot_meta=link))
    destination = directory / f"{role}.json"
    with destination.open("xb") as stream:
        stream.write(parent.to_bytes())


def finalize(root, repository):
    verify_root(root)
    directory = repository / "metadata"
    metadata = {role: Metadata.from_file(str(directory / f"{role}.json"))
                for role in ROLES[1:]}
    for role, item in metadata.items():
        if item.signed.type != role or item.signed.is_expired(datetime.now(timezone.utc)):
            raise ValueError("wrong or expired metadata")
        root.signed.verify_delegate(role, item.signed_bytes, item.signatures)
    targets, snapshot, timestamp = (metadata[r] for r in ROLES[1:])
    for parent, child, role in ((snapshot, targets, "targets"), (timestamp, snapshot, "snapshot")):
        link = (parent.signed.snapshot_meta if role == "snapshot"
                else parent.signed.meta[f"{role}.json"])
        if link.version != child.signed.version:
            raise ValueError("linked metadata version mismatch")
        link.verify_length_and_hashes((directory / f"{role}.json").read_bytes())
    for name, target in targets.signed.targets.items():
        target.verify_length_and_hashes(
            (repository / "targets" / f"{target.hashes['sha256']}.{name}").read_bytes())
    for role in ("targets", "snapshot"):
        item = metadata[role]
        (directory / f"{item.signed.version}.{role}.json").write_bytes(item.to_bytes())
    (directory / f"{root.signed.version}.root.json").write_bytes(root.to_bytes())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    root_cmd = commands.add_parser("root")
    root_cmd.add_argument("--keys", type=Path, required=True)
    root_cmd.add_argument("--previous", type=Path)
    root_cmd.add_argument("--out", type=Path, required=True)
    public = commands.add_parser("public")
    public.add_argument("pem", type=Path)
    sign = commands.add_parser("sign")
    sign.add_argument("metadata", type=Path)
    sign.add_argument("--key", type=Path, required=True)
    check = commands.add_parser("check-root")
    check.add_argument("root", type=Path)
    check.add_argument("--previous", type=Path)
    targets = commands.add_parser("targets")
    targets.add_argument("--root", type=Path, required=True)
    targets.add_argument("--config", type=Path, required=True)
    targets.add_argument("--artifact", type=Path, action="append", required=True)
    targets.add_argument("--version", type=int, required=True)
    targets.add_argument("--previous", type=Path)
    targets.add_argument("--out", type=Path, required=True)
    for role in ("snapshot", "timestamp", "finalize"):
        command = commands.add_parser(role)
        command.add_argument("--root", type=Path, required=True)
        command.add_argument("--repo", type=Path, required=True)
    args = parser.parse_args()
    previous = (Metadata.from_file(str(args.previous))
                if getattr(args, "previous", None) else None)
    if args.command == "root":
        metadata = create_root(json.loads(args.keys.read_bytes()), previous)
        with args.out.open("xb") as stream:
            stream.write(metadata.to_bytes())
    elif args.command == "public":
        key = SSlibKey.from_crypto(load_pem_public_key(args.pem.read_bytes()))
        print(json.dumps({"keyid": key.keyid, "key": key.to_dict()}))
    elif args.command == "sign":
        sign_file(args.metadata, args.key)
    else:
        root = Metadata.from_file(str(args.root))
        if args.command == "check-root":
            verify_root(root, previous)
        elif args.command == "targets":
            metadata, payloads = prepare_targets(root, args.config, args.artifact,
                                                args.version, previous)
            write_targets(args.out, metadata, payloads)
        elif args.command == "finalize":
            finalize(root, args.repo)
        else:
            link_metadata(root, args.repo, args.command)


if __name__ == "__main__":
    os.umask(0o077)
    main()
