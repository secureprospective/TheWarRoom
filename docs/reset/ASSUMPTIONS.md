# Stage 4: old assumptions and task triage (2026-11-23)

Written by Tom (Beelink Claude), read only. Judged against Section 1 of `docs/reset/BUILD-PLAN.md` and
`docs/reset/RULINGS-1122.md`. Cut rule (Christopher 11-22): clashes with a ruling = recommend cut; merely
unproven = keep with a flag; nothing is cut just for being old. Nothing here changes code or the task list.
Line counts are `git ls-files | wc -l` on `origin/main` @ bbe4f298f (Go + TS/TSX, tests included unless said).

## Needs you: rule on 9 assumptions and 12 tasks

| # | What you rule on | Recommendation |
|---|---|---|
| A1 | Remove the desktop app (local mode) from the main build | Retire (already your ruling 30, task T710) |
| A2 | The thin desktop window that docs still call "the product" | Retire with A1; docs reworded so no session rebuilds it |
| A3 | Solo-owner MFL tools live only inside the desktop | Needs ruling: rebuild on the server, or drop with the desktop |
| A4 | Old desktop Control Room | Retire with A1; the hosted one stays until the commissioner redesign |
| A6 | Offline desktop acceptance (T489) | Retire |
| A7 | 3-OS freeze (T535) + desktop packaging in CI | Retire |
| A8 | Flatpak / CasaOS packaging (T457) | Retire for the app; reword as "self-host server package, later" |
| A9 | 52-slice client/server plan (T488, T565) | Retire as a plan (the built code stays) |
| A11 | Every native league is dynasty with contracts (no redraft) | Rebuild as settings, for rehearsal shapes |
| Tasks | 12 tasks marked "needs ruling" in Part B | see Part B |

Brief correction: the desktop-out ruling is task **T710**, not T711 (T711 is the season-in-a-box test).
T320 is not on the open list (already closed or renumbered), so it is not triaged.

---

## Part A. Old assumptions

Counts: retire 8, rebuild 2, keep (with flag) 5, needs ruling 1 (A3). Total 16.

| # | The old idea | Where it lives | How big | Recommendation |
|---|---|---|---|---|
| A1 | **The desktop app (Wails, local all-in-one mode) is a place owners use.** It carries its own copy of the league on the computer. | root `main.go`, `internal/composition`, most of `internal/app/*` (54 packages the server and CLI never use), `frontend/src/app/shell` and desktop screens, `wails.json` | ~55,700 Go lines removable (31,900 code + 23,800 tests) per Tom's keep-list scan (`~/Downloads/TOM-REPORT-desktop-keep-list.md`); `internal/app` alone 43,982; `frontend/src/app/shell` 20,260 TS | **Retire.** Clashes with ruling 30 (desktop out of the main build, T710). Order matters: first re-point the command-table generator (`frontend/scripts/nativegen` reads `composition.CommandRegistry()`), or the CLI and phone app can no longer be regenerated. Claudebox reviews the keep list first. |
| A2 | **"Thin desktop mode is the product"** (ruling 2026-11-14, restated 11-18). | `docs/architecture/desktop-modes.md`, `CLAUDE.md` "What this project is", `internal/composition` `runNative` | part of A1 | **Retire.** Clashes with ruling 30 and ruling 4 (phone target is mobile mock v1; the browser/PWA app is what people use). The words in CLAUDE.md and desktop-modes.md must change in the same PR as A1, or a later session will "fix" the missing window. |
| A3 | **Solo-owner tools (`mflref`, `transactions`, what-if desk) are "trim, never delete".** | `internal/app/mflref/*`, `internal/mflref`, `internal/transactions`, `store/mflref`, `app/offers`; CLAUDE.md line 25, North Star | ~17,000 lines (`internal/mflref` 4,424, `internal/transactions` 5,875, rest in `internal/app`) | **Needs ruling.** Ruling 2 says code serving the MFL companion role is not residue, so it should stay. But it exists only inside the desktop, so A1 deletes it. Options: (a) rebuild the useful parts as server reads for MFL leagues (keep the companion role alive in the phone app); (b) let it go with the desktop and reword CLAUDE.md/North Star. Recommend (a), small: only what a solo MFL owner actually used. |
| A4 | **The old Control Room** (desktop commissioner screen, ADR 0002 "v3 shell owns the Control Room tab"). | `internal/app/control` (3,919 Go), `frontend/src/app/control` (7,442 TS), `docs/ui/control-room/`, `docs/adr/0002-*` | ~11,400 lines | **Retire the desktop copy with A1; rebuild later.** Ruling 6: every commissioner job is redesigned mock-first after a market study (ruling 7). The hosted Control Room the phone uses stays until that redesign. ADR 0002 retirement is already ruled (round 1, item 7); Claudebox edits ADRs. |
| A5 | **MFL is the record for every league** (core plan R2). | Superseded already; leftovers: one hard-coded Legacy league ID `14432` in `internal/ingestion/schema.go`, used by `app/mflref/contracts` and the old Control Room | `internal/ingestion` 13,342 lines (only the constant is the problem) | **Keep the MFL companion role; retire the hard-coded ID with A1.** CLAUDE.md says no code branches on one league's ID. All live uses sit in desktop packages, so A1 removes them. Native leagues already use the native ledger. |
| A6 | **Offline desktop acceptance gate (T489):** a restore check on an offline Claude-OS desktop must pass before slices 28/29/50. | `docs/build-handoffs/Client_Server_27_Evidence_2026-10.md`, `internal/offline` | 139 lines code + docs | **Retire.** The desktop is out (ruling 30) and the server is the record for native leagues (ruling 2). Server backups are proven by the nightly backup to CT105 and the season-in-a-box test instead. |
| A7 | **3-OS acceptance and freeze (T535, slice 52):** ship the desktop on Windows, macOS and Linux, tested on each. | `.github/workflows/desktop-packages.yml`, `scripts/desktop/` (414 lines), Wails steps in `verify.yml` | ~500 lines + a CI job on every main push | **Retire.** Clashes with ruling 30. The phone/browser app runs everywhere without packages. CI gets faster too. |
| A8 | **Release by Flatpak and the CasaOS app store (T457).** | `docs/roadmap/Roadmap_and_Open_Questions.md`, `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` | docs only, no code | **Retire for the desktop app.** Flag for the server: "This assumes commissioners will want to self-host a league server. If wrong, it costs nothing now (no code). Cheapest check: ask 3 commissioners in the market study (T682)." |
| A9 | **The 52-slice client/server build plan is the governing plan (T488, T565 "start step 1").** | `docs/build-handoffs/Client_Server_Build_Plan_2026-10.md`, `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` (already labelled HISTORY) | docs; the 50 merged slices are live server code and stay | **Retire as a plan.** `docs/reset/BUILD-PLAN.md` governs now. The server it produced stays. |
| A10 | **Signed upgrades and the installer drill (T505, "Drill 51").** | `internal/releasetrust` (1,278), installer bits in `deploy/` | ~1,300 lines | **Keep, with flag.** "This assumes server upgrades must be signed. If wrong, it costs ~1,300 lines of upkeep. Cheapest check: confirm the Lab and Season server upgrade path uses it (one `grep` of the swap script)." The desktop installer part goes with A1. |
| A11 | **Every native league is a dynasty league with contracts** (no redraft, no rollover reset). Found by T694. | `internal/engine/*` rollover, `internal/leaguerules` | not separable (engine-wide) | **Rebuild as settings.** Clashes with ruling 3 (test leagues of many sizes and shapes) and the "every rule is a setting" rule. Only the small pieces T694 lists: rookie/taxi off, this-season ranking, one-season mode. |
| A12 | **Desktop-first UI plans** (desktop mock v3, V3 stub plan, UI/UX fold 1-2, "UI to the mocks" on desktop). | `docs/ui/v3/`, `docs/plans/UIUX-MOCK-OUTLINE-2026-11-15.md`, `frontend/src/app/v3` (5,896 TS) | ~5,900 TS + docs | **Keep the code (the browser app renders some v3 screens), retire the desktop plans.** Flag: "This assumes v3 screens are still reached from the browser app. If wrong, ~5,900 lines are dead. Cheapest check: open each v3 route on the Lab in a phone-size browser." Ruling 4: phone target is mobile mock v1 only. |
| A13 | **Legacy will be the first league to move / plans around a "modified Legacy" default (T495).** | `docs/roadmap/`, task T495 | docs only | **Keep, with flag.** Ruling 8: no plan for which league moves first. Reword any doc that predicts the order. Ruling 1: Legacy is never put at risk; it stays read-only on both servers. |
| A14 | **Tiny models / AI harness are a build plan.** | `docs/architecture/infra-map.md` §7, `docs/research/tiny-models-2026-11-22.md`, `docs/plans/FUTURE-PLATFORM-DIRECTION-2026-11-08.md` | docs only | **Keep as direction only.** Ruling 33: directional bias, re-run 2027-01-02 (T707). Ruling 28 gate: no new infrastructure before the canaries exist. |
| A15 | **The game-day release freeze applies to the Lab.** | older resumes and tasks (T677 "no lab releases during games") | docs only | **Retire (already done):** CLAUDE.md Environments moves the freeze to the Season server. Stray wording in old tasks only. |
| A16 | **Piece-tests are the safety net.** | test suites across `internal/` | ~24,000 test lines in desktop packages alone | **Keep, with flag.** Ruling 32: piece-tests made redundant by season-in-a-box (T711) and canaries (T708) are removed carefully, after the season. "This assumes the season box covers the same paths. If wrong, a silent break slips through. Cheapest check: plant one mutant per removed test's area and see the season box go red." |

---

## Part B. Task triage (open TheWarRoom tasks on the Hermes task list, read 2026-11-23)

Counts: still live 57, done 9, superseded 15, needs ruling 12. Total 93. No task should be deleted; Christopher rules, then each is ticked or reworded.

| ID | Gist | Recommendation |
|---|---|---|
| T015 | `CFBD_API_KEY` exported to every shell; move to project `.env` or revoke | Still live (TWR no longer parked; move to `.env`) |
| T378 | League history: full MFL archive done; next player-details pull and write-up | Still live (feeds T698 history import) |
| T399 | Legacy NFL Lab research tool (uncommitted branch), parked as his own tool | Needs ruling: keep parked, or fold into T698 history import |
| T432 | K3: two blind research passes on commissioner control surface | Superseded by T682 market study + ruling 6 (mock-first per commissioner job) |
| T456 | Pre-public gate: MFL→native import | Done in substance (`internal/leaguemigration`, #408 lineups at migration); keep open only as "flawless at every size" check under T699 |
| T457 | Flatpak / CasaOS packaging | Superseded by ruling 30 (desktop out); see A8 |
| T473 | Unmerged docs commit on session/split-scoring | Still live (tiny: drop it or open a docs PR; Stage 1 branch cleanup) |
| T488 | 52-slice client/server plan, slices 51-52 left | Superseded by BUILD-PLAN.md; slice 52 = 3-OS (A7) |
| T489 | Offline desktop restore acceptance gate | Superseded by ruling 30; see A6 |
| T490 | Client/server issues 10-24 (restore drops commissioner, DueWorker stops at first league error) | Needs ruling: item 1 is desktop-only (goes with A1); item 2 (one league's error blocks others) is server and still live — split the task |
| T495 | Scope default mode 2 "modified Legacy" | Needs ruling (A13: presets are only suggested values; is a second preset still wanted?) |
| T498 | Upload TWR profile page to the public resume chatbot | Needs ruling (outward-facing, his call) |
| T503 | Legacy study round-2 owner calls, retention picture | Still live (research; no build) |
| T505 | Drill 51: installer fixes, signed upgrades, header cap | Superseded in part: desktop installer goes with A1; signed server upgrade kept (A10) |
| T535 | 3-OS acceptance + freeze; research macOS VM | Superseded by ruling 30; see A7 |
| T537 | UI data plan on local branch session/ui-ux-groundwork; needs OK to push | Needs ruling: much is desktop-era; recommend archive the branch to `docs/archive/` |
| T538 | UI build 11-05 on build/INTEGRATE-1102; soak clicks | Superseded by Bundle 7/9 PWA work (#396, #442) |
| T559 | Market bid cards: contract years, typed amount, self-refresh | Still live (hosted Market; check on phone app) |
| T560 | Build back 21 parked stat reports as owner inquiries | Still live (after Stage 5; find a phone home) |
| T565 | Client/server plan step 1 | Superseded by BUILD-PLAN.md (A9) |
| T570 | Market slim-down (cold load, memory) | Still live (BUILD-PLAN Stage 5 folds it in) |
| T571 | Pre-existing test failures 11-05 | Needs ruling: recheck on main; several named tests are desktop-only and go with A1 |
| T583 | Client/server step 2 (contract lock, backup-API snapshot) | Superseded by BUILD-PLAN.md; snapshot work is merged |
| T584 | Future platform direction (seams, social, tiny models) | Still live as direction only (A14); re-check 2027-01-02 with T707 |
| T587 | Chatbot question: kicker test | Still live (his answer) |
| T591 | Savage debloat | Still live (BUILD-PLAN Stage 5) |
| T592 | League name setting; join page shows it | Still live (check Lab: join page wording #473) |
| T594 | UI to the mocks on the native league (desktop-era order) | Superseded by ruling 4 (phone mock v1) and Bundle 9 (#442) |
| T596 | warroom.secureprospective.com website with Tom | Still live (pairs with T1817) |
| T600 | Landing page live on lab CT 256 | Done (live since 11-10, PRs #224-#226); tick |
| T601 | Pulse leftovers parked for a smaller model | Still live (after Stage 5; phone first) |
| T603 | Logo (The Roof) films, paused | Still live (paused by him) |
| T610 | Trade desk session: backup QB value, league size scale | Still live |
| T615 | Storage block + one database per league | Done (#229-#238 merged, on Lab); tick |
| T616 | Hosted commissioner seat without a team | Done in substance (#470 no-seat account, #377 commissioner reads, desktop-modes "commissioner without a franchise opens Pulse and rules"); recheck on phone |
| T617 | One account holds commissioner seat + a team | Still live |
| T618 | Pulse game day 11-16 status; F51 player values branch | Superseded by Lab v50-v59 and #468; F51 values: still live under T705 |
| T619 | Desktop UI/UX planning (mock v2 outline) | Superseded by ruling 30 + ruling 4 |
| T621 | Rotate Lab CT 257 TLS + authenticator keys (leaked to transcript) | Still live (security; BUILD-PLAN item 3) |
| T622 | Alpha 0.10 status (stress fail, security 10/14) | Superseded by T631 (Lab v58 load PASS) and T632 |
| T626 | AI environment audit follow-ups (fleet) | Still live (fleet, not TWR code) |
| T631 | Load test: 29x2 PASS; p95 0.85 s vs 200 ms; rerun at 32 | Still live (#475 merged the script; #476 shared results/standings copy; rerun at 32 left) |
| T632 | Cloudflare rule: block plain HTTP and TLS < 1.3 | Still live (security; BUILD-PLAN item 3) |
| T637 | Run lab1 league import on CT 260 | Needs ruling: CT 260 is the Lab now; was it run? If yes tick |
| T638 | Private-notes storage for Franchise HQ (PR #313) | Needs ruling (league data change; panel first) |
| T640 | 0.10 stress superseded; recheck Week 10 vs MFL | Still live (only the Week 10 recheck) |
| T641 | League presets for smaller leagues | Still live (rehearsal shapes; presets = suggested values only) |
| T644 | REAL build of the desktop Control Room | Superseded by ruling 30 + ruling 6 (mock-first redesign); see A4 |
| T648 | Mobile mock v1 theme sweep + LAN dev server | Done in substance (Bundle 9 #442, #429-#437); tick or reword to "theme sweep only" |
| T649 | Remove picomatch audit ignore when fixed | Still live (weekly check) |
| T650 | Three big-play animation versions | Still live (bystander-visible live scoring = full polish, ruling 4) |
| T653 | README PR #339 sign-off | Superseded by #460 and #465 (README rewritten); close #339 |
| T654 | Merge handoff-note PR #342 | Needs ruling: HANDOFF notes retired (Q25); recommend close #342 unmerged |
| T655 | Per-person Home layout table design | Done in substance (#388 sync Home layouts, #405 server-side layouts); tick |
| T656 | 11-18 bundle status (B7-B9) | Done (#396, #442 merged); tick |
| T659 | Server on Samsung S8 Active (Termux) | Needs ruling: the Season server is the Acer laptop now; still wanted? |
| T660 | Alpha phone walk (iPhone 13, S8, S25) | Still live (Dec 6 real-device session) |
| T663 | Lab release automatic on merge to main | Still live |
| T664 | Move fast checks from pre-push to commit time | Still live |
| T665 | Daily cleanup PR job after alpha | Still live, gated by ruling 28 (canaries first) |
| T666 | Rename 96 old-word spots | Still live (Stage 5) |
| T668 | PWA auto-update blocker | Done (#452, #468 "PWA update fix"); tick |
| T673 | Live field first real proof 11-22 on S25 | Done if his 11-22 field check counts (T683 says "wall works as intended"); needs his tick |
| T677 | Startup readings (results/standings/odds/bracket) | Still live in part: warm-up merged #468; confirm first-open time on Lab, then tick |
| T679 | Pre-game preview follow-ups | Still live in part (#468 cites T679; playoff type 3 etc. left) |
| T681 | Test leagues + Stand-ins | Still live (Stage 7; rehearsal) |
| T682 | Market study soak (Sleeper/ESPN/Yahoo) | Still live (ruling 7) |
| T683 | Live wall side panel 50/50 split | Still live in part (#468 cites T683; confirm on Lab) |
| T684 | Yahoo scoreboard URL 404 | Still live |
| T685 | Browser/phone change push | Still live (phone check on Lab, then merge) |
| T688 | Report card checker on Hermes | Still live in part (#469 "Tuesday report card" merged; try on Lab test leagues) |
| T691 | NFL.com-displaced-league campaign after Jan 1 | Still live (outward; his call; gated by rehearsal) |
| T692 | LeagueSafe / escrow research | Still live (money: his call) |
| T694 | Redraft leagues 10/12/16 do not exist natively | Needs ruling: build the small one-season pieces (A11)? |
| T695 | Projections step 1 (Sleeper) | Needs ruling: Sleeper is not in Approved_Sources |
| T696 | Projections step 2 (own model on nflverse) | Still live |
| T697 | Public research site | Still live (future) |
| T698 | GATE part 4: history import ESPN + MFL, full export | Still live |
| T699 | GATE parts 1-3: rehearsal clean, 5 shapes, 32-team load | Still live |
| T700 | Phone market study + mock directions after Dec 6 | Still live |
| T701 | Today tab research + build path | Still live |
| T702 | Mini-games catalogue | Still live (future) |
| T703 | Dynasty stock market on public site | Still live (future) |
| T704 | Wizards after foundation tested | Still live (future) |
| T705 | Native power-rank input (both ranks) | Still live (see on Lab test leagues) |
| T706 | Grill infra map with Christopher | Still live |
| T707 | Re-run tiny-model research 2027-01-02 | Still live (dated) |
| T708 | Canaries (ruling 28) | Still live |
| T709 | Line budget (ruling 29) | Still live in part: guard + baseline merged #469; wire into CI |
| T710 | Desktop app out of main build (ruling 30) | Still live: keep list done; needs Claudebox review + your A3 ruling before removal |
| T711 | Season-in-a-box test (ruling 31) | Still live in part: merged #469; real binary + fake clock left |
| T1817 | Update promo site as recruiting flagship | Still live (outward; his call on content) |
| T1818 | Recruiter track (fleet) | Still live (not TWR code) |

Tasks needing a ruling (12): T399, T490, T495, T498, T537, T571, T637, T638, T654, T659, T694, T695. Also: tick T673 if the 11-22 field check counts, and T710 waits on A3.
