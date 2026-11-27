# Season server install, swap and undo

Scripts for the bare-metal Alpine Season server (`192.168.1.162`, OpenRC). Rules: `CLAUDE.md` "Environments".
Window: **Saturday 18:00–23:59 America/Chicago**, and only with Christopher's explicit OK each time.
Every script prints the exact command to approve and every remote command before running it.

- Exit codes: `0` done or dry run · `1` step failed · `2` bad arguments or artifact refusal · `3` refused by guard (nothing sent).
- Offline tests: `python3 -m unittest discover -s deploy/season -v` (also run by `make test-deploy`).

## First install (`season-install.sh`)

Run from the Beelink (pure POSIX `sh`). The target must always be typed explicitly — there is no default host:

```sh
# Plan only (sends nothing):
sh deploy/season/season-install.sh \
  --target root@192.168.1.162 \
  --version v107 \
  --bundle /path/to/bundle.tar.gz \
  --sha256 <64-hex-sha256> \
  --dry-run

# Real install (only inside Saturday 18:00–23:59 Chicago, after Christopher approves the printed line):
sh deploy/season/season-install.sh \
  --target root@192.168.1.162 \
  --version v107 \
  --bundle /path/to/bundle.tar.gz \
  --sha256 <64-hex-sha256> \
  --christopher-ok
```

Guard rules for `season-install.sh` and `season-uninstall.sh`:

| Time (America/Chicago) | `--dry-run` | `--christopher-ok` | Result |
|---|---|---|---|
| Outside Sat 18:00–23:59 | No | No | Exit 3 (refused) |
| Outside Sat 18:00–23:59 | No | Yes | Exit 3 (refused — real install only inside Sat window) |
| Inside Sat 18:00–23:59 | No | No | Exit 3 (refused — needs `--christopher-ok`) |
| Inside Sat 18:00–23:59 | No | Yes | Runs against `--target` |
| Outside Sat 18:00–23:59 | Yes | No | Exit 3 (refused) |
| Outside Sat 18:00–23:59 | Yes | Yes | Exit 0 (prints plan only, sends nothing) |
| Inside Sat 18:00–23:59 | Yes | No / Yes | Exit 0 (prints plan only, sends nothing) |

Before touching anything on the target, `season-install.sh` verifies Alpine 3 `x86_64` as root and refuses if any TheWarRoom path or user already exists (`/opt/twr`, `/var/lib/twr`, `/var/backups/twr`, `/etc/twr`, `/etc/init.d/twr`, `/etc/init.d/twr-server`, `/etc/logrotate.d/twr`, `/etc/periodic/hourly/twr-logrotate`, `/usr/local/bin/twr-server`, `/usr/local/libexec/twr-sandbox`, or system user `twr`). Pre-existing host tuning such as `/etc/sysctl.d/10-twr.conf` is left untouched.

### Layout created (matches `season-swap.sh`)

- `/opt/twr/versions/<version>/` — extracted `twr-server` (`0755 root:root`) and `endpoint-registry.csv` (`0640 root:twr`); fixed archive members only, never raw `tar -xf .` as root.
- `/opt/twr/current` — symlink to `/opt/twr/versions/<version>` (flipped by `season-swap.sh` on upgrades).
- `/opt/twr/libexec/` — Season `twr-sandbox`, `twr-backup-setup`, and `twr-status` wrappers targeting `/opt/twr/current/twr-server`, port `5199`, and OpenRC service `twr`.
- `/var/lib/twr` (`0700 twr:twr`) — private SQLite state (`server.db` `0600 twr:twr`) and `server.log` (`0600 twr:twr`).
- `/var/backups/twr` (`0700 root:root`) — pre-swap local archives written by `season-swap.sh`.
- `/etc/twr` (`0750 root:twr`) — origin TLS cert/key (`origin.pem` `0640 root:twr`, `origin.key` `0600 twr:twr`), 32-byte `authenticator.key` (`0600 twr:twr`), colocated `endpoint-registry.csv` (`0640 root:twr`, required beside `backup.json` by `cmd/twr-server/backup.go`), and optional `public-host` (`0644 root:root`).
- `/etc/init.d/twr` — OpenRC `supervise-daemon` service named `twr`, running as `twr:twr` with `umask 0077`.
- Health probe: `https://127.0.0.1:5199/app/` verified with `--cacert /etc/twr/origin.pem` (HTTP `200` with `Host: <public-host>` when `--public-host` is configured; HTTP `404` when local-only, matching `internal/serverhttp/hosted.go:142-146`).

### Hardening reused vs dropped from `deploy/server/install.sh`

Reused on bare metal:
- Separate non-login service account (`twr:twr`), private `0700` state directory, `0600` database/keys/logs, `umask 077`.
- `bubblewrap` sandbox (`--unshare-all --share-net --die-with-parent --new-session --cap-drop ALL`, read-only `/usr`, `/lib`, `/bin`, `/opt/twr`, `/etc/twr`, `/etc/ssl/certs`, `/etc/resolv.conf`, disposable `/tmp` and `/run`, single writable `/var/lib/twr`).
- Private P-256 origin TLS keypair (`CN=localhost`, SAN `DNS:localhost,IP:127.0.0.1`) and 32-byte authenticator key.
- Private hourly logrotate (`/etc/logrotate.d/twr`, `create 0600 twr twr`) with `crond`.
- Required Alpine packages including `curl` (absent on the bare laptop) and `bubblewrap`.

Dropped from `deploy/server/install.sh` and why:
- **Proxmox provisioning (`deploy/server/proxmox.sh`) and Proxmox CT firewall:** The laptop is bare-metal Alpine on Wi-Fi (`wlan0`, DHCP), not a Proxmox container. Loopback binding (`127.0.0.1:5199`) and the home router/LAN boundary replace CT firewalling; add a host firewall (e.g. `nftables`) separately if desired.
- **Unprivileged LXC CT isolation and CT zero-swap:** No container layer exists on bare metal; process containment relies on the `twr` user and `bubblewrap` namespaces. Host swap and disk encryption are not modified (`deploy/server/STORAGE.md`).
- **Deleting `openssh-server`:** `install.sh` removes SSH because Proxmox uses `pct enter`. Bare metal is administered from the Beelink over SSH, so `openssh-server` must stay installed.
- **Cloudflare Tunnel (`cloudflared`, `twr-tunnel`, `cloudflare.sh`):** Deferred until Christopher decides local-only vs Cloudflare Tunnel. Without a Tunnel, `-trusted-proxy` is omitted so direct loopback TLS checks work without `CF-Connecting-IP`.
- **Pinning `root.json` into `/etc/twr-release/root.json` and installing `twr-upgrade`:** Season upgrades use `season-swap.sh` from the Beelink, not in-host TUF `twr-upgrade`, and no production TUF root ceremony exists yet (`docs/delivery/Release_Trust.md`).
- **Inherited host NTP:** Proxmox CTs inherit host `chrony`. Bare metal must run its own NTP daemon (`chrony` or `ntpd`) because `twr-server hosted` checks kernel NTP sync (`adjtimex`, `STA_UNSYNC` clear) before admitting timed actions (`cmd/twr-server/README.md`).

### Artifact approval (`LAB-ONLY` vs Season bundle)

`deploy/lab/lab-bundle.sh` calls `deploy/server/bundle.sh` with a throwaway Lab TUF root (`~/fleet/lab-release/LAB-ONLY-root.json`) and names the archive `twr-server-LAB-ONLY-r<N>-<sha>.tar.gz`. The compiled `./twr-server` binary inside is built by the exact same `deploy/server/goreleaser.yaml` config (`CGO_ENABLED=0`, no `practice` build tag) that any Season `bundle.sh` build uses.

Per `CLAUDE.md` Environments rule 1 ("Same binary, no rebuild between Lab and Season"), rebuilding from source between Lab and Season would break binary identity. Instead:
- `season-install.sh` refuses any `LAB-ONLY` filename by default (exit 2) so a Lab tarball can never be deployed by accident.
- Once a Lab build passes its Lab check (health 200, both leagues load, live screen renders) and its SHA-256 is verified against the Lab release's signed `metadata/*targets.json`, Christopher can explicitly approve promoting that exact binary with `--allow-lab-artifact --sha256 <digest>`. `season-install.sh` extracts only `./twr-server` and `./endpoint-registry.csv` and never installs the throwaway `root.json`.
- If Christopher prefers a distinct Season-named bundle (`twr-server-season-r<N>-<sha>.tar.gz`) signed with a production root, complete the offline ceremony in `docs/delivery/Release_Trust.md` first and pass that `root.json` to `deploy/server/bundle.sh` (building once, testing on Lab, then promoting to Season).

### Off-site backups (Cloudflare R2)

Fresh `twr-server hosted` installs start with **league and sign-in writes paused** until `/etc/twr/backup.json` is configured and the first encrypted upload passes full read-back verification (`cmd/twr-server/backup_hosted.go`, `deploy/server/README.md`). No secrets or defaults are ever embedded in scripts.

To open the Season server for writes, Christopher must supply five values following `deploy/server/BACKUPS.md` (never stored in git or passed in argv):
1. Offline-generated **public** age recipient (`age1...` from `age-keygen -o warroom-restore-key.txt` on Christopher's own computer; the private `AGE-SECRET-KEY-...` never touches the laptop).
2. Dedicated Cloudflare **R2 bucket name** (Standard storage, public access disabled; separate from any Lab bucket).
3. Bucket **S3 API endpoint hostname** (hostname only, no `https://` or path).
4. Bucket-scoped R2 **Access Key ID** (Object Read & Write, scoped to this bucket only).
5. Bucket-scoped R2 **Secret Access Key** (typed into hidden prompt).

On the Season laptop as root, run `sh /opt/twr/libexec/twr-backup-setup` (the Season equivalent of `deploy/server/backup-setup.sh`, adapted for `/opt/twr/current/twr-server` and `rc-service twr`), then confirm `Backups` reads `OK` via `sh /opt/twr/libexec/twr-status`.

### Undo (`season-uninstall.sh`)

Removes only what `season-install.sh` installed, using the same Saturday window + `--christopher-ok` guard and explicit `--target`:

```sh
# Remove binary, wrappers, config, OpenRC service and logrotate; KEEP /var/lib/twr and /var/backups/twr:
sh deploy/season/season-uninstall.sh --target root@192.168.1.162 --christopher-ok

# Rehearsal teardown (also deletes /var/lib/twr, /var/backups/twr and system user/group twr):
sh deploy/season/season-uninstall.sh --target root@<rehearsal-host> --purge-data --christopher-ok
```

Never removes `/etc/sysctl.d/10-twr.conf`, `openssh-server`, `crond`, or system packages.

### Rehearsal

See [REHEARSAL.md](REHEARSAL.md) for the exact step-by-step rehearsal on a throwaway Alpine machine built from the Lab golden template.

### Known swap-script alignment before first real Saturday swap

`deploy/season/season-data.sh` line 54 currently probes `curl -fsS http://127.0.0.1:5199/app/`. Because `twr-server hosted` enforces TLS (`internal/serverhttp/hosted.go:121`) and gates `/app/` on `-public-host` (`:142-146`), update `season-data.sh` to probe `https://127.0.0.1:5199/app/` with `--cacert /etc/twr/origin.pem` (and `Host:` when a public host is set) before running the first real swap.

## Decisions for Christopher

1. **Access: Cloudflare Tunnel vs local-only**
   - *Recommendation:* Start **local-only** (`127.0.0.1:5199`, no `--public-host`, no `cloudflared`) for the throwaway rehearsal and initial Dec 5 boot/check; add Cloudflare Tunnel (`cloudflared` + dedicated hostname via `--public-host` + `/etc/twr/trusted-proxy`) only once backups verify and you are ready to open phone/owner access to the Season server.
2. **Off-site backups: dedicated R2 bucket**
   - *Recommendation:* Create a dedicated Season R2 bucket (e.g. `warroom-season-backups`, never shared with Lab) and a dedicated bucket-scoped token, plus confirm the offline `age1...` recipient you want to reuse or mint. Without R2 setup, the server stays read-only/paused by design.
3. **Artifact approval & naming (`LAB-ONLY` vs Season bundle)**
   - *Recommendation:* For Dec 5 rehearsal season, **promote the exact Lab-verified tarball by SHA-256** using `--allow-lab-artifact --sha256 <digest>` (keeps `CLAUDE.md` rule 1: same binary, no rebuild; never pins the throwaway Lab `root.json`). Schedule the offline TUF key ceremony (`docs/delivery/Release_Trust.md`) before admitting real owners outside rehearsal.
4. **Align `season-data.sh` health probe to HTTPS**
   - *Recommendation:* Approve a small follow-up PR updating `deploy/season/season-data.sh:54` from plain `http://` to `https://127.0.0.1:5199/app/` with `--cacert /etc/twr/origin.pem` before the first real Saturday swap (Dec 12).

---

## Season server swap (`season-swap.sh`)

Saturday-night swap of the Season server (Alpine, OpenRC, `192.168.1.162`). Rules: `CLAUDE.md` "Environments".
Window: Saturday 18:00–23:59 America/Chicago, and only with Christopher's OK each time.
A real swap always needs `--christopher-ok`; a dry run needs it only outside the window (else exit 3).
Every run prints the exact command to approve.

- Plan only, sends nothing: `deploy/season/season-swap.sh <version-dir-name> --dry-run`
- Real swap, after his yes: `deploy/season/season-swap.sh <version-dir-name> --christopher-ok`
- `<version-dir-name>` is a folder under `/opt/twr/versions/`, for example `v51`.

Original flow (now stopped for a consistent backup): back up `/var/lib/twr` to `/var/backups/twr`; check the version folder exists; read the current
link; point `/opt/twr/current` at the new version; start `twr`; health check `curl` on port 5199.
It deletes nothing. If a step fails after the link flip, it prints the rollback command.

### Rollback: data and version together

Use the exact rollback command printed by the swap, with a separate Christopher approval:
`deploy/season/season-swap.sh <swap-stamp> --rollback --christopher-ok`.
Add `--dry-run` to print the operation without contacting the server.
The stamp identifies `/var/backups/twr/twr-data-<swap-stamp>.tgz` and its `.record` file.
Keep both; the record binds the archive checksum, previous version and swapped-in version.
Old backups without records cannot use this procedure.

The swap stops `twr` before taking its backup, so SQLite files are copied at rest.
Rollback checks the record, archive and previous folder before stopping anything. It stages
that archive, stops `twr`, moves current data to `/var/lib/twr.before-rollback-<swap-stamp>`,
restores the backup and previous version link, then starts `twr` and checks HTTP health.
It does not delete either data copy or the archive. Run only one swap/rollback at a time.
If a command fails, stop and inspect: the service may remain stopped; do not improvise
or delete saved data. HTTP success does not prove league recovery: verify sign-in and a
league read before accepting the rollback. This is upgrade recovery, not first-install recovery.

Offline integration tests use overridden `SEASON_APP_ROOT`, `SEASON_DATA_DIR` and
`SEASON_BACKUP_DIR` (and `SEASON_ETC_DIR` / `SEASON_INIT_DIR` for install/uninstall) with fake
SSH, OpenRC and curl; defaults remain the Season paths.
`make test-deploy` runs all Season deploy tests without contacting a host.
