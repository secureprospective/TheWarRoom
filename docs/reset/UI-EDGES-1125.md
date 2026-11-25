# Brief for Astra 6.2 — TheWarRoom UI edges → a build plan for the next sessions

**From:** Tom (Claude, head brain on the Beelink) for Christopher Campbell · **Date:** Wed 2026-11-25
**Code state:** `main` at `5068968af` (PR #522). The Lab (the test server, Proxmox CT 260) runs that exact build (v93) and reports "Everything is working".
**Repo copy:** `docs/reset/UI-EDGES-1125.md` (the edge ids E1–E7 are cited by Hermes tasks T1836–T1842). **Read time:** the body is about 25 minutes. Appendices A and B are reference tables to search, not to read through.

---

## 0. Your job

Write a **build plan for the next 5 to 8 working sessions** (one session ≈ one evening or one overnight run).
It must close the most valuable **edges** listed here. An edge is a place where what a person needs, what the
mocks promise, and what the production app actually does stop lining up.

Return one markdown file with:

1. **The sessions in order.** For each session: its goal in one sentence, the items it ships, and what Christopher
   can see or click at the end of it (a Lab link, a screen, a number).
2. **One row per item:**
   - what changes for an owner or commissioner (plain words);
   - the edge ids it closes (from this brief);
   - size (S under 300 lines, M under 1,000, L more);
   - agent (see §8);
   - depends-on;
   - done lines with the check that proves each;
   - whether it **needs Christopher** (a ruling, a spec OK, a sign-off).
3. **A "Needs Christopher" list** at the top. Phrase each item as a pick-from question with your recommendation.
4. **What you chose not to plan, and why.** Name the edges deliberately left for later.
5. **Risks to the calendar in §2.** Name anything in your plan that could endanger the Dec 1 drafts or a Saturday swap.

Assume nothing beyond this brief. Where the brief says "verify", plan a check; don't treat it as fact. Don't
redesign the engines. The UI/UX is the hard part (ruling 6 below).

---

## 1. The product and the rulings that bind the plan

TheWarRoom (TWR) houses a dynasty fantasy-football league: its rules, its record ("the ledger") and everything people
do in it. Every rule is a setting. Any league can be built, and any MFL league can be imported. One server program,
`twr-server`, serves a React app (an installable web app, a PWA) to phones and computers. Every action is a named
command in one catalogue. Screens, the `twr` command line and the AI-helper tools (MCP) are all doors onto the same
commands. The first real league is Christopher's 32-team dynasty IDP salary-cap league, "Legacy", which still lives
on MFL. TWR is its companion until a native season is proven.

**Rulings every plan item must respect** (from `docs/reset/BUILD-PLAN.md` §1, `docs/adr/product/`, `CLAUDE.md`):

1. Through 2027 the aim is one real 32-team league running a full season on TWR. Legacy is never put at risk.
2. There are two kinds of league. **MFL leagues**: TWR is a companion and MFL is the record. **Native leagues**:
   TWR is the record. Companion code is not leftover to delete.
3. Until July 2027: test leagues of many sizes and shapes, with no plan for which real league moves first.
4. **Build for Christopher's solo use first, until about March.** The exception is anything a bystander sees in
   phone live scoring, which gets full polish now. **The phone target is mobile mock v1 only.**
5. A feature should do at least one of three things: save time, make someone ask "what is that?", or protect the league.
6. Off-season direction: redesign every commissioner job until it is faster and clearer than the market. **Mock first.**
7. **Do a market study per commissioner job before any mock** (Sleeper, ESPN, Yahoo; read only).
8. **Stand-in** = an AI holding an empty seat and playing the team like an owner (test leagues only for now).
   **Caretaker** = a commissioner keeping an empty team alive.
9. **The campaign to recruit outside leagues waits on Christopher's phone OK** (ruled 11-24), even if that is later than January.
10. **Settings, not league types.** No code branches on a league's size, preset, source (MFL) or id. A preset is
    only a bundle of starting values.
11. **Never submit anything to MFL.** MFL is read-only reference and links. The My MFL moves helper never sends.
12. Confidence scores are internal; never show them. Solo-owner MFL tools: trim, never grow, never delete.
13. **The desktop app is gone** (ruled 11-22; removed 11-24/25, about 70,700 lines). "Desktop" now means the same PWA
    in a computer's browser. Desktop removal step 6 (**PR-F**: what happens to the remaining solo-owner tools and their docs)
    awaits a ruling.
14. Draft-room undo: **yes** (ruled 11-24). The spec `docs/specs/draft-room-undo-1124.md` awaits approval.
15. No rehearsal playoffs. The automatic season test proves playoffs instead (ruled 11-24).

---

## 2. The calendar (Central time)

| Date | What happens | Constraint on your plan |
|---|---|---|
| Sat 11-28 | Season server (Acer laptop) set up and swapped, with Christopher's OK | Only Saturday nights, and only with his OK each time |
| **Sun 11-29** | Practice drafts on the Lab, by Christopher's hand (copies of both rehearsal leagues) | The draft room and join flow must not regress this week |
| Mon 11-30 | One allowed Season-server fix, only if practice found a problem | — |
| **Tue Dec 1** | **The real rehearsal drafts, hosted on the Lab** | Nothing risky merges Mon–Tue. Lab releases are paused during the drafts |
| Thu Dec 3 | NFL week 13, the first scored rehearsal week | Sleeper accuracy (week 12) checked before this |
| Sat Dec 5 | Swap: wall on real live screens (T689) | — |
| Sun Dec 6 | Rehearsal Sunday 1 | Live scoring polish matters from here |
| Tue Dec 8 | First Tuesday report card | — |
| After Dec 6 | Phone market study and 2–3 mock directions (T700); Today tab research (T701) | Phone redesign starts here, not before |
| Dec 22 | Token renewal | — |
| Jan 2 | Tiny-model / harness research rerun (T707) | — |

---

## 3. Who uses it, and the four ways in

| Role | Who | Ways in today |
|---|---|---|
| **Owner** | a team owner | the phone or browser app |
| **Commissioner** | runs a league | the app's Control Room, **plus the server console for several jobs (edge E1)** |
| **AI helper** | an AI working for one named person in one league, with that person's token | `twr` CLI and MCP tools (`league_read`, `league_workflow`, `league_ledger`) |
| **Operator** | Christopher or Tom at the server's terminal | `twr-server <command>` (28 commands), Lab scripts, status screen |

**Every way in runs the same catalogue.** 263 entries on main (Appendix A). 156 are hosted workflows, which the
phone, browser and AI helpers all use. 58 have a browser table entry. 50 have neither (the console, the MFL
reference and leftovers).

---

## 4. The production UI on main (what a person actually gets)

Startup (`frontend/src/main.tsx` → `control/NativeStartup` → `control/NativeAccount.tsx`) picks one of two layouts.

**Phone (narrow viewport) → `shell/PhoneAppFrame.tsx`.** It has five tabs and two sheets.

| Tab | File | What is on it today (from its own labels) |
|---|---|---|
| Today | `phone/TodayTab.tsx` | My boards; **Needs you** with "Snooze until tomorrow"; "Open to see scores" |
| Live | `phone/LiveTab.tsx`, `LiveField.tsx`, `LivePregame.tsx` | Your matchup (pre-game: win chance, stadium weather, your players); field view; box score; all matchups; NFL games and NFL box score; play by play for the followed game |
| Team | `phone/TeamTab.tsx` | Starters, bench and IR; drag a player to move them, then "Check and save"; "STARTER ON BYE"; player card; My draft picks; Trade. **"Drop is not on the phone yet"** |
| Moves | `phone/MovesTab.tsx`, `MovesSheets.tsx` | Search the free-agent pool; claim and bid; claim order (move up/down, cancel); trade offers (Accept, Counter, Decline, Withdraw); Start a trade |
| League | `phone/LeagueTab.tsx` | Standings with streaks; schedule by week; activity; team rosters; Propose a trade; **Control Room** (commissioner, with open-job count); "Manage this seat" |
| Sheet: Leagues | in `PhoneAppFrame` | switch league |
| Sheet: Ask | `shell/phoneAsk.ts` | **Canned answers built from data already on the phone** ("You are ahead of X, 88–71"). No AI, no free text to a person |

**Browser on a computer → `v3/V3Shell.tsx`.** This is the desktop mock v3 shell, with a rail of five areas, each loaded lazily.

| Rail area | Files | Notes |
|---|---|---|
| Home | `shell/ThinHome.tsx`, `HomeBrief.tsx`, `homeTiles.tsx`, `v3/pieces.tsx` | Boards of "pieces" (tiles) |
| Franchise HQ | `shell/NativeFranchiseHQ.tsx`, `HQShell.tsx`, `v3/HQLineupV3.tsx`, `HQMovesV3.tsx`, `HQContractsV3.tsx`, `shell/MflMovesPanel.tsx` | Lineup and roster, contracts and cap, My moves, **My MFL moves** (new 11-24, #489) |
| War Room | `shell/WarRoomNative.tsx`, `v3/WarDraftV3.tsx`, `WarPoolV3.tsx`, `WarTradeV3.tsx`, `WarPlansV3.tsx` | Draft, pool, trade, plans |
| League Pulse | `shell/NativeLeaguePulse.tsx`, `v3/PulseV3.tsx`, `PulseRaceV3.tsx`, `pulseDrawers.tsx` | Now (live), race, schedule, activity |
| Control Room | `v3/ControlRoomV3.tsx` + `Ctrl{Jobs,Rules,Season,People,Picks,Fix,League,Server,Situations,Feedback}V3.tsx` over `commissioner/HostedControlRoom*.tsx` | The commissioner's room. The phone reaches the same hosted Control Room from League |

The franchise screen inside both layouts is `commissioner/HostedLeague.tsx`. Its tabs: Set lineup · Roster and reserves ·
Draft room · League rules · Free-agent bids · Waivers · Trades · Results and standings · Next season.

**Other pages:** the join page (one league-wide link; "You now own Franchise 2"), the sign-in screen, the "no team
yet" message, and the **wall** (`/app/wall`, `wall/LiveWall.tsx`, a big-screen live display).
**House rules:**
- the entry bundle has a hard cap of 321,818 B (now about 173 kB), so new screens load lazily;
- every endpoint is a row in `docs/ui/endpoint-registry.csv`;
- the command tables are generated (`frontend/scripts/nativegen`); never hand-edit `*.gen.ts`.

---

## 5. The mocks (the targets)

| Mock | Where | What it is | Status |
|---|---|---|---|
| **Desktop mock v3** | `~/Downloads/TheWarRoom-mock-v3-FINAL.html` (tagged copy `…-v3-TAGGED.html`) | The full desktop target: shell, Home boards, Franchise HQ, War Room, League Pulse, Control Room | Inventoried 11-17 into **287 edges** (`docs/ui/v3/v3-stubs.json`, plan `docs/ui/v3/V3-STUB-PLAN.md`, Christopher: "go with yours on all 6" decisions). Re-checked against main in **Appendix B** |
| **Mobile mock v1** | `~/Downloads/TheWarRoom-mobile-mock-v1.html` (copy in `~/archive/2026-11-twr-mock-v1/phone/`) | **The phone target** (ruling 4) | 56 jobs, mock-to-mock parity in `~/fleet/runs/twr-v3-1117/MOBILE-V1-PARITY.md`. Against production in **§6 E2** |
| Older section mocks | `docs/ui/mocks/` (home v5, franchise-hq v10, warroom v6, league-pulse v14, control-room v1, rail v1) | Earlier per-screen targets | Superseded by v3 where they conflict |
| Showcase | `docs/ui/showcase/*.png` | Screens and target images | Reference |

---

## 6. The edges

Each edge has an id you can cite. "Verify" means plan a check; it is not established.

### E1 — Commissioner jobs that only the server console can do

An outside commissioner can't run these without us. The server command exists, but no screen does.

| Edge | Console command today | In-app today | Note |
|---|---|---|---|
| E1.1 | `twr-server join-link` (make, `-rotate`, `-revoke`, `-release` a seat, `-take`) | People screen lists seats, invite/caretaker words exist; **no join link** | Highest value. The commands exist since #474 |
| E1.2 | `create-league` (with a template; 10/12/16 one-season templates exist, #481) | none | Today a person can't start a league by themselves |
| E1.3 | `import-league` (MFL → native, now prints what differs, #507) | none (the desktop import screen was deleted in PR-H) | "The import is the front door and must be flawless" |
| E1.4 | `export-league`, `restore-league`, `delete-league` | Backups / "Back up now" exist in Control Room → Server | Export is GATE part 4 too |
| E1.5 | `reset-password`, `reset-sign-ins` | none | An owner who forgets a password needs the operator |
| E1.6 | `pin-football` (which NFL data version a league reads) | none | Rarely needed by a commissioner; probably stays an operator job |

Operator-only, by design: `hosted`, `upgrade`, `status`, `version`, `verify-release`, `backup-check`, `recover`,
`stand-up`, `restore`, `conversion-proof`, `cancel-move`/`finish-move` (league moves between servers), `mirror-fetch`,
`refresh-nfl-schedule`, `refresh-nfl-injuries`, `requests` (request counters from the running server), `import-model-scores`, `export-model-run`, `sleeper-accuracy`.

### E2 — Mobile mock v1 (the phone target) against the production phone app

"Code exists" means the hosted app has the code. Nobody has re-run the 56-row tap test against the real Lab app.
**Plan that test first** (the mock version came from `t-reach.mjs` in `~/fleet/runs/twr-v3-1117/`). Then fix what fails.

| # | Job (mobile v1) | Mock path | Production phone | Edge |
|---|---|---|---|---|
| 1 | Home board of pieces | Today › My boards | code exists (TodayTab) | — |
| 2 | Build a new board | desktop only by decision | n/a | — |
| 3 | Edit a board | Today › Edit this board | **not found in TodayTab labels — verify** | E2.3 |
| 4 | Live game field | Live | code exists (LiveField) | — |
| 5 | My matchup score and win chance | Live | code exists (LiveTab, LivePregame) | — |
| 6–9 | Matchup box score, other matchups, NFL scores strip, NFL box score | Live › … | code exists | — |
| 10 | Set lineup | Team › MOVE › slot | code exists (drag, Check and save) | — |
| 11 | What-if projection | Team | partial: projection wording exists; **real projections are not shown (Sleeper collected but hidden, E5.4)** | E2.11 |
| 12 | Player card with status | Team › name | code exists | — |
| 13 | **Player notes** ("My note") | Team › name › My note | **missing** | E2.13 |
| 14 | My draft picks | Team › My picks | code exists | — |
| 15 | Free agents, search, filter | Moves | code exists | — |
| 16 | Add / **drop** | Moves › ADD | **partial: "Drop is not on the phone yet"** | E2.16 |
| 17–19 | Waiver claim, blind bid, claim order | Moves › … | code exists | — |
| 20–21 | Answer a trade, build a trade | Moves › Trades | code exists | — |
| 22 | Deadlines and reminders | Today › Needs you | code exists (snooze). **No push alerts anywhere (E5.1)** | E5.1 |
| 23–26 | Standings, schedule, activity, player status feed | League › … | code exists | — |
| 27 | **Ask the assistant** | Ask | **canned answers only; no AI, no conversation** | E5.2 |
| 28 | Switch league | league name | code exists (Leagues sheet) | — |
| 29 | Commissioner jobs list | League › Control Room | code exists (open-job count on the League tab) | — |
| 30–33 | Trade review, stat corrections, bid/waiver tie, quiet owners / lineup holes | Control Room › job | jobs code exists; **verify each job type renders on the phone** | E2.30 |
| 34 | Vacant seat | Control Room › job | code exists (People) | — |
| 35 | **Complaint (collusion)** | Control Room › job | **missing** (no complaint/collusion anywhere) | E2.35 |
| 36 | Notice: trade deadline | Control Room › job | code exists | — |
| 37 | Snooze / reopen a job | Control Room › Later | code exists | — |
| 38 | **Written request (ask box)** | Control Room › ask box | **missing** (no request form found) | E2.38 |
| 39 | Situations (12 stories) | Control Room › Situations | code exists | — |
| 40 | Settings board + source tags | Control Room › Rules | code exists (HostedRules) | — |
| 41 | Presets with compare | Rules › preset | presets exist; **compare: verify** | E2.41 |
| 42–44 | Season dates, playoff bracket, crown champion / rollover | Control Room › Season | code exists | — |
| 45 | Divisions | League › Divisions | code exists | — |
| 46–47 | Fix forms with reason, recent fixes with undo | Control Room › Fix | code exists | — |
| 48 | People: seats, search, filters | Control Room › People | code exists | — |
| 49 | Seat tools (invite, caretaker, replace, remove) | team › Manage this seat | invite/caretaker exist; **replace/remove: verify; join link missing (E1.1)** | E2.49 |
| 50 | Deputies with permissions | Manage this seat | code exists. **`commish.deputy.permissions.set` has no screen caller (E4.1)** | E2.50 |
| 51 | Picks: all, filters | Control Room › Picks | code exists | — |
| 52–53 | League identity / layout, post a notice | Control Room › League | code exists | — |
| 54–56 | Backups / restore (hold), feeds pause / resume, ledger | Control Room › Server | code exists | — |

**Phone polish rule:** anything a bystander sees in **live scoring** (Live tab, wall) gets full polish now (ruling 4).
The rest of the phone redesign follows the market study after Dec 6.

### E3 — Desktop mock v3: 287 edges (full table: Appendix B)

States as of 11-17:
- 74 THIN (a real reading behind a thin UI)
- 44 PARTIAL
- 72 LOCAL (existed only in the deleted desktop local mode)
- 74 NEW (nothing behind it)
- 11 MOCK-only
- 7 need a RULING
- 5 SERVER work

What has changed since 11-17:
- **E3.1 — 59 rows name an owner file that no longer exists**, mostly the old local Control Room views
  (`control/room/JobsView.tsx`, `SeasonView`, `PeopleView`, `RulesView`, `PicksView`, `FixView`, `PresetCompare`,
  `openers.tsx`, `ServerView`, `LeagueView`) and old commissioner forms (`RuleChangeForm`, `NoteForm`, `UndoForm`,
  `BackupNow`, `LeaguePeople`, `LeagueLedger`), plus `ShellRail`, `CommandBar` and `StatusStrip`. 49 of the 59 were LOCAL.
  Their hosted replacements are `v3/Ctrl*V3.tsx` and `commissioner/HostedControlRoom*.tsx`. **The v3 stub plan must
  be re-pointed at the hosted files before anyone builds from it.**
- **E3.2 — 81 rows need a command that doesn't exist** ("NEW:" feeds), e.g. claim-priority reorder (`clMove`, waiting on
  a ruling: are claims reorderable?), board building, pop-out, undo/redo of UI layout. Many are UI-only (no
  command needed). Sort them: UI-only, new command, or needs a ruling.
- **E3.3 — 113 feed references are free text, not command ids** (e.g. "see docket", "account endpoints /v1/auth/*",
  "guide content"). Resolve each to a real command or mark it UI-only.
- Of the commands the v3 rows do name, **338 references resolve to a command a screen already calls** and 16 to a
  command with no screen yet.

### E4 — Command catalogue edges (full table: Appendix A)

- **E4.1 — 26 commands that no screen calls:**
  - `calendar.window.set`/`remove`
  - `commish.controls.read`, `commish.deputy.permissions.set`, `commish.digests.read`
  - `commish.league.seed_held`, `commish.leagues.read`, `commish.rules.preset.read`
  - `commish.succession.accept`/`decline`/`activity`
  - `contract.action.set`, `history.link`, `league.stand_up`
  - `mfl.bid.*` (7), `mfl.pool.read`/`refresh`
  - `native.roster.check`, `native.rules.read`, `score.player.read`

  Some are AI-helper or console only on purpose. Classify each: wanted screen, helper-only, or delete.
- **E4.2 — 19 old desktop-shell chrome commands named only in the registry or search** (`commandbar.open`/`close`,
  `inspector.*`, `surface.open`/`close`, `overlays.close`, `density.set`, `phase.goto`, `preset.apply`, `refresh.run`,
  `cache.run`, `data.run`, `control.read`, `commissioner.read`, `mflkey.connect`/`forget`). These are probably dead
  since desktop removal. Verify, then remove or re-home.
- **E4.3 — 12 commands still marked desktop-only:** `commish.league.seed_held`, `mfl.bid.handoff`/`plans`/`posted`/
  `preview`/`revise`/`submit`/`withdraw`, `mfl.lineup.set`, `mfl.pool.read`/`refresh`, `mfl.trade.accept`. These are
  the old desktop MFL doors. Hosted My MFL moves (`mflmoves.*`, 12 commands) replaced them. Their fate is tied to
  **PR-F (needs a ruling)**.
- **E4.4 — preview and undo:**
  - 134 commands change something; 97 have both a preview and an undo.
  - 11 have neither, including `draft.pick` (an owner's own pick), `trade.draft.delete` and `mflmoves.plan.done`.
  - 12 have a preview but no undo, including the **7 G3 commissioner draft-room commands** `draft.assign`,
    `draft.change`, `draft.insert`, `draft.revert`, `draft.skip`, `draft.timer` and `draft.startup.open` (undo ruled
    yes; spec awaits OK).
  - 14 have an undo but no preview.
  - Source: `docs/architecture/catalogue-preview-undo.md`.

### E5 — Things an owner would expect that don't exist

| Edge | Gap | State | Note |
|---|---|---|---|
| E5.1 | **Phone alerts (push)** | None in the code. No web push | "Sleeper-speed scores + alerts" is a must-match before the campaign. On iPhone, push works only for an installed PWA. Size L. Design first |
| E5.2 | **Chat** (owner↔owner, league board, draft-room chat) | **None.** No command, table, handler or screen. Retired 11-04 (`docs/research/owner-chat-check-1124.md`) | Christopher is discussing chat with Tom now. **Plan a slot but don't design it.** Tom will hand you the decision |
| E5.3 | Ask is canned | `shell/phoneAsk.ts` answers from data already on the phone | An AI helper door already exists (`twr`/MCP, with token). Connecting Ask to a real helper is a product decision tied to E5.2 |
| E5.4 | **Projections not shown** | Sleeper forecasts are collected and scored by league rules on the Lab (#497). The accuracy report exists (#506) | Run it on week 12 before Dec 3; if it holds up, show forecasts on lineup screens (phone Team, HQ lineup) |
| E5.5 | Stat corrections: "what changed" for owners | Engine and season test cover corrections; the owner-visible change is **unverified** | Must-match item |
| E5.6 | Recap share, awards, trophy plates, mini-games | Recap reading exists; the voice and one-tap share do not; awards none; mini-games a catalogue only | Fun layer, no dates. Recommended: Tuesday recap + power-rank arrows into the rehearsal |
| E5.7 | Player notes, drop on phone, complaint, written request | see E2.13, E2.16, E2.35, E2.38 | — |

### E6 — Components next to the UI that a plan must keep whole

| Component | What it is | Edge for the plan |
|---|---|---|
| AI-helper doors | `twr read`/`workflow`/`ledger`/`preview`, MCP `league_read`/`league_workflow`/`league_ledger`; helper token from the app's "helper line" | E6.1: **any new screen action must be a catalogue command**, so helpers get it free. Never a screen-only action |
| Season in a box | real-server season test in CI: draft → lock → live → final → correction → standings → Tuesday check → playoff | E6.2: new owner flows should add a step here |
| Tuesday report card | Hermes, read token, 4 checks, GREEN/RED | — |
| Canary | `cmd/twr-canary`, 7 fixed + 5 random owner journeys, seeded; **not running (needs a Lab token, T708)** | E6.3: once running, new journeys join it |
| Load test | `scripts/load/gameday.py`, 32 teams × 2 devices; PASS p95 193 ms vs 200 target on 2 CPUs | E6.4: **any change to Live/results/standings reads reruns it.** Only 7 ms of headroom. CT 260 to 4 CPUs awaits Christopher |
| Wall | `/app/wall`, the big-screen live display | T689 for the Sat Dec 5 swap |
| Lab kit | `scripts/lab/` (practice drafts, seat links, phone seating) | Lab-only guard: `native.rehtest-*` |
| Server-Timing + slow log | every response carries server time; ≥250 ms logs one line | Use it to judge any read change |

### E7 — Gaps from the 11-25 build map (`~/fleet/runs/buildmap-1125/build-map-1122-1125.html`)

1. Commissioner can't make, rotate or revoke the join link in the app → **E1.1**.
2. No undo on the 7 commissioner draft-room commands → E4.4 (spec awaits OK).
3. The Lab has 2 CPUs and hosts the Dec 1 drafts (Christopher's call on 4 CPUs).
4. Canary built, not running (T708 token).
5. Sleeper forecasts collected, not shown → E5.4.
6. **10/12/16-team one-season leagues exist only as settings.** Tom is standing them up on the Lab **now**
   (GATE part 2); treat as in progress, not yours to plan.
7. History in and out: nothing built (GATE part 4). Spec `docs/specs/history-in-out-1124.md` awaits OK.
8. No owner chat → E5.2 (in discussion now).
9. Football evidence split (GATE part 3 #3): design written (`docs/specs/football-evidence-split-1124.md`), not built; the 32-team load test reruns after it.

---

## 7. Gap register beyond the UI (so the plan doesn't contradict it)

- **GATE** (four parts before any outside league):
  1. Rehearsal clean (in progress with the season).
  2. Every league shape: 10/12/16 (Tom now); **redraft season 2 missing** (rollover keeps every player; no per-season startup draft, no contracts-off switch; L).
  3. One copy of each NFL fact: 3 of 4 done; the split is left.
  4. History in and out: nothing built.
- **Must-match before the campaign:**
  - Live scores: fast. **Alerts: none.**
  - Full lineup on one phone screen: waits on T700.
  - Corrections "what changed": unverified.
  - History: GATE 4.
  - Chat: none.
- **Rollback drill** at a Saturday swap; **token renewal** by Dec 22.
- **Desktop PR-F** ruling; `tools/twr-practice/README.md` still describes desktop click tests in places.

---

## 8. How work gets done here (make the plan executable)

- **One item = one pull request.** It is mergeable only with:
  - CI green;
  - a reviewer-planted break that its test catches;
  - a line-budget reason file if lines grew;
  - the entry bundle under the cap.
- Main merges need Christopher's OK. The Lab upgrades itself after each merge. The Season server changes only on
  Saturday nights with his OK.
- **Agents:**

  | Agent | Takes | Notes |
  |---|---|---|
  | **Luna** (GPT, high effort) | small, well-scoped jobs: one function, one file, one outcome, under 40-line briefs | It quits early and skips tests; Tom writes the tests |
  | **Sol** (GPT, medium effort) | larger jobs across several files | — |
  | **Gemini Neon** | multi-file deletions and refactors | — |
  | **Tom** | reviews every line, plants the break, writes tests, multi-file judgment, ops (Lab, Hermes) | — |
  | **Claudebox** | second reviewer, cold-start tests | — |

  Say which agent each item needs. Never route a design or mock to Luna.
- **Mock first for commissioner jobs** (ruling 6), after a market study per job (ruling 7). A plan that builds a
  commissioner screen with no mock and no study breaks the rulings. Phone redesign waits for the post-Dec 6 study.
  Live-scoring polish does not wait.
- Size guide: S under 300 lines, M under 1,000, L more. Two to four items per session is realistic. One L fills a session.
- Reuse what exists: the reading hooks (`app/data/query.ts`), `HoldToFire`, `PreviewReview`, `ChangeBar`, the
  Control Room job pattern, `SignalChip`. See `docs/ui/v3/UIUX-INVENTORY-2026-11-17.md` §4–5 (building blocks, UX
  laws) and §9 (slop not to copy). Its file names before 11-24 may be stale (E3.1).

## 9. Don't plan these

- Anything that branches on league size, preset, source or id.
- Any MFL write.
- Showing confidence scores.
- Growing the solo-owner MFL tools.
- A second app or a native mobile app.
- Changes to the engines' maths.
- Work on the Season server outside a Saturday swap.
- Starting the campaign.
- Phone redesign beyond live scoring before the Dec 6 study.

## 10. Surface these to Christopher instead of deciding

- PR-F (what remains of the solo-owner tools).
- Claim reordering (are waiver claims reorderable?).
- Chat (being discussed now).
- Push alerts (design and iPhone install path).
- Commissioner self-serve league creation (E1.2): when, and for whom before March, given ruling 4.
- Which E4.1 commands are helper-only versus wanted on a screen.
- 4 CPUs for the Lab.
- The draft-room undo, history and evidence-split specs.

---

## Appendix A — Every league command (263), its doors and its screens

Generated 2026-11-25 from `commandcatalog.GenerationRegistry()` on main `5068968af`. Doors: **B** = browser table, **W** = hosted workflow (phone/PWA and AI helpers via `twr workflow`), **CLI** = `twr` verb. Avail = availability. *Screen* = frontend files (non-test, non-generated) that name the command; `—` = no screen calls it. A name match is a strong lead, not proof that the button is reachable.

| Command | Permission | Grav | Avail | Modes | Undo | Doors | Screen callers |
|---|---|---|---|---|---|---|---|
| `acquisition` |  |  | — |  |  | B | commands/registry.ts, contracts/Acquisitions.tsx, pool/BidPlans.tsx |
| `backup.now` | commissioner-or-co-commissioner | G1 | server | preview/execute | none | B W | commands/registry.ts, commissioner/HostedBackupNow.tsx |
| `bait.set` |  |  | — |  |  | B | commands/registry.ts, personal/PersonalLists.tsx |
| `bid.place` | command:bid.place | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, shell/warroomMarketReads.ts |
| `bid.raise` | command:bid.raise | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, shell/warroomMarketReads.ts |
| `bids.award.override` | command:bids.award.override | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, commissioner/actTable.ts, control/room/guide.ts |
| `bids.process` | command:bids.process | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actBody.ts, commissioner/actReadings.ts, commissioner/actTable.ts, control/room/guide.ts |
| `bids.read` | franchise-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, shell/homeReads.ts, shell/homeTiles.tsx, shell/warroomMarketReads.ts |
| `bracket.advance` | command:bracket.advance | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts, control/room/guide.ts |
| `bracket.build` | command:bracket.build | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts |
| `bracket.edit` | command:bracket.edit | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts |
| `bracket.preset.save` | command:bracket.preset.save | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `bracket.read` | league-view | G0 | both | read | none | W | commissioner/HostedControlRoomSeason.tsx, commissioner/actReadings.ts, shell/pulseReads.ts |
| `bracket.seed` | command:bracket.seed | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts |
| `cache.run` |  |  | — |  |  | B | commands/registry.ts |
| `calendar.read` | league-view | G0 | both | read | none | W | commissioner/HostedControlRoomSeason.tsx, commissioner/hostedControlRoomData.ts, commissioner/hqReading.ts, shell/homeTiles.tsx, shell/warroomCalendar.ts |
| `calendar.summon` |  |  | — |  |  | B | clock/ClockStrip.tsx, clock/deadlineRoute.ts, commands/registry.ts |
| `calendar.window.move` | command:calendar.window.move | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlSeasonV3.tsx |
| `calendar.window.remove` | command:calendar.window.remove | G2 | both | preview/execute | checked-reversal | W CLI | — |
| `calendar.window.set` | command:calendar.window.set | G2 | both | preview/execute | checked-reversal | W CLI | — |
| `commandbar.close` |  |  | — |  |  | B | commands/registry.ts |
| `commandbar.open` |  |  | — |  |  | B | commands/registry.ts |
| `commish.abilities.set` | command:commish.abilities.set | G2 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actBody.ts, commissioner/actTable.ts, v3/CtrlPeopleV3.tsx |
| `commish.actor.read` | session | G0 | both | read | none |  | commissioner/actReadings.ts |
| `commish.calendar.exception` | command:commish.calendar.exception | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.cap.adjust` | command:commish.cap.adjust | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.cap.relief` | command:commish.cap.relief | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.contract.edit` | command:commish.contract.edit | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.controls.read` | session | G0 | both | read | none |  | — |
| `commish.correction.revert` | command:commish.correction.revert | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actReadings.ts, commissioner/actTable.ts, commissioner/proxy.ts, control/room/guide.ts +1 |
| `commish.corrections.read` | league-view | G0 | both | read | none | W | commissioner/hostedControlRoomData.ts |
| `commish.deputy.order` | command:commish.deputy.order | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, v3/CtrlPeopleV3.tsx |
| `commish.deputy.permissions.set` | command:commish.deputy.permissions.set | G3 | both | preview/execute | checked-reversal | CLI | — |
| `commish.digests.read` | league-view | G0 | both | read | none |  | — |
| `commish.feedback.read` | commissioner-or-co-commissioner | G0 | server | read | none | W | commissioner/problemReports.ts |
| `commish.franchise.direction.set` | command:commish.franchise.direction.set | G1 | both | preview/execute | checked-reversal | W CLI | shell/warroomLens.ts |
| `commish.franchise.edit` | command:commish.franchise.edit | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `commish.jobs.read` | league-view | G0 | both | read | none | W | commissioner/hostedControlRoomData.ts, phone/LeagueTab.tsx, shell/homeReads.ts |
| `commish.league.create` | command:commish.league.create | G3 | both | preview/execute | none | CLI | commissioner/proxy.ts |
| `commish.league.seed_held` | command:commish.league.seed_held | G3 | desktop | preview/execute | none |  | — |
| `commish.leagues.read` | league-memberships | G0 | both | read | none |  | — |
| `commish.ledger.filtered.read` | league-view | G0 | both | read | none | W | commissioner/hostedControlRoomData.ts, shell/homeReads.ts, shell/homeTiles.tsx |
| `commish.ledger.read` | league-view | G0 | both | read | none | W | commissioner/actReadings.ts, commissioner/hostedControlRoomData.ts |
| `commish.lineup.correct` | command:commish.lineup.correct | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.lineup.set` | command:commish.lineup.set | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedLeague.tsx, shell/hqNativeLineup.tsx |
| `commish.market.read` | command-policy | G0 | both | read | none | W | commissioner/actReadings.ts, commissioner/hostedRoom.ts |
| `commish.notice.publish` | command:commish.notice.publish | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomLeague.tsx, commissioner/hostedControlRoomData.ts, commissioner/proxy.ts, control/room/guide.ts |
| `commish.notices.read` | league-view | G0 | both | read | none | W | commissioner/HostedControlRoomLeague.tsx |
| `commish.people.read` | league-view | G0 | both | read | none | W | commissioner/HostedControlRoomPeople.tsx, commissioner/actReadings.ts |
| `commish.player.position` | command:commish.player.position | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.player.position.clear` | command:commish.player.position.clear | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.reserve.exception` | command:commish.reserve.exception | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.role.grant` | command:commish.role.grant | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, v3/CtrlPeopleV3.tsx |
| `commish.role.revoke` | command:commish.role.revoke | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, v3/CtrlPeopleV3.tsx |
| `commish.rules.preset.read` | league-view | G0 | both | read | none |  | — |
| `commish.rules.read` | league-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, commissioner/hqReading.ts, shell/warroomMarketReads.ts |
| `commish.rules.set` | command:commish.rules.set | argument-dependent | both | preview/execute | checked-reversal | W CLI | commissioner/HostedRules.tsx, commissioner/hostedControlRoomData.ts, commissioner/proxy.ts, control/room/guide.ts |
| `commish.schedule.read` | schedule.manage | G0 | both | read | none | W | commissioner/HostedControlRoomSeason.tsx, commissioner/actReadings.ts |
| `commish.score.franchise` | command:commish.score.franchise | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/HostedWorkflows.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.score.player` | command:commish.score.player | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.seat.caretaker` | command:commish.seat.caretaker | G2 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPeopleV3.tsx |
| `commish.seat.invite` | command:commish.seat.invite | G2 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPeopleV3.tsx |
| `commish.seat.invite.redeem` | command:commish.seat.invite.redeem | G2 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts |
| `commish.seat.invite.revoke` | command:commish.seat.invite.revoke | G2 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.seat.remove` | command:commish.seat.remove | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPeopleV3.tsx |
| `commish.seat.replace` | command:commish.seat.replace | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPeopleV3.tsx |
| `commish.standings.override` | command:commish.standings.override | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.standings.override.clear` | command:commish.standings.override.clear | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actTable.ts |
| `commish.standings.record.adjust` | command:commish.standings.record.adjust | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/HostedWorkflows.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.stat.correct` | command:commish.stat.correct | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomFix.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlFixV3.tsx |
| `commish.stat.queue.approve` | command:commish.stat.queue.approve | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, commissioner/hostedControlRoomData.ts, control/room/guide.ts |
| `commish.stat.queue.reject` | command:commish.stat.queue.reject | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `commish.succession.accept` | command:commish.succession.accept | G3 | both | preview/execute | checked-reversal | CLI | — |
| `commish.succession.activity` | command:commish.succession.activity | G1 | both | preview/execute | checked-reversal | CLI | — |
| `commish.succession.decline` | command:commish.succession.decline | G3 | both | preview/execute | checked-reversal | CLI | — |
| `commish.succession.election.open` | command:commish.succession.election.open | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, v3/CtrlPeopleV3.tsx |
| `commish.succession.inactivity.set` | command:commish.succession.inactivity.set | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, v3/CtrlPeopleV3.tsx |
| `commish.succession.periods.set` | command:commish.succession.periods.set | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, v3/CtrlPeopleV3.tsx |
| `commish.succession.stepdown` | command:commish.succession.stepdown | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPeopleV3.tsx |
| `commish.succession.takeover` | command:commish.succession.takeover | G3 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.succession.voteout.open` | command:commish.succession.voteout.open | G3 | both | preview/execute | checked-reversal | CLI | commissioner/HostedControlRoomPeople.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPeopleV3.tsx |
| `commish.visibility.set` | command:commish.visibility.set | G2 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts |
| `commish.vote.cast` | command:commish.vote.cast | G2 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts, control/room/guide.ts |
| `commish.vote.close` | command:commish.vote.close | G3 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts |
| `commish.vote.tiebreak` | command:commish.vote.tiebreak | G3 | both | preview/execute | checked-reversal | CLI | commissioner/actTable.ts |
| `commish.whatsnew.post` | command:commish.whatsnew.post | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomLeague.tsx, commissioner/proxy.ts |
| `commissioner.draft` |  |  | — |  |  | B | commands/registry.ts, commissioner/RuleInput.tsx |
| `commissioner.execute` |  |  | — |  |  | B | commands/registry.ts, commissioner/ChangeBar.tsx, commissioner/HoldToFire.tsx |
| `commissioner.read` |  |  | — |  |  | B | commands/registry.ts |
| `commissioner.surface` |  |  | — |  |  | B | commands/registry.ts, commissioner/HostedLeague.tsx, commissioner/HostedWorkflows.tsx, commissioner/SideTray.tsx, control/NativeAccount.tsx +6 |
| `contract.action.set` | command:contract.action.set | argument-dependent | both | preview/execute | checked-reversal | W CLI | — |
| `contract.decision.set` | command:contract.decision.set | G3 | both | preview/execute | checked-reversal | W CLI | shell/hqContractDecisions.ts |
| `contract.decisions.read` | franchise-owner | G0 | server | read | none | W | commissioner/hqReading.ts |
| `contract.quote` | league-view | G0 | both | read | none | W | commissioner/hqReading.ts |
| `contracts.set` |  |  | — |  |  | B | commands/registry.ts, contracts/ContractActs.tsx, contracts/Contracts.tsx, contracts/ReleasePlans.tsx, contracts/WhatIfPanel.tsx +2 |
| `control.read` |  |  | — |  |  | B | commands/registry.ts |
| `correct.pick` | command:correct.pick | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomPicks.tsx, commissioner/actTable.ts, control/room/guide.ts, v3/CtrlPicksV3.tsx |
| `correct.provenance` | command:correct.provenance | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `correct.roster` | command:correct.roster | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `correct.transaction` | command:correct.transaction | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `data.parameter` |  |  | — |  |  | B | commands/registry.ts, control/PickSettings.tsx |
| `data.run` |  |  | — |  |  | B | commands/registry.ts |
| `density.set` |  |  | — |  |  | B | commands/registry.ts, commands/search.ts |
| `draft.archive` | command:draft.archive | G2 | both | preview/execute | none | W CLI | commissioner/hostedControlRoomData.ts, draftroom/trayAction.ts |
| `draft.assign` | command:draft.assign | G3 | both | preview/execute | none | W CLI | commissioner/hostedControlRoomData.ts, draftroom/trayAction.ts |
| `draft.change` | command:draft.change | G3 | both | preview/execute | none | W CLI | commissioner/hostedControlRoomData.ts, control/room/guide.ts, draftroom/trayAction.ts |
| `draft.insert` | command:draft.insert | G3 | both | preview/execute | none | W CLI | commissioner/hostedControlRoomData.ts, draftroom/trayAction.ts |
| `draft.list` |  |  | — |  |  | B | commands/registry.ts, personal/PersonalLists.tsx |
| `draft.order.auto` | command:draft.order.auto | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `draft.order.pattern` | command:draft.order.pattern | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `draft.order.set` | command:draft.order.set | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `draft.order.tiebreak` | command:draft.order.tiebreak | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts |
| `draft.pick` | command:draft.pick | G1 | both | execute | none | W | commissioner/HostedLeague.tsx, commissioner/hostedControlRoomData.ts, commissioner/hostedRoom.ts |
| `draft.queue.read` | franchise-view | G0 | server | read | none | W | shell/warroomDraftReads.ts |
| `draft.queue.set` | franchise-owner | G1 | server | execute | replace-own-private-queue | W | shell/warroomDraftReads.ts |
| `draft.read` | league-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, phone/TeamTab.tsx, shell/homeReads.ts, shell/homeTiles.tsx, shell/warroomDraftReads.ts |
| `draft.reports` |  |  | — |  |  | B | commands/registry.ts, draftreports/Reports.tsx |
| `draft.revert` | command:draft.revert | G3 | both | preview/execute | none | W CLI | commissioner/hostedControlRoomData.ts, control/room/guide.ts, draftroom/trayAction.ts |
| `draft.skip` | command:draft.skip | G3 | both | preview/execute | none | W CLI | commissioner/HostedLeague.tsx, commissioner/hostedControlRoomData.ts, draftroom/trayAction.ts |
| `draft.start` | command:draft.start | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, commissioner/hostedControlRoomData.ts |
| `draft.startup.open` | command:draft.startup.open | G3 | both | preview/execute | none | W CLI | draftroom/trayAction.ts |
| `draft.timer` | command:draft.timer | G3 | both | preview/execute | none | W CLI | commissioner/HostedLeague.tsx, commissioner/hostedControlRoomData.ts, control/room/guide.ts, draftroom/trayAction.ts |
| `endpoint.open` |  |  | — |  |  | B | commands/registry.ts, commands/search.ts, draftreports/Reports.tsx, registry/EndpointIndex.tsx, registry/ReportDrawer.tsx +2 |
| `feedback.report.create` | league-member | G0 | server | execute | none | W | commissioner/problemReports.ts, data/FailureAlert.tsx |
| `franchise.moves.history.read` | league-view | G0 | both | read | none | W | shell/HQMoveHistory.tsx |
| `franchise.set` |  |  | — |  |  | B | commands/registry.ts, shell/FranchisePicker.tsx |
| `game.follow` |  |  | — |  |  | B | commands/registry.ts, now/LiveGame.tsx |
| `history.link` | command:history.link | G2 | both | preview/execute | checked-reversal | W CLI | — |
| `home.layout.read` | account-owner | G0 | server | read | none | W | shell/homeLayout.ts |
| `home.layout.set` | account-owner | G0 | server | execute | replace-own-account-preference | W | shell/homeLayout.ts |
| `inspector.close` |  |  | — |  |  | B | commands/registry.ts |
| `inspector.collapse` |  |  | — |  |  | B | commands/registry.ts |
| `inspector.expand` |  |  | — |  |  | B | commands/registry.ts |
| `inspector.open` |  |  | — |  |  | B | assets/AssetBoardView.tsx, commands/registry.ts, commands/search.ts, contracts/Contracts.tsx, shell/FranchiseHQ.tsx +2 |
| `inspector.pin` |  |  | — |  |  | B | commands/registry.ts, commands/search.ts, now/LiveGame.tsx, personal/PersonalLists.tsx, reports/ReportView.tsx +1 |
| `inspector.toggle` |  |  | — |  |  | B | commands/registry.ts |
| `job.snooze.set` | account-owner | G0 | server | execute | replace-own-account-snooze | W | commissioner/HostedControlRoom.tsx, phone/TodayTab.tsx |
| `job.snooze.wake` | account-owner | G0 | server | execute | clear-own-account-snooze | W | commissioner/HostedControlRoom.tsx |
| `job.snoozes.read` | account-owner | G0 | server | read | none | W | commissioner/HostedControlRoom.tsx, shell/homeReads.ts |
| `league.activity.read` | league-view | G0 | server | read | none | W | shell/pulseReads.ts |
| `league.stand_up` | offline-operator | G3 | server | execute | fresh-folder-only |  | — |
| `lineup.cancel` |  |  | — |  |  | B | commands/registry.ts, shell/FranchiseHQ.tsx, shell/LineupEditor.tsx |
| `lineup.check` | franchise-view | G0 | both | read | none |  | data/lineupProvider.ts |
| `lineup.draft` |  |  | — |  |  | B | commands/registry.ts, shell/FranchiseHQ.tsx, shell/LineupEditor.tsx |
| `lineup.edit` |  |  | — |  |  | B | commands/registry.ts, shell/FranchiseHQ.tsx, shell/HQInspector.tsx, shell/HQRelease.tsx, shell/LineupEditor.tsx |
| `lineup.read` | franchise-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, commissioner/hqReading.ts, shell/pulseReads.ts |
| `lineup.reset` |  |  | — |  |  | B | commands/registry.ts, shell/LineupEditor.tsx |
| `lineup.toggle` |  |  | — |  |  | B | commands/registry.ts, shell/FranchiseHQ.tsx, shell/LineupEditor.tsx |
| `lineupdelta.read` | franchise-view | G0 | both | read | none |  | shell/inspectorFit.ts, shell/warroomTradeReads.ts |
| `market` |  |  | — |  |  | B | assets/AssetBoardView.tsx, assets/contract.ts, assets/state.ts, commands/registry.ts, market/ReadingView.tsx +8 |
| `market.reports` |  |  | — |  |  | B | commands/registry.ts, draftreports/Reports.tsx, reports/MarketReports.tsx |
| `market.wins.read` | league-view | G0 | server | read | none | W | shell/warroomMarketReads.ts |
| `mfl.bid.handoff` | local-desktop-owner | G2 | desktop | execute | withdraw-local-plan |  | — |
| `mfl.bid.plans` | local-desktop-owner | G1 | desktop | read | none |  | — |
| `mfl.bid.posted` | local-desktop-owner | G2 | desktop | execute | withdraw-local-plan |  | — |
| `mfl.bid.preview` | local-desktop-owner | G1 | desktop | preview | none |  | — |
| `mfl.bid.revise` | local-desktop-owner | G1 | desktop | execute | revise-local-plan |  | — |
| `mfl.bid.submit` | local-desktop-owner | G1 | desktop | execute | withdraw-local-plan |  | — |
| `mfl.bid.withdraw` | local-desktop-owner | G2 | desktop | execute | draft-new-local-plan |  | — |
| `mfl.lineup.set` | local-desktop-owner | G1 | desktop | execute | withdraw-local-plan |  | data/lineupReceiptTestData.ts, data/parseEnvelope.ts, shell/moveLabels.ts |
| `mfl.pool.read` | local-desktop-owner | G1 | desktop | read | none |  | — |
| `mfl.pool.refresh` | local-desktop-owner | G1 | desktop | execute | refresh-again |  | — |
| `mfl.trade.accept` | local-desktop-owner | G2 | desktop | execute | withdraw-local-plan |  | data/parseEnvelope.ts, data/tradeTestData.ts, shell/EnvelopeRail.tsx, shell/OfferCard.tsx, shell/moveLabels.ts +1 |
| `mflkey.connect` |  |  | — |  |  | B | commands/registry.ts |
| `mflkey.forget` |  |  | — |  |  | B | commands/registry.ts |
| `mflmoves.bid.check` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.lineup.check` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.plan.done` | franchise-owner | G0 | server | execute | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.plan.link` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.plan.save` | franchise-owner | G0 | server | execute | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.plan.withdraw` | franchise-owner | G0 | server | execute | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.plans.read` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.pool.read` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.refresh` | franchise-owner | G0 | server | execute | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.roster.read` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.team.set` | franchise-owner | G0 | server | execute | none | W | shell/MflMovesPanel.tsx |
| `mflmoves.teams.read` | franchise-owner | G0 | server | read | none | W | shell/MflMovesPanel.tsx |
| `move.handoff` |  |  | — |  |  | B | commands/registry.ts, shell/EnvelopeRail.tsx, shell/FranchiseHQ.tsx, shell/LineupPlan.tsx, shell/MyMoves.tsx +2 |
| `move.withdraw` |  |  | — |  |  | B | commands/registry.ts, shell/EnvelopeRail.tsx |
| `moves.read` | franchise-view | G0 | both | read | none | W | commissioner/hqReading.ts |
| `native.assets.read` | league-view | G0 | both | read | none | W | commissioner/HostedControlRoomPicks.tsx, commissioner/HostedLeague.tsx, commissioner/actReadings.ts, commissioner/hostedRoom.ts, commissioner/hqReading.ts +7 |
| `native.cap.check` | league-view | G0 | both | read | none | W | commissioner/hqReading.ts, shell/homeReads.ts, shell/warroomMarketReads.ts, shell/warroomTradeReads.ts |
| `native.directory.read` | league-view | G0 | both | read | none | W | commissioner/HostedControlRoomSeason.tsx, commissioner/actReadings.ts, commissioner/hostedControlRoomData.ts, control/ProblemReport.tsx, data/FailureAlert.tsx +1 |
| `native.reserve.check` | league-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, commissioner/hqReading.ts |
| `native.roster.check` | league-view | G0 | both | read | none | W | — |
| `native.rules.read` | league-view | G0 | both | read | none |  | — |
| `nav.open` |  |  | — |  |  | B | clock/deadlineRoute.ts, commands/registry.ts, commands/search.ts, contracts/contractActState.ts, shell/EntityInspector.tsx +5 |
| `news.feed.read` | league-view | G0 | server | read | none | W | research/espnNews.ts |
| `nfl.game.pregame.read` | league-view | G0 | server | read | none | W | shell/hostedGames.ts |
| `nfl.game.read` | league-view | G0 | both | read | none | W | shell/hostedGames.ts, wall/useWallGames.ts |
| `nfl.games.read` | league-view | G0 | both | read | none | W | phone/TeamTab.tsx, shell/hostedGames.ts, wall/useWallGames.ts |
| `nfl.player-status.read` | league-view | G0 | server | read | none | W | phone/playerStatus.ts |
| `note.add` |  |  | — |  |  | B | commands/registry.ts, personal/PersonalLists.tsx |
| `now.reports` |  |  | — |  |  | B | commands/registry.ts, draftreports/Reports.tsx |
| `odds.read` | league-view | G0 | both | read | none | W | shell/homeTiles.tsx, shell/pulseReads.ts |
| `oline.auction.close` | command:oline.auction.close | G2 | both | preview/execute | none | W CLI | commissioner/actTable.ts |
| `overlays.close` |  |  | — |  |  | B | commands/registry.ts |
| `personal.handoff` |  |  | — |  |  | B | commands/registry.ts, personal/PersonalLists.tsx |
| `phase.goto` |  |  | — |  |  | B | commands/registry.ts |
| `player.agearc.read` | league-view | G0 | both | read | none | W | market/api.ts |
| `player.news.read` | league-view | G0 | server | read | none | W | research/espnNews.ts |
| `player.note.read` | franchise-owner | G0 | server | read | none | W | shell/playerPersonal.ts |
| `player.note.set` | franchise-owner | G0 | server | execute | none | W | shell/playerPersonal.ts |
| `player.usage.read` | league-view | G0 | both | read | none | W | research/api.ts |
| `player.watch.order` | franchise-owner | G0 | server | execute | none | W | shell/warroomMarketReads.ts |
| `player.watch.read` | franchise-owner | G0 | server | read | none | W | shell/warroomMarketReads.ts |
| `player.watch.set` | franchise-owner | G0 | server | execute | none | W | shell/playerPersonal.ts, shell/warroomMarketReads.ts |
| `preset.apply` |  |  | — |  |  | B | commands/registry.ts, commands/search.ts |
| `projections.read` | league-view | G0 | both | read | none | W | commissioner/hqReading.ts, shell/pulseReads.ts, shell/warroomInspectorReads.ts |
| `race.reports` |  |  | — |  |  | B | commands/registry.ts, draftreports/Reports.tsx |
| `refresh.run` |  |  | — |  |  | B | commands/registry.ts |
| `report.sort` |  |  | — |  |  | B | commands/registry.ts, reports/ReportView.tsx |
| `research.read` |  |  | — |  |  | B | commands/registry.ts, draftreports/Reports.tsx, research/PlayerInspector.tsx |
| `results.read` | league-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, commissioner/hqReading.ts, shell/pulseReads.ts |
| `roster.add` | command:roster.add | G3 | both | preview/execute | checked-reversal | W CLI | shell/warroomMarketReads.ts |
| `roster.cut` | command:roster.cut | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx |
| `roster.ir` |  |  | — |  |  | B | commands/registry.ts, commands/sessionMoves.ts, shell/EnvelopeRail.tsx, shell/HQInspector.tsx, shell/HQPanels.tsx +3 |
| `roster.reserve` | command:roster.reserve | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedLeague.tsx, shell/hqNativeLineup.tsx |
| `roster.taxi` |  |  | — |  |  | B | commands/registry.ts, commands/sessionMoves.ts, shell/EnvelopeRail.tsx, shell/HQPanels.tsx, shell/PlayerAct.tsx +2 |
| `rule.exception` | command:rule.exception | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actTable.ts, control/room/guide.ts |
| `schedule.edit` | command:schedule.edit | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actBody.ts, commissioner/actTable.ts |
| `schedule.franchise.read` | league-view | G0 | both | read | none | W | shell/pulseReads.ts |
| `schedule.generate` | command:schedule.generate | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts |
| `schedule.publish` | command:schedule.publish | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts |
| `schedule.read` | league-view | G0 | both | read | none | W | shell/pulseReads.ts |
| `score.player.read` | league-view | G0 | both | read | none |  | — |
| `season.rollover` | command:season.rollover | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/HostedWorkflows.tsx, commissioner/actBody.ts, commissioner/actTable.ts, v3/CtrlSeasonV3.tsx |
| `standings.read` | league-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, commissioner/hqReading.ts, shell/pulseReads.ts, shell/warroomTradeReads.ts |
| `stats.queue.read` | statistics.manage | G0 | both | read | none |  | commissioner/actReadings.ts |
| `surface.close` |  |  | — |  |  | B | commands/registry.ts |
| `surface.open` |  |  | — |  |  | B | commands/registry.ts |
| `title.set` | command:title.set | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedControlRoomSeason.tsx, commissioner/actTable.ts, v3/CtrlSeasonV3.tsx |
| `trade.accept` | command:trade.accept | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, phone/MovesTab.tsx, shell/WarRoomDrawers.tsx |
| `trade.accept.plan` |  |  | — |  |  | B | commands/registry.ts, shell/OfferCard.tsx |
| `trade.appeal` | command:trade.appeal | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/actReadings.ts, control/room/guide.ts |
| `trade.appeal.rule` | command:trade.appeal.rule | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, control/room/guide.ts |
| `trade.builder` |  |  | — |  |  | B | commands/registry.ts, shell/EnvelopeRail.tsx, shell/OfferCard.tsx, shell/TradePlan.tsx |
| `trade.decline` | command:trade.decline | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, phone/MovesTab.tsx, shell/WarRoomDrawers.tsx |
| `trade.draft.delete` | franchise-owner | G1 | server | execute | none | W | shell/warroomDrafts.ts |
| `trade.draft.read` | franchise-owner | G0 | server | read | none | W | shell/warroomDrafts.ts |
| `trade.draft.save` | franchise-owner | G1 | server | execute | delete-own-private-proposal | W | shell/warroomDrafts.ts |
| `trade.hold` | command:trade.hold | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, commissioner/actTable.ts |
| `trade.offer` | command:trade.offer | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, shell/warroomTradeReads.ts |
| `trade.override` | command:trade.override | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, control/room/guide.ts |
| `trade.recuse` | command:trade.recuse | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, control/room/guide.ts |
| `trade.review` | command:trade.review | G3 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, control/room/guide.ts |
| `trade.withdraw` | command:trade.withdraw | G1 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, phone/MovesTab.tsx, shell/WarRoomDrawers.tsx |
| `tradelens.read` | franchise-view | G0 | both | read | none |  | shell/warroomLens.ts |
| `trades.read` | franchise-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, phone/MovesTab.tsx, shell/WarRoomDrawers.tsx, shell/homeReads.ts, shell/warroomPlansReads.ts +1 |
| `value.read` | league-view | G0 | both | read | none | W | commissioner/hqReading.ts, shell/warroomInspectorReads.ts, shell/warroomMarketReads.ts |
| `waivers.award.override` | command:waivers.award.override | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, commissioner/actTable.ts, control/room/guide.ts |
| `waivers.claim` | command:waivers.claim | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx |
| `waivers.claim.cancel` | command:waivers.claim.cancel | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, phone/MovesTab.tsx |
| `waivers.claim.order.set` | command:waivers.claim.order.set | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, phone/MovesTab.tsx |
| `waivers.order.auto` | command:waivers.order.auto | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actTable.ts |
| `waivers.order.set` | command:waivers.order.set | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actTable.ts |
| `waivers.process` | command:waivers.process | G2 | both | preview/execute | checked-reversal | W CLI | commissioner/HostedWorkflows.tsx, commissioner/actReadings.ts, commissioner/actTable.ts |
| `waivers.read` | franchise-view | G0 | both | read | none | W | commissioner/hostedRoom.ts, phone/MovesTab.tsx, shell/warroomMarketReads.ts, v3/model.ts |
| `watch.add` |  |  | — |  |  | B | commands/registry.ts, personal/PersonalLists.tsx, personal/WatchButton.tsx, shell/HQInspector.tsx |
| `week.read` | league-view | G0 | both | read | none | W | commissioner/hqReading.ts, shell/pulseReads.ts |
| `whatif.stack.read` | franchise-owner | G0 | server | read | none | W | shell/HQNativeWhatIf.tsx |
| `window.edit` |  |  | — |  |  | B | clock/WindowRow.tsx, commands/registry.ts |
| `window.set` |  |  | — |  |  | B | clock/WindowRow.tsx, commands/registry.ts |

## Appendix B — Desktop mock v3: all 287 edges, re-checked against main

Source rows: `docs/ui/v3/v3-stubs.json` (built 11-17 from `TheWarRoom-mock-v3-TAGGED.html`). *11-17 state*: THIN = real reading, thin UI; PARTIAL; LOCAL = existed only in the deleted desktop/local mode; NEW = nothing behind it; MOCK = mock-only; RULING = needs Christopher; SERVER = server work. *Feeds now*: each command the mock needs → `ui` (exists and a screen calls it), `cmd` (exists, no screen), `NEW` (no command yet), `?` (free-text, not a command id). *Owner file now*: ✓ present on main / ✗ deleted (most ✗ are the old local Control Room removed in desktop removal PR-H; the hosted replacements are `v3/Ctrl*V3.tsx` and `commissioner/HostedControlRoom*.tsx`).

| Edge id | Area | What it does | 11-17 state | Feeds now | Owner file now |
|---|---|---|---|---|---|
| `franchise` | pieces | Record, seed, power and cap room for one league in a strip | THIN | standings.read:ui; odds.read:ui; native.cap.check:ui | ✓ HQHeader.tsx, ✓ WarRoomHeader.tsx |
| `race` | pieces | Seed now, projected seed, playoff odds, next five opponents | THIN | standings.read:ui; schedule.read:ui; odds.read:ui; bracket.read:ui | ✓ PulseRace.tsx, ✓ homeTiles.tsx |
| `docket` | pieces | Everything waiting on you or others, three lanes, across all leagues | PARTIAL | moves.read:ui; trades.read:ui; bids.read:ui; waivers.read:ui; calendar.read:ui; commish.jobs.read:ui; NEW:one:NEW; NEW:private:NEW | ✓ HomeBrief.tsx, ✓ jobs.ts, ✓ EnvelopeRail.tsx |
| `week` | pieces | Every matchup you play this week, all leagues, one plate | PARTIAL | results.read:ui; standings.read:ui; week.read:ui; NEW:cross-league:NEW | ✓ homeTiles.tsx |
| `today` | pieces | Your starters across every league grouped by kickoff | NEW | lineup.read:ui; nfl.games.read:ui; projections.read:ui; NEW:one:NEW | ✓ PulseNow.tsx |
| `cards` | pieces | One card per league: record, seed, next game, waiting count | PARTIAL | standings.read:ui; schedule.read:ui; native.directory.read:ui; commish.leagues.read:cmd | ✓ leaguePicker.tsx, ✓ NativeAccount.tsx |
| `deadlines` | pieces | Next clocks that matter, soonest first; remind toggle | THIN | calendar.read:ui; NEW:reminder:NEW | ✓ ClockStrip.tsx, ✓ CalendarPanel.tsx |
| `cjobs` | pieces | Commissioner jobs across every league you run | LOCAL | commish.jobs.read:ui; stats.queue.read:ui; trades.read:ui | ✓ jobs.ts, ✗ JobsStrip.tsx, ✗ JobsView.tsx |
| `shortcuts` | pieces | Pinned places, one press away | LOCAL | NEW:layout:NEW | ✓ homeTiles.tsx |
| `notes` | pieces | Free-form private scratch pad, autosaves | PARTIAL | NEW:private:NEW | ✓ PersonalLists.tsx |
| `clock` | pieces | Countdown to the next deadline | THIN | calendar.read:ui; week.read:ui | ✓ ClockStrip.tsx, ✓ ticker.ts |
| `matchup` | pieces | Your live game: score, win chance, who is left to play | THIN | results.read:ui; lineup.read:ui; projections.read:ui; odds.read:ui; nfl.games.read:ui | ✓ PulseNow.tsx |
| `board` | pieces | Every matchup in the league this week | THIN | results.read:ui; schedule.read:ui; week.read:ui | ✓ PulseNow.tsx, ✓ PulseSchedule.tsx |
| `nfl` | pieces | NFL scoreboard and the ball on the field | THIN | nfl.games.read:ui; nfl.game.read:ui | ✓ LiveGame.tsx, ✓ NativeLeaguePulse.tsx |
| `pbp` | pieces | Play by play, newest first, with fantasy effect | PARTIAL | nfl.game.read:ui; results.read:ui | ✓ LiveGame.tsx |
| `ingame` | pieces | Your starters in one NFL game | THIN | lineup.read:ui; nfl.game.read:ui | ✓ LiveGame.tsx, ✓ PulseNow.tsx |
| `statchg` | pieces | Live numbers vs final numbers with source | NEW | results.read:ui; stats.queue.read:ui | — |
| `winp` | pieces | Win chance needle | PARTIAL | odds.read:ui | ✓ homeTiles.tsx |
| `heat` | pieces | Live matchups ranked by closeness and time left | NEW | results.read:ui; odds.read:ui; NEW:heat:NEW | — |
| `standings` | pieces | Seeds and records | THIN | standings.read:ui | ✓ PulseRace.tsx |
| `power` | pieces | Power rankings with components | THIN | odds.read:ui; standings.read:ui | ✓ PulseRace.tsx |
| `bracket` | pieces | Seed now/projected and bracket | THIN | bracket.read:ui; odds.read:ui; standings.read:ui | ✓ PulseRace.tsx, ✓ PulseRaceDrawers.tsx |
| `road` | pieces | Your remaining schedule | SERVER | schedule.franchise.read:ui | ✓ PulseSchedule.tsx |
| `form` | pieces | Last five weeks vs projection; click a week | PARTIAL | results.read:ui; projections.read:ui | — |
| `trophy` | pieces | Titles, awards, custom trophies; award one | NEW | NEW:trophies/awards:NEW; NEW:award:NEW | — |
| `archive` | pieces | Seasons, eras, champions | NEW | NEW:league:NEW | — |
| `wire` | pieces | Every transaction as it lands | THIN | commish.ledger.filtered.read:ui; franchise.moves.history.read:ui | ✓ HQMoveHistory.tsx |
| `lineup` | pieces | Starters; drag to change; save | THIN | lineup.read:ui; lineup.check:ui; projections.read:ui; native.reserve.check:ui; commish.lineup.set:ui | ✓ LineupEditor.tsx, ✓ HQBoard.tsx, ✓ hqNativeLineup.tsx, ✓ hqPlacement.ts |
| `roster` | pieces | Everyone you hold with taxi and IR | THIN | native.assets.read:ui; lineup.read:ui; native.reserve.check:ui; roster.reserve:ui; roster.cut:ui | ✓ HQPanels.tsx, ✓ HQRelease.tsx, ✓ HostedLeague.tsx |
| `caprun` | pieces | Cap room this year and next four | THIN | native.cap.check:ui | ✓ HQPanels.tsx |
| `contracts` | pieces | Every deal year by year | PARTIAL | native.assets.read:ui; native.cap.check:ui; value.read:ui; contract.action.set:cmd | ✓ Contracts.tsx, ✓ HQPanels.tsx |
| `expiring` | pieces | Deals that end and the calls you owe (extend/tag/let walk) | SERVER | native.assets.read:ui; calendar.read:ui; contract.action.set:cmd | ✓ HQPanels.tsx |
| `deadcap` | pieces | Money still owed for players gone | PARTIAL | native.cap.check:ui | ✓ HQPanels.tsx |
| `whatif` | pieces | Stack cuts/restructures, watch the cap answer | PARTIAL | contract.quote:ui; native.cap.check:ui; NEW:stacked:NEW | ✓ HQNativeWhatIf.tsx, ✓ WhatIfPanel.tsx |
| `picks` | pieces | Every pick you own; click for detail | THIN | draft.read:ui; native.assets.read:ui | ✓ DraftBoard.tsx |
| `pcard` | pieces | One player, front and back, with a picker | PARTIAL | value.read:ui; projections.read:ui; player.usage.read:ui; player.agearc.read:ui; contract.quote:ui; player.note.read:ui | ✓ HQInspector.tsx, ✓ WarRoomInspector.tsx, ✓ PlayerCard.tsx |
| `offers` | pieces | Offers in and out; accept, decline, withdraw, counter | THIN | trades.read:ui; trade.accept:ui; trade.decline:ui; trade.withdraw:ui | ✓ HostedWorkflows.tsx, ✓ WarRoomDrawers.tsx |
| `desk` | pieces | Trade desk: send/get, two lenses, verdict, tune it | THIN | tradelens.read:ui; value.read:ui; lineupdelta.read:ui; native.cap.check:ui; native.assets.read:ui; trade.offer:ui | ✓ WarRoomTrade.tsx, ✓ warroomTradeReads.ts, ✓ warroomLens.ts |
| `review` | pieces | Trades waiting on your ballot (DOT seat) | THIN | trades.read:ui; trade.review:ui; trade.recuse:ui; trade.appeal:ui | ✓ HostedWorkflows.tsx, ✗ JobsView.tsx |
| `pool` | pieces | Free agents and what they are worth; bid/claim | THIN | native.assets.read:ui; value.read:ui; waivers.read:ui; bids.read:ui; bid.place:ui; waivers.claim:ui; player.watch.set:ui | ✓ WarRoomMarket.tsx, ✓ warroomMarketReads.ts |
| `bids` | pieces | Your open bids; raise, withdraw | THIN | bids.read:ui; bid.raise:ui; bid.withdraw:? | ✓ WarRoomMarket.tsx |
| `waivers` | pieces | Your claims and the order; reorder, cancel | PARTIAL | waivers.read:ui; waivers.claim:ui; NEW:claim:NEW; NEW:claim:NEW | ✓ HostedWorkflows.tsx |
| `draft` | pieces | Draft clock and board; make the pick | THIN | draft.read:ui; draft.pick:ui; draft.queue.read:ui; draft.queue.set:ui | ✓ DraftRoom.tsx, ✓ WarRoomDraft.tsx |
| `plans` | pieces | Private plans on the calendar | NEW | calendar.read:ui; NEW:plans:NEW | ✓ WarRoomPlans.tsx |
| `looking` | pieces | Past offers and partners | THIN | trades.read:ui; moves.read:ui | ✓ WarRoomPlans.tsx |
| `nflnews` | pieces | News on your players, filters | NEW | NEW:hosted:NEW; practice:?; depth):? | — |
| `headlines` | pieces | What happened in your league this week | NEW | NEW:week:NEW | — |
| `depth` | pieces | NFL depth charts coloured by owner | NEW | NEW:depth:NEW; native.assets.read:ui | — |
| `nflstats` | pieces | Stats scored with your league rules, filter and sort | NEW | NEW:league-scored:NEW | — |
| `describe` | pieces | Say what you want; see what would change | PARTIAL | commish.rules.read:ui; commish.controls.read:cmd; commish.rules.set:ui; NEW:AI:NEW | ✗ RuleChangeForm.tsx, ✓ ruleDraft.ts |
| `house` | pieces | House rules in plain words with firing log | PARTIAL | native.rules.read:cmd; commish.rules.set:ui; NEW:house-rule:NEW | ✗ RulesView.tsx |
| `layout` | pieces | Every screen/piece the league gets, and why; set on/off/auto | NEW | NEW:league:NEW | — |
| `statq` | pieces | Stat corrections waiting for yes/no | LOCAL | stats.queue.read:ui; commish.stat.queue.approve:ui; commish.stat.queue.reject:ui | ✗ openers.tsx, ✗ FixView.tsx |
| `rules` | pieces | Rulebook, editable, with preset compare | THIN | commish.rules.read:ui; commish.rules.preset.read:cmd; commish.rules.set:ui; rule.exception:ui | ✓ HostedRules.tsx, ✗ RulesView.tsx, ✗ PresetCompare.tsx |
| `season` | pieces | Windows of the season; move them | LOCAL | calendar.read:ui; commish.schedule.read:ui; calendar.window.move:ui; calendar.window.set:cmd; calendar.window.remove:cmd | ✗ SeasonView.tsx |
| `fix` | pieces | Recent fixes and exceptions in force | LOCAL | commish.corrections.read:ui; commish.ledger.read:ui; commish.correction.revert:ui | ✗ FixView.tsx, ✗ UndoForm.tsx |
| `people` | pieces | Who sits where and when last seen; seat tools | LOCAL | commish.people.read:ui; commish.seat.invite:ui; commish.seat.replace:ui; commish.seat.remove:ui; commish.seat.caretaker:ui; commish.role.grant:ui; commish.role.revoke:ui | ✗ PeopleView.tsx, ✗ LeaguePeople.tsx |
| `ledger` | pieces | The record of everything, filterable | LOCAL | commish.ledger.read:ui; commish.ledger.filtered.read:ui | ✗ LeagueLedger.tsx, ✗ LedgerFilters.tsx |
| `notices` | pieces | What you told the league; publish | LOCAL | commish.notices.read:ui; commish.notice.publish:ui | ✗ NoteForm.tsx, ✓ notes.ts |
| `health` | pieces | Stalled matters, quiet owners, orphan teams | LOCAL | commish.people.read:ui; commish.jobs.read:ui | ✓ jobs.ts, ✓ people.ts |
| `server` | pieces | Backups, feeds, upgrades | LOCAL | NEW:server:NEW | ✗ ServerView.tsx, ✗ BackupNow.tsx |
| `hq:lineup` | screens | Lineup and roster: starters/bench, drag to set, live game, win chance, form, taxi/IR | THIN | lineup.read:ui; lineup.check:ui; projections.read:ui; native.reserve.check:ui; commish.lineup.set:ui; roster.reserve:ui | ✓ NativeFranchiseHQ.tsx |
| `hq:contracts` | screens | Contracts and cap: deals by year, runway, decisions, release/restructure/extend forms | PARTIAL | native.cap.check:ui; native.assets.read:ui; contract.quote:ui; value.read:ui; roster.cut:ui; contract.action.set:cmd | ✓ HQPanels.tsx, ✓ HQRelease.tsx, ✓ HQNativeWhatIf.tsx |
| `hq:moves` | screens | My moves: what you did and what is pending | THIN | franchise.moves.history.read:ui; moves.read:ui | ✓ HQMoveHistory.tsx, ✓ HQMoves.tsx |
| `war:pool` | screens | Valuations and pool: free agents, targets, bids, budget split, recent winning bids | THIN | native.assets.read:ui; value.read:ui; bids.read:ui; waivers.read:ui; player.watch.read:ui; player.watch.order:ui; bid.place:ui; bid.raise:ui; bid.withdraw:?; NEW:recent:NEW | ✓ WarRoomMarket.tsx |
| `war:trade` | screens | Trade desk: shelves, ledger, two lenses, verdict, tune, versions, pocket, negotiations | THIN | tradelens.read:ui; value.read:ui; lineupdelta.read:ui; native.cap.check:ui; trades.read:ui; trade.offer:ui; NEW:saved:NEW | ✓ WarRoomTrade.tsx, ✓ WarRoomDrawers.tsx |
| `war:draft` | screens | Draft room: queue, scouting card, make the pick | THIN | draft.read:ui; draft.queue.read:ui; draft.queue.set:ui; draft.pick:ui; NEW:ADP/tiers:NEW; NEW:scouting:NEW | ✓ DraftRoom.tsx, ✓ WarRoomDraft.tsx |
| `war:plans` | screens | Plans: private plans, steps on the calendar, past offers | PARTIAL | calendar.read:ui; trades.read:ui; NEW:plans:NEW | ✓ WarRoomPlans.tsx |
| `pulse:now` | screens | Now: league scoreboard, your matchup, NFL field, plays, stat pulls | THIN | results.read:ui; lineup.read:ui; odds.read:ui; nfl.games.read:ui; nfl.game.read:ui; week.read:ui | ✓ NativeLeaguePulse.tsx, ✓ LiveGame.tsx |
| `pulse:race` | screens | Race: standings, power + why it moved, tiebreaks, bracket | THIN | standings.read:ui; odds.read:ui; bracket.read:ui; schedule.read:ui | ✓ PulseRace.tsx, ✓ PulseRaceDrawers.tsx |
| `pulse:archive` | screens | Archive: seasons, champions, trophy case | NEW | NEW:archive:NEW; NEW:trophies:NEW | ✓ nodes.ts |
| `control:jobs` | screens | Jobs: morphs by clock; lanes NOW/HOLDING/TODAY/WEEK/LATER; open the tool that settles it | LOCAL | commish.jobs.read:ui; stats.queue.read:ui; trades.read:ui; calendar.read:ui | ✗ JobsView.tsx, ✓ jobs.ts |
| `control:situations` | screens | Situations: 12 stories, each walks the tools that settle it | LOCAL | guide:? | ✗ SituationsView.tsx, ✓ guide.ts |
| `control:rules` | screens | Rules: settings board with source tags, preset compare, house rules | PARTIAL | commish.rules.read:ui; commish.rules.preset.read:cmd; native.rules.read:cmd; commish.rules.set:ui; rule.exception:ui | ✗ RulesView.tsx, ✓ HostedRules.tsx |
| `control:season` | screens | Season: windows, schedule, bracket, title, rollover | LOCAL | calendar.read:ui; commish.schedule.read:ui; schedule.edit:ui; schedule.generate:ui; schedule.publish:ui; bracket.build:ui; bracket.edit:ui; title.set:ui; season.rollover:ui; calendar.window.move:ui | ✗ SeasonView.tsx |
| `control:fix` | screens | Fix: correction forms with preview, reason, safeguard; undo | LOCAL | commish.corrections.read:ui; correct.roster:ui; correct.transaction:ui; correct.pick:ui; commish.stat.correct:ui; commish.correction.revert:ui | ✗ FixView.tsx, ✓ actTable.ts |
| `control:picks` | screens | Picks: all picks at 32 teams, filters; grant, swap, forfeit, condition, void, create | LOCAL | draft.read:ui; native.assets.read:ui; correct.pick:ui; draft.order.set:ui; NEW:pick:NEW; not:? | ✗ PicksView.tsx |
| `control:people` | screens | People: seats, deputies, permissions, succession | LOCAL | commish.people.read:ui; commish.role.grant:ui; commish.role.revoke:ui; commish.abilities.set:ui; commish.deputy.order:ui; commish.succession.*:?; commish.seat.*:? | ✗ PeopleView.tsx |
| `control:league` | screens | League: identity, divisions, layout, colour, ledger | LOCAL | commish.rules.read:ui; commish.franchise.edit:ui; commish.ledger.filtered.read:ui; NEW:league:NEW; NEW:league:NEW | ✗ LeagueView.tsx |
| `control:server` | screens | Server: backups, feeds, upgrades, metrics | LOCAL | NEW:commissioner:NEW; NEW:feed:NEW; NEW:restore:NEW | ✗ ServerView.tsx |
| `shell:home-boards` | shell | App Home above all leagues: boards (tabs), free grid of 61 pieces, edit, stack, full, undo | NEW | NEW:layout:NEW | ✓ HomeBrief.tsx, ✓ homeLayout.ts |
| `shell:rail` | shell | Sidebar: app zone (Home + boards), my shortcuts, league zone with picker and screens | LOCAL | commish.leagues.read:?; NEW:per-league:NEW | ✗ ShellRail.tsx, ✓ leaguePicker.tsx |
| `shell:league-chip` | shell | League switcher grouped NEEDS YOU / LIVE NOW / EVERYTHING ELSE | PARTIAL | NEW:per-league:NEW | ✓ leaguePicker.tsx |
| `shell:search` | shell | Search or type a command (Ctrl K) | LOCAL | NEW:search:NEW | ✗ CommandBar.tsx, ✓ search.ts, ✓ commandbar.gen.ts |
| `shell:needs-you` | shell | Needs you count and Docket drawer | PARTIAL | see:? | ✓ HomeBrief.tsx |
| `shell:alert-band` | shell | NOW band: top alert with its one action, cycle | PARTIAL | NEW:alert:NEW | ✓ HomeBrief.tsx |
| `shell:assistant` | shell | Assistant drawer: scripted requests that build boards or settle jobs | NEW | NEW:in-app:NEW | ✓ NativeAccount.tsx |
| `shell:feeds` | shell | Feed tags, peek strip, feed index (mock-only wiring view) | MOCK | —:? | ✓ EndpointIndex.tsx |
| `shell:account` | shell | YO menu: labels, AI key, devices, sign out | THIN | account:? | ✓ NativeAccount.tsx |
| `shell:status-bar` | shell | CONNECTED · LIVE n s behind · week clock; each opens its panel | PARTIAL | NEW:per-league:NEW; UI:spoiler:? | ✗ StatusStrip.tsx, ✓ freshness.ts |
| `shell:inspector` | shell | Right panel: full player/asset card | THIN | value.read:ui; projections.read:ui; contract.quote:ui; player.usage.read:ui; player.agearc.read:ui; player.note.read:ui | ✓ HQInspector.tsx, ✓ WarRoomInspector.tsx, ✓ PlayerInspector.tsx |
| `shell:popout` | shell | Pop a piece into its own window; keep live score on top | NEW | same:? | — |
| `shell:cards` | shell | Click any player/team/club/transaction name: popover card | PARTIAL | see:? | ✓ EntityInspector.tsx |
| `shell:tour` | shell | First-login tour and labels toggle | NEW | —:? | — |
| `shell:mock-controls` | shell | MOCK CONTROLS tab: moment, league count, sample switches | MOCK | —:? | — |
| `closeLayer` | NAV | Close the top popover/drawer, focus returns | THIN | —:? | — |
| `edit` | shell:home-boards | Toggle Home edit mode | LOCAL | —:? | ✓ homeLayout.ts |
| `undo` | shell:home-boards | Undo last layout change | NEW | —:? | — |
| `redo` | shell:home-boards | Redo layout change | NEW | —:? | — |
| `board` | shell:home-boards | Switch Home board | NEW | —:? | — |
| `boardAdd` | shell:home-boards | Open starter-board menu | NEW | —:? | — |
| `boardNew` | shell:home-boards | Add a starter board (or switch to it if open) | NEW | —:? | — |
| `boardMenu` | shell:home-boards | Board menu: rename, duplicate, publish, reset, close | NEW | —:? | — |
| `bm` | shell:home-boards | Run a board-menu choice; 'publish to league' sends the board to the league | NEW | NEW:publish:NEW | — |
| `boardReset` | shell:home-boards | Reset board to its starter | LOCAL | —:? | ✓ homeLayout.ts |
| `boardClose` | shell:home-boards | Close a board with Undo | NEW | —:? | — |
| `boardMore` | shell:home-boards | Overflow menu for boards that do not fit | NEW | —:? | — |
| `pset` | shell:home-boards | Change a piece setting (filter, sort, size) | NEW | —:? | — |
| `pflip` | shell:home-boards | Flip a piece to its back side | NEW | —:? | — |
| `stk` | shell:home-boards | Switch tab inside a stacked piece | NEW | —:? | — |
| `premove` | shell:home-boards | Remove a piece | LOCAL | —:? | ✓ homeLayout.ts |
| `pgear` | shell:home-boards | Open piece settings in the inspector | NEW | —:? | — |
| `pmenu` | shell:home-boards | Piece menu: bind league, size, full, pop out, keep on top | NEW | —:? | — |
| `pm` | shell:home-boards | Run a piece-menu choice (full, pop out, bind pin/follow/all) | NEW | —:? | — |
| `fullOpt` | shell:home-boards | Library choice: stack or place a piece | NEW | —:? | — |
| `pvApply` | shell:home-boards | Apply a previewed layout (from assistant or library) | NEW | —:? | — |
| `pvCancel` | shell:home-boards | Cancel layout preview | NEW | —:? | — |
| `bringBack` | shell:popout | Bring a popped-out piece back | NEW | —:? | — |
| `home` | NAV | Go Home | LOCAL | —:? | — |
| `nav` | NAV | Go to a screen tab | THIN | —:? | ✓ browser.gen.ts |
| `railBoard` | shell:home-boards | Open a board from the sidebar | NEW | —:? | — |
| `railCol` | NAV | Collapse/expand the sidebar | LOCAL | —:? | ✗ ShellRail.tsx |
| `pickLeague` | NAV | Select a league (Follow pieces switch) | LOCAL | —:? | ✓ leaguePicker.tsx |
| `goLeague` | NAV | Open a league's screen | LOCAL | —:? | ✓ leaguePicker.tsx |
| `star` | NAV | Pin a tab to My shortcuts | LOCAL | NEW:shortcut:NEW | ✓ homeTiles.tsx |
| `inspect` | shell:inspector | Open the player in the right panel | THIN | value.read:ui; projections.read:ui; contract.quote:ui; player.usage.read:ui; player.agearc.read:ui; player.note.read:ui | ✓ HQInspector.tsx, ✓ WarRoomInspector.tsx |
| `inspClose` | shell:inspector | Close the right panel | THIN | —:? | — |
| `pmSel` | shell:home-boards | Piece-menu choice from the inspector | NEW | —:? | — |
| `say` | control:people | Commissioner/account action box by type (approve, reject, vote, fix, rule, seat, revert, retry, devices, sign out…) | LOCAL | commish.stat.queue.approve:ui; commish.stat.queue.reject:ui; trade.review:ui; commish.rules.set:ui; commish.correction.revert:ui; correct.pick:ui | ✓ CommissionerActTray.tsx, ✓ ActFields.tsx |
| `feeds` | MOCK | Show/hide feed tags (mock wiring view) | MOCK | —:? | — |
| `switcher` | shell:league-chip | League switcher with groups | PARTIAL | commish.leagues.read:cmd; NEW:per-league:NEW | ✓ leaguePicker.tsx |
| `swPick` | shell:home-boards | Pick a league in the switcher (or pin a piece to it) | LOCAL | —:? | ✓ leaguePicker.tsx |
| `account` | shell:account | Account menu | THIN | —:? | ✓ NativeAccount.tsx |
| `noop` | MOCK | Nothing (rocker label) | MOCK | —:? | — |
| `acct` | shell:account | Account menu choice: labels, AI provider key, devices, sign out | PARTIAL | account:?; device:?; NEW:AI:NEW | ✓ NativeAccount.tsx |
| `map` | shell:tour | Help map of the screen | NEW | —:? | — |
| `cmdk` | shell:search | Command palette: screens, commands, players | LOCAL | NEW:search:NEW | ✗ CommandBar.tsx, ✓ search.ts |
| `cmdRun` | NAV | Run a palette result | LOCAL | —:? | ✓ commandbar.gen.ts |
| `docket` | p:docket | Open the Docket drawer, three lanes | PARTIAL | see:? | ✓ HomeBrief.tsx |
| `dkLane` | p:docket | Switch Docket lane | NEW | —:? | — |
| `dkLg` | p:docket | Docket: this league only / all | NEW | —:? | — |
| `dkAdd` | shell:home-boards | Add the Docket as a Home piece | NEW | —:? | — |
| `dgo` | p:docket | Open a Docket item's place and mark it read | NEW | NEW:private:NEW | — |
| `library` | shell:home-boards | Open the piece library (greys pieces a league cannot use, with reason) | LOCAL | commish.rules.read:? | ✓ homeLayout.ts |
| `assistant` | shell:assistant | Open the assistant drawer | NEW | NEW:in-app:NEW | — |
| `askSend` | shell:assistant | Send typed request to the assistant | NEW | NEW:in-app:NEW | — |
| `ask` | shell:assistant | Send a suggested request | NEW | NEW:in-app:NEW | — |
| `asstApply` | shell:home-boards | Apply what the assistant prepared (board, lineup, pop out) | NEW | —:? | — |
| `feedIndex` | MOCK | Feed index drawer (mock wiring view) | MOCK | —:? | ✓ EndpointIndex.tsx |
| `peek` | MOCK | Peek a feed tag (mock wiring view) | MOCK | —:? | — |
| `peekClose` | MOCK | Close peek | MOCK | —:? | — |
| `demo` | MOCK | Mock controls panel | MOCK | —:? | — |
| `dset` | MOCK | Mock control switch (moment, league count) | MOCK | —:? | — |
| `notesBuild` | MOCK | 'Notes for the real build' drawer | MOCK | —:? | — |
| `tourStart` | shell:tour | Start first-login tour | NEW | —:? | — |
| `tourNext` | shell:tour | Next tour step | NEW | —:? | — |
| `tourSkip` | shell:tour | Skip tour | NEW | —:? | — |
| `etab` | NAV | Switch an inner tab on a screen | THIN | —:? | — |
| `pcol` | NAV | Collapse a screen panel | NEW | —:? | — |
| `sset` | NAV | Change a screen panel setting | NEW | —:? | — |
| `spmenu` | shell:home-boards | Screen panel menu (send to Home, pop out) | NEW | —:? | — |
| `spm` | shell:home-boards | Run panel menu choice | NEW | —:? | — |
| `pSel` | pulse:now | Pulse: select which matchup follows | THIN | —:? | — |
| `pull` | pulse:now | Pulse drawer for a matchup's stat pulls | PARTIAL | results.read:ui; nfl.game.read:ui | ✓ PulseRaceDrawers.tsx |
| `hqAct` | p:lineup | Lineup: reset / review / save | THIN | lineup.check:ui; commish.lineup.set:ui | ✓ hqNativeLineup.tsx, ✓ LineupEditor.tsx |
| `jobOpen` | control:jobs | Open a job drawer (old v2 jobs) | LOCAL | commish.jobs.read:ui | ✗ JobsView.tsx |
| `jobDone` | control:jobs | Mark a job settled | LOCAL | the:? | ✗ JobsView.tsx |
| `cmTry` | control:rules | Describe-your-league: preview a proposal | LOCAL | commish.rules.read:ui | ✓ useChangePreview.ts |
| `cmCancel` | control:rules | Cancel proposal | LOCAL | —:? | — |
| `cmAsk` | shell:assistant | Describe-your-league: ask in words | NEW | NEW:in-app:NEW | — |
| `cmApply` | control:rules | Apply the proposal | THIN | commish.rules.set:ui | ✗ RuleChangeForm.tsx |
| `asstCm` | shell:assistant | Apply a commissioner request from the assistant | NEW | commish.rules.set:ui | — |
| `surf` | control:league | League layout: set a surface on/off/auto | NEW | NEW:league:NEW | — |
| `lcolor` | control:league | Change league identity colour | NEW | NEW:league:NEW | — |
| `trDir` | war:trade | Trade desk: lens direction | THIN | —:? | ✓ WarRoomTrade.tsx |
| `trFlt` | war:trade | Trade desk: shelf filter | THIN | —:? | ✓ WarRoomTrade.tsx |
| `trVis` | war:trade | Trade desk: open or private offer | THIN | —:? | — |
| `trVer` | war:trade | Trade desk: switch/add version | PARTIAL | NEW:saved:NEW | — |
| `trClear` | war:trade | Clear the deal | THIN | —:? | ✓ tradeBuilderState.ts |
| `trTune` | war:trade | Apply a tune-it suggestion | THIN | tradelens.read:? | ✓ warroomLens.ts |
| `trInspect` | shell:inspector | Inspect an asset on the desk | THIN | value.read:ui; contract.quote:ui | ✓ WarRoomInspector.tsx |
| `trToggle` | war:trade | Add/remove an asset to the deal | THIN | —:? | ✓ warroomDrag.ts, ✓ tradeBuilderState.ts |
| `trSave` | war:trade | Save deal to pocket | PARTIAL | NEW:saved:NEW | — |
| `trSend` | war:trade | Send the offer | THIN | trade.offer:ui | ✓ warroomTradeReads.ts |
| `trLoad` | war:trade | Load a saved deal or a live thread | PARTIAL | trades.read:ui; NEW:saved:NEW | ✓ WarRoomDrawers.tsx |
| `trPocket` | war:trade | Pocket sheet: saved deals re-read | PARTIAL | NEW:saved:NEW | — |
| `trNeg` | war:trade | Negotiations sheet: offers in motion | THIN | trades.read:ui | ✓ WarRoomDrawers.tsx |
| `stConn` | shell:status-bar | League servers drawer: one row per server, ping | NEW | NEW:per-server:NEW | ✓ freshness.ts |
| `stLive` | shell:status-bar | Spoiler delay menu | NEW | —:? | — |
| `stDelay` | shell:status-bar | Set spoiler delay (hold live scores back) | NEW | —:? | — |
| `stWeek` | NAV | Week clock → Season calendar | LOCAL | —:? | — |
| `ent` | shell:cards | Popover card for a player, fantasy team, NFL club or transaction | PARTIAL | value.read:ui; projections.read:ui; standings.read:ui; nfl.game.read:ui; commish.ledger.filtered.read:ui; NEW:status:NEW | ✓ EntityInspector.tsx, ✓ PlayerCard.tsx |
| `eGo` | NAV | Card button: go to lineup/standings/game | THIN | —:? | — |
| `eTrade` | NAV | Card button: open trade desk with player | THIN | —:? | — |
| `eWatch` | p:pool | Watch/unwatch a player | THIN | player.watch.set:ui | ✓ playerPersonal.ts, ✓ WatchButton.tsx |
| `eDepth` | p:depth | Depth chart for a club | NEW | NEW:depth:NEW | — |
| `eBid` | p:bids | Bid/claim form from a card | THIN | bids.read:ui; native.cap.check:ui; waivers.read:ui | ✓ WarRoomMarket.tsx |
| `eBidGo` | p:bids | Place the bid | THIN | bid.place:ui; bid.raise:ui | ✓ warroomMarketReads.ts |
| `bidStep` | p:bids | Raise or lower an open bid | THIN | bid.raise:ui | — |
| `bidDrop` | p:bids | Withdraw a bid with Undo | THIN | bid.withdraw:? | — |
| `clMove` | p:waivers | Reorder waiver claims | NEW | NEW:claim:NEW | — |
| `clDrop` | p:waivers | Cancel a waiver claim with Undo | NEW | NEW:claim:NEW | — |
| `offAcc` | p:offers | Accept a trade offer (confirm) | THIN | trade.accept:ui | — |
| `offDec` | p:offers | Decline with optional note | THIN | trade.decline:ui; NEW:decline:NEW | — |
| `offWd` | p:offers | Withdraw your offer | THIN | trade.withdraw:ui | — |
| `offSet` | p:offers | Confirm accept/decline | THIN | trade.accept:ui; trade.decline:ui | — |
| `offCtr` | p:offers | Counter: put their offer on the desk | NEW | —:? | — |
| `decide` | p:expiring | Expiring deal choices: extend, tag, let walk | SERVER | native.assets.read:ui; contract.quote:ui | — |
| `decSet` | p:expiring | Record a contract decision | SERVER | contract.action.set:cmd | — |
| `decUndo` | p:expiring | Undo a contract decision | SERVER | contract.action.set:cmd | — |
| `wifAdd` | p:whatif | What-if: add a cut/restructure to the stack | PARTIAL | contract.quote:ui; native.cap.check:ui; NEW:stacked:NEW | ✓ HQNativeWhatIf.tsx |
| `wifDel` | p:whatif | What-if: remove a move | PARTIAL | —:? | — |
| `wifReset` | p:whatif | What-if: clear | PARTIAL | —:? | — |
| `pick` | p:picks | Pick detail popover (projected slot, from whom) | THIN | draft.read:ui | ✓ EntityInspector.tsx |
| `dlGo` | NAV | Deadline row → open what is due | THIN | —:? | ✓ deadlineRoute.ts |
| `remind` | p:deadlines | Set a reminder 15 min before a deadline | NEW | NEW:private:NEW | — |
| `cmOk` | control:people | Confirm a commissioner action box (seat change needs a reason) | LOCAL | commish.seat.replace:ui; commish.seat.remove:ui; commish.seat.caretaker:ui | ✗ openers.tsx |
| `devOut` | shell:account | Sign out a device with Undo | THIN | account:? | ✓ NativeAccount.tsx |
| `seatGo` | control:people | Seat tool: invite link, caretaker, replace, remove | LOCAL | commish.seat.invite:ui; commish.seat.caretaker:ui; commish.seat.replace:ui; commish.seat.remove:ui | ✗ openers.tsx, ✓ join.go |
| `copyLink` | control:people | Copy the invite link | NEW | —:? | — |
| `cmDone` | control:people | Finish invite | LOCAL | commish.seat.invite:ui | ✗ openers.tsx |
| `signin` | MOCK | Mock sign in after sign-out | MOCK | —:? | ✓ NativeAccount.tsx |
| `healthGo` | NAV | League health row → its tool | LOCAL | —:? | — |
| `formWk` | p:form | Form: open one week's result | PARTIAL | results.read:ui | — |
| `tOk` | FORM | Submit the open tool form | LOCAL | —:? | ✓ ActFields.tsx |
| `cmGo` | p:contracts | Release / restructure / extend form with cap quote | PARTIAL | contract.quote:ui; native.cap.check:ui; roster.cut:ui; contract.action.set:cmd | ✓ HQRelease.tsx |
| `cmUndo` | p:contracts | Undo a release/restructure | RULING | NEW:owner:NEW | — |
| `pBid` | p:bids | Pool: bid form with budget left | THIN | bids.read:ui; native.cap.check:ui; bid.place:ui; bid.raise:ui | ✓ WarRoomMarket.tsx |
| `dqAdd` | war:draft | Queue a prospect | THIN | draft.queue.set:ui | ✓ warroomDraftReads.ts |
| `dqDel` | war:draft | Remove from queue | THIN | draft.queue.set:ui | ✓ warroomDraftReads.ts |
| `dqMove` | war:draft | Reorder queue | THIN | draft.queue.set:ui | ✓ warroomDraftReads.ts |
| `scout` | war:draft | Scouting card for a prospect | PARTIAL | draft.read:ui; NEW:scouting:NEW; NEW:ADP/tiers:NEW | ✓ WarRoomScouting.tsx |
| `dPick` | war:draft | Make the pick (final) | THIN | draft.pick:ui | ✓ YourPick.tsx |
| `whyMoved` | pulse:race | Power: why each team moved | PARTIAL | odds.read:ui; NEW:why-moved:NEW | ✓ PulseRaceDrawers.tsx |
| `tieBreak` | pulse:race | Tiebreak order and which step decided | PARTIAL | standings.read:ui; NEW:tiebreak:NEW | ✓ PulseRaceDrawers.tsx |
| `brkFull` | pulse:race | Full bracket drawer | THIN | bracket.read:ui; odds.read:ui | ✓ PulseRace.tsx |
| `trAward` | p:trophy | Award a trophy | NEW | NEW:award:NEW | — |
| `preCmp` | control:rules | Compare your league to presets | LOCAL | commish.rules.preset.read:cmd | ✗ PresetCompare.tsx |
| `preApply` | control:rules | Apply a preset column | LOCAL | commish.rules.set:ui | ✗ PresetCompare.tsx |
| `houseNew` | control:rules | Write a new house rule | PARTIAL | commish.rules.set:?; NEW:house-rule:NEW | ✗ RulesView.tsx |
| `schedTool` | control:season | Schedule drawer weeks 12–14 | LOCAL | commish.schedule.read:ui | ✗ SeasonView.tsx |
| `schedSwap` | control:season | Swap a matchup | LOCAL | schedule.edit:ui | ✗ SeasonView.tsx |
| `brkTool` | control:season | Bracket settings: teams, reseed | LOCAL | bracket.build:ui; bracket.edit:ui; bracket.preset.save:ui | ✗ SeasonView.tsx |
| `titleTool` | control:season | Crown a champion by ruling | LOCAL | bracket.advance:ui; commish.standings.override:ui | ✗ SeasonView.tsx |
| `rollTool` | control:season | Rollover preview | THIN | season.rollover:? | ✓ CommissionerActTray.tsx |
| `rollRun` | control:season | Run rollover (hold to fire) | THIN | season.rollover:ui | ✓ HoldToFire.tsx |
| `pkF` | control:picks | Picks filter | LOCAL | —:? | ✗ PicksView.tsx |
| `pkMenu` | control:picks | Pick menu | LOCAL | —:? | ✗ PicksView.tsx |
| `pkDo` | control:picks | Grant/swap/forfeit/condition/void/restore a pick | PARTIAL | correct.pick:ui; NEW:pick:NEW | ✗ PicksView.tsx |
| `pkNew` | control:picks | Create a pick | NEW | NEW:create:NEW | — |
| `permT` | control:people | Toggle a deputy permission | LOCAL | commish.abilities.set:ui | ✗ PeopleView.tsx |
| `depAdd` | control:people | Add a deputy with permissions | LOCAL | commish.role.grant:ui; commish.abilities.set:ui | ✗ PeopleView.tsx |
| `depDel` | control:people | Remove a deputy | LOCAL | commish.role.revoke:ui | ✗ PeopleView.tsx |
| `sucMove` | control:people | Reorder the succession line | LOCAL | commish.deputy.order:ui | ✗ PeopleView.tsx |
| `bkNow` | control:server | Take a backup now | LOCAL | backup.now:?; NEW:hosted:NEW | ✗ BackupNow.tsx |
| `bkRestore` | control:server | Restore preview | RULING | NEW:backup:NEW | — |
| `bkRun` | control:server | Restore a backup | RULING | NEW:restore:NEW | — |
| `feedT` | control:server | Pause/resume a feed | RULING | NEW:feed:NEW | — |
| `feedSw` | control:server | Switch live scoring to the backup feed | RULING | NEW:truth-ladder:NEW | — |
| `upChk` | control:server | Check for a server update | PARTIAL | NEW:update:NEW | ✓ UpdateRequired.tsx |
| `upRun` | control:server | Install server upgrade | RULING | NEW:upgrade:NEW | — |
| `hJob` | NAV | Health row → open its job tool | LOCAL | —:? | — |
| `cjFilter` | control:jobs | Jobs filter chips | LOCAL | —:? | ✗ JobsView.tsx |
| `cjHome` | NAV | Cross-league jobs → that league's Jobs | LOCAL | —:? | — |
| `cjReopen` | control:jobs | Reopen a settled job | NEW | —:? | — |
| `cjSnooze` | control:jobs | Move a job back a day | NEW | NEW:job:NEW | — |
| `cjOpen` | control:jobs | Open a job drawer | LOCAL | commish.jobs.read:ui | ✗ JobsView.tsx |
| `cjGo` | NAV | Job → open its tool | LOCAL | —:? | ✗ openers.tsx |
| `cjMark` | control:jobs | Mark a job settled with a note | LOCAL | the:?; NEW:settle:NEW | ✗ JobsView.tsx |
| `cjTrade` | control:jobs | Trade review drawer: both sides, ballots | LOCAL | trades.read:ui | ✗ JobsView.tsx |
| `cjTx` | control:jobs | Approve / veto / override / extend review | PARTIAL | trade.review:ui; trade.override:ui; NEW:extend:NEW | ✓ HostedWorkflows.tsx |
| `cjStat` | control:jobs | Stat corrections drawer | LOCAL | stats.queue.read:ui | ✗ openers.tsx |
| `cjSq` | control:jobs | Approve/reject one or all corrections | LOCAL | commish.stat.queue.approve:ui; commish.stat.queue.reject:ui | ✗ openers.tsx |
| `cjTie` | control:jobs | Settle a blind-bid tie | LOCAL | bids.award.override:ui; waivers.award.override:ui | ✗ SeasonView.tsx |
| `cjLine` | control:jobs | Lineups with a hole drawer | NEW | NEW:league-wide:NEW | — |
| `cjLn` | control:jobs | Set a lineup for an owner or nudge in app | PARTIAL | commish.lineup.set:ui; commish.notice.publish:ui | ✓ HostedLeague.tsx |
| `cjCap` | control:jobs | Over-the-cap ruling | LOCAL | commish.cap.relief:ui; rule.exception:ui; commish.notice.publish:ui | ✓ guide.ts |
| `cjNotice` | control:jobs | Publish a notice | LOCAL | commish.notice.publish:ui | ✗ NoteForm.tsx |
| `cjOrder` | control:jobs | Set next year's draft order | LOCAL | draft.order.set:ui; draft.order.auto:ui; draft.order.pattern:ui | ✓ actTable.ts |
| `askC` | shell:assistant | Ask bar on a Control Room tab | NEW | NEW:in-app:NEW | — |
| `askGo` | NAV | Assistant routes to a tool | LOCAL | —:? | ✓ search.ts |
| `askKeep` | shell:assistant | Keep a request in writing (becomes a job) | NEW | NEW:written:NEW | — |
| `askHouse` | shell:assistant | Turn a request into a house rule | NEW | commish.rules.set:ui | — |
| `cfgEdit` | control:rules | Edit one setting with timing and reason | THIN | commish.rules.set:ui | ✗ RuleChangeForm.tsx, ✓ RuleInput.tsx |
| `divTool` | control:rules | Divisions drawer | LOCAL | commish.rules.read:ui | ✗ LeagueView.tsx |
| `divMove` | control:rules | Move a team to another division | RULING | commish.rules.set:?; NEW:if:NEW | — |
| `pfS` | control:people | People filter | LOCAL | —:? | ✗ PeopleView.tsx |
