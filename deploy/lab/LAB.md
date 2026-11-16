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
| 4 | 20 KiB headers accepted, 24 KiB refused | Go reads past `MaxHeaderBytes`: ~4 KiB on a new connection, ~8 KiB on a reused one | Field set to 8 KiB; test refuses 17 KiB on both (#218) | Product, minor |
| 5 | CT DHCP/NAT broken: `UNREPLIED`, un-NATed source, DHCP `INVALID` | Host routed + bridged the same VLAN | `lab-gw` CT routes; host only bridges (`lab-gw.sh`) | Lab |
| 6 | Install fails at `cloudflare.sh`: `Could not resolve host: api.cloudflare.com` (11-08, CT 201; CT 202 beside it passed) | `proxmox.sh` runs install.sh right after `pct start` without waiting for the CT network; DNS answered 7.8 s after start. With the golden template apk is a no-op, so nothing absorbs the wait (apk already warns "DNS: transient error") | **Open, needs ruling:** a network wait in `proxmox.sh` changes production installer behaviour | Product bug |
| 7 | `curl https://twr-lab-<id>…/not-a-route` → 404, empty body | Path is outside `^/v1/`: cloudflared's ingress `http_status:404`, not the app | For the app's own 404 use `/v1/not-a-route` (CSP + `nosniff` headers) | By design |
| 8 | Heredoc to the lab ran nothing | Outer `ssh -n` replaced stdin | scp the script, then `ssh … bash -s < file` | Operator |
| 9 | Slow-request timing misread | Measured the client's own `sleep` | Watch the server socket close (`netstat`) | Operator |
| 10 | A failed CT stays behind, stopped | Installer keeps failed CTs for diagnosis | Use a new CT number; remove the old one with `twr-down` | By design |
| 11 | Logins and changes answer 503 on a fresh server | Changes pause until off-site (R2) backups verify | Expected on a lab server without R2 | By design |
| 12 | Agent session cannot remove CTs | Beelink hook blocks any `pct destroy` by an agent | Christopher runs `twr-down` / `golden-smoke.sh` | Operator |

`stresscheck` reads SQLite integrity, ledger heads and request IDs, including split league files.
Build: `CGO_ENABLED=0 go build -o .scratch/stresscheck ./tools/stresscheck`.
Copy to the lab, then `pct push 258 stresscheck /usr/local/bin/stresscheck --perms 0755`.
Lab-only: `su -s /bin/sh twr -c "stresscheck -db /var/lib/twr/server.db"`; never migrates or writes data.

## Request storm and security sweep (lab only)

Build/test locally; **do not run these during another lane's CT drill**. Run the tools below only
on the lab host, against a disposable league, after the operator releases the target.

```sh
python3 deploy/lab/stress.py storm https://twr-lab-258.<zone> /dev/shm/STATE --out /root/storm
bash deploy/lab/security-sweep.sh 258 twr-lab-258.<zone> --state /dev/shm/STATE --lan-peer 192.168.1.5:443
# Disruptive reboot check is opt-in: append --with-reboot.
# Offline tests (only loopback fake HTTP and mocked console/curl commands):
python3 -m unittest discover -s deploy/lab -v
```

`storm` requires HTTPS `twr-lab-<digits>.<domain>` on port 443 and at least two already-signed-in
owners in the existing onboard STATE. The first owner is reserved for observation; the rest supply
load bearers. No accounts or devices are created and no sign-ins happen during measurement. It uses
10 baseline reads two seconds apart, ramps 16/32/64/128/256/512 clients for 60 seconds each, holds the
first step with at least 10% 429/503 answers (or the maximum) for 600 seconds, and samples recovery
for 180 seconds. `--max-step`, `--step-seconds`, `--hold-seconds` bound rehearsal load. Requests already
in flight drain before recovery; served RPS includes that drain time. No retries conceal connection
errors. `storm.json` contains bounded latency histograms' aggregates (p50/p95 rounded up to 1 ms),
status/error counts, refusal rates, timed observer samples, baseline and gate inputs. Gate 11 uses
completion time of the first successful sample within 1.5× baseline median, not its launch time.

The cheap flood read remains `POST /v1/native/ledger` (the existing tool has no GET league read).
The signed-in observer now uses `POST /v1/native/workflow`, traced from
`frontend/src/app/shell/pulseReads.ts`, `NativeLeaguePulse.tsx`, `PulseNow.tsx` through
`internal/serverhttp/workflow.go` and `internal/commandcatalog/workflow_commands.go`.
Every baseline, hold and recovery open runs the same **sequential** hosted Now read set:
`native.directory.read` → `week.read` → `standings.read` → `bracket.read` → `odds.read` →
`results.read` (when a current/next week exists) → `lineup.read` (selected matchup's own franchise
only) → `projections.read` (selected starters, batches of 120) → `nfl.games.read` → `nfl.game.read`
(when a game is live). Season and week come from the directory and server-selected week, not constants;
week uses `at=""`, `localZone="UTC"`. The first owner's STATE entry must contain its `franchise`.
An open is 200 only when all its applicable requests are 200 with readable dependency fields.
Failure stops that open; timings cover the whole attempted set, with per-request status/time/error
in `storm.json`. `pulse_endpoint`, `pulse_commands`, and `pulse_scope` identify the measured work.
Unlike the UI's concurrent reads, the tool serializes them to time the set. It follows the first live
NFL game rather than switching to the one with most own starters. Rendering, SSE, drawer reads and
interactive navigation are not measured: operator click-through and crash/OOM inspection remain manual.
No server routes are changed.

The sweep shell entrypoint delegates to stdlib Python to keep passwords/bearers out of curl argv;
curl config and grep patterns travel over stdin. It refuses CT 110/256/257, non-200–299 IDs, a host
other than `twr-lab`, mismatched CT hostname and mismatched public hostname. It prints one line per
check, exits nonzero on any FAIL, and does not treat MANUAL as a pass. No response bodies or secret
file contents are printed. File checks use `stat -c '%n %U %a'`; log scanning uses count-only
`grep -c -F -f - /var/lib/twr/server.log`. Run without shell tracing.

- Check 4 is automated with `--lan-peer HOST:PORT`, a home-LAN machine with an open TCP port.
  The lab host must reach the peer (`nc -4 -z -w 3`, exit 0), while the same connection inside the CT
  must fail (exit 1, not a missing utility). Optional `--v6-peer '[IPv6]:PORT'` applies the same IPv6
  control/refusal checks. Without it, `pct exec CT -- ip -6 route show table all scope global` must
  succeed with empty output: the container has no global IPv6 route, not a measured peer refusal.
  Inside the CT, `nslookup example.com 1.1.1.1` and `curl --head --fail --max-time 15 https://example.com`
  must both succeed. Without `--lan-peer`, check 4 remains MANUAL. Choose a peer known to be on the
  home LAN; the flag is an operator-supplied reachability target, not automatic network discovery.
- Check 5 needs `--with-reboot`, otherwise MANUAL. It waits up to two minutes after `pct reboot`, then
  requires both OpenRC services and a healthy `/usr/local/libexec/twr-status`.
- Check 6 inventories known values recursively in STATE fields named password/bearer/token/secret/
  code/key. Include once-shown codes and key material privately (never in git). Without the operator's
  `security_secrets_complete: true` declaration, zero matches remains MANUAL, not a full leak proof.
  This scans the current log only; separately inventory rotated logs if retained.
- Check 9 permits an HTTP redirect to the same HTTPS host, not plaintext application content, and
  requires TLS 1.2 handshake rejection (not a DNS/timeout error). It measures the public edge.
- Check 10 intentionally spends the address's sign-in allowance: 12 wrong `POST /v1/auth/login`s,
  valid Retry-After (wait capped at 120 seconds), then one correct login. Coordinate with other users.
- Check 11 requires a working signed-in ledger control. Client-side slow-upload timeout is MANUAL,
  never evidence of a server drop. The documented Go header over-read may make the 16 KiB test FAIL;
  the tool records the real result rather than weakening the requested test.
- Check 12 needs an owner `principal: acct.<id>` and a `co_commissioner` object with `bearers` in STATE.
  Add `recent_auth` to an actor when TOTP requires a step-up proof (do this check within its lifetime;
  checks 10/11 take time). People reads verify the actual roles before a `commish.role.grant` refusal
  attempt. An unexpectedly successful preview stops without execute. Otherwise preview must be 403,
  execute 409, and each actor's people/ledger readings and revision must remain unchanged. Missing
  identities/proofs are MANUAL; the script never creates a deputy or elevates an account.
- Check 13 enumerates the route table's read doors: ledger, people, workflow and invalidations; all
  require 401 without credentials. Workflow read commands share the same authentication gate.

The three NOTE lines distinguish code facts from live evidence: restart throttle state, token
lifetimes, and known outbound services. Exact deployed outbound host inventory remains operator work
using private configuration and DNS/egress evidence; printing credential-bearing config is forbidden.
