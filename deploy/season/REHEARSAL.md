# Season first-install rehearsal (Lab throwaway Alpine)

Paper plan only. Rehearse `deploy/season/season-install.sh` and `deploy/season/season-uninstall.sh` on a throwaway Alpine container built from the Lab golden template (`deploy/lab/golden-build.sh`) **before** touching the real Season laptop (`192.168.1.162`) on Sat Dec 5.

Never run any command in this document against `192.168.1.162`.

## Roles and prerequisites

- **Christopher:** approves each real install/uninstall command, creates or destroys the throwaway CT on VM 503 `twr-lab` (`192.168.1.182`, since agent hooks block `pct destroy` per `deploy/lab/LAB.md`), and signs off the result.
- **Operator on Beelink (`chris@192.168.1.191`):** runs the dry-run and approved `season-install.sh` / `season-uninstall.sh` commands, verifies bundle SHA-256, records outputs and the `deploy/CHANGE-LOG.md` entry.
- **Candidate artifact on Beelink:** the newest Lab-verified release tarball (for example `~/fleet/runs/labrel-v107-auto-1126/release-v107/twr-server-LAB-ONLY-r11-a43521f7.tar.gz`) whose commit has passed the Lab check (health 200, both leagues load, live screen renders) and whose SHA-256 matches its signed `metadata/*targets.json`.

## Step 1 — Spin up a throwaway Alpine machine from the Lab golden template (Christopher on `twr-lab`)

Choose an unused Lab CT ID in `200–299` (example: `295`, never `110`, `256`, `257`, or live Lab `260`) and the newest golden template in `/var/lib/vz/template/cache/`:

```sh
# On twr-lab (192.168.1.182), as root:
[ "$(hostname)" = twr-lab ] || exit 97
TPL=$(ls -1 /var/lib/vz/template/cache/alpine-3.23-twr-golden_*_amd64.tar.gz | sort | tail -1)
(cd /var/lib/vz/template/cache && sha256sum -c "$(basename "$TPL").sha256")

(umask 022; pct create 295 "local:vztmpl/$(basename "$TPL")" \
  --hostname twr-season-rehearsal \
  --unprivileged 1 --features nesting=1 --cores 2 --memory 1024 --swap 0 \
  --rootfs lab-thin:4 --onboot 0 --nameserver 1.1.1.1 \
  --net0 name=eth0,bridge=vmbr1,tag=51,ip=dhcp,ip6=manual)
pct start 295
```

Because `golden-build.sh` strips `openssh-server` (for Proxmox CTs), enable temporary SSH inside the throwaway CT only so it mimics the bare-metal laptop's SSH administration from the Beelink, and place a sentinel `/etc/sysctl.d/10-twr.conf` to verify it is preserved across install and undo:

```sh
# On twr-lab, as root:
pct exec 295 -- sh -euc '
  apk add --no-cache openssh-server
  mkdir -p /root/.ssh && chmod 700 /root/.ssh
  rc-update add sshd default
  rc-service sshd start
  printf "# sentinel pre-existing file\n" > /etc/sysctl.d/10-twr.conf
  ip -4 addr show eth0
'
# Copy Beelink public key into /root/.ssh/authorized_keys inside CT 295 (or use SSH jump via twr-lab
# if VLAN 51 is isolated from the home LAN by lab-gw).
```

Note on network reachability: VLAN 51 (`10.51.0.x`) sits behind `lab-gw` (CT 110) and blocks direct LAN traffic (`deploy/lab/LAB.md`). If the Beelink cannot reach `10.51.0.x` directly, either attach CT 295's temporary rehearsal NIC to the rehearsal-accessible bridge or verify `ssh root@<rehearsal-ip> hostname` returns `twr-season-rehearsal` before Step 2.

## Step 2 — Verify the candidate artifact and dry-run from Beelink (Operator on Beelink)

```sh
cd ~/work/TheWarRoom
BUNDLE=~/fleet/runs/labrel-v107-auto-1126/release-v107/twr-server-LAB-ONLY-r11-a43521f7.tar.gz
SHA=$(sha256sum "$BUNDLE" | cut -d ' ' -f 1)
# Compare $SHA byte-for-byte with release-v107/metadata/107.targets.json before continuing.

sh deploy/season/season-install.sh \
  --target root@<rehearsal-ip> \
  --version v107 \
  --bundle "$BUNDLE" \
  --sha256 "$SHA" \
  --allow-lab-artifact \
  --dry-run
```

Confirm:
- Dry run exits `0` (inside Sat 18:00–23:59 Chicago; outside that window add `--christopher-ok` to preview dry run only).
- Every remote command is printed (`== 1/7` through `== 7/7`) and zero packets are sent.
- Hand the printed `Exact command for Christopher to approve` line to Christopher.

## Step 3 — Run the real rehearsal install (Operator on Beelink, inside Saturday 18:00–23:59 Chicago after Christopher approves)

```sh
sh deploy/season/season-install.sh \
  --target root@<rehearsal-ip> \
  --version v107 \
  --bundle "$BUNDLE" \
  --sha256 "$SHA" \
  --allow-lab-artifact \
  --christopher-ok
```

(If rehearsing outside Saturday night on this throwaway CT only, do **not** bypass the guard on `192.168.1.162`; schedule the throwaway rehearsal during Saturday 18:00–23:59 Chicago — e.g. Sat Nov 28 — so the real wall-clock guard is exercised too.)

## Step 4 — Post-install verification checks (Operator from Beelink)

Run these read-only checks against `root@<rehearsal-ip>`:

1. **Service and process isolation:**
   ```sh
   ssh root@<rehearsal-ip> 'rc-service twr status; rc-update show default; ps -ef | grep -E "bwrap|twr-server" | grep -v grep'
   ```
   Expect: `status: started`, `twr` in `default`, `twr-server` running under user `twr` inside `bwrap`.

2. **Exact Season layout and permissions (`season-swap.sh` contract):**
   ```sh
   ssh root@<rehearsal-ip> '
     readlink /opt/twr/current
     ls -ld /opt/twr /opt/twr/versions /opt/twr/versions/v107 /opt/twr/libexec /var/lib/twr /var/backups/twr /etc/twr
     ls -la /etc/twr /var/lib/twr
   '
   ```
   Expect:
   - `/opt/twr/current` -> `/opt/twr/versions/v107`
   - `/var/lib/twr` `drwx------ twr twr`, `server.db` `0600 twr:twr`, `status.sock` present
   - `/var/backups/twr` `drwx------ root root`
   - `/etc/twr` `drwxr-x--- root twr`, `origin.key` & `authenticator.key` `0600 twr:twr`, `origin.pem` & `endpoint-registry.csv` `0640 root:twr`
   - `/etc/twr-release/root.json` does **not** exist (no throwaway Lab root pinned).

3. **TLS health check on `127.0.0.1:5199/app/` and console dashboard:**
   ```sh
   ssh root@<rehearsal-ip> '
     curl --silent --cacert /etc/twr/origin.pem --output /dev/null --write-out "%{http_code}\n" https://127.0.0.1:5199/app/
     sh /opt/twr/libexec/twr-status
   '
   ```
   Expect: HTTP `404` (local-only without `--public-host`) over verified TLS, `twr-status` prints installed server build and reports backups unconfigured / writes paused (expected until R2 setup).

4. **Re-install refusal check:**
   Re-run the exact Step 3 command once. Expect **exit 1** at `== 1/7` (`Refused: existing TheWarRoom path found: /opt/twr`) with existing keys/DB untouched (`sha256sum /etc/twr/origin.key /etc/twr/authenticator.key` unchanged).

5. **Pre-existing host files untouched:**
   `ssh root@<rehearsal-ip> 'cat /etc/sysctl.d/10-twr.conf; rc-service sshd status'`
   Expect: sentinel line intact and `sshd` still started.

## Step 5 — Rehearse undo (`season-uninstall.sh`)

First dry-run, then run after Christopher approves:

```sh
sh deploy/season/season-uninstall.sh --target root@<rehearsal-ip> --purge-data --dry-run
sh deploy/season/season-uninstall.sh --target root@<rehearsal-ip> --purge-data --christopher-ok
```

Verify clean removal:

```sh
ssh root@<rehearsal-ip> '
  ls -ld /opt/twr /var/lib/twr /var/backups/twr /etc/twr /etc/init.d/twr /etc/logrotate.d/twr /etc/periodic/hourly/twr-logrotate 2>&1 || true
  id twr 2>&1 || true
  cat /etc/sysctl.d/10-twr.conf
  rc-service sshd status
'
```

Expect: every TheWarRoom path and user `twr` gone, `/etc/sysctl.d/10-twr.conf` intact, `sshd` still running.

## Step 6 — Destroy throwaway CT (Christopher on `twr-lab`)

```sh
# On twr-lab (192.168.1.182), as root:
pct stop 295
pct destroy 295
```

## Step 7 — Log the rehearsal in `deploy/CHANGE-LOG.md`

Add a new entry at the top of `## Entries` in `deploy/CHANGE-LOG.md` (matching the template from `session/env-change-log-1126`):

```markdown
### YYYY-MM-DD HH:MM · lab · install

- **From → to:** none → v107 (a43521f7, throwaway CT 295 from Lab golden template; Season first-install rehearsal)
- **Why:** Rehearse `deploy/season/season-install.sh` and `season-uninstall.sh` before Sat Dec 5 Season laptop install
- **Approved by:** Christopher (<message / tracker card>)
- **Commands:**
  - `sh deploy/season/season-install.sh --target root@<rehearsal-ip> --version v107 --bundle <bundle> --sha256 <sha> --allow-lab-artifact --dry-run`
  - `sh deploy/season/season-install.sh --target root@<rehearsal-ip> --version v107 --bundle <bundle> --sha256 <sha> --allow-lab-artifact --christopher-ok`
  - `sh deploy/season/season-uninstall.sh --target root@<rehearsal-ip> --purge-data --christopher-ok`
- **Took:** <N> minutes (install to healthy)
- **Checks after:** `rc-service twr status` started; `/opt/twr/current -> /opt/twr/versions/v107`; TLS probe `https://127.0.0.1:5199/app/` returned 404 (local-only); `twr-status` confirmed writes paused without R2; re-install refused cleanly; undo removed all TWR files and preserved `/etc/sysctl.d/10-twr.conf` and `sshd`; CT 295 destroyed
- **What broke:** <nothing, or exact issues found>
- **Rollback needed:** no (undo rehearsed and verified)
- **Next time:** <any script/doc adjustments before Sat Dec 5>
```
