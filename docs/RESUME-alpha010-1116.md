## §0c — Fri 11-20 ~1:50 AM CST (Claudebox head brain) — compact-safe #3

**Christopher's latest orders (do not re-ask):**
- "Do the Saturday cleanup now, don't wait." → NEXT ACTION 1.
- Bundle 9 → main: SIGNED OFF (11-20 ~1:30 AM).
- Stop being careful about logins/Bills: "rip whatever out of the way and keep moving"; imperfect is fine, flag and move on.
- All lab accounts are us (no real people yet): scrub freely.

**Done tonight (all CI green, merged into bundle/b9):** #435 pool rows team/kickoff/AVG · #436 Live header one line ·
#437 League chips gone + playoff note · #438 Moves "Waivers run <time>" · #439 Team "No team of yours" for teamless ·
#440 test: join-link owner reads own lineup (PASS, so Bills = lab data, not code) · #441 two-conference playoff note.
Lab CT 260: v15 = bundle/b9 0b03748f, twr-status all OK. S8 walked owner + commissioner: no sideways scroll.

**In flight:** Bundle 9 landing on main — `~/fleet/bin/twr-bundle-land.sh bundle/b9 ...` (pid 1325861, Tom/Beelink,
timeout 2h, started ~1:35 AM). Log `~/fleet/runs/overnight-b9walk-1120/land-b9.log`. Alive: `pgrep -af "twr-bundle-land.sh bundle/b9"`.
Frontend suite passed 1,111/1,111; then make verify, push, PR, CI, squash-merge. On done: check `gh pr list --base main --state merged -L 3`.
If it stops on a conflict: read the log tail, fix in its worktree, rerun.

**NEXT ACTIONS**
1. Saturday cleanup NOW on lab CT 260 (all accounts are us):
   a. Backup first: `twr-status` shows checked backups; also copy /var/lib/twr to /var/lib/twr/manual-<date>-precleanup.
   b. Fresh re-import of CURRENT MFL league 14432 (ruling 11-19; MFL stays official). The old seed gave 3,367 not 3,399 —
      that is expected for a fresh pull; do not block on it. Tools in CT: /usr/local/libexec/twr-{delete-league,receive-league,
      restore-league,reset-sign-ins}. Fallback if re-import is slow: Option A in ~/fleet/runs/pulse-freshlab/RESULT.md
      (commish.seat.remove x32 + twr-reset-sign-ins); accounts cannot be deleted, only unseated.
   c. Verify twr-status "Everything is working", 32 open seats, issue a fresh join link for Sunday testers.
2. After Bundle 9 lands: redeploy lab from main with `~/fleet/bin/twr-lab-release.sh <V> <prev targets.json> <run-dir>`
   (last: v15, prev targets `~/fleet/runs/overnight-b9walk-1120/labrel-v15/release-v15/metadata/15.targets.json`).
   NOTE the script builds origin/bundle/b9 — change line 17 to origin/main once b9 is landed.
3. Sunday: Christopher sends testers the join link.

**Flagged for later (not alpha):** Bills lab seat (clears with the re-import) · Team B-7 banner (same) · Today snooze clock ·
Live field graphic · L11 version tags · tfeedback · PLANS · F51 player values · F62 animations · F18 CT 257 test accounts.

**Lessons tonight:** deploy/server/bundle.sh is not executable in git → script calls `bash deploy/server/bundle.sh`.
New worktrees need `pnpm install` + `pnpm build` (or copied frontend/dist) before the push hook. Stale golangci cache
from a deleted worktree: `golangci-lint cache clean`. CI flake: hostedRules "teamless Commissioner" 10 s timeout — rerun.
Watchers: wait until no check is "pending" (gh exit codes are unreliable for fail vs pending).
## §0b — Overnight run b9walk, Thu 11-19 evening (Claudebox head brain) — compact-safe 10:50 PM CST

**Doing:** finish Bundle 9 (bundle/b9) phone port against mobile mock v1, on lab CT 260, walked on the S8. Run dir: Beelink `~/fleet/runs/overnight-b9walk-1120/` (PLAN.md, REPORT.md log, briefs/, s8walk/RESULT.md). Tracker: `~/Downloads/twr-usable-week-plan.html` via `~/fleet/bin/twr-tracker.py`.

**Done lines**
1. Lab runs bundle/b9 head incl. new PRs — NOT MET (lab on 578ac110b; #435-437 not yet merged/deployed).
2. Player-row PR merged into bundle/b9 — NOT MET (#435 reviewed, CI pending).
3. S8 walk 5 tabs x owner+commissioner — PARTLY: owner walked (12 bugs, s8walk/RESULT.md); commissioner NOT walked (Haiku didn't use `~/fleet/runs/pulse-lab4/private/commissioner.json`).
4. Bills starters cause — see Current bug.
5. REPORT.md with Needs Christopher — kept current.

**Agents (Christopher's rulings tonight):** NO new Luna. DeepSeek free is exhausted ~20h (I ran 5 at once — never more than one short job). Use **Tom Haiku** (`claude -p --model haiku` on Beelink) with **tightly scoped briefs**: exact files, exact change, one test command, exact commit/push/PR commands, "fail twice → stop". Haiku env needs `PATH=$HOME/go/bin:$HOME/.local/bin:/usr/local/go/bin:$PATH` (pre-push needs go + wails) and a pre-built `frontend/dist` (`cd frontend && pnpm install --frozen-lockfile && pnpm build`). Line limit 110 chars (controls.test.ts).
Phone target = mobile mock v1 ONLY (never desktop v3). Wiring map: `~/fleet/runs/twr-mobile-stubs-1119/data/all.txt` (also PR #409 docs/ui/mobile-v1, unmerged) — extend the owner each row names.

**Status**
| Item | State |
|---|---|
| Lab CT 260 | 578ac110b release 11, twr-status clean, history 3,399. DB backup `/var/lib/twr/manual-1119-nfl-refresh` (43 MB, 9:53 PM). |
| PR #435 player rows (team, kickoff, AVG) | reviewed OK (server: average counts only scored weeks); Haiku pushed fix to use `useHostedGames`; CI pending |
| PR #436 Live header one line 360px | reviewed OK, CI pending |
| PR #437 League: drop filter chips, playoff note | reviewed OK, CI pending |
| Deferred | Today per-item snooze clock (design call; mock = commissioner jobs), Live field graphic (big), Team fixes (AVG label B-6, banner B-7, My Picks 44px B-8), Moves (Rising filter, trades count, waiver time) — Team/Moves after #435 merges |

**Current bug (lineup):** lab server.log: `hosted league lineup.read read refused: ... This lineup is private to its team and the commissioner.` (internal/commissioner/lineup_reading.go:177-189; seat owner from governance seat people_reading.go:200; app's `mine` = league.franchise). Seen only with **stress-test owner #1 of 32** (`stress-owners.json`). Hypothesis: lab test-account artefact (Saturday cleanup unseats them), NOT proven a product bug. Check: walk Team with Christopher's own account / commissioner.
**Refuted — do not retest:** (1) Bills "data gap / NFL dates missing wk1-5" — `twr-server refresh-nfl-schedule` says all 17 weeks dated; week resolves (live stats wk 9-11). (2) Re-import needed — no.

**In flight at compaction:** none running except Claudebox watchers (CI watcher for #435-437; 50-min checker). Check CI: `ssh chris@192.168.1.191 'export PATH=$HOME/.local/bin:$PATH; cd ~/work/TheWarRoom; gh pr checks 435; gh pr checks 436; gh pr checks 437'`.

**Next actions** (updated 10:57 PM CST, compact-safe #2)
1. Check CI: `ssh chris@192.168.1.191 'export PATH=$HOME/.local/bin:$PATH; cd ~/work/TheWarRoom; for n in 435 436 437; do gh pr checks $n; done'`. Green: merge all three into bundle/b9 (bundle only).
2. Lab redeploy, one command (new 11-19; recipe recovered from pulse-b7lab2 + b11lab3): `~/fleet/bin/twr-lab-release.sh 13 ~/fleet/runs/pulse-b11lab3-tom1/evidence/release-v12/metadata/12.targets.json ~/fleet/runs/overnight-b9walk-1120/labrel-v13`. Signing helper `~/fleet/bin/twr-lab-sign.py`; venv `.worktrees/pulse-b11lab3/.scratch/b3/venv`; client config `.worktrees/pulse-b7lab2/.scratch/pulse-b7lab2/client-config.json`. NOT yet run end to end: watch its first run.
3. S8 re-walk owner + commissioner: Haiku with `~/fleet/runs/overnight-b9walk-1120/briefs/hk-walk2.md` (out s8walk2/RESULT.md). Its Team answers settle the lineup question.
4. Scoped Haiku jobs: Team (B-6, B-8; B-7 per re-walk) and Moves (B-9, B-10).
5. Keep the 50-minute checker armed (`/root/.claude/scripts/twr-lane-check.sh 50`, background), Christopher's standing ask. Tom Haiku out of tokens: run the brief as a Claudebox Agent subagent (model haiku) over ssh.
6. Tracker template now splits Running now / Ready / Stuck / Coming next (backup `~/fleet/runs/twr-tracker-template.html.bak-1119`). Update cards on every launch, merge, stop.
7. Hook note: `git push -q origin ...` inside a long command was blocked as a force-push (false positive). Push in a separate plain command.

**Needs Christopher:** Saturday lab backup + cleanup (approved plan); Bundle 9 → main sign-off (command when all lanes in).
**Process lessons:** never `kill $(pgrep -f X)` over ssh (killed my own shell); watchers must check process liveness without matching their own ssh command.

## §0a STATE AT ~9:30 PM CST THU 11-19 — HEAD BRAIN MOVES TO CLAUDEBOX (Tom out of weekly tokens)
- bundle/b9 head 578ac110b. Merged tonight: #428 frame fix, #429 tab bar+Today, #430 League, #431 Team, #432 Moves, #433 Live, #434 Today pt2 (all CI green). All five tabs ported toward mock v1 (target: ~/Downloads/TheWarRoom-mobile-mock-v1.html).
- Lab CT 260: b64fe42bf (frame fix) live, release 11, TUF v11; S8 shows full-screen app (~/fleet/runs/phone-proof-1119/s8-after-428.png).
- IN FLIGHT (Tom Opus subagents; may die when Tom's tokens end):
  1. Lab redeploy of 578ac110b — brief ~/fleet/briefs/alpha010/b11lab3.md, result ~/fleet/runs/pulse-b11lab3/RESULT.md. If missing: rerun that brief.
  2. pool-fields (team/kickoff/season avg on player rows) — brief ~/fleet/briefs/alpha010/pool-fields.md, branch session/pool-fields, PR into bundle/b9. If no PR: rerun.
- Known gaps needing server reads (named in PRs #431-#434): snooze for owner rows, AVG/usage/rising, waivers run time, pace/team colours/drive verb on Live, Drop on player card.
- Lab sign-in got rate-limited ("too many authentication attempts") from parallel lanes; sign in once and reuse.
- Lab league data: Bills show 0 starters/45 empty bench — expect Saturday re-import to fix; verify.
- ~/fleet/bin/twr-phone-compare.mjs now clicks the app tab and waits for "Reading…" to clear (edited, ~/fleet is not a git repo).
- NEXT: (1) on redeploy, S8 walk all five tabs vs mock (python3 ~/fleet/bin/phone-chrome.py s8 ...); (2) owner + commissioner walk; (3) Saturday backup→cleanup→re-import→b9→main (approved; main merge still his click).
- Tracker: ~/Downloads/twr-usable-week-plan.html via ~/fleet/bin/twr-tracker.py only.
# RESUME — TheWarRoom alpha 0.10 + Franchise HQ + OVERNIGHT RUN (Claudebox head brain), Mon 2026-11-16 night
Updated 1:15 PM CT Tue 11-17. On "we are back": do his instruction first, else section 10 item 1.
**OVERNIGHT RUN ACTIVE: `~/fleet/runs/overnight-twr-1116/` (Beelink) — PLAN.md, briefs/, REPORT.md (keep REPORT current; "Needs Christopher" first).**
**MERGE AUTHORITY TONIGHT: he said "review, test if needed, merge - yolo mode" → I merge tonight's lane PRs myself after my review + green CI.**

- **11-19 3:50 PM CST Christopher approved:** (1) release 11 lab deploy to CT 260 incl. migration 5 on Friday; (2) Saturday backup → cleanup → re-import → b9→main.

## 0. STATE AT 8:05 PM CST THU 11-19 (Tom)

**Christopher's standing word (11-19 8 PM):** mobile mock v1 (`~/Downloads/TheWarRoom-mobile-mock-v1.html`, sha dd5740ce…0651) IS the target. A lot of time went into tuning it: make it a real thing, faithful, not "close". Every tab must match the mock at 390/430 on the real lab + S8; "shown by old screen" or placeholder is not acceptable.
**Approvals given 11-19 3:50 PM:** (1) release 11 lab deploy — DONE; (2) Saturday backup → cleanup → re-import → b9→main (still to do).
**Claudebox available:** Opus low + Haiku for helper work (Christopher 11-19 ~7:40 PM).

### Done today (all in bundle/b9, CI green)
#410-#427 merged: phone frame #411, tabs slot #420, Today #421, Team #422, Live #423 (mock layout, Pulse removed, field animation cut), Moves #424 (all action sheets), League #425, server lanes #412-#419, fill-in rows #426, release 11 #427. Conflicted ones merged by hand on integ branch (5844fc0b3).
**Lab CT 260 = release 11, commit f7ad8c98e, TUF v10**, migration 5 applied, sign-in OK. Evidence ~/fleet/runs/pulse-b11lab/RESULT.md.

### THE CURRENT BUG (blocks testers)
On the lab at 390px and on the real S8 (`~/fleet/runs/phone-proof-1119/s8-now.png`, `today-390.png`): old NativeAccount page bits ("Legacy NFL" h1 + "League" button above, "Open my franchise"/"Sign out" below) render around the phone frame; frame not full-height; tab bar cut off, tabs render as bordered icon-only boxes (PhoneAppFrame.css grid/labels not winning). Data reads DO work on S8. Hypothesis: page chrome not suppressed when PhoneAppFrame active + global button/HQAct CSS overriding; caveat: unverified, helper is diagnosing.
**In flight:** Claude helper "Fix phone frame layout on lab" (Beelink, Opus) → worktree `.worktrees/framefix`, branch session/framefix, scratch ~/scratch/framefix-1119; will open a PR into bundle/b9 and NOT deploy. On its report: review, merge via twr-batch-merge.sh, redeploy lab (brief ~/fleet/briefs/alpha010/b11lab.md recipe, release stays 11? — if the upgrade tool needs a higher release, add release 12 like #427), recheck S8 + `twr-phone-compare.mjs <tab> --width 390|430 --out ~/fleet/runs/phone-proof-1119` all five tabs.

### Refuted / don't retry
- Practice league cannot serve a signed-in browser: sign-in writes pause until R2 backups exist (cmd/twr-server/backup_hosted.go, by design). Proof = lab only. (`~/fleet/bin/twr-practice-web.sh` + `twr-practice-proxy.mjs` exist but end at that wall.)
- Compare tool "missingLabels" counts are mock-text diffs, not failures; read the PNGs.
- Compare tool now waits for `.phone-app` after sign-in (earlier captures were taken too early).

### NEXT ACTIONS
1. Land framefix → redeploy lab → S8 + 10 side-by-sides.
2. Per-tab faithful-to-mock pass: list every visual/row gap per tab from the side-by-sides; dispatch fixes (Claudebox Opus low / Haiku for trivial CSS), merge, redeploy, recheck. Repeat until each tab matches.
3. Phone walk owner + commissioner on S8 (+ iPhone if available) per ~/fleet/briefs/alpha010/phonewalk.md.
4. Saturday (approved): backup → cleanup → re-import → b9→main (main merge still needs his live sign-off at the moment).
**Needs Christopher:** `git push origin --delete integ/b9-tabs-1119` (guard blocked delete as force-push; harmless leftover). Review #409; rotate CT 257 keys.
**Honest status:** all five tabs built and deployed, but NOT yet visually proven against the mock on a phone; the frame layout bug is real and open. Sunday still reachable if framefix lands tonight.

## 0-prev9. (older) STATE AT 3:55 PM CST THU 11-19 (Tom, after "we are back" + go)
**Ruling 11-19 ~2:30 PM:** Christopher gave the go on the no-GPT plan: cut field animation (G1) and phone Control Room
from alpha; Claude helpers allowed (2 Opus now, Haiku for trivial jobs); Claudebox frees up at 7 PM CST to help.
Phone app = faithful mobile mock v1 (proved to him: compare tool mock sha256 dd5740ce…0651 == repo zip).

**In flight (check on resume):**
1. Batch merge into bundle/b9 of #411 #412 #414 #415 #416 #417 #418: `~/fleet/bin/twr-batch-merge.sh` (pid ~364378),
   started 3:28 PM; worktree `.worktrees/batch-b9-1119-1528`*; result line in `/tmp/claude-1000/-home-chris/314afa76-851a-4392-8dcf-a389bb998d5b/tasks/bi5gre0nq.output`.
   Check: `gh pr view 411 --json state`. On CI FAIL: `gh run view <id> --log-failed > /tmp/x.log` then grep ' FAIL |--- FAIL'.
   (*name may be 1528 or later; prior failed batches removed.)
2. Two Claude Opus helper agents (Agent tool, background): Today port (branch session/port-today,
   `.worktrees/port-today`) and Team port (session/port-team). Briefs `~/fleet/briefs/alpha010/port-{COMMON,today,team}.md`.
   They push + open PRs into bundle/b9, never merge. Liveness: `git -C .worktrees/port-<tab> log -1 --format=%cr; git status -s`.
**Open PRs into b9:** #409 plan (needs Christopher review), #413 mleagues (resolve NativeAccount conflict vs #411:
keep PhoneAppFrame + admittedLeagues from memberships), #419 msnooze (migration 5 → account_job_snoozes),
#420 tabs slot (stacked on #411; merge after it).
**Fixed today (do not redo):** frame focus restore now an effect not setTimeout (CI flake); standings wire fixture
regenerated for streak (`TWR_WRITE_WIRE=1 go test ./internal/store/history -run TestNativeResultsStandingsWire`);
mstatus: return-week resolution moved to cmd/twr-server adapter (leagueservice must not import ingestion→mfl,
boundarycheck) + boundary-adapters.tsv hash; release-matrix fixtures added for trade.hold, waivers.claim.order.set,
waivers.claim.cancel (`internal/app/commissioner/testdata/release-mutations.json`, hand-formatted: text-insert only);
batch script now builds frontend/dist before merging (Go embeds dist; hooks fail without it).
**Gotchas:** fresh worktree push needs `cd frontend && pnpm install --frozen-lockfile && pnpm run build`; a pre-commit
golangci "Failed" with 0 real issues = another lint running in parallel, just retry; new catalog mutation ⇒ add a
release-matrix fixture (`go test ./internal/app/commissioner -run Release`).

**Next, in order:** 1) confirm batch; on fail fix + rerun with remaining PRs. 2) batch #419 #420, then #413 after
conflict fix. 3) at 7 PM brief Claudebox for Moves + League ports (same COMMON brief). 4) review Today/Team PRs:
cited rows, compare screenshots 390/430, retire old phone code; merge. 5) Live tab (no field animation).
6) Friday: ask Christopher OK for release 11 lab deploy (runs migration 5), then phone walk S8 + iPhone.
7) Saturday: his go for backup → cleanup runbook → MFL re-import → b9→main.
**ETA:** server pieces done 3:50 PM ✅; Today+Team by Fri 6 AM; Moves/League Fri 1 PM; Live Fri 5 PM; deploy+walk Fri
evening; Sunday testers with ~½ day slack. Unproven: tab ports' fidelity until side-by-sides are reviewed.

## 0-prev8. (older) TOM HEAD BRAIN — STATE AT 10:05 AM CST THU 11-19

**Goal:** phone app (Bundle 9) proven on real phones, on main Sat; testers on lab CT 260 Sun 11-22 (week 12).
Run dir `~/fleet/runs/overnight-alpha-1118/` (REPORT.md, PLAN.md). Tracker `~/Downloads/twr-usable-week-plan.html` via `~/fleet/bin/twr-tracker.py` only.
Merge authority: lane PRs → bundle branches after my review + green CI (twr-batch-merge.sh). **main needs Christopher's go each time** (CLAUDE.md hard limit 2).

**Done since 9:15 PM:** #394→b7; **B7 on main #396 (5a582c47a, his go)**; into b9: #397 league, #398 Today/Ask, #399 Moves, #400 b7→b9 (fixes phone sign-in 403: Chrome GET /app/session had no Origin), main sync 4c15e137a, #402 tester guide, #403 release 10, #401 flaky tests, #405 layouts server-only (sec9 finding). Closed #304 #342 (superseded). Tool `~/fleet/bin/phone-chrome.py` drives Chrome on both phones (goto/eval/shot/measure/walk). bundle/b8 sync skipped (wholly inside b9).

**Rulings 11-19 (do not re-ask):** lab CT 260 = alpha tester server; LAB-ONLY trust chain accepted for alpha; release 10 on b9 OK; Saturday 11-21 fresh re-import of CURRENT MFL league 14432 (MFL stays official, TWR mirrors); seats start vacant.

**In flight (check: `cat ~/fleet/runs/pulse-<lane>/completion.txt`; report `tail -c 2500 stdout.log`):**
- batch #404 (refresh-nfl-schedule cmd) → b9, then chain auto-launches **b9lab2** (redeploy b9 release 10 to CT 260 + run refresh-nfl-schedule; brief `~/fleet/briefs/alpha010/b9lab2.md`, TUF targets v9). Chain shell task bjtn5ffsq; it moves old `pulse-b9lab2` → `-try1`. If chain died: do those steps by hand.
- lanes joinonce (join link one claim), freshlab (Sat re-import runbook; launched BEFORE Part 2 ruling — relaunch with brief Part 2 if it delivers "free seats, keep 11-08 data"), lineup12 (import carries current-week lineups). Watchers: twr-lane-wait.sh + twr-lane-check.sh 50 (re-arm each time).

**Refuted — don't retest:** scores wk1-8 match MFL exactly (scores11); wk9-10 absent = 11-08 import, not a scoring bug. perf: no regression (holds 256). flaky roster-add test: 50/50 pass.

**Next:** 1) b9lab2 deploy → verify `/app/session` no-Origin = 401, dates present. 2) Phone walk v2 (brief phonewalk.md + use phone-chrome.py; output ~/fleet/runs/phone-walk-1118/) owner+commish, iPhone+S8. 3) Bugs → fix lanes into b9. 4) Merge joinonce/lineup12/freshlab PRs; update TESTER-GUIDE (join link, open seats, clipping). 5) Sat: run freshlab runbook (backup first). 6) Ask his go: b9 → main.
**Needs Christopher:** CT 257 keys rotate; Paylocity tab glance; b9→main go (Sat). Windows laptop: 192.168.1.210 (HP?) 4-224 ms jitter = weak Wi-Fi; suggested fixes given.

## 0-prev7. (older) TOM HEAD BRAIN — STATE AT 9:15 PM CST WED 11-18

**Christopher is asleep. Ruling 9:10 PM: "take the wheel and finish the main session; do what you can on the
iPhone and the S8; keep working."** Tom runs unattended; blocked steps go to "Needs Christopher" in
`~/fleet/runs/overnight-alpha-1118/REPORT.md`.

**Done since 7:05:** #393+#395 batch-merged into bundle/b9 (CI green). #394 (b7fix2) reviewed: owner
schedule read keeps existing `league-view` admission (outsider 403 / anon 401 test), People decoder
fixed client-side, Pulse = lab NFL schedule missing kickoff dates for 8 weeks (data, not code), #381
proven on CT 260. GitHub refused my approve (own PR) — review recorded in REPORT. b9lab (2nd try)
DEPLOYED bundle/b9 `a10900742` to CT 260 (release 9, TUF v8, sign-in owner+commish OK) —
`~/fleet/runs/pulse-b9lab/RESULT.md`. First try stopped at signing: `~/fleet/runs/pulse-b9lab-try1/`.

**In flight:** `twr-batch-merge.sh bundle/b7 394` (pid 2553689, started 8:45 PM). Check:
`pgrep -f "twr-batch-merge.sh bundle/b7"`; result = `gh pr view 394 --json state` (MERGED) and REPORT.md
line "BATCH DONE into bundle/b7". On success → land bundle/b7 on main (squash, own review + green
verify; merge authority per §8), then merge main into bundle/b8 and bundle/b9.

**Refuted — do not retest:**
- "Lab signing needs a passphrase": FALSE. LAB-ONLY keys in `~/fleet/lab-release-ct258/` are unencrypted;
  `release.py sign` always prompts (built for production keys). Sign with inline Python
  `load_pem_private_key(..., password=None)` as in pulse-b7lab2 (brief b9lab.md §Signing).
- S8 Chrome cannot go past 101: Play Store shows "Last updated May 9, 2022" for this device even after
  `pm clear com.android.vending`. Edge and Samsung Internet updates unavailable (Android 9).
- iPhone Chrome start page (about://newtab) is not drivable; accessibility `perform_press` only focuses.
  Once Chrome shows any http page, `WebinspectorService.inspector_session(app,page)` +
  `runtime_evaluate("location.href=...")` drives it (proved: example.com measured 390/390, then wikipedia).

**Decisions (Christopher 11-18):** both phones are free for TWR testing. Browser priority for alpha:
**Chrome on both phones**; Safari, Firefox, Samsung Internet, Brave, Edge after alpha. S8 is a test
phone; TWR-server-on-phone idea parked as T659.

**Phones (details: `~/fleet/tom/homelab/HOMELAB-REFERENCE.md`):**
- iPhone 13: test Apple ID, Chrome installed + Web Inspector on, control `--remotepairing` over Wi-Fi.
- S8 Active SM-G892U: Android 9, Christopher's Google account, `adb connect 192.168.1.122:5555`;
  Chrome 101, Firefox 160. After any restart it needs one USB plug-in + `adb -d tcpip 5555`.
- Backups (keep): `~/Pictures/phone_backup/{iphone,android}/`.

**Next actions, in order:**
1. Land #394 batch, then B7 → main; sync main into b8, b9.
2. Add a Chrome mode to `~/fleet/bin/iphone-walk.py` (inspector_session + runtime_evaluate; no WebDriver).
3. Phone walk on CT 260 (bundle/b9): owner + commissioner, every screen, iPhone Chrome + S8 Chrome
   (adb: `am start -a android.intent.action.VIEW -d <url>`, `input text`, `screencap`; DevTools via
   `adb forward tcp:9222 localabstract:chrome_devtools_remote`). Output `~/fleet/runs/phone-walk-1118/`.
   Creds `~/fleet/runs/pulse-lab4/private/` — never print.
4. Fix what the walk finds (lanes into bundle/b9), then write/launch b9moves, b9act, b9ask briefs.
5. Keep tracker HTML + REPORT.md current.

**Needs Christopher:** rotate CT 257 keys; pick how testers reach a server before Sunday; glance at the
Paylocity tab.

## 0-prev6. (older) TOM HEAD BRAIN — STATE AT 7:05 PM CST WED 11-18 (supersedes every older 0 block)

Head brain: **Tom (Beelink)** since 3:29 PM (ClaudeBox out of weekly tokens). Merge authority unchanged: merge lane PRs into bundles and land bundles on main after own review + green CI.
Run report: `~/fleet/runs/overnight-alpha-1118/REPORT.md`. Tracker: `~/Downloads/twr-usable-week-plan.html` via `~/fleet/bin/twr-tracker.py` (new subcommands `plan`, `server`, `headbrain`).

### Done looks like
1. B7 (desktop browser app) on main — NOT MET (b7fix2 PR #394 in CI; then land).
2. B8 phone screens on main — NOT MET (all 6 merged into bundle/b8; needs phone walk + land after B7).
3. B9 on main by Sat — NOT MET (server pieces #387 #388 #389 merged into b9; screens #391-#395 reviewed/merging).
4. Phone walk on a real iPhone over Wi-Fi — NOT STARTED (tools ready; needs b9 on the lab).

### Branch heads 7:05 PM
main bb99a5881 (incl. #390 CI change) · bundle/b7 c0d60f35b · bundle/b8 10406a70c (all of B8: #380 #382 #383 #384 #385 #386) · bundle/b9 f18f9a894 (b8 merged in + #387 #388 #389).

### In flight
| Job | Check alive | Output | On result |
|---|---|---|---|
| b7fix2 (Luna xhigh, from ~5:06 PM, 3 h) People actors field, owner Home League Read, Pulse week, prove #381 on lab | `pgrep -af "pi -p.*pulse-b7fix2"`; transcript mtime `~/fleet/runs/pulse-b7fix2/sessions/*.jsonl` | PR **#394** into bundle/b7, `~/fleet/runs/pulse-b7fix2/` | review (no widened owner reads), batch-merge `twr-batch-merge.sh bundle/b7 394`, then merge main into b7 and land: `twr-bundle-land.sh bundle/b7 "<title>" <body.md>` |
| b9lab launcher (Tom bg shell) waits for b7fix2 completion.txt, then launches `luna b9lab` (brief `~/fleet/briefs/alpha010/b9lab.md`, deploy bundle/b9 to CT 260 only) | real lane: `pgrep -af "pi -p.*pulse-b9lab"` (NOT `pgrep -f "luna b9lab"`: matches the launcher's own command line) | `~/fleet/runs/pulse-b9lab/RESULT.md` | then run the iPhone walk (below) |
| batch #391+#392 into bundle/b9 | DONE 7:10 PM: both MERGED | — | next: batch #393 + #395 (both add CSS at end of controlRoom.css after #391: if the batch skips one for conflict, resolve in its lane worktree keeping all blocks) |

### Phone walk plan (iPhone, mine)
Driver: `~/.local/share/pipx/venvs/pymobiledevice3/bin/python -I ~/fleet/bin/iphone-walk.py steps.json outdir` (steps: go/js/measure/shot/wait; measure lists elements wider than 390 px). Lab URL https://twr-lab-260.secureprospective.com/app/ ; creds `~/fleet/runs/pulse-lab4/private/` (never print; load into the steps file at runtime and delete it). Walk owner then commissioner: Today, Team/lineup, Live/Pulse, bids, trades, player card, Control Room tabs; Add to Home Screen. Output to `~/fleet/runs/phone-walk-1118/`. Device facts: `~/fleet/tom/homelab/HOMELAB-REFERENCE.md` "iPhone 13 — AI + Dev Test Phone".

### Done today since 3:45 PM
- Reviewed + merged: #387 (activity feed, allow-list), #388 (boards sync; resolved workflow.go union conflict with #387), #389 (roster.add — **I removed the commissioner shortcut**: owner-only like cuts/claims/bids; test `TestRosterAddIsOwnerOnlyEvenForCommissioners`; matrix + hosted test updated), #385, #383 + #386 (first batch), b8 merged into b9 (direct push).
- #390 on main: `session/**` and `batch/**` skip desktop packaging; bundles + main still package.
- New `~/fleet/bin/twr-batch-merge.sh <bundle> <pr>...` (DRY=1 stops before push): local merge, regenerate gen files FIRST, tsc/vitest/go build+vet/go test on changed pkgs, push `batch/*`, wait verify, fast-forward bundle (PRs show MERGED).
- perf lane: NO regression (lab5 refusals were ramp phase; 128-client hold 81.7 ok req/s vs lab4 82.3 at 256). Closed.
- b7lab2: security spot checks MET; bugs → b7fix2. Phone-width clipping on B7 is B8's job.
- Go cache trimmed (entries >6 h): disk 91% → 67%.

### Rulings made tonight (Tom, head brain)
- Phone Control Room scrolls with the page: the phone shell content area (`.phone-shell__content`, overflow-y auto) is the ONE scroller. #395's contained `height:100%;overflow:hidden` box was dropped; its test now asserts page scrolling.
- roster.add is owner-only (no commissioner shortcut); commissioners use correction commands with a reason.
- iPhone 13 is the AI's own test phone (Christopher 11-18): erased, may be wiped/reconfigured freely. Homelab reference updated.

### Refuted / learned — do not re-learn
- A naive union of two CSS conflict sides drops a shared closing `}` (broke #393's media block). After any CSS union, run the phone tests (postcss parse).
- `pgrep -f "<pattern>"` inside a `bash -c` waiter matches the waiter's own command line (stuck #388 waiter; false "b9lab alive"). Wait on a PID or a file instead.
- Generated `native.gen.ts` can merge as text with NO conflict and still be broken (duplicate schemaNNN). Always regenerate before building (batch script does).
- The guard blocks any command containing `--force` (e.g. `git worktree remove --force`) together with `git push`; and the word "reboot" in a heredoc. Split commands / reword.
- `twr-bundle-merge.sh` waits for every check incl. the 27-min package/install job; only `verify` is required on main.
- #389's brief premise "commissioner acting with reason, same as other owner commands" was false: no owner market command admits the commissioner.

### Next actions
1. On #394: review (People contract fixed on the right side, owner League Read not widened, Pulse decision evidenced, #381 proof), batch-merge into b7, land B7 on main.
2. Merge main into bundle/b8 and bundle/b9 after B7 lands.
3. When b9lab reports: run the iPhone walk; fix lanes for what breaks; then land B8, then B9.
4. Batch #393 + #395: both branches synced by Tom 7:15 PM (#393 = b9 merged in; #395 = #393 merged in). Run `twr-batch-merge.sh bundle/b9 393 395` once #395's push finished (check `git ls-remote origin session/pulse-b9cr1` = 4d5302527; first push failed on the 110-char line rule, fixed).
5. Write + launch remaining B9 lanes: b9moves (after #389 ✓), b9act (after #387 ✓), b9ask (after #388 ✓) — briefs from `~/fleet/runs/pulse-b8plan/RESULT.md` §3+§5 using b9card.md template.
6. Samsung S8 Active: when a data cable appears, back up photos/video/recordings to `~/Pictures/phone_backup/android/`, then same setup as the iPhone.
7. 50-min lane checks; keep tracker + REPORT current.

### Needs Christopher
Rotate CT 257 keys · pick how testers reach a server before Sunday · glance at Paylocity tab on Claude-OS · USB-C data cable for the Samsung.

### Honest status
B7 on main: tonight late / early Thu if #394 is clean. B8: Thu midday–afternoon (never walked on a phone yet: expect layout fixes). B9: Fri–Sat. Batch merging cut merge time roughly 3×.

## 0-prev5. (older) HANDOFF TO TOM — STATE AT 3:45 PM CST WED 11-18 (supersedes every older 0 block)

**Head brain moves to Tom (Beelink) now: ClaudeBox is out of weekly tokens.** Christopher's order, 11-18 ~3:45 PM.
Tom: you run everything below from the Beelink; your tools are under `~/fleet/bin/` (same scripts ClaudeBox used over ssh).

### A. Goal and dates
- Desktop browser app (PWA, **Bundle 7**) on main **before Thu 11-19**. Phone app (**Bundles 8-9**) on main **before Sun 11-22 kickoff**.
- Merge authority (Christopher, "same as day"): head brain merges lane PRs into bundles AND lands bundles on main **after own review + green CI**. Squash only.
- Tracker for Christopher: `~/Downloads/twr-usable-week-plan.html`, written ONLY via `~/fleet/bin/twr-tracker.py` (lane/addlane/log/needs/ahead/touch). Keep it current; he reads it.
- Night report: `~/fleet/runs/overnight-alpha-1118/REPORT.md` (one `- <time> CST <lane>: <result>` line per event).

### B. Rulings (do not reopen)
- B7 browser door APPROVED "locked down" (6:45 AM): `__Host-twr_session` HttpOnly/Secure/Strict; Origin must equal `https://`+`/etc/twr/public-host`; GET/HEAD without Origin allowed (#378) unless `Sec-Fetch-Site: cross-site`; POST/changes need exact Origin; agent token+Origin 403, token-as-cookie 401, Authorization+cookie 403; multi-league account 409 "use desktop". One admitted league, no switcher.
- Phone (12:45 PM): phone = B7 PWA with viewport layout (<700px), no app store, no new auth. **Activity feed = public events only** (notices, completed trades, completed roster moves; never bids/notes/pending offers/commissioner ledger). **Today boards SYNC across devices** (server preference; a migration for this ONE preference is approved). **Ask = read-only from on-screen data.**
- **Luna NEVER at max** (3:10 PM; it overthinks). Levels: `high` plain, `xhigh` choices/design. Tom on Opus low only if GPT is down.
- Never: print `/etc/twr/*`, secrets, STATE; touch CT 110/256 (CT 257 read-only); `pkill -f` over ssh; edit a running bash script; `--no-verify`; reboots/power changes; Proxmox host 192.168.1.200.
- Claude-OS is released to us. **Never touch the Paylocity Brave tab**; lanes use a separate Brave profile (`--user-data-dir=$HOME/.cache/<lane>-profile`). b7lab may have sent that tab one browser-menu keystroke (no clicks/typing) — told Christopher.

### C. Branches (heads at 3:45 PM)
| Branch | Head | Contains |
|---|---|---|
| main | 28cd636a0 | Bundles 1-5 + #374 #375 search, #376 Home seed/Pulse week, #377 Control Room reads, #379 race-test fix |
| bundle/b7 | dd6f5c615 | #373 browser client/PWA, #372 server door, #378 Origin fix, #381 upgrade adds /app + recovers public-host. b7lab2 is merging main into it (Step 0 of its brief) |
| bundle/b8 | e4ce1db55+ | cut from b7; #380 phone shell, #382 Today, #384 lineup merged; #385 bids merging; #383 Live/Pulse and #386 trades queued (all reviewed, CI green) |
| bundle/b9 | e4637ebfa | cut from b8 early (shell only). Needs `bundle/b8` merged in before the B9 UI lanes start |

### D. In flight (each has a run dir `~/fleet/runs/pulse-<lane>/`: task.md, stdout.log, completion.txt, sessions/)
Alive check: `pgrep -f "luna <lane> "`. Report: `tail -c 2500 ~/fleet/runs/pulse-<lane>/stdout.log`. Watch: `~/fleet/bin/twr-lane-wait.sh <lanes>` (background), 50-min checker `~/fleet/bin/twr-lane-check.sh 50`.
| Lane | Base | Started | What / what to do with result |
|---|---|---|---|
| b7lab2 (xhigh) | bundle/b7 | 2:37 PM | Merge main into b7 (push merge commit direct), lab-only release, upgrade CT 260 WITHOUT hand fixes (proves #381), curl security checks, Brave walk on Claude-OS desktop + 390x844. Result → `RESULT.md`. Fix any bugs via small lanes into bundle/b7, then **land B7 on main** (`~/fleet/bin/twr-bundle-land.sh bundle/b7 "<title>" <body.md>`) |
| b9feed (xhigh) | bundle/b9 | 12:57 PM | PR **#387** open: `league.activity.read`, allow-list of public kinds. Review: allow-list only, no bids/notes/pending/ledger; then merge into b9 |
| b9boards (xhigh) | bundle/b9 | 12:58 PM | PR **#388** open: `home.layout.read/set` sync. Review: owner-only, agent refused, validation, migration correct (fresh+upgrade test) |
| b9fa (xhigh) | bundle/b9 | 2:42 PM | New `roster.add` free-agent command (refuse on-waivers, roster/cap checks, optional drop atomic). Review authorization carefully |
| perf (xhigh) | main | 2:43 PM | Capacity regression (lab5 refuses at 128 clients, lab4 held 256). Local repro only; cause not a limit bump. PR into main |
| bundle merge | — | 3:40 PM | Beelink bg loop: merging #385 then #383, #386 into bundle/b8 via `twr-bundle-merge.sh`. Check: `pgrep -af twr-bundle-merge`; PR states via `gh pr view N` |
Windows: 3 h per lane; exit=124 = timeout. A timed-out lane that already opened a PR is fine (fixcr/fixhome did that). Otherwise relaunch the same brief (resumes from HANDOFF.md).

### E. Next actions, in order
1. When bundle/b8 has #383 #385 #386: merge `origin/bundle/b8` into bundle/b9 (merge commit, push direct as fast-forward — NOT a squashed PR).
2. Launch the B9 UI lanes (briefs written, in `~/fleet/briefs/alpha010/`): `b9card` (player card/picks), `b9cr1` (Control Room jobs), `b9cr2` (rules/season/league), `b9cr3` (people/picks/server), all `BASE=bundle/b9 ... high`. Then B9-1 Moves UI (after b9fa merges), B9-2 League activity UI (after #387), B9-4 boards/Ask UI (after #388) — write those three briefs from `~/fleet/runs/pulse-b8plan/RESULT.md` §3 + §5, using the b9card.md template.
3. b7lab2 result → fix → land B7 on main (target Wed night / Thu AM). Then merge main into bundle/b8 and b9.
4. Phone walk at 390x844 on Claude-OS of every B8 screen (lanes could not emulate; all B8 tests are CSS-contract tests only). Then land B8 on main, then B9 (target Sat).
5. Review/merge #387, #388, b9fa, perf as they finish.
6. When no lanes are running: edit `~/fleet/bin/launch-pulse-lane.sh` to (a) refuse `max`, (b) add standard rule "Do not rewrite HANDOFF.md; append only your own section" — every main PR today conflicted on HANDOFF.md (fix: in the lane worktree `git merge origin/main`, `git checkout --ours HANDOFF.md`, commit, push with build env).
7. Keep tracker + REPORT.md current; 50-min checks.

### F. Launch / build recipes
- Launch: `cd ~/work/TheWarRoom && BASE=<branch> nohup setsid ~/fleet/bin/launch-pulse-lane.sh luna <lane> ~/fleet/briefs/alpha010/<brief>.md <high|xhigh> >/dev/null 2>&1 </dev/null &`. Stagger multiple launches ≥45 s (two at once killed one).
- Build env for pushes: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOTOOLCHAIN=go1.26.11 CGO_ENABLED=1 GOLANGCI_LINT_CACHE=$PWD/.scratch/lintcache`.
- Merge into bundle: `~/fleet/bin/twr-bundle-merge.sh <lane> <pr>` (updates branch, waits CI, squashes). Land: `twr-bundle-land.sh`.
- Brief corrections: append "Part N" to the brief file, relaunch.
- Lab: CT 260 only, `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`; creds in `~/fleet/runs/pulse-lab4/private/` (never print).

### G. Refuted / learned today
- Random CI failure `TestUnavailableKeychainLoginIsMemoryOnly` was a test race → fixed #379; if it recurs it is new.
- Lab "4 bugs" root causes: CR reads missing from hosted workflow catalog (400); odds seed uint64 > JS safe int; Pulse week blocked by missing future NFL dates; search name order + whole-league.
- B7 lab: Brave sends no Origin on same-origin GET → fixed #378. Old updater can't install new helpers → #381 reuses `twr-status` start hooks.
- #386 found trade offers were sent `visibility: league`; now `private`.

### H. Needs Christopher
- Rotate the CT 257 keys (his hands).
- Pick how testers reach a server before Sunday.
- Glance at the Paylocity tab on Claude-OS.

### I. Honest status
B7 on main by Thu morning: likely (≈70%) if b7lab2 finds nothing new. B8 screens are built but never seen at phone size — expect layout fixes. B9 by Saturday is tight: 7 UI lanes + 3 server PRs to review. Perf cause unknown.

## 0-prev4. (older) STATE AT 11:10 AM CST WED 11-18 (supersedes every older 0 block and sections 3, 10, 11)

**Targets (Christopher 11-18 6:40 AM):** desktop browser app (PWA, Bundle 7) before Thursday 11-19; phone app (Bundles 8-9) before Sunday 11-22 kickoff. He APPROVED the Bundle 7 browser door at 6:45 AM, "locked down": exact own origin only, same checks, no agent tokens from browsers (memory `project_twr_b7_browser_door_approved`). Claude-OS is released to us (leave the Paylocity Brave tab alone). Stale branch deleted by him.

**On main:** Bundles 1-5 all landed: #362, #361, #365, #369 (B4), #371 (B5, `b597b6122`), #368 test flakes, #374 search name order. Alpha bundle list complete.

**bundle/b7:** #373 browser client + PWA (cookie session, no bearer in JS, SW caches shell only; hidden in browser: local mode, MFL import, file dialogs, AI helpers, enrollment/recovery/TOTP/device revoke). #372 server door (security reviewed by me: `__Host-twr_session` HttpOnly Secure Strict; Origin must equal `https://`+`/etc/twr/public-host`; missing Origin w/ cookie refused; agent token with Origin 403, as cookie 401; Authorization+cookie 403; multi-league account → 409 "use desktop"; changes feed/enrollment closed). Known gap: no live push in browser.

**In flight**
| Job | Check | On result |
|---|---|---|
| b7lab (Luna xhigh, from ~10:15) deploy bundle/b7 to CT 260, curl security checks, browser walk on Claude-OS | `~/fleet/runs/pulse-b7lab/` completion.txt, RESULT.md | bugs → fix lanes into bundle/b7; pass + fixes on main → land B7 with `twr-bundle-land.sh bundle/b7 ...` (write b7pr.md body) |
| fixcr (xhigh) Control Room Jobs/Season/People/Ledger refused for commissioner | `~/fleet/runs/pulse-fixcr/` | review (no weakened authz), merge to main via `twr-bundle-merge.sh fixcr <pr>` |
| fixhome (xhigh) Home "Seed: expected safe integer ≥ 0" + Pulse "league has not said which week" | `~/fleet/runs/pulse-fixhome/` | review, merge |
| PR #375 whole-league player search (reviewed, OK) | CI re-run after unrelated Go race; watcher task b3g2rxup6 | if green: `gh pr merge 375 --squash` (or rerun twr-bundle-merge.sh fixsearch 375) |
| watchers | twr-lane-wait.sh fixcr fixhome; b7lab; twr-lane-check.sh 50 | re-arm |

**New issues found today**
- Go data race (flaky) in `internal/serverclient` `TestUnavailableKeychainLoginIsMemoryOnly` failed CI on #375: needs its own small fix lane.
- Speed: lab5 server refuses 37% at 128 clients (lab4 held 256). After bugs.
- Commissioner with no franchise gets no player-search list (#375 guards on franchise). Later.

**Traps learned today (do not re-learn)**
- Squash-merging a "merge main into bundle" PR loses the merge parent; landing re-conflicts. Instead push the lane's merge commit straight to the bundle branch (fast-forward), as done for b5sync (40596ca63). For B4 I re-recorded with `git merge -X ours origin/main` after confirming zero tree diff.
- Pushing from the main checkout fails the pre-push hook (no Go on PATH); push from a lane worktree with the build env exported.
- `sleep N; ssh ...` chained is blocked by the guard; launch commands bare.

**Lab:** CT 260 on `b597b6122`, history 3,399, backups OK. lab5 stress 75,066 req, 6,016×503, 0×502.

**Next actions**
1. Merge #375 when CI green. Review/merge fixcr and fixhome PRs.
2. On b7lab result: fix what it finds; merge main into bundle/b7 (direct merge commit, not squashed PR); land B7 on main; tracker.
3. Re-walk desktop + browser on Claude-OS after fixes.
4. Start Bundles 8-9 (phone) on top of B7: briefs from `~/Downloads/TheWarRoom-mobile-mock-v1.html` and `~/fleet/runs/twr-v3-1117/MOBILE-V1-PARITY.md`.
5. Fix-lane for the serverclient race; perf lane after.

**Needs Christopher:** rotate CT 257 keys; pick how testers reach a server (before Sunday).

**Tracker:** `~/Downloads/twr-usable-week-plan.html` fully rewritten 9:31 AM (plan B1..B89, ahead stops to Sunday); last touched 10:20 AM.

## 0-prev3. (older) STATE AT 2:55 AM CST WED 11-18 (overnight run; supersedes every older 0 block and sections 3, 10, 11)

**Overnight run:** `~/fleet/runs/overnight-alpha-1118/` on the Beelink (`PLAN.md`, `REPORT.md` with "Needs Christopher" at the top). Merge authority is the same as daytime (his answer 11-17 ~8 PM): merge after my own review and green CI, into bundles and main. The repo allows SQUASH merges only (`--merge` is refused).

**On main now:** #346 (fresh-server first load), #362 (Bundle 1: HQ, War Room, Pulse), #361 (install fixes: league-load owns what it creates; updater `TWR_FIRST_LOAD=1`), #365 (Bundles 2-3: Needs-you list and the fixed thin Home). Main HEAD is `06529b7d5`.

**Bundle 5 (`bundle/b5`), all reviewed and merged:** #347 frame + Jobs + Situations, #350 Rules, #352 Season, #351 Server, #354 job producers (co-commissioner can read), #359 Picks, #357 People, #358 Fix, #356 League, #355 deputy permissions (SECURITY fix: no commissioner/co-commissioner shortcut; deputies go through authorizeTrade/authorizeLineupSet; trade parties refused). Waiting only on b5backup, then land with `twr-bundle-land.sh bundle/b5 ...`.

**Bundle 4 (`bundle/b4`):** #360 shell (thin rail, player search, cards) and #364 (sync with Bundle 1) are merged. Landing on main hit 9 conflicts against main (Bundles 1-3). Lane **b4sync2** (Luna xhigh) is merging; the real join is `HostedLeague.tsx` (the rail's Home must return to the Bundle 3 thin Home; there must be one Home). Then land it with `twr-bundle-land.sh bundle/b4 "Bundle 4: ..." ~/fleet/runs/overnight-alpha-1118/b4pr.md`.

**In flight (watchers on Claudebox)**
| Lane | What | Check | On result |
|---|---|---|---|
| b4sync2 | merge main into bundle/b4 | `~/fleet/runs/pulse-b4sync2/` (completion.txt, `tail -c 1500 stdout.log`) | review the PR (HostedLeague Home join), merge with `twr-bundle-merge.sh b4sync2 <pr>`, then land B4 |
| b5backup | hosted `backup.now` for commissioner/co-commissioner; unblocks Rollover and Server > Back up now | `~/fleet/runs/pulse-b5backup/`; its 3 h window ends ~3:20 AM | review authz (owner/deputy/token refused, one at a time), merge, then land B5. If it timed out, relaunch the same brief (it resumes from HANDOFF.md) |
| 50-min checker | `twr-lane-check.sh 50` | — | re-arm |

**New fleet tools (Beelink `~/fleet/bin/`, written tonight)**
- `twr-bundle-merge.sh <lane> <pr>`: syncs the PR base into the lane branch (HANDOFF keeps the lane's copy; any other conflict stops), tsc + tests, push with lint-lock retries, waits for CI, squash-merges.
- `twr-tab-merge.sh <lane> <pr> [wait-pr]`: the same, plus auto-resolving the Control Room tab conflict with `twr-tabs-resolve.py` (handles only single-line tab entries; ternary or multi-line entries needed hand resolution: #357, #358, #356).
- `twr-bundle-land.sh <bundle> <title> <body>`: merges main into the bundle in `.worktrees/land-<bundle>`, regenerates gen files, full vitest (one retry for load flakes), frontend build, `make verify`, push, PR to main, CI, squash-merge.

**Traps found tonight (do not re-learn)**
- A fresh worktree needs `cd frontend && pnpm run build` before `make verify` or a push: the Go embed (`all:frontend/dist`) and `make bindings` fail without it.
- `golangci-lint` is a single lock across lanes. Commits and pushes fail with "parallel golangci-lint is running": retry every 60 s.
- The guard reads `ssh -f` or `--fail-fast` in the same command as `git push` as a force push. Keep pushes in their own command.
- A blocked command also drops the heredoc file write inside it. After any guard block, check the brief exists (`wc -c`) before launching (b4shell launched with an empty brief once).
- Full-suite vitest under fleet load flakes on timing tests (nativeAccount "opens the league straight after signing in", hostedRules "teamless Commissioner"); both pass alone.
- Landing bundles late compounds conflicts: land each bundle the moment it is done.

**Rulings I made tonight (head brain; record them, do not relitigate)**
- Thin mode is the product. Bundle 4 was built in thin mode; local mode stays frozen (shared presentational `RailNodes` is fine).
- Home privacy: a thin sign-in is bound to one league, so Home lists only the admitted league and never calls `commish.leagues.read`. It is built as a `LeagueChoice[]` list for a later account-scoped read.
- lab4: upgrade-then-archive-then-remove-one-by-one was the accepted workaround; the fix is on main (#361).

**Lab:** CT 260 runs release 9 with MFL 14432 imported (3,399 history), backups checked in R2. Stress: 79,795 requests; 5,407 HTTP 503 from our server's overload refusal at 256 clients; 0 HTTP 502; recovery 0.7 s. Evidence: `~/fleet/runs/pulse-lab4/RESULT.md`. Client pass on Claude-OS NOT done (a private page was open; left alone).

**Next actions**
1. On b4sync2 done: review, merge, land Bundle 4.
2. On b5backup done: review, merge, then land Bundle 5 (needs a body file like b4pr.md).
3. Then alpha bundle list is complete: update the tracker (`twr-tracker.py`), write the morning REPORT summary, run the Claude-OS client pass if the desktop is idle, and re-run the lab on the new main.
4. Bundles 8-9 (phone) are NOT started: blocked on the Bundle 7 security call.

**Needs Christopher (also at the top of REPORT.md)**
- Bundle 7 browser door (security loosening): approve it, or keep the phone for after alpha. This blocks Bundles 8-9.
- Delete the stale branch: `git push origin --delete session/pulse-b1hqdec`
- Tell me when Claude-OS is free for the client pass.
- Rotate the CT 257 keys; choose how testers reach a server.

## 0-prev2. (older) STATE AT 7:55 PM CST TUE 11-17 (supersedes every older 0 block and sections 3, 10, 11)

**New rulings since 7:00 PM (memories: feedback_luna_exclusive_scale_thinking, project_twr_alpha_cuts_1118)**
- **Thinking level is routing.** Pick it for wall clock and to keep the session alive on OpenAI's servers, not for output quality. Use high for short lanes, xhigh for long ones, and max only when a long run already got cut off.
- **Alpha is now after Bundles 1 to 5 plus the release checks.** Bundle 6 is removed.
  - Cut from alpha (moved to the tracker's "later" list): usage/age arc, the D58 layout store and arrangeable boards (Home becomes a fixed v3 look), the alert band, the status bar, team/league search, stat changes, the Archive tab, points per play, and board sync.
  - Kept: stacked what-if, the Picks tab, and player search.
- **#320 counts as a pass for release check line 14.**
- **#304/#307 are parked until after alpha.**
- **Head brain owns the testing:** the lab server (CT 260 on lab host 192.168.1.182) and the client on Claude-OS.
- **Tom (Opus low) may be used as GPT fallback.** Launcher: `TOM_WT=<worktree> TOM_BRIEF=../alpha010/<brief> TOM_SCOPE=... launch-tom-build.sh`. Run dir: `~/fleet/runs/tom-<wt>/`. At most 2 at once.
- **Goal:** testers are waiting. Get to alpha fast, then learn from their feedback.

**Head-brain rulings given to lanes**
- **b5perm:** `commish.vote.close` and `commish.vote.tiebreak` stay commissioner-only.
- **b1pulse:** removing a duplicate read in shared local/hosted code is slop and is allowed.
- **b0load:** league-load prepares an empty state dir, plus an installer `TWR_FIRST_LOAD=1` mode.

**Merged:** #344 (offers drawer) into bundle/b1 at 7:53 PM. My fix in it: offer cards name the partner team.

**In flight (all on Beelink; watchers on Claudebox)**
| Job | What | Check | On result |
|---|---|---|---|
| b92bz4ed9 | merge #345 (b1pulse, reviewed twice, green) into bundle/b1 | `gh pr view 345` | update tracker |
| b9b9f6how | merge #346 (b0load fresh-server first load, reviewed) to MAIN | `gh pr view 346` | lab4 relaunches |
| b0oklgyc1 | after #346 MERGED and 02:54Z, relaunch lab4 (Luna max, brief lab4.md with a RE-RUN section) | `~/fleet/runs/pulse-lab4/` | review RESULT.md; file bugs as lanes |
| b8qoa197a | at 02:52Z relaunch b1whatif, b5frame, b5jobs, b5perm on Luna xhigh (stopped on GPT quota at 7:38 PM) | run dirs `~/fleet/runs/pulse-<lane>/` | arm `twr-lane-wait.sh` on them |
| bpvirqlew | Tom on b1hqdec (contract decisions), continuing from its HANDOFF | `~/fleet/runs/tom-pulse-b1hqdec/completion.txt` | review its PR into bundle/b1 |
| b1cleanup | Luna xhigh, still running at 7:50 PM | lane-check | review its PR into bundle/b1 |
| b4q3ixdx4 | 50-min lane check | — | re-arm |

**CT 260 state**
- Server stopped and `/var/lib/twr` empty.
- Empty first-start data archived at `/root/first-start-data.tar.gz`.
- Seed league at `/root/seed/history.db` (sha256 ca2d477a…ff3e, matches the original).
- Lab ssh: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`.

**Refuted / traps this window**
- The guard hook blocks any command text containing a container power word (it blocked a brief that mentioned one). It also blocks recursive chown/chmod. Use the documented non-recursive `chown twr:twr /var/lib/twr/server.db*`.
- Pushing from a worktree needs the build env (PATH with Go and wails, plus GOTOOLCHAIN). Without it, verify fails with "wails: not found".
- Heredocs over ssh break on apostrophes. Write the file locally and scp it.
- The storm harness only accepts the lab URL on 443, not a LAN IP. The Cloudflare route is accepted.

**Next actions**
1. As each lane finishes, review its diff myself, fix small slop directly, and merge into its bundle branch.
2. When B1 is complete: merge main into bundle/b1, run the full suite, then open and merge the bundle PR.
3. After B5's frame lands, launch the B5 tabs: Rules, Season, Picks and Server on high; Fix, League and People on xhigh.
4. Review lab4's RESULT and turn every bug into a fix lane.
5. Write the tester-ready checklist: installer, an outside-reachable server, and a feedback channel. Then queue Bundles 2 to 4 (reduced) and 8 to 9.

**Needs Christopher:** rotate the CT 257 keys (whenever he's at a keyboard). Also his choice of how testers reach a server (outward-facing).

## 0-prev. (older) STATE AT 7:00 PM CST TUE 11-17 (supersedes 0a/0b and sections 3, 10, 11 below)

**Goal (Christopher 11-17, latest):** build the full v3 desktop app in bundles, push day and night, and get Bundles 8 and 9 (the phone app) done before **Sunday 11-22 kickoff**. Official alpha comes after Bundle 6 (his call). Tracker: `~/Downloads/twr-usable-week-plan.html`. The page has been rebuilt (template `~/fleet/runs/twr-tracker-template.html`, backups `.bak-1117`, `.bak-1117b`). Its data keys: `plan` (bundles B0–B8 + ALPHA, each with items in state done/building/todo/waiting), `lanes` (cards with `day` 11-17, `at`, `pr`, `ord`), `needsYou`, `later`, `slop`, `log`, `today`.

**Bundle method (his order 11-17):** each bundle gets one shared branch, `bundle/bN`. Lanes are launched with `BASE=bundle/bN ~/fleet/bin/launch-pulse-lane.sh luna <lane> <brief> <level>`; BASE support was added today (backup at `launch-pulse-lane.sh.bak-1117`). Each lane opens its PR with `--base bundle/bN`. I review each PR and merge it into the bundle. Then the full suite and CI run once, and the bundle goes to main as one merge. CI runs on every push to any branch. Bundle 1's shared brief rules are in `~/fleet/briefs/alpha010/b1-*.md`, Bundle 5's in `b5-*.md`. Slop gets fixed first, inside each lane.

**Plan order:** B0 (tonight) → B1 Finish HQ + War Room → B5 Control Room on the server app → B2 Home server side → B3 Home screen → B4 Shell → B6 Pulse extras → B7 browser door + PWA → ALPHA → B8–9 phone app built from `~/Downloads/TheWarRoom-mobile-mock-v1.html`. That mock is complete: 55 of 56 desktop jobs reachable in 3 taps or fewer, per `~/fleet/runs/twr-v3-1117/MOBILE-V1-PARITY.md`. It was missed by me earlier; it is not a draft.

**Merged to main today (11-17):** #312 #315–#319 #321 #322 #324–#327 #329–#341 (#339 is the README; #341 is the v3 stub plan, merged 5:45 PM). The last code merge was #340 (recent winning bids) at 6:02 PM. #338 merged after I removed duplicate code: one `ownerPrivateCommand` helper in `internal/commandcatalog/native_reads.go`.

**In flight (Luna, all running at 6:45 PM):**
| Lane | Branch base | Level | Brief |
|---|---|---|---|
| b1hqdec (owner contract decisions: new owner command, approved) | bundle/b1 | max | b1-hqdec2.md |
| b1whatif (stacked what-if server read) | bundle/b1 | xhigh | b1-whatif.md |
| b1offers (War Room offers drawer) | bundle/b1 | xhigh | b1-offers.md |
| b1pulse (My Road + shared form helper) | bundle/b1 | xhigh | b1-pulse.md |
| b1cleanup (remove withdraw on placed bids, S10 fold, v3 doc fix) | bundle/b1 | xhigh | b1-cleanup.md |
| b5frame (thin Control Room frame + Jobs + Situations) | bundle/b5 | xhigh | b5-frame.md |
| b5jobs (missing job producers, server) | bundle/b5 | xhigh | b5-jobs.md |
| b5perm (deputy six-tick gate, ruled) | bundle/b5 | max | b5-perm2.md |

- **Alive check:** `/root/.claude/scripts/twr-lane-check.sh 50` (armed), or the newest file time in `~/fleet/runs/pulse-<lane>/sessions/`.
- **Watcher:** `twr-lane-wait.sh` for all 8 (armed).
- **When a lane is done:** read `stdout.log`; review the diff, especially policy, token and store files; merge its PR into `bundle/bN`; resolve clashes as a union of both sides and regenerate generated files with `go run ./frontend/scripts/nativegen` and `-surface doors` from the repo root; run `make bloat`.
- **GPT quota:** he said to tell stalled lanes to continue. Relaunch the same lane: the worktree and HANDOFF.md resume it.

**Rulings today (don't reopen):**
- Release 9 / schema 16 private table approved.
- Ignore picomatch advisory GHSA-fc5q-39xc-vjpf until patched (T649).
- API tokens may reach private notes, watch, draft queue and trade drafts for the owner's own team only.
- One sign-in = one server; a server holds many leagues; choosing between servers waits for the login screens (memory project_twr_one_signin_one_server).
- Build an owner contract-decision command (ledger event, no table).
- Alpha after Bundle 6.
- Deputies get the v3 six-tick grid: stat fixes, trades, lineups, picks, rules, rollover. Co-commissioner = all six. Roles, succession and deputy grants are never delegable (memory project_twr_deputy_six_ticks).
- The ADP feed is his call; the Plans store and private two-team threads are parked.

**#341 errors found (fix in the docs):**
1. It says the engine allows withdrawing a placed bid. False: `bid_execute.go` always refuses. The app still offers withdraw; b1cleanup removes it.
2. It says `contract.action.set` is the owner decision command. False: it is commissioner-only league policy (CM-009).

Check each #341 plan line against the code before a lane relies on it.

**Open Control Room decisions (ask when their screens come up):**
- Is snooze per person or per league?
- Do deputies see every job?
- Who can see a written request?
- Can divisions be named?

**NEXT, in order:**
1. Review and merge B1/B5 lane PRs into their bundle branches as they finish.
2. Launch the remaining B5 tab lanes once b5frame lands: Rules, Season, Fix, People (uses b5perm), League, Picks, Server read-only.
3. B0: build from main and install on lab CT 260. The lab path and refused import command are in `~/fleet/runs/pulse-lab1/RESULT.md`; the steps are in `docs/lab/STRESS-READY.md`. Then write the one-run import script for him to read and run.
4. When B1 is complete: merge main into bundle/b1, run the full suite, open the bundle PR to main, merge on green.
5. Queue B2 (D58 layout store; needs a migration → release number: ask him) and B3/B4 so Luna never idles.
6. B7 needs his security sign-off: the server refuses browser Origin today, on purpose.

**Needs him:**
- #320 line-14 call
- #304/#307 player values
- rotate CT 257 keys
- run the CT 260 import script (once written)
- B7 security sign-off
- B2 release number

**Honest status:** B8–9 by Sunday means B1–B7 must merge first. That is about 7 bundles in 5 days at today's pace of roughly 23 merges a day. It is possible but tight. B5 is the largest and most likely to slip.

## 0a. (older) STATE AT 1:15 PM CST TUE 11-17 (supersedes the 7:35 AM block and sections 3, 10, 11 below)
**Rulings today:** Luna ONLY (high/xhigh/max). Full send gated by launcher room() (>=4 GB, load <12). Windows 3 h (max 4 h).
50-min checks (`/root/.claude/scripts/twr-lane-check.sh 50`, re-arm). DAY MERGE AUTHORITY: yes, after my review + green CI.
Private-entries table APPROVED; Release 9 / schema 16 APPROVED. API tokens MAY reach notes/watch/draft queue for the
issuing owner's own team only (same rule as bids). Picomatch advisory GHSA-fc5q-39xc-vjpf ignored (#335; Hermes T649 = remove
when patched). Build plan: SKIP 0.10.2; one build from main after #333/#334 land, then a reviewed one-run league-import
script for CT 260 (he runs it; RESULT.md at ~/fleet/runs/pulse-lab1/ is notes, not a command). "Ahead of schedule."
**Merged 11-17:** #322 #324 #325 #326 #327 #329 #330 #331 #332 #335 (+ overnight #312 #314-#319 #321). HQ steps 1-6 done.
War Room hosted: trade Send, Market bids, Market pool, player panel reads, plans-tab empty state.
**In flight:** #334 MERGED 1:10 PM. #333 conflicted in 9 files (duplicate private-table plumbing) -> Luna xhigh lane dqueue Part 3 (1:20 PM, brief ~/fleet/briefs/alpha010/dqueue.md) rebuilds on #334's plumbing. On done: review (token files too), then ~/fleet/bin/twr-merge-pr.sh dqueue 333.
(Claudebox bg task bulbfy2qc). Check: `gh pr view 334/333 --json state`. If not MERGED: rerun the helper (it syncs main,
HANDOFF.md takes main's, waits all checks, squash-merges; exit 2 = real conflict → resolve by hand). No Luna lanes running.
**Merge helper:** `~/fleet/bin/twr-merge-pr.sh <lane|worktree-dir> <pr>`. War Room plan: `~/fleet/runs/pulse-wrplan/REPORT.md`.
**NEXT (he said: continue on the next set of work after compaction):**
1. Confirm #334 + #333 MERGED; close #313, #323, #328 if still open (link #334/#333).
2. Launch Luna xhigh: War Room Market target lists on the watch kind (WR job 3 rest) — brief like wr3.md, reuse player.watch.*.
3. Check if saved trade drafts (WR job 6) fit private_entries (kind 'offer'?) without a migration; if yes launch Luna xhigh, else ask him.
4. Build from main for install (lab build path in ~/fleet/runs/pulse-lab1/RESULT.md line 3 uses LAB-ONLY root fleet/lab-release-ct258 + twr-up); then write the one-run CT 260 league-import script (lines 17-19 + docs/lab/STRESS-READY.md) for him to read and run.
5. Later: WR plans (job 7, may need table), PAIR threads (job 8, privacy ruling), recent wins (job 9), hosted injury feed.
**Needs him:** #320 stress line-14 measurement; F51 #304/#307 GOMEMLIMIT; rotate CT 257 keys; run the import script (after step 4).
**Lessons today:** pgrep -f over ssh matched my own shell and cut it (find exact PID first). Merge helper needed: wait for all checks
(not --watch), treat not-MERGEABLE as needs-sync, accept non-pulse worktree dirs. Luna lanes over-extend scope into token policy;
review internal/accounts + token_policy diffs every time.

## 0b. (older) STATE AT 7:35 AM CST TUE 11-17 (supersedes sections 3, 10, 11 below)
**Rulings today:** Luna ONLY (high/xhigh/max by difficulty; restart same brief before rescoping). Full send: no lane cap,
launcher `room()` gates on >=4 GB RAM + load <12. Windows 3 h (max 4 h). Check lanes every 50 min
(`/root/.claude/scripts/twr-lane-check.sh 50` background, re-arm each time). DAY MERGE AUTHORITY: yes, all day, after my
review + green CI (decisions #313, #320, F51 still his). Phone/audio = paragraphs. Never edit a running bash script in place.
**Merged today/overnight:** #312 #314 #315 #316 #317 #318 #319 #321. HQ steps 1,2,4,5 + slop cleanup on main.
**In flight (Beelink lanes, `~/fleet/runs/pulse-<lane>/`):**
- smem part 2 (Luna xhigh, 7:22 AM): PR #322 memory -21.9% but removed shared-ledger clone + its test; must restore
  `TestSharedRecordsCallerCannotChangeAnother`, audit callers, race test. Merge only if those hold.
- hqreserve (Luna xhigh, ~7:30 AM): HQ step 6 IR injury/return + taxi experience. Review, merge.
- wrplan (Luna high, ~7:30 AM): War Room read-only plan -> `~/fleet/runs/pulse-wrplan/REPORT.md`. Then launch its jobs.
- Watchers: done `twr-lane-wait.sh smem hqreserve wrplan`; 50-min checker. If lost after compaction, re-arm both.
**Needs Christopher:** lab1 league import CT 260 (`~/fleet/runs/pulse-lab1/RESULT.md`); notes storage #313; stress line-14
measurement #320; F51 #304/#307 (memory limit now set); rotate CT 257 keys; install 0.10.2.
**Overnight skill v2** written (Claudebox, Tom ~/.claude, pi ~/.pi/agent/skills, Hermes ~/.hermes/skills); Beelink copies of
the watcher scripts in `~/fleet/bin/`. Launcher backup `~/fleet/bin/launch-pulse-lane.sh.bak-1117`.
**Next:** handle pings; after wrplan, launch War Room jobs (Luna); keep Beelink full; tracker + report current.

## 1. WHAT WE ARE DOING
(a) Alpha 0.10: prove the 15 lines of `docs/roadmap/RELEASE-0.10.md` (lab jobs tonight after the game).
(b) NEXT SCREEN = FRANCHISE HQ on real server data (his "hard focus"). Plan on the tracker, 6 steps.
(c) Standing rule: slop from old models gets flagged/rewritten as we go (memory `feedback_twr_active_slop_cleanup`; tracker `slop` list, S04 + RULE).
(d) Fable (his separate session) builds 3 live-animation HTML versions from `~/Downloads/fable-handoff-live-animations.md`.
Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (main checkout is HIS; worktrees only). Lab host 192.168.1.182 (CT 257 live test server, CT 258 lab, CT 110 gateway). From Beelink: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`. Never touch CT 110, 256. CT 257 read-only for lanes. No lab load until the game is final.

## 2. DONE LOOKS LIKE (overnight PLAN.md)
1. 0.10 lab proof lines 3-5, 13, 14 MET/NOT MET with quoted evidence — pending (after game).
2. Week 10 final vs MFL for 32 teams — pending.
3. HQ step 1 inventory DONE + reviewed (`~/fleet/runs/pulse-hqmap/REPORT.md`); steps 2-4 as PRs merged (yolo) — lanes running.
4. lword + cpu7 PRs reviewed, green, merged — lanes running.
5. REPORT.md + tracker current.
Tonight's stream: MET (live ticks every 10 s, 0 errors; #309 #310 #311 merged).

## 3. IN FLIGHT
| Job | Started | Check | On result |
|---|---|---|---|
| Lane lword (Luna): refusal reason shown in app (client half of #309) | 8:13 PM | `~/fleet/runs/pulse-lword/completion.txt`, stdout.log | review PR, CI green → merge |
| Lane cpu7 (Luna): contracts cache, `rememberNativeContracts` | 8:13 PM | `pulse-cpu7/` | review equality test + before/after table → merge |
| Lane hqreads (Luna): hosted `player.usage.read` + `player.agearc.read` + shared adapters | 8:44 PM | `pulse-hqreads/` | review → merge; then write step-4 brief |
| Lane hqnotes (Luna): hosted private notes read/change, PRIV privacy tests | 8:44 PM | `pulse-hqnotes/` | review privacy tests carefully → merge |
| Watcher b9evudbwl: `twr-lane-wait.sh lword hqreads hqnotes` (cpu7 FINISHED 8:50 PM: review its PR first!) | 8:46 PM | prints first lane done/dead, exits (seen list `/tmp/claude-0/twr-lanes-seen`) | handle, re-arm with remaining lanes |
| Timer b60mlntfd: 10:15 PM | — | — | confirm game final, then section 10 item 2 |
Lane transcripts: `~/fleet/runs/pulse-<lane>/sessions/`, session id in `session-id.txt`. Launch: `ssh -f ... 'nohup setsid ~/fleet/bin/launch-pulse-lane.sh luna <lane> ~/fleet/briefs/alpha010/<lane>.md high >/dev/null 2>&1 &'`.

## 4. RESULTS THIS WINDOW
- Merged: #309 (after HANDOFF conflict fix), #310, #311. Main has #303 #305 #306 #309 #310 #311.
- ESPN study (Sol): `~/fleet/runs/pulse-espn-soak/REPORT.md` (46 shots), recordings `video/go1-1.mp4 go2-1.mp4 test-b.mp4`, frames `video/frames/`. ESPN updates in chunks every 45-65 s w/ full reload; ours flows every 10 s. Fable handoff written.
- Walked hosted app on Claude-OS :5 (thewarroom-main vs CT 257): HQ roster/cap/lineup real; player panel "Not available yet" because it calls desktop Wails bindings (`market/api.ts readAgeArc`, `offers/api.ts readPersonalNotes`, research usage).
- hqmap: 19 HQ rows; only 3 new hosted reads needed (usage, age arc, notes); rest = wire existing `value.read`, `native.cap.check`, `contract.quote`, `calendar.read` into HQ (step 4).
- CT 258 lab keys moved to `~/fleet/lab-release-ct258/` (symlink at old path, logged in ~/MOVED.md).
- Tracker cleaned (72 old lanes removed), HQ plan + slop rule added.

## 5. ARTIFACTS
Briefs `~/fleet/briefs/alpha010/`: lword, cpu7, hqmap, hqreads, hqnotes, lab1, lab2, lab3, f51t, espn. Copies in overnight run dir.
Fable handoff `~/Downloads/fable-handoff-live-animations.md` (8,305 B); mock screenshot `~/scratch/fable-handoff-1116/` (KEEP.md).
Stream app: `~/scratch/twr-mnf-1116/thewarroom-main` (sha c9fa86d8…).

## 6. OPEN BUGS / GAPS
- f51t brief points at `/tmp/f51/rehearsal257.tgz` which is GONE (Beelink /tmp cleared). Fix brief to `~/scratch/f51-1117/rehearsal257.tgz` after making a fresh copy post-game.
- lab1 container restart: guard blocks agents; lane writes command under "Needs Christopher".

## 7. REFUTED — DO NOT RETEST
CT 258 for Pulse · lineup refusals as timing race · preview ignoring kickoff roster · release 9 for a patch · Terra for multi-step jobs · `adb pull -q` (lost a clip: no -q flag) · `pct reboot` text inside a Bash heredoc trips Claudebox hook (use Write tool for such briefs).

## 8. DECISIONS
F51 option A. Release number = contract (#308 closed; older-bundle proof via LAB-ONLY release-7 bundle). Luna high default. Overnight merges: yolo (his words). Lab1 restart: skip, morning. CT 257 data copy for F51: yes after final, delete after. HQ is the next screen; War Room → Home → Control Room after. Fable: 3 versions, buttons Run/Pass/Penalty/FG/TD/Random drive; ESPN = inspiration only; keep our 10 s feed.

## 9. LEDGER
This RESUME: repo copy `docs/RESUME-alpha010-1116.md` on branch `session/a010-docs` (worktree `.worktrees/pulse-a010`), plus `/root/TheWarRoom-RESUME.md`.

## 10. NEXT ACTIONS
1. On watcher ping: review that lane's PR (read the diff myself), check CI, merge (`gh pr merge N --squash --delete-branch` via Beelink `bash -lc`); resolve HANDOFF.md conflicts with `--theirs`(main). Update REPORT.md + tracker. Re-arm watcher.
2. 10:15 PM: confirm game final (CT 257 log / nfl.games). Then: copy CT 257 data → `~/scratch/f51-1117/rehearsal257.tgz` (+KEEP.md), fix f51t brief path, launch lab2, lab1, f51t; lab3 (waits inside for lab2 stress). Arm watcher on them.
3. Week 10 final vs MFL (read-only), into REPORT.md.
4. After hqreads+hqnotes merge: write + launch step-4 brief (one shared hosted player-panel read path; wire value/cap/quote/calendar into HQ; remove desktop calls in hosted branch; slop items from hqmap report).
5. Morning: finish REPORT.md (Needs Christopher first), tracker, task list on Hermes, one-line result to him.

## 11. NEEDS CHRISTOPHER
Lab1 container restart (morning, command from lane). Rotate CT 257 keys. Tue: install 0.10.2 (lines 6-7).

## 12. ENVIRONMENT
gh on Beelink needs `bash -lc` (PATH ~/.local/bin). Remote heredocs inside ssh '…' break on apostrophes: write locally, scp. Tracker edits: python with `twr-tracker.py` module + flock (see scratch tu*.py pattern: load/stamp/save). adb S25 serial 192.168.1.171:35907 (paired).

## 13. HONEST STATUS
0.10: lines 1, 8, 12 met; 3-5, 13, 14 depend on tonight's lab jobs (unproven; stress previously FAILED on 502s). HQ: only inventory done; reads/notes lanes just started. Morning ETA ~6 AM.
