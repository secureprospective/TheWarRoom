# 32-team game-day load test (`gameday.py`)

> **WARNING: DO NOT RUN DURING GAMES.**
> Never run this against the lab while live football is on or real owners are using it.

`gameday.py` uses Python 3 stdlib only. It onboards `1..32` owners via a join link,
opens `N` devices per team, holds an `/v1/native/invalidations` SSE stream per device,
and runs the phone Live tab reads (`nfl.games.read`, `nfl.game.read`,
`nfl.game.pregame.read`, `results.read`, `lineup.read`, `standings.read`) on the
app's cadence plus bursts when a score changes.

Credentials are stored in `/dev/shm` (`0600`) and deleted when the run ends. Logs hold
only franchise IDs, device indices, HTTP status codes, and timings. Total request time minus `server_ms` is network and tunnel time.

## Running against the lab (after v51, outside game hours)

1. Generate a fresh join link on the lab host (into `/dev/shm`, `umask 077`):
   ```sh
   pct exec <CT> -- sh -c \
     "/usr/local/bin/twr-server join-link -db /var/lib/twr/server.db \
      -league <LEAGUE_ID> -rotate < /dev/shm/commissioner.json" > /dev/shm/join.txt
   ```
2. Run the 32-team load test (2 devices per team, 15 minutes):
   ```sh
   python3 -u scripts/load/gameday.py \
     --target https://<LAB_HOST> \
     --join-file /dev/shm/join.txt \
     --teams 32 --devices-per-team 2 --minutes 15
   ```
3. Run `tools/stresscheck` on the lab database after stopping/quiescing:
   ```sh
   go run ./tools/stresscheck -db /var/lib/twr/server.db
   ```

## Expected duration

- **Onboarding (32 teams, 2 devices/team):** ~10–11 minutes. The server throttles
  `/v1/auth/*` to 10 requests/minute/IP, so 96 auth calls (list + 32 claims + 64
  logins) are paced at 9/minute.
- **Load phase:** `--minutes` (e.g. 15 minutes).
- **Total:** ~25 minutes.

## What to watch

- **Pass bar:** The final line must begin with `PASS: 0 x 502, 0 x 5xx, ...`. Any `502`
  or `5xx` fails the run.
- **Latency:** Watch `p95` (target `< 200 ms` for screen reads).
- **Server RSS & CPU:** Watch `twr-server` RSS (`GOMEMLIMIT=700MiB` on lab; peak should
  stay well under container limit) and `cloudflared` tunnel health.
- **Integrity:** `tools/stresscheck -db <server.db>` must print `integrity ok`.
