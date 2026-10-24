# Encrypted league disk and backup unlock

**Preproduction. Real Proxmox cold-boot/deputy proof is still required.**
The Alpine CT remains on OpenRC. Proxmox's systemd owns encrypted storage and
container startup. This does not switch the CT runtime or claim Alpine supports
systemd credentials.

## What installation does

Run the existing `proxmox.sh` installer with the trusted release bundle. It now
requires the Proxmox host's TPM 2.0 (`/dev/tpmrm0`) and creates a new, dedicated
8 GiB LUKS2 file. It never formats an existing disk, migrates a running CT or
reuses existing league storage. The CT's 4 GiB root disk lives entirely inside
that encrypted filesystem: databases, WAL/SHM, configuration, TLS/TOTP/Tunnel
credentials, temporary files and logs. Host release binaries, container/network
configuration and the sealed key are not confidential league data.

The installer explains and confirms a backup passphrase of at least 16
characters, including two paper copies: commissioner and sealed deputy copy.
It tests that passphrase before starting the CT. No password is supplied on a
process command line, put in shell history, printed in logs or saved on disk.
Short-lived key/passphrase files use private `/run` tmpfs directories and are
removed on both success and failure.

It explicitly asks permission to disable **host-wide swap**. Declining stops the
installation. Configured swap units and swap.target are masked, swapoff is
required to succeed, and every disk-unlock attempt refuses active swap. The CT
also has swap=0. Do not re-enable host swap while this deployment is in use.
This avoids pretending that CT swap=0 alone protects the host's helpers. An
administrator can defeat this policy; there is no protection claim against root.

## Flow and ownership

| Boundary | Contract / effects / failure |
|---|---|
| `install-storage.sh` → cryptsetup | Fresh image only; random 32-byte automatic key in LUKS slot 0, commissioner passphrase in slot 1. Prove both before deployment. Failed provisioning retains the image; never silently reformat it. |
| `cmd/twr-unlock` → TPM | `github.com/anatol/clevis.go` binds/unbinds the standard Clevis JWE, SHA-256 PCR 7. A different TPM or changed measured security state refuses automatic recovery. No custom key protocol. |
| Host systemd → `storage.sh` | `LoadCredential` delivers only the sealed JWE in `CREDENTIALS_DIRECTORY`. The helper streams the recovered key directly to cryptsetup. No secret argument/environment variable. Unlock/mount must succeed before container startup. |
| Host storage → Proxmox | Private image `/var/lib/twr-storage/<id>.luks`, mounted at `/mnt/twr-<id>`, registered as `twr-<id>` with mountpoint checking. Missing/locked storage is not an empty writable fallback directory. No namespace sandbox on this privileged mount helper: its mount must be visible to Proxmox. The CT service retains its existing sandbox. |
| `twr-container@<id>` → CT | Requires/BindsTo/PartOf storage; reverse stop ordering shuts down the CT before unmount/close. Proxmox onboot stays off so it cannot race unlock. The installer exercises this unit before enabling it. |
| CT OpenRC → server | Private state directory and umask 077. DB/WAL/SHM/server log 0600. File-based stdout/stderr stays inside the encrypted disk; hourly log rotation keeps three copies with private permissions. These diagnostic logs are not the authoritative security audit. |
| `twr-unlock-league` → backup recovery | Console prompts, verify backup before stopping anything, unlock with slot 1, repair slot 0/JWE on the current TPM, and start the dependent container automatically. Slot 1 is never removed. An interrupted slot-0 repair still leaves backup recovery. Without a usable TPM, this boot can run but the message says backup unlock is required next reboot. |

Dependency identity/source checked against the publisher's README, source and
module at `anatol/clevis.go` commit `8cd07c1db068fff94299b5c83eb4b39e124e2e9e`;
module checksum and transitive versions are in go.sum. It is a library, not a
LUKS installer. This project uses cryptsetup for LUKS and the library only for
TPM binding. TPM-only is the selected Q1 alternative; no Tang server or key
custody service is required. PCR 7 binding is not proof of a complete verified
boot chain or resistance to malicious firmware.

## If the league does not come back after a restart

From the Proxmox console (LAN/VPN administration, never a public web route), run:

```sh
twr-unlock-league
```

Enter the container number and passphrase from the sealed paper. The tool
starts the container and, where possible, repairs automatic unlock itself; no
key copying, crypttab editing or service configuration is needed. Keep the paper
copies afterwards. This unlocks storage, not an owner's account. If both copies
and the original TPM are lost, the data cannot be recovered from this disk.

## Evidence and release gate

Executed against this candidate:

- `go test -race ./cmd/twr-unlock`: invalid modes, short keys, empty/oversized and
  corrupt bindings return no key bytes.
- `bash deploy/server/test-storage.sh`: disposable Alpine with a software TPM,
  real Clevis library and real LUKS2 keyslots. Automatic key roundtrip succeeds;
  changed PCR and replacement TPM return no bytes; wrong passphrase fails;
  deputy passphrase still works and refused operations leave disk bytes intact.
  This test has no host devices or privileges and cannot prove a mounted disk.
- `bash deploy/server/test-alpine.sh`: real hosted server/OpenRC sandbox; assert
  0600 DB, live WAL/SHM and log, rotate log and assert 0600 copies. Existing TLS,
  filesystem and simulated Cloudflare regression coverage is retained.

Still **unverified**: run the installer on a disposable Proxmox node, completely
power-cycle it, observe automatic host unlock and CT readiness, stop
`twr-storage@<id>` and prove the CT stops and its encrypted filesystem cannot be
read while locked, then cold-boot with TPM unavailable and have the deputy use
`twr-unlock-league`. Check host `/proc/swaps` is empty, CT swap=0, systemd
credential delivery/cleanup, both DB sidecars/log permissions, and wrong backup
passphrase leaves the stopped/locked state unchanged. Test TPM-replacement repair
and interrupted repair. A CT restart and a software-TPM roundtrip are not this
hardware proof. No real owners until this gate passes.

No protection claim against Proxmox/CT root, an unlocked/running compromised
server, old plaintext copies or a hostile boot chain. This is not an encrypted
backup: host loss destroys this disk. Off-host matched encrypted backup,
retention and host-loss recovery remain slices 36–37; do not treat ordinary
plaintext Proxmox dumps as protected backups.

## Debt left

- `deploy/server/proxmox.sh`, `install-storage.sh`, `storage.sh`,
  `recover-storage.sh`, and host units: actual Proxmox/cold-power/deputy and
  systemd-credential acceptance remain unverified because no disposable Proxmox
  host was available. This is an evidence gap, not a manual setup step deferred
  to the commissioner.
