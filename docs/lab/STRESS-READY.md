# Stress run: ready-to-use steps (proven on lab CT 252, 11-08)

What the 32-owner stress run will do, using only doors that exist and were exercised on the real
league (Legacy NFL, MFL 14432) on lab CT 252 through the public Tunnel. Lab only: never CT 110,
never production DNS names other than `twr-lab-<ctid>`, never the real `thewarroom.db`.

## 1. Server with the real league

```sh
# Beelink
bash deploy/lab/lab-bundle.sh                      # needs commits 3fa33d6d8..375da6cae
scp .scratch/lab/twr-server-LAB-ONLY-*.tar.gz lab:/root/bundles/
ssh lab /root/twr-lab/twr-up <ctid> /root/bundles/<bundle> < ~/.config/thewarroom/cloudflare-token
ssh lab "pct set <ctid> --memory 4096 --cores 2 && pct resize <ctid> rootfs 8G"
# seed: stream, never copy to the lab host
ssh lab "pct exec <ctid> -- sh -c 'umask 077; mkdir -p /root/seed; cat > /root/seed/history.db'" \
  < ~/fleet/lab-seed/twr-seed-20261108/history.db           # check sha256 against SHA256SUMS
```

Commissioner credentials are a JSON object `{"username","password"}` made in `/dev/shm` on the
Beelink and pushed to the CT's `/dev/shm`; never an argument. Load with the service stopped:

```sh
pct exec <ctid> -- sh -c 'rc-service twr-server stop </dev/null; cd /var/lib/twr; setsid sh -c \
 "twr-server league-load -db /var/lib/twr/server.db -mirror /root/seed/history.db -mfl-league 14432 \
  -football-pin /etc/twr/football.sha256 < /dev/shm/c.json > /tmp/load.out 2>&1; rm -f /dev/shm/c.json; \
  chown twr:twr /var/lib/twr/server.db*" </dev/null >/dev/null 2>&1 &'
# then: rc-service twr-server start; rc-service twr-tunnel start   (stopping the server stops the Tunnel)
```

Measured on CT 252 (2 cores, 4 GB): **642 s**, peak RSS **188 MB**, league
`native.mfl-14432-2026`, 32 franchises, 3,367 ledger entries, rosters with contracts. Run it detached:
an ssh session may time out first.

Backups (writes and sign-ins stay paused until one verifies): build `backup.json` in memory on the
Beelink from `~/.config/thewarroom/r2-lab` plus a throwaway age recipient, pipe it to
`/root/twr/backup.json` in the CT, run `sh /usr/local/libexec/twr-backup-setup </dev/null`. Wait for
`twr-status` to show `OK Backups` and `OK Tunnel`.

## 2. Join link

```sh
pct exec <ctid> -- su -s /bin/sh twr -c \
  'twr-server join-link -db /var/lib/twr/server.db -league native.mfl-14432-2026 -rotate' < commish.json
```

Prints `league` and `code` (43 characters). Keep it in `/dev/shm`; it lets anyone claim an open team.
`-revoke` kills it, `-release franchise.NN` reopens a team (a `commish.seat.remove` ledger event).

## 3. How 32 simulated owners claim teams

All requests: `POST`, JSON, headers `X-TWR-Protocol: 1.1`, `X-TWR-Commands: 1`, **no `Origin`**.

1. `/v1/auth/join/franchises` `{"protocol":1,"join":{"league":L,"code":C}}` → 32 franchises,
   `taken` flags.
2. `/v1/auth/join/claim` `{"protocol":1,"claim":{"league":L,"code":C,"franchise":"franchise.NN",
   "username":U,"password":P}}` → 201; 409 if taken (the race is serialized, so exactly one of
   simultaneous claims wins) or the username is used; 403 for a bad link.
3. `/v1/auth/login` `{"protocol":1,"username":U,"password":P,"device":"stress"}` → `bearer`.

**Throttle:** every `/v1/auth/*` request counts against **10 per minute per client IP**
(`CF-Connecting-IP` through the Tunnel), with at most 6 in flight server-wide. 32 owners × (claim +
login) = 64 auth calls from one IP takes **≥ 7 minutes**. Either pace at 10/minute or drive the
owners from several source IPs. Claims hash a password, so expect `503 server is busy` under
concurrency; retry after `Retry-After`.

## 4. Reads and clicks that exist for an owner (bearer only, no TOTP needed)

| Door | Body | Seen on CT 252 |
|---|---|---|
| `/v1/native/ledger` | `{"protocol":1,"league":L}` | 200, 3,370 entries |
| `/v1/native/workflow` `native.assets.read` | `{"league":L}` | 200; 1,462 league players; own roster with contracts (34–51) |
| `workflow lineup.read` | `{"league":L,"franchise":F,"week":W}` | 200 for weeks the pinned schedule holds |
| `workflow bids.read` | `{"league":L,"franchise":F}` | own 200, **another owner's 409** |
| `workflow commish.rules.set` preview | (commissioner command) | **409** for owners |
| `workflow commish.lineup.set` preview → execute | `RawCommand` with `expected` = assets revision, `visibility:"private"`, `football` = the response's digest | preview `blockers:[]`, execute 200, retry returns the same receipt, re-read `set:true` |

The Legacy lineup is 21 starters: 8 offense (QB/RB/WR/TE; the kicker is not offense), 12 defense,
1 PK. Build it from `lineup.read`'s bench by position. Other owner reads from the catalog: `moves.read`,
`draft.read`, `trades.read`, `results.read`, `standings.read`, `value.read`, `waivers.read`; owner
writes include `bid.*`, `trade.offer/accept/decline/withdraw`, `waivers.claim`, `roster.cut`, `roster.reserve`.

## 5. Known gaps before the run

- The held copies have no NFL **week 12** schedule, so `lineup.read` for week 12 refuses. Weeks
  that are held work (13 was used).
- Football evidence carries no statistics (only the player directory and kickoffs), so results and
  scoring reads show what the ledger recorded, not fresh stats.
- `twr-server league-load` is CPU-heavy (642 s). Re-run only on a fresh CT.

## 6. Cleanup

Delete `/root/seed` and `/dev/shm/*` in the CT. Delete the CT's own Tunnel and DNS record with
`cloudflare.sh --rollback`, the same block `twr-down` runs (token on stdin), then `pct stop <ctid>`.
Run `twr-down` only when the CT may also be destroyed.
