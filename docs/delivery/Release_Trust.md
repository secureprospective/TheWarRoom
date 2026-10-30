# Release trust and offline ceremony

Status: tooling and throwaway-key tests only. **No production root, hostname, signing key,
owner installer or custody ceremony has been created or verified.** Christopher must perform
and record the ceremony below before an owner build. This is free TUF integrity, not Apple
notarization or Authenticode. Unsigned-app warnings remain; packaging/install drills are slice 45.
The native verification library and persistent trust cache are in `internal/releasetrust`;
desktop wiring, staged installation and launch are slice 44.

## Trust boundary and contract

Reviewed packages → SHA-256 targets (including `client-config.json`) → signed targets →
signed snapshot (length/hash/version of targets) → signed timestamp (length/hash/version of
snapshot). An authenticated installer embeds the initial **public root** and public repository
URLs. The client refreshes through sequential, dual-threshold root rotations, then verifies
fresh timestamp/snapshot/targets and artifact bytes before use. Persist trusted metadata across
restarts; never replace it with downloaded bootstrap metadata or reset it after an error.
Denial must not launch an artifact, apply configuration or write native league data.

`release.py` prepares local files, prompts to sign with operator-held encrypted keys, checks
thresholds/links/artifact hashes and writes consistent-snapshot filenames. It does not upload,
install, change server configuration or contact MFL. `tuf.ngclient.Updater` is the independent
reference verifier in the tests, not a replacement implementation of TUF. Its fetcher is an
in-memory repository, so the tests make no network calls. Publisher input files and output
folder selection are operator-controlled; use a private workspace, not an uploaded directory.
An incomplete output directory is never a publishable release. Publish only after `finalize`.

Policy: Ed25519; root **2 of 3**, three independently held keys; targets, snapshot and timestamp
one distinct key each. Root expires in 365 days, targets 30 days, snapshot seven days, timestamp
one day. Renew well before expiry. The simple publisher increments all three release-role
versions together, even for freshness-only releases; it does not implement independent timestamp
renewal. Retain the last authenticated targets metadata as the next invocation's `--previous`.
Clients enforce rollback independently of publisher input selection. A fresh install cannot
detect an otherwise valid older release within its expiry window; distribute current bootstrap
material through the personally authenticated install handoff. A correct UTC client clock is
required. Hostile clocks and stolen signing thresholds are outside this slice's guarantees.

## Prepare the release workstation

Commands below run from the repo root. Use a reviewed checkout on a Linux x86-64 CPython 3.13
workstation (the hashed binary-wheel lock is for that environment, not owner OS dependencies).
The new dependencies are the upstream reference `tuf` package from
`github.com/theupdateframework/python-tuf` and `securesystemslib` from
`github.com/secure-systems-lab/securesystemslib`; package metadata/source identities were checked.
`cryptography`, `cffi`, `pycparser` and `urllib3` are pinned transitively. The native verifier separately pins `github.com/theupdateframework/go-tuf/v2` v2.4.2.
Upstream identity, latest release and API were checked against its official GitHub release and
module source. This release fixes duplicate-public-key threshold counting; do not downgrade it. Review and re-lock deliberately when updating these tools.

```sh
scripts/release/test.sh
# Before disconnecting the ceremony workstation, stage these pinned wheels:
scripts/release/.venv/bin/python -m pip download --require-hashes --only-binary=:all: \
  -r scripts/release/requirements.txt -d /media/TRANSFER/wheels
# On the offline workstation, create the venv and install from that reviewed medium:
python3.13 -m venv scripts/release/.venv
scripts/release/.venv/bin/python -m pip install --no-index --find-links=/media/TRANSFER/wheels \
  --require-hashes --only-binary=:all: -r scripts/release/requirements.txt
```

Do not place private keys, decrypted backups, passphrases or production staging folders in the
checkout. The commands below are instructions for Christopher, **not evidence they ran**.
`$CEREMONY` and `$KEYS` must name private directories on encrypted removable storage outside
any repository; `$PUBLIC` and `$STAGE` must name new private staging directories. `umask 077`.
Do not pass passphrases through flags, environment variables, logs or shell history.

## First root: Christopher and deputy present

1. Disconnect networking. Check the reviewed tool checkout and offline dependency hashes.
   Confirm correct UTC time. Prepare three independent encrypted root custody media: Christopher,
   his wife/deputy, and a separately secured sealed recovery copy. One custodian's daily machine
   must not hold two root private keys. Record physical locations and access instructions privately.
2. For each of `root-a`, `root-b`, `root-c`, `targets`, `snapshot`, `timestamp`, generate a distinct
   encrypted Ed25519 PEM on its intended custody medium. Substitute the key name below; never
   use test keys. OpenSSL prompts for encryption passphrases:

   ```sh
   umask 077
   openssl genpkey -algorithm ED25519 -aes-256-cbc -out "$KEYS/root-a.pem"
   chmod 600 "$KEYS/root-a.pem"
   openssl pkey -in "$KEYS/root-a.pem" -pubout -out "$PUBLIC/root-a.pub.pem"
   scripts/release/.venv/bin/python scripts/release/release.py public \
     "$PUBLIC/root-a.pub.pem" > "$PUBLIC/root-a.json"
   ```

   Keep targets signing offline with reviewed release artifacts. Snapshot/timestamp keys may be
   on a dedicated release-signing workstation, never on the public repository/Tunnel host or in
   ordinary CI secrets. This implementation prompts for their passphrases too.
3. Assemble `public-keys.json` **using only** the six public JSON outputs. Format:
   `{"root":[<root-a JSON>,<root-b JSON>,<root-c JSON>],"targets":[<targets JSON>],"snapshot":[<snapshot JSON>],"timestamp":[<timestamp JSON>]}`.
   The `public` command emits each complete `{"keyid":...,"key":...}` entry. Do not insert PEM
   private bytes. Review the six public key IDs independently with the deputy.
4. Create the first unsigned root. Each custodian signs the **same bytes**, preserving signatures:

   ```sh
   P=scripts/release/.venv/bin/python
   R=scripts/release/release.py
   "$P" "$R" root --keys "$PUBLIC/public-keys.json" --out "$PUBLIC/1.root.json"
   "$P" "$R" sign "$PUBLIC/1.root.json" --key "$KEYS/root-a.pem"
   # Transfer public metadata to the second custodian; do not consolidate private keys.
   "$P" "$R" sign "$PUBLIC/1.root.json" --key "$KEYS/root-b.pem"
   "$P" "$R" check-root "$PUBLIC/1.root.json"
   sha256sum "$PUBLIC/1.root.json"
   ```

5. Record that public root digest, key IDs, thresholds, expiry and tool commit in the ceremony
   record. Both people compare the digest out of band. Print/seal deputy recovery instructions
   with custody/access details; never publish those details or passphrases. Test decrypt/sign
   on a throwaway metadata copy, then remove that copy. Unmount custody media.
6. Bundle this authenticated public root in the first installer via slice 44/45. Personally hand
   the installer and independently confirmed root digest to each owner. Downloading root.json
   from the same untrusted update endpoint is **not** first-install trust. Do not substitute a TLS
   leaf fingerprint: public HTTPS certificate renewal must not require an application release.

## Build and sign one repository

Create a public config JSON with **exactly** `server_url`, `metadata_url`, `targets_url`, each
an HTTPS public hostname URL. Repository paths may be `/metadata/` and `/targets/`; no IPs,
credentials, query strings, fragments, arbitrary paths, certificate pins or extra fields.
Use your actual approved public names, not examples. This is client routing only: never copy
cloudflared credentials, enrollment codes, session tokens, TOTP seeds or host configuration.
The signed root is public trust material distributed separately, not a secret config field.

```sh
P=scripts/release/.venv/bin/python
R=scripts/release/release.py
"$P" "$R" targets --root "$PUBLIC/1.root.json" --config "$PUBLIC/client-config.json" \
  --artifact "$PUBLIC/warroom-linux-amd64.deb" --version 1 --out "$STAGE"
# Repeat --artifact for warroom-windows-amd64.exe and warroom-darwin-arm64.dmg.
# These are the exact desktop-packages.yml outputs; approve the owner inventory first.
# Basenames must be unique. For server delivery, add --server-bundle <archive>
# (or use it instead of --artifact). Its signed custom.twr_server_release comes
# from internal/serverversion, not by running the candidate. Build the archive
# with deploy/server/bundle.sh <archive> <authenticated-public-root.json>.
# Deliver the original-basename server archive plus the finalized metadata folder.
# The installed CT tool authenticates them before unpacking or running anything;
# deploy/server/UPGRADE.md documents that flow and the persistent trust cache.
"$P" "$R" sign "$STAGE/metadata/targets.json" --key "$KEYS/targets.pem"
"$P" "$R" snapshot --root "$PUBLIC/1.root.json" --repo "$STAGE"
"$P" "$R" sign "$STAGE/metadata/snapshot.json" --key "$KEYS/snapshot.pem"
"$P" "$R" timestamp --root "$PUBLIC/1.root.json" --repo "$STAGE"
"$P" "$R" sign "$STAGE/metadata/timestamp.json" --key "$KEYS/timestamp.pem"
"$P" "$R" finalize --root "$PUBLIC/1.root.json" --repo "$STAGE"
```

For the next release choose a **new** staging directory, set `--version 2`, and add
`--previous <retained-authenticated-targets.json>` to `targets`. Archive the full signed metadata
and artifact set outside the web host; never reuse a version for changed bytes. Finalization
checks all role signatures, expiry, linked metadata hashes/versions and payload bytes before
writing versioned metadata. Never modify or re-sign children after linking a parent; rebuild in
a new stage if anything changes. Do not publish private working files or the whole ceremony folder.

Upload only hash-prefixed `targets/` files and versioned root/targets/snapshot metadata first;
retain older versioned files and every sequential root. Publish **timestamp.json last**, with
an atomic replacement. Disable intermediary caching of timestamp.json (and bound metadata
cache lifetimes). Keep the last good publication on any failed signing/upload step. Inspect
public content for secrets. The tool does not implement hosting credentials or upload automation.
Record a real reference-client download against the hosted HTTPS repository before owner use;
local tests do not prove proxy behavior, public TLS, cache policy or outage recovery.
Use the same native verifier that slice 44 will consume (from the repo root):

```sh
go run ./scripts/release -root "$PUBLIC/1.root.json" \
  -config "$PUBLIC/client-config.json" -cache "$VERIFY_CACHE" -target client-config.json
go run ./scripts/release -root "$PUBLIC/1.root.json" \
  -config "$PUBLIC/client-config.json" -cache "$VERIFY_CACHE" -target warroom-linux-amd64.deb
```

`$VERIFY_CACHE` is a dedicated, private persistent folder outside the repository. Keep it for
subsequent release/rotation checks; never clear it to fix a rejected update. The command only
verifies/downloads, never installs or starts an application. Run the native race tests with
`go test -race ./internal/releasetrust/...`; see its README for the slice 44 cache ownership and
configuration-validation contract.

## Root rotation and loss

Before root expiry, generate three replacement root public keys using the same offline ceremony.
Keep non-root role keys unchanged for this root-only rotation. Build a new public manifest and:

```sh
"$P" "$R" root --keys "$PUBLIC/next-public-keys.json" \
  --previous "$PUBLIC/1.root.json" --out "$PUBLIC/2.root.json"
# On separate custody stations, sign 2.root.json with TWO old and TWO new root keys.
"$P" "$R" sign "$PUBLIC/2.root.json" --key "$KEYS/root-a.pem"
"$P" "$R" sign "$PUBLIC/2.root.json" --key "$KEYS/root-b.pem"
"$P" "$R" sign "$PUBLIC/2.root.json" --key "$KEYS/new-root-a.pem"
"$P" "$R" sign "$PUBLIC/2.root.json" --key "$KEYS/new-root-b.pem"
"$P" "$R" check-root "$PUBLIC/2.root.json" --previous "$PUBLIC/1.root.json"
```

Publish `2.root.json` before releases using it; keep `1.root.json` and all intervening roots
forever. Subsequent releases use the new root. Verify a retained old client crosses the rotation
before retiring old custody media. Changing targets/snapshot/timestamp keys also needs an
operator-reviewed TUF role-key transition and fresh role metadata; this root-only recipe must
not be treated as that drill. If a root key is lost, the remaining two can rotate; if two are
lost/compromised, stop updates and perform a newly authenticated personal reinstall handoff.
Never disable thresholds, expiry or rollback checks to recover. These release keys are not the
account console-recovery key; do not reuse either key set.

## Evidence and remaining gates

`scripts/release/test.sh` runs ten unittest cases using freshly generated, temporary keys:
valid signed package/config downloads; tamper rejection at all three release roles; expiry at
all four roles; corrupt package/config leaving no download; timestamp/snapshot/targets rollback
across client restart preserving trusted targets; and dual old/new threshold root rotation.
The encrypted-PEM signing path and owner-only permission rejection are exercised too.

These tests do not invoke the desktop updater, OS trust dialogs, a hosted repository, hardware
custody or a real offline ceremony. Do not mark those gates passed. Run this test command in
release review; the release-tools CI job also executes these tests with the hashed Python dependencies.
The normal Go race shards execute the native verifier tests.
The plan's slice boundary holds: production offline ceremony requires Christopher, and native
client integration/clean-machine installation remain slices 44/45.
