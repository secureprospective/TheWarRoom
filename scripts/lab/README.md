# Lab kit: drive test leagues by script (target: lab only)

Small scripts that ran the 11-23 Lab test drafts and seated the S8 and iPhone (task T715). They
talk to the Lab server only. They never print a password, code, join link or token.

Secrets never live here. Every script reads them from a kit folder you name with `TWR_LAB_KIT`
(mode 0700), which holds `server.url`, `lab-commish.json` (the Lab commissioner's username and
password), `totp.seed`, `phone-<name>.json`, `helper-commands.json`, `bin/twr` and `cfg/`.

| Script | What it does |
|---|---|
| `labcommish.py enroll` | Sets up the authenticator for the Lab commissioner account. |
| `labcommish.py helper <league>` | Issues a record-level helper line for franchise.01 (24 h). |
| `labcommish.py reader <league>` | Issues a read-only token for the Tuesday report card (30 days). |
| `labcommish.py confirm <file>` | Signs a G2/G3 confirm request (`twrc1.…`), prints only the code. |
| `labcommish.py revoke <league>` | Disconnects the helper made for a league. |
| `apply.sh <change.json>` | Preview, confirm when asked, execute, through `twr`. |
| `cycle.sh` | One command: deletes and recreates `native.rehtest-default` and `native.rehtest-twin`, sets the schedule and the 13-day pick clock, opens both drafts, applies the 14-day draft timers, loads NFL kickoff dates (needs `LAB_SSH_CONF`). Refuses any id not starting `native.rehtest-`. |
| `seat-link.sh <league>` | Prints one new join link on the terminal only (stops the Lab server, starts it again). |
| `oline-close.sh <league> <default\|twin>` | Closes the O-Line auction once every bid is decided (`commands/50-oline-close.json`). |
| `lib.sh` | Shared helpers: the `native.rehtest-` id guard and the Lab server stop/start with restart on early exit. |
| `watch-draft.sh` | Prints draft progress and flags a clock that stops moving. Last round comes from `ROUNDS` (default 43). |
| `seat_phone.py`, `cdp_join.py`, `cdp_app.py` | Claim a seat and sign in on the iPhone / S8 through their browser devtools. |

Making the Lab commissioner: `twr-server create-league -commissioner` (see `cmd/twr-server/README.md`).

## Sunday 11-29 practice drafts (target: `lab`, CT 260 only)

Never the Season server. The real `native.rehearsal-*` leagues are never touched; the scripts refuse them.

1. Morning of Sunday (helper lines last 24 h): copy `cycle.sh`, `apply.sh`, `lib.sh`, `watch-draft.sh`,
   `seat-link.sh`, `oline-close.sh`, `labcommish.py` and `commands/` from this folder into the kit folder
   and check each is identical (`cmp`).
2. `cd "$TWR_LAB_KIT" && LAB_SSH_CONF=<ssh conf for alias lab> bash ./cycle.sh`. It stops the Lab server,
   recreates both practice leagues, loads kickoff dates (a failure there is a warning), starts the server,
   sets the 13-day clock rule, opens both drafts and sets the 14-day timers. If it stops early, the Lab
   server is started again; a failed delete stops the run with a plain message.
3. `bash ./seat-link.sh native.rehtest-default` (and `-twin` if wanted). Copy the link from the terminal
   at once; it is never saved. Open it on your device and pick a franchise. Test one claim before the
   day: older notes say a link is single use, newer ones say one link serves every open seat.
4. `TWR_LAB_KIT=… bash ./watch-draft.sh` (set `ROUNDS=<n>` if the round count differs from 43). It ends
   with `BOTH DONE` or `STALLED <league>`.
5. After the draft and every O-Line bid: `bash ./oline-close.sh native.rehtest-default default`, then
   `native.rehtest-twin twin`.
6. Check health: `ssh lab "pct exec 260 -- /usr/local/libexec/twr-status"` shows no PROBLEM.

Warning (G9, not fixed here): `deploy/lab/twr-down` accepts ids 200-299, which includes CT 260. Do not
run any `deploy/lab/` script against 260.
