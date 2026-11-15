#!/usr/bin/env bash
# LAB-ONLY release bundle: a throwaway TUF root (never the real one) + bundle.sh from this worktree.
# Keys live in ~/fleet/lab-release (0700). Output: .scratch/lab/twr-server-LAB-ONLY-r<release>-<sha>.tar.gz
set -euo pipefail
cd "$(dirname "$0")/../.."
keys=${TWR_LAB_RELEASE:-$HOME/fleet/lab-release}
venv=$PWD/.scratch/lab-release-venv
(umask 077; mkdir -p "$keys"); chmod 700 "$keys"
[[ -x $venv/bin/python ]] || {
  python3.13 -m venv "$venv"
  "$venv/bin/python" -m pip install -q --require-hashes --only-binary=:all: -r scripts/release/requirements.txt
}
root=$keys/LAB-ONLY-root.json
if [[ ! -f $root ]]; then
  # Same shape as the real ceremony (3 root keys, 2-of-3; one key per other role), throwaway keys.
  (umask 077; "$venv/bin/python" -I - "$PWD/scripts/release" "$keys" <<'PY'
import sys
from pathlib import Path
sys.path.insert(0, sys.argv[1])
import release
from cryptography.hazmat.primitives import serialization as s
from securesystemslib.signer import CryptoSigner
out = Path(sys.argv[2])
keys = {r: [CryptoSigner.generate_ed25519() for _ in range(3 if r == "root" else 1)] for r in release.ROLES}
for role, signers in keys.items():
    for i, k in enumerate(signers):
        pem = k._private_key.private_bytes(s.Encoding.PEM, s.PrivateFormat.PKCS8, s.NoEncryption())
        (out / f"LAB-ONLY-{role}-{i}.pem").write_bytes(pem)
root = release.create_root({r: [{"keyid": k.public_key.keyid, "key": k.public_key.to_dict()} for k in v]
                            for r, v in keys.items()})
for k in keys["root"][:2]:
    root.sign(k, append=True)
release.verify_root(root)
(out / "LAB-ONLY-root.json").write_bytes(root.to_bytes())
PY
  )
fi
"$venv/bin/python" -I scripts/release/release.py check-root "$root"
rel=$("$venv/bin/python" -I -c 'import sys; sys.path.insert(0, "scripts/release"); import release; print(release.server_release())')
mkdir -p .scratch/lab
bash deploy/server/bundle.sh ".scratch/lab/twr-server-LAB-ONLY-r$rel-$(git rev-parse --short HEAD).tar.gz" "$root"
