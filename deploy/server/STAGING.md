# Staging drill server (Proxmox in QEMU)

A throwaway Proxmox VE node on the Beelink that runs the **real** `proxmox.sh` installer with a
**real** Cloudflare Tunnel, so release-blocker drills test the path an owner runs. Never a real
league, never real owners. First used for drill 51 (2026-10-29); evidence lives in
`~/logs/TheWarRoom/drill51-2026-10-29.md` on the Beelink.

## Build it

| Step | Command / action | Why |
|---|---|---|
| ISO | Download `proxmox-ve_<ver>.iso` from enterprise.proxmox.com/iso and check `SHA256SUMS` into `/QEMU/iso` | VM disks and ISOs live on /QEMU, not /home |
| Unattended ISO | `dpkg-deb -x` the `proxmox-auto-install-assistant` .deb from the pve-no-subscription repo; `prepare-iso --fetch-from iso --answer-file answer.toml` | No GUI clicking; repeatable |
| answer.toml | `fqdn`, `root-password` (random, kept in `/QEMU/scratch/<drill>/root-pw`, mode 600), `root-ssh-keys` = Beelink key, `network source="from-dhcp"`, `disk-list=["vda"]`, ext4 | Gives `local-lvm` for CTs |
| VM | `virt-install --memory 4096 --vcpus 4 --cpu host-passthrough --boot uefi`, disk on /QEMU/scratch, passt user network with port forwards 127.0.0.1:2251→22 and :8051→8006 | Claude-OS can stay running; 4 GiB fits beside it |
| First boot | The installer powers the VM **off** when done. `virsh start` it | Off is success, not a crash |
| Lab network | `bash staging-lab.sh` inside the VM (below) | The installer requires an isolated VLAN with DHCP, the PVE firewall on and a synced clock; it never creates them |
| Bundle | `bash deploy/server/bundle.sh .scratch/server.tar.gz` in a worktree, copy it plus `proxmox.sh` to the VM | Same artifact owners get |
| Install | Run `proxmox.sh` in `tmux new -s install`; the operator types the Cloudflare token at the hidden prompt | Watchable: `TERM=xterm-256color ssh -t -p 2251 root@127.0.0.1 tmux attach -t install` |

Cloudflare: use the permanent `thewarroom` API token (Tunnel Edit, DNS Edit, Zone Read on one
zone) and the hostname `twr-staging.<zone>`. Delete that Tunnel and DNS record when the lab is retired.

## Troubles and fixes

| # | Trouble | Cause | Fix | Kind |
|---|---|---|---|---|
| 1 | Every install failed: `tar: /var/lib/lxc/<id>/rootfs: Cannot open: Permission denied` | Installer `umask 077` reached `pct create`; the unprivileged mapping could not enter the rootfs | `pct create` runs under `umask 022` (#216) | **Product bug** |
| 2 | Valid pasted token: "does not look right" | Terminal bracketed-paste markers / whitespace reached the hidden `read` | Installer strips paste markers and whitespace before validating (#216) | **Product bug** |
| 3 | `upgrade.sh` installed a tampered binary and ran a fake one as root | Candidate is not authenticated before execution | Installed binary verifies signed bundle + metadata against the root pinned at install; candidate runs only as `twr` in the sandbox (#217; staging retest 10-30: 7/7 refused, good accepted) | **Product bug (RB7)** |
| 4 | 20 KiB headers accepted, 24 KiB refused | Go reads past `MaxHeaderBytes`: ~4 KiB on a new connection, ~8 KiB on a reused one | Field set to 8 KiB; test refuses 17 KiB on both (#218) | Product, minor |
| 5 | `apt update` 401 Unauthorized in the VM | Fresh PVE points at the enterprise repo | Lab script swaps to pve-no-subscription | Lab |
| 6 | CT got no DHCP lease | Cluster firewall dropped DHCP to the host's dnsmasq | Cluster rule `IN ACCEPT -p udp -dport 67` (no `-i`) | Lab |
| 7 | New SSH into the VM timed out once the CT's tunnel ran; existing sessions lived | passt user networking stops forwarding new inbound connections under the CT's NATed flows | Open one ControlMaster before starting the CT and send every command over it | Lab |
| 8 | `kitty`: "missing or unsuitable terminal" on attach | VM has no kitty terminfo | Prefix `TERM=xterm-256color` | Lab |
| 9 | Heredoc to the VM ran nothing | Outer `ssh -n` replaced stdin with /dev/null | scp the script, then `ssh ... bash -s < file` | Operator |
| 10 | Console typing via `virsh send-key` dropped and reordered keys | No delay between key events | 40 ms between keys; screenshot to confirm | Operator |
| 11 | Slow-request timing first read "30 s / 40 s" | Measured the client's own `sleep`, not the server | Watch the server socket close (`netstat`) | Operator |
| 12 | A failed CT stays behind | Installer deliberately keeps failed CTs for diagnosis | Use a new CT number; destroying CTs is Christopher's call | By design |
| 13 | Logins and changes answer 503 on a fresh server | Changes pause until off-site (R2) backups verify | Auth, flood-hash, MFL-input and audit-anchor checks wait for R2 on the real server | By design |

## Retire it

Stop the VM, then Christopher deletes `/QEMU/scratch/<drill>`, the `twr-staging` Tunnel and its
DNS record. Keep the evidence log.
