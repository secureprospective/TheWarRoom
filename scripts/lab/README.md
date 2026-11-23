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
| `cycle.sh` | Deletes and recreates both test leagues and opens their drafts (needs `LAB_SSH_CONF`). |
| `watch-draft.sh` | Prints draft progress and flags a clock that stops moving. |
| `seat_phone.py`, `cdp_join.py`, `cdp_app.py` | Claim a seat and sign in on the iPhone / S8 through their browser devtools. |

Making the Lab commissioner: `twr-server create-league -commissioner` (see `cmd/twr-server/README.md`).
