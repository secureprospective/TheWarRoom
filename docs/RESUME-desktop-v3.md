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
