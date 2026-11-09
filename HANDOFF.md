# HANDOFF — 2026-11-05 ~13:30 CT (session close)

## Where we are
- `build/INTEGRATE-1102` = `session/solo-native` = **0457b992**. The cleanup docket, items 1–7, is done, passed live by Christopher, and merged.
  - Soak: Legacy 27/27 and MFL 27/27 (`docs/audit/SOAK-2026-11-05.md`).
  - Market rework (b825060e).
  - Item 5: keep-alive for HQ, Home, Pulse and War Room (4005c83d).
  - Item 6: War Room › Plans gets "Looking back": past offers, counterparties, saved proposals (721da53a).
  - Item 7: the Home alert tray adds trade events and deadlines in the next 7 days. Also fixes the Control trade alert, which pointed at a node "trade" that doesn't exist; it now points at "war" (0457b992).
- Known limits:
  - On MFL leagues, Looking back has no finished offers (only live offers are read).
  - On native leagues, saved proposals have no read yet.
  - Viewing a franchise that isn't your seat makes `trades.read` refuse, so there are no trade alerts and Looking back shows the refusal.
- Tests that already failed on the base before item 7 (not filed):
  - `TestOfflineFeedsSayPlainWords`: "dial tcp" reaches the screen.
  - `appLeaguePulse`: "no schedule published" wording.
  - `emptyStates`: quiet War Room date strip.
  - `warroomStyle`: Market mock rule `.fa-h` is missing from the CSS.
- Measured speed: Home revisit about 2.0 s, which misses the 200 ms target. First-open time hasn't been measured.

## Environment facts learned today
- Christopher's desktop on Claude-OS is **DISPLAY=:0**. Agent and test walks use :5, which he can't see. To let him test, launch his copy on :0 with `ssh -f`:
  `ssh -f claudeos 'cd ~/work/TheWarRoom && setsid nohup env DISPLAY=:0 XDG_CONFIG_HOME=$HOME/f2click/config XDG_DATA_HOME=$HOME/f2click/data build/bin/thewarroom >/tmp/twr-live.log 2>&1 </dev/null &'`
- The revamped Home only appears on **app (native) leagues**. The app copy of Legacy is `native.mfl-14432-2026`, and it lives in the `~/f2click` profile. The default `~/.config` profile has only the MFL-synced Legacy, which shows a league list.
- Run only one app per profile. A second copy loses the profile lock and shows "No leagues available", and it doesn't retry. Candidate debt task.
- Builds need `PATH=$HOME/.local/bin:/usr/local/go/bin:$HOME/go/bin:$PATH GOTOOLCHAIN=go1.26.10`, then plain `make build`.

## Item 8: client/server, new session
Governing doc: `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md`. Inputs are in `docs/audit/CLIENT-SERVER-*`. Rulings: memory `project_warroom_client_server_plan_1105`.
- **Order:** 1 → 2 → 5 → 3 → 4 → 6. These are Tom's changes, adopted. Nothing is up for re-vote.
- **Step 1, start here:**
  1. Measure the real-tunnel RTT distribution on home broadband, on desktop and on a modern phone (p50/p95/p99/max).
  2. Compare it with the 100 ms confirmed-action budget. Report every breach, and escalate if the budget is physically impossible; never redefine success.
  3. Lock the server entry and command contract: hosted `runNative` and the command catalog (`commandcatalog`/`nativegen`). The locked screens and the CLI/MCP must reach the same service and get the same outcomes and rejections.
  4. Delete duplicate native adapters only by tracing their callers, never by grep alone.
- **Before starting:** ask Christopher who builds (Sol medium by default; GPT was low on 11-05) and where the tunnel endpoint is. Memory `project_warroom_staging_vm` covers twr-staging and CT152 with the tunnel.
- **Open point for step 2's gate:** a daily evidence manifest alone can't meet the 15-minute RPO for newly referenced evidence. Settle the cadence before admitting writes.
- **Step 2's gate also includes memory bounds:** CT memory limit, held seasons, and decoded generations. Today's Market cost (19–25 s cold load, +697 MiB) is on the Hermes debt task.
- **Each slice records production LOC before and after.** The tree must shrink (11-04 reset).
