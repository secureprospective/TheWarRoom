# Alpine league-server deployment (slices 34–36)

Encrypted Cloudflare R2 recovery and CT-only setup: [BACKUPS.md](BACKUPS.md).
Replacing a lost server, and signing everyone out after an incident: [RESTORE.md](RESTORE.md).
The daily look at the server's health, and what to do when away: [CHECKLIST.md](CHECKLIST.md).
New installations serve with changes paused until that setup verifies an off-site set.

**Preproduction. Not approved for real owners.** League files are private inside
the container, not disk-encrypted; see [STORAGE.md](STORAGE.md) for the limits.
This is an Alpine 3.23 amd64,
OpenRC, unprivileged Proxmox CT installer. It deliberately replaces the plan's
systemd service and separate reverse-proxy assumption. The Go origin already
speaks TLS; cloudflared verifies its private origin certificate directly. No
`noTLSVerify`, inbound forwarding, privileged CT, SSH or public admin listener.

## Flow and ownership

| Boundary | Contract / effects / failure |
|---|---|
| `bundle.sh` → release bundle | GoReleaser OSS builds the static Linux binary. Official GoReleaser and Cloudflare assets have pinned SHA-256 digests. Package scripts and the existing preview registry alongside them; no state or secrets. |
| `proxmox.sh` → Proxmox | Validate settings, require a running Proxmox firewall and synchronized host chrony, select/download Alpine template, allocate a fresh CT on the selected VLAN. Never overwrite another CT/firewall. On failure stop and retain this CT, without enabling boot. |
| Proxmox firewall → CT | Default-deny ingress/egress. DHCP is the bootstrap exception; IPv6 egress is denied entirely. IPv4 private/shared/link-local/reserved/multicast and every directly connected host subnet are denied before public HTTPS/Tunnel permits. DNS is restricted to public resolver 1.1.1.1. No CT-to-LAN DNS dependency. |
| `install.sh` → CT | Install OpenRC/bubblewrap; remove any template SSH server; create separate service identities, private SQLite state, TLS and authenticator keys. Refuse reinstall before modifying state. Start real hosted mode and verify its TLS/application response. |
| OpenRC → `twr-sandbox` → Go | Non-root, no capabilities, private PID/user/mount/IPC/UTS namespaces. Only persistent writable mount is `/var/lib/twr`; `/tmp`, `/run` and minimal `/dev` are disposable. No host root/home, Tunnel credential or external filesystem is mounted. Share the CT network only for loopback TLS; Proxmox owns lateral-network containment. |
| `cloudflare.sh` → Cloudflare API | API token is an owner-only transient file/header, not a process argument. Create a locally managed Tunnel and new proxied DNS record; refuse an occupied hostname. Install its separate scoped credential, local ingress rules and origin CA automatically. Delete API token input on exit. |
| cloudflared → origin | Only the configured hostname's `/v1/` routes reach `https://127.0.0.1:8443`, verified as `localhost` with the installed CA. Everything else is an edge 404. Origin trusts exactly `127.0.0.1` for `CF-Connecting-IP`; missing/forged peer metadata is refused. Metrics/readiness bind `127.0.0.1:20241`, never public ingress. |
| `restore.sh` → CT (`twr-restore`) | Console answers → `twr-server restore` as the service user → newest usable R2 set, checkpoint compared, sign-ins reset in the staged copy → installed database, authenticator key and new backup settings. Refuses a target holding a league; a failure leaves the fresh install running. [RESTORE.md](RESTORE.md). |
| `status.sh` → CT (`twr-status`) | Console dashboard: `twr-server status` as the service user reads the owner-only `/var/lib/twr/status.sock` (backups, history chains, resets), the origin's TLS readiness, cloudflared `/ready`, clock and disk. Loopback and socket only; nothing is sent anywhere. [CHECKLIST.md](CHECKLIST.md). |
| `reset-sign-ins.sh` → CT (`twr-reset-sign-ins`) | Incident reason → new auth epoch for every account; passwords, authenticators and league unchanged. |
| Installer → reboot | Reboot CT; require both OpenRC services and cloudflared's connected `/ready` before declaring success/enabling on-boot. A dead Tunnel is not a successful installation. |

Administration is the Proxmox console (`pct enter <number>`) through the existing
LAN/VPN-restricted Proxmox management plane. Host administration is not an account
or league role. CT ingress rules allow only DHCP replies; loopback never crosses
the Proxmox firewall. No SSH port, recovery console, metrics or host-management
route is published through the Tunnel. Public account/TOTP/device routes are
owner-facing APIs, not host administration.

## One-command installation

From an already provisioned Proxmox host, with the trusted release bundle and
this installer delivered together:

```sh
bash deploy/server/proxmox.sh ./twr-server-alpine-amd64.tar.gz
```

Prompts collect the unused container number, bridge, isolated VLAN, storage,
Cloudflare zone/new hostname, whether a lost server is being replaced and API token (hidden input). Token permissions are
Cloudflare Tunnel Edit for the account, plus DNS Edit and Zone Read for that zone.
The installer creates the Tunnel, TLS material, DNS record and services itself;
there is no dashboard ingress editing, certificate copying or manual init setup.
Complete the documented R2 setup inside the CT before moving an active league.
It does not change the host's management network, firewall policy or storage
security. A pre-existing VLAN/DHCP network, firewall-enabled Proxmox node, template
storage and synchronized host clock are infrastructure prerequisites, not claims
that this script can safely reconfigure a whole homelab. No host packages are installed;
JSON tooling runs only inside the container.

It installs 2 cores / 1 GiB RAM / 4 GiB disk / no CT swap, unprivileged with
`nesting=1` for bubblewrap's namespaces. The firewall allows public TCP 443 and
TCP/UDP 7844 plus DNS; it does not allow arbitrary LAN, NTP, SSH or IPv6 access.
Alpine's package repositories are switched to HTTPS before package installation.
If the existing firewall/clock is unavailable, installation stops, rather than
quietly weakening containment or trusting a fake userspace NTP client. A CT
cannot synchronize its own kernel clock: it inherits Proxmox's NTP state.

Do not rerun over an existing installation. Failure retains the stopped CT and
its authoritative state for diagnosis. If Cloudflare creation succeeded before a
later failure, its DNS/Tunnel identifiers are recorded in `/etc/twr-cloudflare.ids`. Failure automatically
stops cloudflared and attempts to delete only this install's DNS and Tunnel.
If the API is unreachable during cleanup, identifiers remain in the stopped CT
for recovery; they are not secret. On the CT, supply a fresh owner-only API input
file with the same fields and run `sh /root/twr/cloudflare.sh --rollback <input>`.
The cleanup entry retries only recorded resources and reports failure plainly;
normal `cloudflare.sh` entry also completes pending cleanup before creation.
Successful deletions are removed from the ledger; already-absent resources count
as cleaned. An existing DNS record is never overwritten, except when replacing a
lost server: then only a single CNAME to a disconnected `twr-…` Tunnel in the same
account is repointed in place and that old Tunnel deleted ([RESTORE.md](RESTORE.md)).

## Build and test the candidate

Inside this worktree, with Go on PATH:

```sh
bash deploy/server/bundle.sh .scratch/server.tar.gz
bash deploy/server/test-alpine.sh
```

The builder uses checksum-pinned GoReleaser OSS 2.19.0 and cloudflared 2026.10.3,
both from their publishers' GitHub repositories. `goreleaser.yaml` is build
configuration, not a complete signed release pipeline. `bundle.sh` assembles the
server, Tunnel binary and deployment files. Backup dependencies and their source checks are documented in BACKUPS.md. TUF trust/distribution is slice 43, not provided by a checksum
printed next to an unsigned archive.

The test uses a digest-pinned disposable Alpine Docker container, with an init
reaper and privileges confined to that container for nested namespace setup. Its
only host mount is this candidate read-only; it does not use host networking,
production databases, Cloudflare credentials or real MFL calls. It executes:

- GoReleaser-built real hosted binary, OpenRC start/restart and verified TLS;
- reinstallation refusal with unchanged key/database digests;
- 0600 database/key modes, successful state write, refused `/usr` and config
  writes/mounts; no home/Tunnel credential visibility inside the service sandbox;
- refusal of origin TLS via the CT's non-loopback interface and of wrong CA;
- the console dashboard `twr-status` after a restart: owner-only socket, healthy
  server, the container's real faults (no backups, no Tunnel), a stopped server,
  and refusal when not run as root;
- automatic Cloudflare configuration against a **simulated API**, parsed/validated
  by the real pinned cloudflared binary; native routes match, admin/metrics/wrong
  hostname do not, occupied DNS is preserved, service cannot read Tunnel secrets;
  explicit rollback deletes both newly created resources and injected DNS failure
  automatically deletes the newly created Tunnel; an API outage retains identifiers,
  and a second run cleans exactly those resources without touching occupied DNS.

## Evidence and remaining gates

Executed 2026-10-24: Alpine test above passed with OpenRC 0.63-r1 and bubblewrap
0.12.0-r0. Re-run 2026-10-25 with the dashboard checks: passed. `shellcheck` checks the shell files separately. This is **not** the full
slice proving test: Docker privileges are not Proxmox's unprivileged LXC AppArmor
profile, and service restart is not a CT cold boot.

Unproven: actual `pveam`/`pct` provisioning, firewall compile/enforcement and real
CT reboot, nested user namespaces under Proxmox, host clock propagation, real
Cloudflare API/DNS/Tunnel connectivity, and IPv4/IPv6 lateral denial against a
known reachable disposable peer. Those require a disposable Proxmox node/VLAN
and test Cloudflare zone; neither was available to this worktree. Do not accept
RB1 or the slice-34 deployment gate from the Docker test alone. In particular,
Cloudflare shared-edge scans cannot prove origin isolation.

## Debt left / scope boundaries

- `proxmox.sh`: real Proxmox isolation/reboot acceptance is not yet established;
  the installer fails closed on missing prerequisites but cannot substitute for
  that host-level proving test.
- First-league activation belongs to desktop→server transfer (slices 39–41),
  which moves the commissioner's league and grants to the server. `install.sh`
  initializes empty hosted storage and directs the commissioner to that next step;
  deployment must not invent accounts or ledger grants.
- Private container files (35) are described in [STORAGE.md](STORAGE.md).
  Release integrity (43) remains its own gate; GoReleaser OSS builds the artifacts.
- Health (38) is the console dashboard in [CHECKLIST.md](CHECKLIST.md). The 10-25
  ruling is no outbound notifications, so ntfy is not used. Gatus was considered and
  not adopted: a web dashboard needs a port reachable from the VPN into the isolated
  VLAN, which means network changes outside this installer, while the Proxmox
  console is already the LAN/VPN-only admin door. There is still no public
  `/healthz` or status route; the Tunnel reaches only the owner-facing `/v1/` API.

The origin certificate is private, pinned only to this local Tunnel hop, and has
a long lifetime to avoid an unattended-expiry outage. It is not desktop/public
trust material. Public TLS and renewal belong to Cloudflare. This slice makes no
claim against Proxmox root, CT root, a running compromised process, host swap or
unencrypted backing storage. No owners connect until the integrated release gates.

## Updating an installed server

Update the server before desktop apps. Use [UPGRADE.md](UPGRADE.md) and the new bundle's
`upgrade.sh`; never rerun the installer or replace only the binary over newer data.

Staging drills (throwaway Proxmox VM, real installer and Tunnel): [STAGING.md](STAGING.md).
