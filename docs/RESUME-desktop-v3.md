# RESUME — desktop v3 / phone live

## COMPACT POINT 11-21 ~8:30 PM CT — NEXT: start T672 (phone Live pre-game preview, design B "Broadcast")
Christopher's instruction: after compaction start T672.
- Spec: task T672 on Hermes (`cc task show T672`). Approved mock: Beelink `~/work/TheWarRoom/design-mocks/live-pregame-preview-B.html` (28012 B).
  Rulings: no betting odds; weather = game-time conditions (NWS hourly; ESPN day forecast abroad; "Indoors" for domes, no invented temps);
  ESPN predictor drives the win-chance bar (drive bar becomes win %); scrolling marquee of the selected matchup's players in that NFL game.
  Must be on theme (tokens in LiveTab.css / liveField.css / fieldView.ts), sits where the field sits before kickoff.
- Branch from origin/main (cc7898ca2+). Work worktree suggestion: `.worktrees/live-pregame`. Pushing needs frontend/dist copied in +
  node_modules symlinked to `readlink -f ../dv3-shell/frontend/node_modules` + PATH=$HOME/go/bin:/usr/local/go/bin (wails).
- Do NOT release to the lab during Sunday games (12:00 PM–~10 PM CT): a restart makes the first app open time out (T677).

## STATE AT THIS POINT
- main = cc7898ca2 (#454 matchup/headshots, #455 FINAL HUD, #456 server CSP = app build policy). Lab CT260 = v46, all status OK, 5.8 GB free.
- T671, T674, T675, T676 checked. T676: all sessions of the leaked lab account revoked (S25 + Beelink dev page must sign in again — Christopher);
  R2 token "twr-lab objects (lab only)" rolled in Cloudflare, new values in Beelink ~/.config/thewarroom/r2-lab and lab /etc/twr/backup.json,
  backups verified in R2 01:15 UTC. Caveat: page lookup showed key ID + ~first quarter of secret in this transcript; offered a re-roll.
- T677 open: build results/standings/odds/bracket readings at server start (cold 3.5-5 s vs 5 s limit). After Sunday.
- T673 open: live proof Sun 11-22 12:00 PM CT. Wall (/app/wall) uses same nfl.games.read/nfl.game.read path; never run on real data; needs sign-in.
- Tracker (http://192.168.1.222:8790): source Beelink ~/fleet/tracker (commit there), deploy = copy collect.py/index.html to Hermes ~/fleet/tracker.
  Board, plan cards (by T### in title), Needs Christopher (report <12 h else 🔴 tasks) and idle alarm now derive from Task_list. cc stamps dates in Central.
  "Update the tracker" = audit every section + screenshot; fix slop without asking (memory feedback_update_tracker_means_whole_page).

## IN FLIGHT
- Lab dev server: vite pid 557753 on 127.0.0.1:5199, cwd `~/work/TheWarRoom/.worktrees/lab-dev/frontend` (branch session/lab-dev = main),
  log ~/logs/vite-lab-dev.log. KEEP running (Christopher's /app/ and /app/wall?demo). Stop: kill 557753.
- Harness ~/scratch/lineup-1121 (w40.js walk, wf.js read timings, csp.js console check, state-lab.json — session now REVOKED, re-capture after sign-in).

## NEEDS CHRISTOPHER
1. Sign in at http://127.0.0.1:5199/app/ (Beelink) and on the S25 lab app. Then I check /app/wall shows the real slate.
2. Sunday 12:00 PM CT: open Live on S25 during a game (T673).


## NEXT (his instruction): work through T673, T674, T675, T676 in that order
- T673 🔴 Live field proof Sun 11-22 12:00 PM CT on S25. Already proven 11-21 on finished IND-HOU (ESPN 401873066):
  collector polling (checkpoint /var/lib/twr/live-scores/checkpoint.json, week 11 board, 13 games), summary held with
  159 plays all with yard lines, nfl.game.read returned 160 plays + state, phone drew field (ball IND 30).
  Unproven: 5-s in-game poll on v43. During games: watch lab log `grep "server live" /var/lib/twr/server.log`.
- T674 ⚪ finished game shows "1ST & 11" (END GAME play down=0 → render; hide down/dist when status post) in
  frontend/src/app/phone/LiveField.tsx / fieldView.ts HUD (hDown/hDist). Console: CSP "Applying inline style" warnings
  + one 404 on Live tab (find which URL; harness lv2.js logs >=400).
- T675 🟡 lab CT 260 disk 2.0 GB free / 8.4. Candidates seen: /twr-b7lab-new (old tarballs v4), labrel-* dirs,
  first-start-data.tar.gz at / of CT. Check which are referenced by the running release before deleting.
- T676 🔴 his steps: sign out other sessions on the lab account; rotate lab R2 keys. Hand him numbered steps.

## IN FLIGHT AT COMPACTION
- Push of squash commit e371ff5f to main, run by Christopher via `!` from Claudebox chat (background task output
  /tmp/claude-0/-root/61ea614a-bffb-482b-b76e-4b768b2d36b1/tasks/bez3vfvd1.output). Pre-push checks were running
  (go test on 9 changed pkgs). Check: `ssh beelink 'ps -eo args | grep -c "[h]ook-type=pre-push"'` (0 = done) and
  `git -C ~/scratch/merge-pc fetch -q origin && git -C ~/scratch/merge-pc log --oneline -1 origin/main`
  (landed = e371ff5f-equivalent "Mobile alpha walk..." on top of eb6dd752d). Main at compaction: eb6dd752d.
- If it landed: release main to lab (REF=origin/main twr-lab-release-ref.sh 44 with prev
  ~/fleet/runs/labrel-v43-1121/release-v43/metadata/targets.json), walk w40.js, `cc task check T671`, then
  `git worktree remove ~/scratch/merge-pc` and delete ~/scratch/lineup-1121.
  If push failed: read the output file; branch protection may need the browser PR path
  (https://github.com/secureprospective/TheWarRoom/compare/main...session/player-card, Squash and merge).
- Never run the push or anything containing its text yourself: hook blocks it (even a pgrep with that string).

## SHIPPED SINCE LAST COMPACT (branch session/player-card 1b4e2fcd3, lab v43, walked)
- Live: starter tiles removed from My matchup (8fd71daee).
- Fix 1: score row follows matchup picked in All matchups; header "My matchup" is a button (aria "Show my matchup"),
  shows "‹ My matchup" when away. Walked: BUF 4.2 98% NYJ → back to ARZ 32.3 93% CAR 38.0.
- Fix 2: headshots 350x254 (was 96x70) in package subfolder headshots/350; route /v1/headshot/350/<id>; ESPN jersey
  in Bio.Jersey; phone card hero (club colour --tc, #7, ghost number). Walked: Stroud natural 350x254, #7.
- Tests: frontend 807 pass in touched dirs; Go tests pass for changed pkgs.

## REFUTED / LEARNED — DO NOT RETEST
- `?s=350` query on photo URL: hosted server refuses ANY query string (hosted.go RawQuery check) → 400. Don't loosen.
- Cloudflare caches /v1/headshot/<id> (max-age 86400, cf-cache-status HIT) → use versioned path, not purge.
- boundarycheck refuses changes to internal/composition/platform_host.go startHeadshots → put policy in the package.
- Push hooks need PATH=/usr/local/go/bin:$HOME/go/bin (wails missing otherwise). Fresh worktree needs frontend/dist
  copied for ifaceguard.
- Lab release script builds what is PUSHED: a failed push silently released old code (v39).

## DECISIONS
- Pre-game preview: direction B (Broadcast) picked; later build T672. No betting odds. Game-time weather (NWS hourly),
  not day high/low. Mock ~/work/TheWarRoom/design-mocks/live-pregame-preview-B.html.
- Small fixes logged to ~/work/TheWarRoom/design-mocks/FIXES-mobile-batch-1121.md (1-2 done).
- He approved this merge to main 11-21 ("go ahead and merge it. i approve").

## TASK LIST (Hermes) this session
T671 rewritten (merge state), T668, T660 updated; T672 B picked; added T673-T676.

---
# COMPACT POINT 11-21 ~1:05 PM CT — mobile walk on S25, lab v37 (newest; READ FIRST)

## WHAT WE ARE DOING
Christopher walks the PHONE app (target: mobile mock v1 only) on his S25 against the LAB:
https://twr-lab-260.secureprospective.com/app/ (no IP; Cloudflare tunnel). He sends error codes; Claudebox
finds them in the lab log, fixes on `session/player-card`, releases the branch to the LAB only, verifies.
Repo: Beelink ~/work/TheWarRoom/.worktrees/player-card. Branch `session/player-card` = a42fd1594, pushed,
NOT merged. Desktop v3: he says good enough for alpha; still needs lots of work later.

## NEXT ACTION (his instruction after compaction): work on LIVE / field view
His question: "Does the server serve the live game to the mobile app with a field view?" Answer given:
path exists end to end, never proven on an in-progress game. Path: cmd/twr-server/live_collector.go polls
ESPN (scoreboard Set line ~284, details SetDetails ~371: plays, possession, down, distance, yardLine) ->
reads `nfl.games.read` / `nfl.game.read` -> frontend/src/app/now/liveContract.ts (FieldState) ->
phone/LiveTab.tsx -> phone/LiveField.tsx + fieldView.ts. Offered check (he had not answered): open the
FINISHED Thursday week-11 game in phone Live on the lab and confirm the field draws from real server data.
Harness ready: ~/scratch/lineup-1121/lv.js (phone viewport, state-lab.json, Live tab, screenshots
live1.png/live2.png). It was interrupted mid-run once; read-only. First real live proof: Sun 11-22 noon CT.
RULING 11-21: Hermes live-recorder has NOTHING to do with the phone walk. Do not look at Hermes for this.

## LAB STATE
- Lab CT260 runs v37 = a42fd1594 (branch release). All 8 status OK.
- Next release prev targets: `~/fleet/runs/labrel-v37-1121/release-v37/metadata/targets.json`.
- Release cmd: `R=~/fleet/runs/labrel-v38-1121; mkdir -p $R; REF=origin/session/player-card
  ~/fleet/bin/twr-lab-release-ref.sh 38 <prev targets> $R > $R/release.log 2>&1 </dev/null` (~3 min, blocks).
- Lab log: `S="ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab"; $S pct exec 260
  -- sh -s` with a heredoc script (quoted inside ssh '...' runs on the Beelink by mistake). Log
  /var/lib/twr/server.log. Grep the error code he sends.
- Lab tunnel only forwards ^/(v1/.*|join|password|site.css|...|app(/.*)?)$ — new routes must live under /v1/ or /app/.

## FIXES THIS SESSION (all on session/player-card, all lab-verified)
| Commit | Fix | Proof |
|---|---|---|
| eb6e4f3fa,3c2e85d29 | Player photos: hosted /v1/headshot/<mfl id> (desktop cache reused) | browser: 96x70 photo on card |
| 13e97f695 | Lineup unlock 12-16 s -> 3.3 s: identical in-flight reads shared (nativeApi.ts), recent form read 1 week at a time | browser timing x2 |
| 1fab5c051 | Log why native request refused + missing app file | log lines seen |
| c2c05470e | vite:preloadError reloads once (pwaUpdate.ts); League season reads 2 weeks at a time | tests |
| 086b32844 | Recorded starters memo per records mark+week gen, 1 h (history/native_results.go) | timed: RecordedStarters was 4.2 s/week |
| 231f8b654 | Server warms weeks 1-18 at start +20 s and every 50 min (football_pins.go) | League cold 3.1 s after release |
| 0a1711134 | Contract quotes carry viewer revision (StampRevision) — "quote belongs to another decision" gone | lab reading + Team no banner |
| a42fd1594 | week.read allows 5 min device clock skew (S25 fast); missing /app/assets file answers Clear-Site-Data cache+storage | curl shows header |

## OPEN / UNVERIFIED
- His S25 kept loading an OLD page (asked for assets/commissioner.M5lPOGmH.js, control.ChxWdhoY.js).
  v37 Clear-Site-Data should clear it; NOT yet confirmed on his phone. Manual fallback: Brave site settings -> Clear & reset.
- 503 request-deadline bursts on odds/standings/bracket/results when phone loads (2-core lab). Watch.
- Desktop known: Home news cold load; HQ value.read 409.
- Phone hold-to-drag only tests. Field view never seen live.

## REFUTED — DO NOT RETEST
- 400 TWR-BVLR/PP99 was NOT the problem-report body (same body 200 from harness, direct lab too). PP99 = week.read future `at` (clock skew).
- results.read slowness is NOT queueing alone: RecordedStarters = 4.2 s/week (timed on lab).
- Lone results.read "0.3 s" was a cached answer; uncached ~3.2 s before memo.
- /headshot/ 404 was the Cloudflare tunnel allowlist, not the server route.

## HARNESS
~/scratch/lineup-1121 (KEEP.md): state.json (walk server login; NEVER print), state-lab.json (same cookie on
lab domain, chmod 600; its cookie was printed once in chat by mistake — rotate when convenient).
t4.js = lineup unlock timing; m8/m9/m10 = phone Team/League on lab; q1.js = contract decisions shape; lv.js = Live tab.
Walk server: vite on player-card worktree port 5199 (pid 49543); stop `kill 49543`.

## NEEDS CHRISTOPHER
- Go for merge of session/player-card -> main (squash), then normal lab release from main.
- Rotate lab R2 keys; sign out other sessions on lab account (harness cookie printed).
- Confirm S25 now on current build (Team + League).

## RULES LEARNED TODAY
- Never --no-verify (hook blocks). Remote branch delete is blocked: hand him `git push origin --delete <b>`.
- Go not on PATH over ssh: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH` before commit/push hooks.
- Bloat gate counts new small non-test files: fold into an existing file.

# HANDOFF 11-21 ~8:00 AM CT — walk-and-fix cadence, ESPN news live on lab v26 (newest; READ FIRST)

## How we work right now (the cadence — keep it)
1. Christopher walks http://127.0.0.1:5199/app/ (desktop) and the phone layout, and sends what he finds
   (often with screenshots: "check SS" = newest files in Beelink /home/chris/content/feedback).
2. Small fixes: Claude does them directly in the `player-card` worktree. Fresh/larger coding: write a brief
   file, launch Gemini (AGY Neon High) with `~/fleet/bin/agy-run.sh <job> <brief> "Gemini 4 Neon (High)" <wt>`
   and arm a done-watcher on `~/fleet/runs/<job>-1121/exit` in the same step. Reuse existing code first.
3. Claude verifies every fix itself: tsc + vitest + go test, then a real browser check with the harness
   `~/scratch/lineup-1121` (playwright-core + headless brave; state.json = commissioner login, never print).
4. Push the branch `session/player-card` (allowed). For him to see server-side changes, release the branch
   to the LAB only (see below). NOTHING merges to main until he says go: all walk fixes ship as one batch.
5. Report short: what works (with evidence), what's not verified, one pick-from question with a recommendation.

## Branch / server state
- Branch `session/player-card` = 6a14f7676, pushed, NOT merged. Contains origin/main + lab v25 (eb6dd752d).
  Worktree: Beelink ~/work/TheWarRoom/.worktrees/player-card. Also `session/lineup-standings` 1bbb5edfd
  (subset of player-card; superseded).
- Lab server (CT 260) runs **v26 = 6a14f7676 (the branch, LAB-ONLY test release, Christopher picked "1")**.
  Status all OK at 7:48 CT. Run dir ~/fleet/runs/labrel-news-1121 (MANIFEST, upgrade.log, status.txt).
  Next release must pass `~/fleet/runs/labrel-news-1121/release-v26/metadata/targets.json` as previous.
- Branch release script (new): `REF=origin/session/player-card ~/fleet/bin/twr-lab-release-ref.sh <V>
  <prev targets.json> <run-dir>` (copy of twr-lab-release.sh with REF; default still origin/main).
- Walk server: vite on player-card worktree, port 5199, proxies to lab. Restart:
  `cd ~/work/TheWarRoom/.worktrees/player-card/frontend && pnpm exec vite --config vite.lab.config.ts`
  (ssh -f + nohup setsid). Stop: `kill $(pgrep -f vite.lab.config)`. vite.lab.config.ts is not in git.

## Walk fixes done this session (all on session/player-card)
| # | Fix | Proof |
|---|-----|-------|
| 1 | Lineup locked: standings parser refused new `streak` field → allowed (hostedMarkets.ts + test) | browser: lineup editable |
| 2 | One-click lineup: starter name→bench; bench name→first fitting open slot (`quickMove`, shell/hqPlacement.ts) | browser desktop |
| 3 | "Reading your lineup from the league server…" while locked | browser |
| 4 | One position order (`LINEUP_ORDER`/`slotRank`, shell/pulseModel.ts) for lineup, live scoring, phone Live + Team | browser desktop + phone |
| 5 | Desktop player card drawer (names on Contracts/lineup/news open it) — Neon | browser |
| 6 | Phone: MOVE quick-moves, hold-to-drag, full card sheet — Neon | browser except drag (tests only) |
| 7 | ESPN news: server cache (league 15m, player 15m, bio 24h, stale on failure), `player.news.read` + `news.feed.read`, card News tab + bio header, Home NEWS panel with real chips — Neon (37a28c9fe, 6a14f7676) | browser on lab v26: Home ALL 50 / MY PLAYERS 1 / INJURIES 12 / ON THE WIRE 8; Bowers card Ht/Wt/Born/College/Draft + ESPN items |

Gates at 6a14f7676: go vet + go test (espnnews, leagueservice, twr-server) ok; tsc clean; vitest 1,284/1,284
(Neon's run); push hook passed.

## Known, not fixed (candidates for next turns)
1. Home news: first load after release showed "Reading the news…" >10 s (cold cache). Re-check a normal load.
2. Player photos blocked by app CSP (a.espncdn.com) → default image. Fix: serve via our own `/headshot/` proxy.
3. Lineup editor takes 12–16 s to unlock (room reads ~10 calls incl. many results.read).
4. HQ read warnings: `value.read` 409; "Contract decision … quote belongs to another decision".
5. Phone card on the phone layout not browser-checked after the news change (shared code, tests pass).
6. cmd/twr-server/espn_news.go makes its own `http.Client{Timeout:10s}` as the livegame transport (fine;
   parsing reuses livegame.FetchHTTPS).
ESPN does not give: Acquired, Contract Notes, draft for undrafted players.

## Rulings (do not relitigate)
- Phone keeps name-tap = card (no ⓘ button). News from ESPN only for now; keep a `source` field per item.
- Gemini does fresh coding; reuse existing code first. Batch walk fixes; one merge on his go.
- Cap: runway = MFL contracts; what-if = calculator; dead cap = MFL adjustments + TWR cuts; committed = TWR-only.

## Next actions, in order
1. Take Christopher's next walk items (the cadence above).
2. Fix items from "Known, not fixed" he cares about (recommend 2 photos, then 3 slow unlock).
3. On his go: PR session/player-card → main (squash only), merge, then normal lab release from main
   (`twr-lab-release.sh 27 ~/fleet/runs/labrel-news-1121/release-v26/metadata/targets.json <run>`).
4. Carry-overs: DEMO mark on ?demo wall; OBS wall test; Sunday 11-22 ~11 AM live check (wall, phone,
   recorder), noon first plays; Argon review of #453.

## Needs Christopher
- Go for the merge to main. Rotate lab R2 keys. OBS sign-in test.
Task list: T671 (Hermes) = this walk-fixes batch.

# COMPACT POINT 11-21 ~7:10 AM CT — walk fixes in progress (newest section; older below)

**State:** #453 merged (main eb6dd752d), lab on v25. Christopher is walking http://127.0.0.1:5199/app/
and sending fixes. Walk server now runs on the **player-card** worktree (vite.lab.config.ts copied in).
Restart: `cd ~/work/TheWarRoom/.worktrees/player-card/frontend && pnpm exec vite --config vite.lab.config.ts`
(launch with ssh -f + nohup setsid). Stop: `kill $(pgrep -f vite.lab.config)`.

**Branches (both pushed, none merged; he chose to batch walk fixes into one release):**
- `session/lineup-standings` (1bbb5edfd, worktree alpha-2, off main): (1) lineup locked because standings
  parser refused new `streak` field (from #442) → added; (2) one-click: starter name→bench, bench name→
  first fitting open slot (`quickMove`); (3) "Reading your lineup…" message while locked (was "No changes");
  (4) one `LINEUP_ORDER`/`slotRank`/`bySlot` in shell/pulseModel.ts drives lineup, desktop My matchup,
  phone Live tiles, paired box score; kickers own group last. Test shell/lineupOrder.test.ts.
- `session/player-card` (0eb4271a8 pushed; worktree player-card) = lineup-standings + Neon desktop player
  card drawer (dd321d4e7) + Neon phone quick Move / hold-drag / full card sheet (1d4571bb0) + phone Team tab
  lineup order (0eb4271a8). Browser-verified: desktop Contracts name→card; phone Move benches; phone card.
  NOT verified: phone hold-to-drag (tests only).

**In flight:** Neon High `espn-news-1121` in player-card worktree (started ~7:00 CT; brief
`~/fleet/runs/espn-news-1121/brief.md`; run dir `~/fleet/runs/espn-news-1121-1121/`, done = `exit` file;
out.log = report). Builds ESPN player news + bio service, `player.news.read` + `news.feed.read`, card News
tab + bio header, Home NEWS panel with real chips. On done: read report, review diff, run tsc/vitest/go test,
browser-check card news + Home feed, push branch, tell Christopher. Quota-hit → err.log "Resets in".

**Rulings today:** phone keeps name-tap = card (option 2, no ⓘ button); news from ESPN only for now
("first-class from that single source", other sources later, keep a source field); lean on Gemini for
fresh coding; reuse existing code before building.

**Cap Q answered:** runway = MFL contracts; what-if = calculator; dead cap = MFL salary adjustments + TWR
cuts (team checked had none); committed moves = TWR-only decisions, empty until owners act.

**Known, not fixed:** lineup editor takes 12–16 s to unlock (room readBundle ~10 reads incl. many
results.read); value.read 409 and "Contract decision … quote belongs to another decision" warnings on HQ read.

**Refuted:** lineup lock was NOT permissions, NOT HQAct dispatch, NOT unstable port/room recreation —
it was the strict parser (verified by logging the hidden failure cause).

**Test harness:** `~/scratch/lineup-1121` (playwright-core + brave headless, state.json = commissioner
login; p5/p6 lineup, p10/p11 order, p12 card+phone). KEEP until walk fixes merge.

**Next:** 1) finish ESPN news on done; 2) take Christopher's next walk items; 3) fix slow lineup unlock;
4) on his go: PR player-card → main, merge, lab release (`twr-lab-release.sh 26 <v25 targets> <run>`).

# COMPACT POINT 11-21 ~5:15 AM CT — ALPHA ON MAIN + LAB; walk next

**WHAT:** TWR alpha live on Legacy lab before Sun 11-22 12 PM CT kickoff. Repo ~/work/TheWarRoom (Beelink). Main = 877683d50 (#452 squash: desktop v3, glossary, MFL sync, error codes, phone live w1+w2+fixes, PWA update, live wall v1).
**DONE LOOKS LIKE:** (1) alpha on main — MET 877683d50. (2) lab runs main — MET v24 (0.10.1-60-g877683d50), status all OK. (3) lab matches MFL — MET: real refresh 68 adds/50 removes/17 status/9 cap/29 picks/352 lineups/10 wks scores; rerun dry = "already up to date". (4) follow-ups #453 on main + lab v25 — IN FLIGHT. (5) Christopher walk + fixes — NEXT. (6) Sunday live proof (wall + phone + recorder) — NOT MET until noon Sunday.

**IN FLIGHT**
- `~/fleet/bin/merge-453-release.sh` (pid 2732379, Beelink, started 5:10 AM): waits for PR #453 CI green -> `gh pr merge 453 --squash` (Christopher authorized 11-21) -> `twr-lab-release.sh 25 ~/fleet/runs/labrel-alpha-1121/release-v24/metadata/24.targets.json ~/fleet/runs/labrel-453-1121`. Log: `~/fleet/runs/labrel-453-1121/log.txt`. Alive: `pgrep -f merge-453-release`. CI fail -> exit 3 "CI FAILED" in log: read `gh run view <id> --log-failed`, fix on session/alpha-2, push.
- Walk dev server: vite on alpha-2 worktree (`frontend/vite.lab.config.ts`, excluded from git), http://127.0.0.1:5199/app/ (lab proxy; wall /app/wall, demo /app/wall?demo). Stop: `pkill`-free: `kill $(pgrep -f vite.lab.config)` locally on Beelink.
- Hermes: live-recorder (~/live-recorder, data/YYYY-MM-DD), fleet tracker (port 8790, ~/fleet/tracker; source Beelink ~/fleet/tracker branch session/tracker-v1 head de5ce1b).

**#453 contents (session/alpha-2, 220ceb295):** refresh test store pinned to fixture clock (Christopher: "the clock should be pinned to the server"), v3 Control Room error codes with Copy (d2034e904), wall v2 fluid full-screen + ticker + spotlight + big-plays rail (18eac730a). Local gates: tsc clean, 108 vitest, go Refresh tests ok.

**LAB FACTS:** release script `~/fleet/bin/twr-lab-release.sh <V> <prev targets.json> <run>` builds origin/main. Prev targets also on lab `/root/labrel-<sha>-v<V>/metadata/<V>.targets.json`. Backup before alpha: `~/fleet/runs/labrel-alpha-1121/lab-backup-before.tgz` (325 MB, tar of /var/lib/twr). MFL refresh: build twr-server, `mirror-fetch -db <h.db> -mfl-league 14432`; copy to lab `/var/lib/twr/manual-1121-mfl-refresh/` (chown twr — /root is unreadable to twr); `rc-service twr-server stop`; `su -s /bin/sh twr -c "twr-server league-refresh -db /var/lib/twr/server.db -mirror <file> -league native.mfl-14432-2026 [-dry-run]" < ~/fleet/runs/pulse-lab4/private/commissioner.json`; start. Lab is Alpine/busybox (no bash, no sqlite3).

**REFUTED — DO NOT RETEST**
- "AGY signed out": agy keeps sign-in in GNOME keyring but uses empty file store when SSH_CONNECTION/SSH_CLIENT set. Fixed in agy-run.sh (unset + `agy models` preflight, exit 4 = SIGNED_OUT). Fallback: `systemd-run --user -p KillMode=process`.
- CI flake TestLeagueRefreshStopsOnOpenTradeOffers: not a code bug; fixture clock vs wall clock. Fixed in #453.
- Argon "162 h lock" from ledger was false; AGY screen said 100%. Trust the AGY screen.
- `gh pr merge --merge` fails: repo allows squash only.

**DECISIONS (11-21):** tracker usage bars removed (no real quota numbers — facts only). Wall: real games only; ?demo dev-only, add a visible DEMO mark. Christopher live-streams the wall weekly. Gemini when available, Tom/Haiku fallback. Hydrogen shares Neon's bucket; Argon separate; AGY Claude/GPT locked to ~Fri 11/27.

**NEXT ACTIONS**
1. Take Christopher's walk fixes and thoughts (he walks after compaction) — triage into lanes (Neon/Hydrogen), verify each myself.
2. Check merge-453 log; confirm lab v25 status OK; walk server stays on alpha-2 (or main after merge).
3. DEMO mark on ?demo wall; test wall in OBS signed in.
4. Sunday ~11 AM: wall shows today's slate, recorder live mode, phone check on lab; noon watch first plays.
5. Argon review of #453 pieces post-merge (v3 errors, wall v2).

**NEEDS CHRISTOPHER:** walk + fixes; rotate lab R2 keys; OBS sign-in test with me.
**HONEST STATUS:** wall/phone never seen on a live game; first proof Sun noon. PWA self-update unproven on a real phone. #453 not merged until CI green.

# COMPACT POINT 11-20 ~8:40 PM CT — alpha push, Sunday 11-22 kickoff (12 PM CT)

## WHAT WE ARE DOING
Alpha for Sunday kickoff + phone live scoring that matches mobile mock v1 (THE marketing hook, 7 weeks of season left).
Christopher walks desktop v3 + phone and signs off the ONE alpha merge Saturday 11-21 MORNING.

## STATUS / BRANCHES (all on Beelink ~/work/TheWarRoom/.worktrees/)
| Branch | State |
|---|---|
| session/desktop-v3-shell (dv3-shell) | pushed e74d8b7dc; RESUME commits local (6489793c8, 139e1336c, this one) |
| session/glossary-adr | pushed 12ccd292f |
| session/mfl-sync (mfl-sync) | DONE + PUSHED d85693da1. Runs 1-3 (engine d85711676+04bcb4761, league-refresh 58766113a+3351322a4, mirror-fetch 72e12e715+d85693da1), every Argon finding fixed, verified by Claudebox. |
| session/error-codes (error-codes) T667 | built 401bf3b1c; Argon: fix first (server code lost in UI etc). Fix run cut off by Neon quota with ~34 UNCOMMITTED files, tsc error HQInspector.tsx:208 setErrorCode, 4 failing vitest. Resume run t667fix2 auto-launches ~10:00 PM (brief ~/fleet/briefs/agy-t667-fix2-1121.md, watcher blwwlshkq). Then verify (go test serverhttp/serverclient, vitest src/app/data src/app/phone, tsc). |
| session/phone-live (phone-live) wave 1 | RUNNING: AGY Claude Opus 5.5 Low, run ~/fleet/runs/phonelive-w1-1120 (brief ~/fleet/briefs/agy-phone-live-w1-1121.md, watcher bfhgvju2m). Live field + big-play engine from real plays + reduced motion. Gap list: ~/fleet/runs/bee-bee-mockgap-1121-1120/stdout.out; mock screenshots ~/fleet/runs/mockgap-1120/screenshots/. |
| T668 PWA auto-update | NOT STARTED. frontend/pwa/sw.js cache name hardcoded v1, cache-first. Needed so weekly mock waves reach phones. |
| T670 live wall | NOT STARTED: one phone per live NFL game on a Beelink page for OBS; after wave 1. |

## LAB PROOF (MFL sync) — on a COPY, live lab untouched
~/scratch/mflsync-dry-1121 (KEEP.md): lab backup tgz, fresh mirror history.db (32 franchises, 2762 players, 976 tx, 10 weeks), dry run = no data change (sqlite .dump identical), real refresh 68 adds/50 removes/29 picks/352 lineups, rerun "already up to date". Arizona 47, Cleveland 55, New Orleans 43 match MFL exactly. Real lab refresh ships with the alpha bundle (needs reference.db beside server.db; commissioner creds ~/fleet/runs/pulse-lab4/private/commissioner.json, never print).

## RUNNING SERVICES
- Live score recorder T669: Hermes `live-recorder.service` (~/live-recorder, data ~/live-recorder/data/YYYY-MM-DD/packets.jsonl.gz). Source ~/fleet/live-recorder (Beelink, git session/recorder-v1). Check Sunday: `journalctl --user -u live-recorder -n 5` on Hermes — expect mode=LIVE polls every 5 s.
- Tracker: http://192.168.1.222:8790/ (Hermes fleet-tracker.service + collect timer). My file /home/hermes/fleet/tracker/data/plan.json — read data/answers.jsonl FIRST each update.
- Desktop v3 dev server was STOPPED 8:25 PM (memory). Restart before Christopher's Saturday walk: `cd ~/work/TheWarRoom/.worktrees/dv3-shell/frontend && npx vite --port 5199` (check the RESUME section below for the exact earlier command).

## ROUTING / QUOTA (see ~/fleet/ROUTING.md, memory reference_gemini_pacing_1120)
- Neon locked until ~10:02 PM (5-hour window). Argon locked until Fri 11/27 2:24 PM (weekly). Max 2 Neon lanes at once.
- Bee/Luna (fleet-dispatch.sh) is READ-ONLY and blocked when swap > 80% → builds via agy-run.sh (Claude Opus/Sonnet via AGY).
- Usage ledger: `python3 ~/fleet/bin/agy-usage.py` (agy-run.sh now records model/minutes/quota per run).
- Beelink: cleared 84 GB go-build cache (disk 69%); swap ~88% from Claude-OS VM + Christopher's Brave — leave them.

## DECISIONS 11-20 evening (do not relitigate)
- Direction (memory project_twr_direction_1120): Legacy stays on MFL for years; real native league = July 2027 startup draft, target full 32 teams; Christopher is solo tester until ~March; sequence = phone live scoring hook now → off-season commissioner-work proof → draft.
- Phone mock v1 starts NOW, not after the reset. Weekly reveals via auto-update (T668).
- Live wall for streaming (T670). Recorder data can replay into it.
- Rotate lab R2 backup keys (they were printed in session output 6:20 PM; not sent anywhere) — flagged.

## NEXT ACTIONS, IN ORDER
1. When wave 1 lands: verify (vitest src/app/phone, tsc, screenshots vs mock), review with Sol/Terra or Opus (Argon locked), fix, push branch.
2. ~10 PM: T667 fix2 lands → verify → push session/error-codes.
3. Start T668 PWA auto-update (Neon or Opus via AGY), small.
4. Build the ALPHA BUNDLE branch: desktop-v3 + glossary + mfl-sync + error-codes + phone-live w1 + T668; full tests; restart dev server; Saturday morning walk.
5. After sign-off (Christopher runs main merge): lab release + real lab MFL refresh (backup first) + phone check before kickoff.
6. Wave 2 (matchup header, win-chance swing, score animations) + T670 wall.
7. Direction grilling continues (open question: which commissioner jobs eat most time).

## NEEDS CHRISTOPHER
- Saturday morning: walk + alpha merge sign-off (main merge command to be handed over).
- Rotate lab R2 backup keys.

# ALIGNMENT CONVERSATION 11-20 ~4:00-5:40 PM CT (Christopher's rulings — do not relitigate)

**Alpha (scope frozen; Sunday 11-22 kickoff is a target, not a hard deadline)**
- Order: MFL sync runs 2-3 -> lab dry run (backup first) + Arizona roster check -> T667 error codes -> T668 PWA auto-update + hot fixes -> Christopher walks desktop v3 -> ONE alpha merge bundle (desktop v3 + glossary + MFL sync + T667 + T668), one sign-off -> lab release.
- Claudebox drafts a one-page alpha checklist (league, owners, screens, actions) for his approval.
- T667: every error shows a short code + Copy button (code, time, screen, app version); server logs same code; "Send to commissioner" files it as a problem report in the Feedback tab. He does most testing: make it easy for him = easy for everyone.
- T668 (BLOCKER): frontend/pwa/sw.js cache name hardcoded `twr-static-shell-v1`, assets cache-first -> fixes never reach phones. Fix: build stamps version, app checks on open/focus, swaps + "Updated" note, server minimum version. Hot-fix lane: production still needs his sign-off, made one-tap (fix + plain what-it-fixes + one command).

**How we work** (`~/fleet/AGENTS.md` "How we build together", loose habits not gates)
- Christopher = strategic partner (real world, what people need); cannot code. AI = tactical partner / logic engine.
- Keep model creativity; don't stop on small stuff; pitch ideas he wouldn't know (engineering AND how real people operate) as "would this work?" with a recommendation. Talk in behaviour; show results.
- Memory: feedback_feature_loop_1120.md.

**Post-alpha: the realignment session** (memory project_twr_realign_session_1120.md)
1. Clean workspace: main checkout parked on session/league-history-pwa with 595 broken files; 136 worktrees (22 GB), 19 merged; 353 branches; 21 RESUME docs; 3 front doors (README/North_Star/HANDOFF).
2. De-slop code from early weak models. 3. Remove old assumptions (legacy desktop mode, old Control Room). 4. Realign code/docs/glossary to one direction. Start with a grilling session on real-world direction.
- Codebase facts: ~163k Go + 62k TS production lines, ~124k test lines; SYSTEM_MAP.md good but dense, header says current as of 10-20.

**Live tracker** (replaces ~/Downloads/twr-usable-week-plan.html; memory reference_fleet_tracker_1120.md)
- http://192.168.1.222:8790/ on Hermes (fleet-tracker.service + fleet-tracker-collect.timer). Facts every minute, zero tokens. My file: /home/hermes/fleet/tracker/data/plan.json, updated at start/finish/blocked/direction change only.
- Every update: read data/answers.jsonl FIRST, act, list in plan.json "seen".
- v2 in flight (Hydrogen, run tracker2-1120): War Room dark moulded-rubber look, collapsible status groups, interactive agent track. Deploy: tar ~/fleet/tracker Beelink -> Hermes ~/fleet/tracker, keep data/, restart fleet-tracker.service.

**In flight at 5:40 PM**
- MFL run 2 (league-refresh command, Neon High): ~/fleet/runs/mflsync-b2-1120. Then Argon review -> Neon Medium fix -> verify -> run 3 (mirror fetch).
- Run 1 done: d85711676 + fix 04bcb4761 (10/10 Argon findings fixed, verified by Claudebox).
- Go on the Beelink over ssh: export PATH=$PATH:/usr/local/go/bin.

# RESUME — Desktop v3 finish (and phone v1) — updated 2026-11-20 ~09:40 CT

## NOW (read first after a restart)
- **COMPACT POINT 11-20 ~3:55 PM CT.** Christopher next wants a CONVERSATION to get TWR back on track (he has ideas; find his alignment — ask, don't build). TWR building resumes after.
  - **Desktop v3 branch** `session/desktop-v3-shell` PUSHED at e74d8b7dc (Control Room v3 + Neon polish: names not ids, plain wording, Contracts row, shared people query, line-length fix). Earlier "pushed c8fcdd4f0" claim was WRONG; remote had been b5b2fa53a until e74d8b7dc.
  - **Sign-in on Beelink dev page** works with the lab commissioner creds in `~/fleet/runs/pulse-lab4/private/commissioner.json` (protocol:1, no code). "sign-in refused" = wrong creds.
  - **MFL sync (in flight):** worktree `~/work/TheWarRoom/.worktrees/mfl-sync`, branch `session/mfl-sync` from main ee3461c84. Design = `~/fleet/runs/mflsync-plan-1120/report.md` (Argon High). Christopher's rulings: add a pull-from-MFL command; MFL scores count for weeks played on MFL; refresh REFUSES while in-app trade offers are open; server keeps seats/roles/rules/settings/ledger; MFL wins rosters/contracts/salaries/MFL adjustments/picks/schedule/results/played lineups; franchise names: in-app edit else MFL; idempotent; dry run. Run 1 (engine) = AGY Neon High, run dir `~/fleet/runs/mflsync-b1-1120` (alive check: `ps -p $(cat ~/fleet/runs/mflsync-b1-1120/pid)`; done when `exit` file appears; 12 uncommitted files at 3:53 PM). NEXT: verify its report vs tree → Argon review on spec/standards/security → run 2 (league-refresh console command, brief to write) → run 3 (mfl mirror fetch command) → dry run on lab (backup first) → check Arizona roster vs MFL.
  - **Glossary + 8 ADRs** on branch `session/glossary-adr` PUSHED (12ccd292f): GLOSSARY.md, docs/adr/0001-0008, docs/glossary-followups.md (96 old-word spots = Hermes T666). Rulings: Co-commissioner vs Deputy (deputy priority → Co-commissioner order); Mirror = MFL copy; Twin-pool league; Ledger = league history only (Cap schedule / Dead-cap schedule; You send / You get); Dead cap; Cut / Vacate seat / Release(software); Native league / Hosted server / Thin client.
  - **Workflow adopted (Matt Pocock loop):** in Beelink `~/fleet/AGENTS.md` "How features are built" + memory feedback_feature_loop_1120. Specs in docs/specs, tickets = AGY briefs + Hermes lines; cross-model adversarial review (spec/standards/security); PR with merge danger (`pr` skill); main sign-off stays; lab release by tooling (T663); fast checks at commit (T664); daily cleanup job after alpha (T665).
  - **Skills imported** (mattpocock/skills abeeaf9, adapted copies): grill-with-docs, grill-me, grilling, domain-modeling, diagnosing-bugs, tdd, codebase-design, wizard, retro, writing-for-agents, to-tickets, pr — Claudebox + Beelink ~/.claude/skills, Pi, AGY plugin.
  - **AGY routing learned:** Neon High builds well but misses repo style (≤110-char lines) → put it in briefs; Argon High = accurate reviewer; Hydrogen Medium fine for writing to a settled spec; zero Claude/GPT quota used this afternoon.
  - **Tracker** `~/Downloads/twr-usable-week-plan.html` rebuilt forward-looking (old copy `~/scratch/twr-tracker-before-1120.html`). Lab = v23; desktop v3 NOT on lab.
  - Vite dev server pid 1663256 still running (port 5199) for his walk; stop with `kill 1663256` on Beelink when done.
- **COMPACT POINT 11-20 ~11:35 CT.** Christopher's direction: continue the build toward an ALPHA PRODUCTION release; UI built on the real Legacy league; FLAG side quests, do not chase them; production thinking, no mock or test-only special cases. He has low trust in correctness right now: verify every screen on Claude-OS against server replies before claiming it. Nothing in flight. Branch in sync with origin at 5cd31f667.
- **RUNNING (keep): vite dev server pid 1663256 in frontend/ (`pnpm exec vite --config vite.lab.config.ts`, 127.0.0.1:5199, proxies to lab). Christopher walks it. Stop: kill 1663256.**
- **11-20 ~12:15 CT Christopher:** AGY is now in the fleet; Gemini NEON is something he is working with now (do not assume details; ask before dispatching to either). After compaction KEEP MOVING on Control Room v3 without check-ins; he will walk the app at http://127.0.0.1:5199/app/ soon and report side quests.
- **COMPACT POINT 11-20 ~12:10 CT — War Room v3 DONE, all pushed (c8fcdd4f0).** Trade desk = `v3/WarTradeV3.tsx` on `useWarRoomTrade` hook (split out of WarRoomTrade.tsx); Pocket/Negotiations reuse WarRoomDrawers; mock classes (tstrip, tcol, tmid, leds, lensw, aft, tsec, tpart, ttray) come from the v3 CSS. All 4 War Room tabs + 3 HQ tabs verified on Claude-OS with Arizona. Nothing running. NEXT: Control Room v3 — capture mock tabs with `T.go('control',TAB,'A')` for TAB in jobs, situations, rules, season, fix, picks, people, league, server (mock tab 3E62B5A61141A02864D78CA1CABB823C; Haiku agent can capture, see how war mocks were captured into scratchpad); Control Room currently still mounts via legacy-host HostedLeague (V3Shell `active ? legacy-host`); build like HQ/War (native mount, v3 bodies, reuse existing reads/actions via hooks). Include the MFL sync/refresh setting as a flagged feature (ask Christopher). Still waiting on Christopher: run `! bash ~/Downloads/take-arizona.sh` (seat on Arizona). Side quests open: season ledger read hangs (>3 min, jams browser pool); lab has no player values/weekly stats; mock private plans/notes have no read; browser seat/role commands blocked (DelegableCommand); Restructure/Extend no server command; MFL refresh setting.
- **11-20 ~12:00 CT — HQ My moves + War Room (pool, plans, draft) in v3. Pushed 1ffd7dc4d; draft 433f308cb local.** My moves = `v3/HQMovesV3.tsx` (Needs you / Waiting on / Landed; season ledger on request — server `franchise.moves.history.read` hangs >3 min on the lab and jams the browser's connection pool: SIDE QUEST). Commissioner franchise pick persists (localStorage `twr.hqTeam`) and is shared by HQ and War Room. War Room mounts natively in V3Shell (entry war/bids/trades) with v3 tab strip + rail leaves + TeamPick; WarRoom.tsx returns v3 bodies when `native.v3`: pool `v3/WarPoolV3.tsx` (market logic split into `useWarRoomMarket` hook in WarRoomMarket.tsx), plans `v3/WarPlansV3.tsx` (looking back + calendar; mock's private plans/notes have no read), draft `v3/WarDraftV3.tsx` (clock/picks, board, my picks, queue; picking stays in DraftRoom opened below). Screens now stay mounted (hidden) under account panels (`keep()` in V3Shell). nativeAccount tests watch a mocked WarRoomNative. Christopher has NO seat: wrote `~/Downloads/take-arizona.sh` (twr-server join-link -take franchise.25 on lab, prompts for his password, backs up first); he has not run it yet. NEXT: Trade desk v3 (mock `scratchpad/war-trade-mock.*` lost on compaction: recapture with `T.go('war','trade','A')`), then Control Room v3. He can walk the build at http://127.0.0.1:5199/app/ on the Beelink.
- **11-20 ~11:30 CT — HQ Contracts v3 DONE + PUSHED (c19ad4c10).** `v3/HQContractsV3.tsx` (table All/Expiring/Top, cap runway, expiring + Decide drawer (owner only), What-if opens HQNativeWhatIf, dead cap per season (no per-player ledger read exists), committed moves; Release acts on the player picked in the table). Checked on Claude-OS with Arizona. Mock has Restructure/Extend buttons: no server command found, left out (ask Christopher). nativeAccount test now opens War Room (still the hosted room). NEXT: My moves v3 (capture mock `T.go('hq','moves','A')`), then War Room v3, Control Room v3. Christopher asked: we ARE on the PWA; he has not seen it; offer a lab release of this branch so he can open the lab URL.
- **11-20 ~11:20 CT — Franchise HQ v3 lineup DONE (142d3a878, local; branch pushed through 14d1ce838).** Christopher: production thinking, real league only, no test seats; Arizona Cardinals (franchise.25) is HIS team; he wants complete commissioner control. Built `v3/HQLineupV3.tsx` (plate, starters in mock order, bench, taxi, IR, This week, Win chance gauge, Form last five; drag or click-pick to move; Check-and-save in the plate when dirty) mounted by V3Shell for entry hq/lineup via NativeFranchiseHQ `v3` prop. Commissioner gets a FRANCHISE picker in the HQ tab strip (any team; view-only unless it is their own seat, because value.read / moves.read / contract.decisions.read are owner-only on the server). Checked on Claude-OS with real AZ roster. Contracts and My moves tabs still render the old panels inside the v3 frame (next to restyle). NEXT: HQ Contracts + My moves v3 (mock captured: `~/fleet/runs/dv3/hq-contracts-mock.html` + `.png`; panels: Contracts table ALL/EXPIRING/TOP SALARIES by season, Cap runway bars, Contracts expiring/decide, What-if, Dead-cap ledger, Committed contract moves; reuse NativeFranchiseHQ reads: snapshot rosters, reading.contracts, runway, contractDecisions, HQNativeWhatIf), then War Room v3, then Control Room v3. Verify on Claude-OS with the FRANCHISE picker set to Arizona Cardinals.
- **SIDE QUESTS FLAGGED (not started, Christopher decides):** (1) **Seat/role/succession commands cannot run from the browser app at all**: `/v1/native/workflow` only admits `accounts.DelegableCommand` (AI-helper-safe) commands, which excludes `commish.seat.*`, `commish.role.*`, `commish.succession.*` (`internal/commandcatalog/workflow_commands.go:41`, `internal/accounts/tokens.go:77`). Fix = let human cookie sessions run them while AI tokens stay blocked; security change. This is why the commissioner cannot put Christopher's account into Arizona. (2) Lab has **no weekly NFL player stats** (projections.read says "No NFL weeks held for this player"), so projections and player form are blank. (3) **MFL sync setting** for Control Room (Christopher asked: keep the league current from MFL; today only one-off `league-load`). (4) Commissioner editing another team's lineup (commish.lineup.correct exists). (5) Earlier list: owner test seat (superseded by 1), news branch review, Archive, phone. Small fix shipped: the commissioner action tray now shows the server's refusal reason instead of silently greying out.
- **COMPACT POINT 11-20 ~10:50 CT.** Christopher: UI first, side quests later (owner seat, news wiring review, Archive). Branch `session/desktop-v3-shell` (pushed up to drawers; Pulse commits local). DONE + checked on Claude-OS 1920x1080 with the REAL lab league (Legacy NFL, not mock data): frame, Home board, Docket drawer + Waiting on/Landed lanes, Assistant drawer (not on server yet), Feeds key removed (his call), `v3/fixes.css` (approved mock CSS clash fixes), People Vacant-seat bug fixed, League Pulse NOW in v3 (`v3/PulseV3.tsx`) with pull drawers (`v3/pulseDrawers.tsx`: matchup box, schedule, stat changes), Pulse RACE in v3 (`v3/PulseRaceV3.tsx`: standings, power, bracket, my road, tiebreaks + full bracket drawers). Wiring: V3Shell mounts NativeLeaguePulse with `v3={badge,kind}` for entry pulse/league (bypasses HostedLeague); PulseBody returns v3 layout for tab now/race. Left out (no data source, tell Christopher): Who holds the ball, Week story, NFL box score, Archive tab, Dynasty/Blend power, League headlines, Why it moved. NEXT: screenshot Race after bracket-column fix, then Franchise HQ (lineup) v3, War Room v3, Control Room v3 (each: capture mock DOM via mock tab in Claude-OS Brave, `T.go(screen,tab,'A')`, outerHTML of .work; build from existing reads). News branch `session/nfl-news-read` pushed (not reviewed, not wired). Mock tab id on claudeos 3E62B5A61141A02864D78CA1CABB823C (close when done).
- **11-20 ~10:05 CT: refactor DONE and branch PUSHED** (v3 split into model.ts, pieces.tsx, overlays.tsx, slim V3Shell.tsx; hostedRules.test.tsx moved to the v3 flow). Lab on v23 shows week 11 (refresh rerun, backup manual-1120-nfl-refresh-v23). Claude-OS frame + Home checked after refactor: no regressions. Next: batch-1 open items below.
- **PR #451 MERGED ee3461c84.** Lab release v23 ran from it (`~/fleet/runs/overnight-b9walk-1120/labrel-v23.out`). Next: copy lab-nfl-1120.sh with a new backup dir name, run it via `labrun.sh`, then `python3 ~/fleet/runs/dv3/wk.py` and expect current week 11.
- **Progress 11-20 ~09:50 CT (branch `session/desktop-v3-shell`, pushed up to b4fab897f; later commits local until batch push):** ca706f3be frame+Home+PostCSS fence; 17f57ae03 switcher, Ctrl K, account menu; 052df7e03 account actions in the Account menu + account screens in the work area, nativeAccount tests rewritten for v3 (control+clock 120/120 pass); legacyHost.css anchored on #app. Claude-OS app window: Brave `--app=http://127.0.0.1:5199/app/` fullscreen = true 1920x1080, CDP tab id changes per window (list via `curl -s 127.0.0.1:9222/json`).
- **CSS isolation design:** `frontend/postcss.config.js` fences every stylesheet outside `app/v3/` with `:where(:not(#app, #app *), .legacy-host, .legacy-host *)`. Old screens render inside `.legacy-host` in the v3 work area (interim until batch 2); `app/v3/legacyHost.css` fits the old fixed-position league shell into the work area and hides its rail.
- **Batch 1 still open:** (1) real-data side-by-side needs an owner seat with a lineup on the lab (commissioner account has no team, so This week / My players are empty) and the week fix live (PR #451 -> v23); (2) NFL news piece: wire `nfl.news.read` when lane `hk-newsread` lands (branch `session/nfl-news-read`), review it first; (3) Needs-you key opens the Docket drawer, Feeds and Assistant keys are inert; (4) docket Waiting on / Landed lanes from trades, bids, waivers, moves reads; (5) board tab strip shows only Today: Game day, All leagues live, Commissioner desk, Front office boards are batch 3.
- **Build branch:** `session/desktop-v3-shell`, worktree `~/work/TheWarRoom/.worktrees/dv3-shell` (from origin/main 796887798). Gap list committed: `docs/desktop-v3-gaps.md` (regions, grades, 4 batches). Read it.
- **Method (Christopher 11-20: "hi-fidelity textures and real UI match the mock, no shortcuts"):** port the mock's CSS verbatim (`.scratch/mock/v3.css`, extracted from the mock) and emit the mock's DOM and class names from React (`.scratch/mock/v3.js` shell.js l.760-905 = frame; grid.js l.484-760 = boards; pieces.js l.345-484). Swap sample data for hosted reads. Legacy `TargetApp`/`homeLayout` is the OLD v5 mock: do not reuse its visuals; reuse data layer only (`homeReads`, `homeNeedsYou`, pulse reads).
- **Integration point:** `frontend/src/app/control/NativeAccount.tsx` ~l.183-260: on desktop (not phoneFrame) with homeReady, mount the v3 frame instead of header/nav/ThinHome/league section; work area = Home board, or `HostedLeague` when a league is open (reskin in batch 2). Phone path untouched.
- **Batch 1 = frame + Home** (top bar, band, rail, board tab strip, 24x14 grid with default Home board + 6 pieces, inspector, status line, switcher, Ctrl K). One PR when the batch is complete and side-by-side checked. No small merges (Christopher 11-20).
- **Work in the open on Claude-OS** (Christopher 11-20): visible Brave window, fullscreen F11 = true 1920x1080. Dev loop: `cd dv3-shell/frontend && pnpm exec vite --config vite.lab.config.ts` (port 5199, proxies /app/session,/v1,/headshot to the lab; file is local only, do not commit) -> on CT105 `ssh -f -N -L 5199:127.0.0.1:5199 chris@192.168.1.191` and `ssh -f -N -R 5199:127.0.0.1:5199 claudeos` (claudeos cannot ssh to the Beelink). App URL on claudeos `http://127.0.0.1:5199/app/`. Drive: `node ~/cdp/cdp.cjs <tab> nav|eval|click|type|shot`; sign in: `ssh chris@192.168.1.191 cat ~/fleet/runs/pulse-lab4/private/commissioner.json | ssh claudeos node ~/cdp/signin.cjs <tab>` (never print creds). Mock copy on claudeos `~/scratch/TheWarRoom-mock-v3-FINAL.html`, shot `~/scratch/dv3/mock-home.png`.
- **Week stuck on 9 — ROOT CAUSE FOUND:** hosted `week.read` reads only the pinned football evidence (`internal/leagueservice/football.go:41` -> `PinnedFootball.ScheduleDates`). `refresh-nfl-schedule` re-fetches stale weeks (`refetchStaleNFLWeeks`, discards result) and only re-pins MISSING weeks. MFL shows weeks 9 and 10 all final (checked 11-20). Fix lane `hk-weekpin` (Haiku) on branch `session/refresh-pins-final-weeks`: re-fetched final weeks replace pinned weeks in a new evidence digest. Review its diff yourself, then it ships with a lab release (v23) and a rerun of `lab-nfl-1120.sh` in `~/fleet/runs/overnight-b9walk-1120/`. Lab refresh ran 11-20 15:07 UTC (backup `/var/lib/twr/manual-1120-nfl-refresh`).
- **Rehearsal seat freed (by accident, 11-20 ~09:00 CT):** importing `~/fleet/runs/hk-seats/free_seats.py` runs its main body. franchise.01 freed; 32/32 seats open. Join link NOT yet rotated. Never import that file; my own helper is `~/fleet/runs/dv3/wk.py`.
- **Batch 1 spec map (`.scratch/mock/v3.js` lines):** icons IC l.13-27, helpers seg/flap/ladder/rock l.64-68; frame top/band/rail/status/insp l.760-905 (+ v3 overrides l.1395-1445: board-tab close x, rail board x, status readouts as buttons stConn/stLive/stWeek, tab overflow +N); board grid COLS 24 ROWS 14 GAP 4, Home.frame/tabs/render/layout/place l.486-560; default Today board = docket(0,0,7,9) week(7,0,10,6) today(17,0,7,9) nflnews(7,6,10,8) deadlines(0,9,7,5) shortcuts(17,9,7,5); pieces dkRow+R.docket/week/today/shortcuts l.358-384, R.nflnews FINAL l.1462, R.deadlines FINAL l.1606; docket() lanes need/wait/land l.181-221. v3.css now at `frontend/src/app/v3/v3.css` (verbatim).
- **PR #451** (week pin fix) open, CI watcher armed; merge when green, then lab release v23 + rerun `lab-nfl-1120.sh`, expect week.read current 11.
- **News source research:** Haiku lane `hk-newsrc` (brief `briefs/hk-newsrc.md`, input `~/Downloads/news_api.md` from Gemini) -> `~/fleet/runs/hk-newsrc/RESULT.md`. Verify its top pick yourself before building the news read.
- **Open question for Christopher (superseded: he asked for research, see above):** NFL news piece has no hosted read (none in `internal/commandcatalog/native_reads.go`). Needs a source decision.

## 0. Christopher's rulings for this workstream (2026-11-20, read first)
- **Finish the product to the mock. No slop, no "ship less" options, the date never cuts quality.** Sunday 11-22 is a checkpoint, not the finish. (memory: feedback_twr_finish_product_not_ship_date)
- **Claude (head brain) does the UI work itself.** No Luna. Haiku agents on Tom only for easy, mechanical jobs (inventories, greps, wiring checks, CI/merge chains).
- **Desktop target = `~/Downloads/TheWarRoom-mock-v3-FINAL.html` (Beelink).** Phone target = `~/Downloads/TheWarRoom-mobile-mock-v1.html`, never a blend.
- **All UI testing on Claude-OS at 1920x1080** (`ssh claudeos`; shared desktop — check idle first, leave it running). Done for a screen = side-by-side screenshot with the mock at 1920x1080, taken by Claude, before Christopher sees it.
- His verdict on the current desktop (thin mode PWA, screenshot `/home/chris/content/feedback/2026-11-20_08-51.png`): "the UI is dogshit, not close". **The wiring (server reads/commands) is likely in place; the UI is the job.**
- Merges for this work are approved (squash, after green CI). Hard limits in CLAUDE.md still apply (no email, no Proxmox host changes, never print secrets/join codes/passwords, never --no-verify).

## 1. What we are doing
Bring the thin-mode desktop app (the product, `docs/architecture/desktop-modes.md`) up to mock v3 region by region, then the phone to mobile v1. Repo: Beelink `~/work/TheWarRoom`, worktrees in `.worktrees/`, branch per piece `session/<desc>`, never work on main.
Key fact: most v3-looking pieces exist in the **legacy local shell** `TargetApp.tsx` (Home, War Room, HQ, Pulse, Control Room, command bar). Thin mode (`NativeStartup.tsx` → `NativeAccount` → `ThinHome` → `HostedLeague`/`HostedWorkflows`) got a bare rail + Pulse. Plan: bring the v3 shell into thin mode, feed it from hosted reads, then polish each panel to the mock. Legacy mode must not grow (desktop-modes.md).

## 2. Done looks like (all NOT met yet)
1. Every region of mock v3 exists in the thin app with real data: top bar (league picker, search/command, Needs you, Assistant, Feeds, avatar), NOW ticker, tab strip (Home, Today, Game day, All leagues live, Commissioner desk, Front office, Board, Add a piece, Edit), left rail (App nav, My shortcuts, league picker list), Home panels (Docket, This week across my leagues, My players today, NFL news, Deadlines, Shortcuts), status bar. Side-by-side at 1920x1080 on Claude-OS.
2. Phone: every tab matches mobile v1, side by side.
3. Data behind the panels is right on the lab (current week, lineups, projections, values) — empty panels from missing data are defects.
4. Tracker `~/Downloads/twr-usable-week-plan.html` lists each region matched/not with screenshots.

## 3. Next actions, in order
1. **Gap list (Claude, by hand):** open mock v3 at 1920x1080 and the thin app (commissioner sign-in) at 1920x1080; for each region record: mock look, existing code (thin or legacy file), the hosted read that feeds it, grade match/partial/missing. Write to `docs/desktop-v3-gaps.md` on a branch.
2. Build the shell first (top bar, tab strip, rail, status bar) in thin mode, then Home panels, then the other tabs. One PR per piece, screenshot-checked against the mock before merge.
3. Lab week fix (see §4) — must be done before any screenshot judging "week" panels.
4. Small open item: sign-in screen label. Worktree `.worktrees/signin-label`, branch `session/signin-code-optional`, **no changes made yet**. Change `NativeAccount.tsx:307`: on the login screen label "Authenticator code (only if you set one up)"; keep "Six-digit authenticator code" on the authenticator screen. Add a test in `frontend/src/app/control/nativeAccount.test.tsx`. (Rehearsal also saw a button click not submit on the first try; unproven, don't chase without evidence.)
5. Lab housekeeping before any outside join: free the rehearsal seat (franchise.01 BUF, creds `~/fleet/runs/pulse-lab4/private/rehearsal-owner.json`; use `~/fleet/runs/hk-seats/free_seats.py` pattern — it skips only the commissioner seat), then rotate the join link (summary of CLI in §6).
6. Hermes weekly job (Tue 03:00 CT) to run refresh-nfl-schedule on the lab (stop server, refresh, start) so the week keeps advancing until the server refreshes on its own (post-fix design item).

## 4. In flight at handoff
- **PR #450** MERGED 7968877; lab release **v22 DONE** (79688779) at ~09:00 CT 11-20.
- **After v22:** run on the lab (server must be stopped): copy of `~/fleet/runs/overnight-b9walk-1120/lab-nfl.sh` (it backs up, stops, runs `twr-server refresh-nfl-schedule -db /var/lib/twr/server.db -league native.mfl-14432-2026 -season 2026` as user twr, starts, prints status) via `bash ~/fleet/runs/overnight-b9walk-1120/labrun.sh <script>`. Expect "Re-fetched results for N past week(s) that were not final: 9, 11." (11 only if its games are over). Then `week.read` should give current week 11 (Sunday 11-22 is week 11). If the clock still says 9: the archived re-fetch is not what the schedule cache reads — next suspect is `internal/app/commissioner/native/schedule_cache.go` (reads `History.ReferenceBodies`, newest copy first; cache keyed on archive mark) vs where the transport sink writes. Medium confidence on that hypothesis.

## 5. State / gates
| Item | State |
|---|---|
| Lab CT 260 | release v22 (79688779) live; schedule refresh NOT yet run. 1 commissioner account + rehearsal owner on BUF. 31 open seats. |
| PR #448 Live follows selected league | merged, lab v20 |
| PR #449 phone Team skips value.read (banner gone) | merged, lab v21 |
| PR #450 stale-week re-fetch | merged, v22 in flight |
| CI on main | green at #450 merge |
| Backup | `/var/lib/twr-backup-sat-1120/twr-full.tgz` 203,087,044 B, sha256 73da36…; restore verified (16 dbs integrity ok) |
| Rate limits (labcfg lane) | fine: per-IP via CF-Connecting-IP, 10 auth/min/IP, 64 in-flight global, 6 auth in-flight; sessions 7d idle/30d |
| Server drift since 11-16 | no migrations, no TODO/panic; draft.queue seat check concern REFUTED (seat comes from caller's own principal, `draft_queue.go:88-111`) |

## 6. Refuted / known — do not retest
- Team-tab "Some reads are unavailable" = `value.read` 409 for everyone: lab has no values feed ("player records and fitted parameters are not available on this host"). Fixed by #449 for phone; desktop HQ still reads values — the values feed itself is a data item.
- Week stuck on 9 = held nflSchedule copy for week 9 fetched before games ended (`gameSecondsRemaining` != 0); week 11 copy fetched Thu 11-19 16:02Z. `leagueweek.LineupWeek` needs all games Final.
- Empty starters / PROJ 0.0 on lab = no lineup set + no NFL stats held (projections.read available:false). Data, not code.
- Live "updated 9:12 AM · ESPN · provisional" = board line, as designed; "Updated n min ago" only inside a followed game.
- Commissioner sign-in does NOT need an authenticator (ruling 11-14, `accounts/totp.go:221`). Leave the code box empty. Browser sign-in verified 200 with stored creds.
- Blank screen after sign-in on the installed PWA = stale page across the v21 restart; fresh browser renders Home fine.
- golangci-lint-full "153 issues naming a deleted worktree" = stale cache: `golangci-lint cache clean`, then commit normally.

## 7. Tools and access
- Lab shell: `bash ~/fleet/runs/overnight-b9walk-1120/labrun.sh <script-file>` (busybox; no python/sqlite/journalctl; log `/var/lib/twr/server.log`). Lab URL `https://twr-lab-260.secureprospective.com/app/`, league `native.mfl-14432-2026`.
- API helpers: `~/fleet/runs/hk-seats/free_seats.py` (login/post; workflow reads: season/week are integers). Creds in `~/fleet/runs/pulse-lab4/private/` — never print.
- Lab release: `~/fleet/bin/twr-lab-release.sh <V> <prev>/release-v<prev>/metadata/<prev>.targets.json <run-dir>` (builds origin/main).
- Haiku launcher: `~/fleet/runs/overnight-b9walk-1120/hkx.sh <lane>` with brief `briefs/hk-<lane>.md` → worktree `.worktrees/hk-<lane>`, output `~/fleet/runs/hk-<lane>/{stdout.log,completion.txt}`. Launch: `setsid nohup bash .../hkx.sh <lane> >/dev/null 2>&1 </dev/null &`.
- Headless screenshot without playwright: node 24 + brave `--headless=new --remote-debugging-port=…` over CDP (built-in WebSocket); fill React inputs via the native value setter + input event, `form.requestSubmit()`. Redact the password from output. (For the real check use Claude-OS 1920x1080.)
- Christopher installed the PWA on the Beelink in Brave (shows "Uninstall" in title bar).

## 8. Honest status
Wiring and server: solid, CI green, lab healthy. Desktop UI: far from mock v3 — essentially unstarted against the mock. Phone: close to mobile v1 per the earlier walks, not yet side-by-side verified at full fidelity. ETA for desktop v3 match: several days of focused work; not Sunday.

---
## COMPACT POINT 2026-11-20 ~13:40 CT — Control Room v3 built with AGY (Claudebox = head-brain)

**State:** Control Room v3 is DONE on the branch locally, verified on Claude-OS against the lab league (all 10 tabs,
rail leaves + strip + body agree). Local commits (NOT pushed): 51d035df6 hook, b41ec5a6c shell mount, 5769e9b7b Jobs,
ffa67fde1 ten bodies, fdac8da89 tests, 72d21415d Rules/Season/League pass 2. Also 64d8815ee (`join-link -take -no-login`).
Last pushed: b5b2fa53a.

**PUSH FAILED** at 13:38: pre-push `Test failed` — test name not captured (output was tail -3). FIRST NEXT ACTION:
`cd frontend && npm run build`, then `git push 2>&1 | grep -E "FAIL|×" -A15` to see which test; fix; push. Known flaky under
load: `commissioner/hostedRules.test.tsx > NativeAccount admits a teamless Commissioner...` (passes alone and on rerun).

**Files:** v3/ControlRoomV3.tsx (BODIES map, shell owns tab), v3/Ctrl{Jobs,Situations,Rules,Season,Fix,Picks,People,
Feedback,League,Server}V3.tsx (+tests); commissioner/HostedControlRoom.tsx exports useHostedControlRoom,
HostedControlRoomState, HostedControlRoomBody; shared hooks useProblemReports, useHostedSeason (+ League/Rules hooks).
V3Shell: crTab state, hostedControlTabs strip + rail leaves, ControlRoomV3 lazy mount for entry 'control'.

**Seat:** Christopher is COMMISSIONER + OWNER of Arizona (franchise.25) on the lab via new `join-link -take -no-login`
(backup /var/lib/twr/manual-1120-seat-az). Owner-only flows now testable.

**AGY (fleet agent, Beelink):** plugin ~/.gemini/config/plugins/fleet (rules, skills links, guard.sh PreToolUse,
stopguard.sh Stop); launcher ~/fleet/bin/agy-run.sh <job> <brief> [model]; briefs ~/fleet/briefs/agy-cr-*-1120.md;
runs ~/fleet/runs/agy-cr-*-1120/. Routing + calibration in memory feedback_agy_routing_1120.md. AGY Claude/GPT
5-hour bucket EXHAUSTED at 13:30 (resets ~18:10 CT) → use Hydrogen for coding until then. Christopher will not use
AGY while Claudebox runs it (his sign-in knocks ours out). Test sign-in with a Flash "Reply: OK" before launching.

**Side quests found (not chased):** ledger/deputies show raw account ids (acct.…) not names — server read;
Beelink disk 87% (go cache 79G: `go clean -cache` when no build runs); fallback text "this part … not on the server app
yet" appears in Season Divisions and League; Season calendar first row has no name in the data.

**NEXT ACTIONS:** 1) fix the push (above). 2) Christopher's walk of Control Room on http://127.0.0.1:5199/app/.
3) Then phone mock / side quests per his list. Dev server (vite pid 1663256) still running on purpose.
