# TWR lab (nested Proxmox, real installer, real Tunnel)

The lab is VM 503 `twr-lab` (Proxmox VE 9.2, nested) at `192.168.1.182`. It runs the **real**
`proxmox.sh` from a release bundle with a **real** Cloudflare Tunnel, so drills test the path an owner
runs. It never holds a real league or real owners. It replaces the retired QEMU staging VM.
Root SSH from the Beelink: `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@192.168.1.182`;
check `hostname` = `twr-lab` first (every lab script refuses to run anywhere else).

| CT IDs | Use |
|---|---|
| 110 `lab-gw` | Network gateway. Never touch. |
| 200–299 | Lab league servers (`twr-up` / `twr-down`) |
| 9000–9099 | Template builder (9000) and smoke (9001) |

Storage: CT disks on `lab-thin`, templates on `local`.

## Network: why a gateway CT

Servers sit on bridge `vmbr1`, VLAN `51`, DHCP `10.51.0.100–150` from `lab-gw` (CT 110, `lab-gw.sh`),
NATed out through its home-LAN leg and barred from `192.168.0.0/16`. The old staging design had the
PVE host itself route and NAT the same VLAN it also bridges; netfilter's bridge pass then fixed the
NAT decision before the routed pass, so replies came back `UNREPLIED` with an un-NATed source on
`vmbr0` and DHCP was marked `INVALID` (proven 11-08). A real host has its router upstream of the
bridge, so the lab now does too: the host only bridges, `lab-gw` routes. A ping to `10.51.0.1` from a
server failing is **normal** (the installer's CT firewall drops it); test with HTTPS to the internet.

## Golden template

`deploy/lab/golden-build.sh` (on the lab, as root) builds
`local:vztmpl/alpine-3.23-twr-golden_<date>_amd64.tar.gz` plus `.sha256` and `.manifest`: Alpine 3.23,
fully upgraded, the installer's packages preinstalled, no OpenSSH at all, no secrets (install.sh still
mints every key, the database and the release pin per server). `deploy/lab/golden-smoke.sh` proves it:
creates CT 9001, checks network, no sshd/host keys, packages, nothing upgradable, then removes it.
`proxmox.sh` uses it only when `TWR_TEMPLATE=<file>` is set, after checking `<file>.sha256`.

## Bundle (on the Beelink)

```sh
bash deploy/lab/lab-bundle.sh   # → .scratch/lab/twr-server-LAB-ONLY-r<release>-<sha>.tar.gz
```

A throwaway **LAB-ONLY** TUF root (3 root keys at 2-of-3, one per other role, same shape as the real
ceremony) is made once in `~/fleet/lab-release/` (0700) and reused; never the real root. The venv lives
in `.scratch/lab-release-venv` from the hash-locked `scripts/release/requirements.txt`. Copy the bundle
to `/root/bundles/` and `deploy/lab/twr-*` to `/root/twr-lab/` on the lab.

## Up, list, down (on the lab, as root)

```sh
ssh lab /root/twr-lab/twr-up 201 /root/bundles/<bundle> < ~/.config/thewarroom/cloudflare-token
ssh lab /root/twr-lab/twr-list
ssh lab /root/twr-lab/twr-down 201 < ~/.config/thewarroom/cloudflare-token
```

- `twr-up` runs the `proxmox.sh` inside the bundle with `TWR_TEMPLATE` (newest golden), answering its
  prompts: CT, `vmbr1`, `51`, `lab-thin`, `local`, `secureprospective.com`,
  `twr-lab-<ctid>.secureprospective.com`, `no`, token. The token comes from stdin only. Prints seconds.
- `twr-list`: CT, hostname, bundle, state, cloudflared `/ready`, memory (the CT cgroup's
  `memory.current` read on the host; `free` in a CT shows the host), disk used.
- `twr-down` stops cloudflared and deletes **only** the Tunnel and CNAME ids the CT recorded in
  `/etc/twr-cloudflare.ids` (`cloudflare.sh --rollback`, token input in tmpfs), then removes the CT.
  Refuses IDs outside 200–299.

Cloudflare: the zone is production. Lab names are `twr-lab-<ctid>` only; tunnels are named
`twr-twr-<ctid>`. Never touch `command-center`, `homelab-nextcloud`, `homelab-postiz`, `twr-twr-152` or
`twr-staging`. End every run with a read-only listing showing no `twr-lab-*` record or tunnel.

## Troubles and fixes

| # | Trouble | Cause | Fix | Kind |
|---|---|---|---|---|
| 1 | `tar: /var/lib/lxc/<id>/rootfs: Cannot open: Permission denied` | Installer `umask 077` reached `pct create` | `pct create` under `umask 022` (#216) | Product bug, fixed |
| 2 | Valid pasted token: "does not look right" | Bracketed-paste markers / whitespace in the hidden `read` | Installer strips them (#216) | Product bug, fixed |
| 3 | `upgrade.sh` ran a tampered candidate as root | Candidate not authenticated before execution | TUF verification against the pinned root; candidate never runs as root | Product bug (RB7), fixed |
| 4 | 20 KiB headers accepted, 24 KiB refused | Go adds 4 KiB slack to `MaxHeaderBytes` | Set the field 4 KiB lower or document 20 KiB | Product, minor |
| 5 | CT DHCP/NAT broken: `UNREPLIED`, un-NATed source, DHCP `INVALID` | Host routed + bridged the same VLAN | `lab-gw` CT routes; host only bridges (`lab-gw.sh`) | Lab |
| 6 | Install fails at `cloudflare.sh`: `Could not resolve host: api.cloudflare.com` (11-08, CT 201; CT 202 beside it passed) | `proxmox.sh` runs install.sh right after `pct start` without waiting for the CT network; DNS answered 7.8 s after start. With the golden template apk is a no-op, so nothing absorbs the wait (apk already warns "DNS: transient error") | **Open, needs ruling:** a network wait in `proxmox.sh` changes production installer behaviour | Product bug |
| 7 | `curl https://twr-lab-<id>…/not-a-route` → 404, empty body | Path is outside `^/v1/`: cloudflared's ingress `http_status:404`, not the app | For the app's own 404 use `/v1/not-a-route` (CSP + `nosniff` headers) | By design |
| 8 | Heredoc to the lab ran nothing | Outer `ssh -n` replaced stdin | scp the script, then `ssh … bash -s < file` | Operator |
| 9 | Slow-request timing misread | Measured the client's own `sleep` | Watch the server socket close (`netstat`) | Operator |
| 10 | A failed CT stays behind, stopped | Installer keeps failed CTs for diagnosis | Use a new CT number; remove the old one with `twr-down` | By design |
| 11 | Logins and changes answer 503 on a fresh server | Changes pause until off-site (R2) backups verify | Expected on a lab server without R2 | By design |
| 12 | Agent session cannot remove CTs | Beelink hook blocks any `pct destroy` by an agent | Christopher runs `twr-down` / `golden-smoke.sh` | Operator |
