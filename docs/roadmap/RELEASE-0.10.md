# Alpha 0.10 — what "done" means
**Ruled by Christopher 2026-11-16.** This file is the finish line. If a line here passes, stop working on it.
If work is not on this list, it does not belong in 0.10.

## How releases are numbered
| Number | Meaning | Tag |
|---|---|---|
| 0.10 | First alpha. Cut from main at #294 (17c9109ef) plus the version stamp. | `v0.10.0` |
| 0.10.1, 0.10.2 … | Small patch: one PR. May happen hourly. Exercises the update path. | `v0.10.1` |
| 0.11, 0.12 … | Recommended update. | `v0.11.0` |
| 0.20, 0.30 … | A full field of fixes, features and upgrades. | `v0.20.0` |

Alpha is everything below 1. Beta runs until beta is done; it is not tied to the numbers.
This extends `Alpha_Versioning_and_Releases_DECIDED.md` (D-V1, D-V2); it does not replace it.

## Two pairs
- **Real:** one server and one desktop client that Christopher tests live. It only moves by a numbered release,
  and only when he says so. Nothing experimental is installed on it. Testers: Christopher only, until he opens it up.
  League mates will look in on it to see progress, so it must stay up.
- **Dev:** one server and one client where work continues. Everything lands here first.

Every piece of work names the release it targets.

## 0.10 is done when all of these pass
**Build**
1. Main is tagged `v0.10.0`, CI (`verify`) is green on that commit, and `desktop-packages` produced the Linux package.
2. The desktop app (`thewarroom -version`) and the server (`twr-server version`) both report 0.10.0.

**Real server**
3. `deploy/server/proxmox.sh <bundle>` completes on a fresh container and prints its web address.
4. After a container reboot the server and tunnel come back by themselves and `twr-status` shows zero problems.
5. The league is loaded and `twr-status` shows backups current.

**Real client**
6. The CI-built app, installed on Christopher's screen and started with `-native <server> <league>`, opens the hosted sign-in, not the local app.
7. He signs in and lands on his team.

**Game day**
8. League Pulse updates scores during a live game without a reload.
9. Badges read correctly: LIVE, DONE, NO LINEUPS.
10. A finished week turns official, and its winners match MFL.
11. Standings are right, and odds show labelled "Reduced".

**Update path**
12. A 0.10.1 bundle installs on the real server through `twr-upgrade`, and the league data is still there afterwards.
13. A tampered or older bundle is refused before any service stops.

**Stress test** (ruled 2026-11-16: the server is not called permanent until this passes)
14. The server survives a written stress test on a copy of the real setup. The exact loads and pass numbers live in
    `RELEASE-0.10-STRESS.md` and are approved by Christopher before the run. At minimum: a full league of signed-in
    clients during a live-game window, no crash or out-of-memory kill, live scores never older than two poll
    intervals, backups keep running, and the server recovers by itself from a forced kill.

Lines 6, 7 and 8–11 are only done when Christopher has seen them himself.

## Not in 0.10
- Player values on the server (F51, branch `session/server-values`).
- The formal Update button in the commissioner console, with a version display (ruled 2026-11-16: required;
  headline of 0.11). Until then, 0.10.x patches install through `twr-upgrade` on the console.
- Typing the server address in a screen.
- The non-Pulse hosted task screens as proven features; hosted bids and waivers.
- An automated check of the 200 ms screen / 100 ms click budget.
- A real off-host restore drill.

## How a later release gets its own finish line
Copy this file to `RELEASE-<number>.md`, list the checks, get Christopher's approval before building.
