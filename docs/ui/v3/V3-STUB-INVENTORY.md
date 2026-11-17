# TheWarRoom desktop mock v3 — stub inventory

**Written:** 2026-11-17 by Tom (Claude on the Beelink). **Mock:** `~/Downloads/TheWarRoom-mock-v3-FINAL.html` (sha256 c7663b1a…2292, unchanged). **Real app:** TheWarRoom `origin/main` at `64e0507fa` (2026-11-17 4:43 PM CT). **Command catalog:** 158 command names on that commit (`internal/commandcatalog`, `cmd/twr/verbs.gen.go`, `frontend/src/app/commands/native.gen.ts`).

**On screen:** `~/Downloads/TheWarRoom-mock-v3-TAGGED.html`. Press **Feeds** (top right). Every control gets a tag, the bottom bar says what the thing under the mouse does and what it needs, and **INDEX** lists all 287 stubs with filters.

## How this was counted
- The mock was opened in a headless browser and its own tables were listed: **192 actions** (every `data-a` handler), **61 pieces**, **19 screen tabs**. Plus **15 shell parts** (top bar, rail, status bar, cards, assistant, pop out, tour, mock controls). Total **287 stubs**. A script checks that no action is missing and none is extra.
- Every data name below was checked by script against the 158 catalog names on `main`. Anything not in the catalog is written **NEW**.
- Every "real code" file named was checked to exist on `main`.

## Words used
| Word | Means |
|---|---|
| **Stub** | One thing in the mock that acts or shows data: a piece, a screen tab, a button/handler, or a shell part. |
| **Kind: UI** | Works with no data at all (layout, filters, navigation, menus). |
| **Kind: UI·S** | No league data, but saved for the person (boards, pins, preferences). Needs a per-person store to follow you across devices. |
| **Kind: IN** | Reads data in from the league server. |
| **Kind: OUT** | Sends a command out (changes something). |
| **Kind: IN+OUT** | Reads and sends. |
| **Kind: MOCK** | Exists only to demo the mock. Not in the real app. |
| **State: Hosted** | Built and wired in the thin (hosted) desktop app on `main` today. |
| **State: Partly** | Some of it is on `main`; the row says what is missing. |
| **State: Local only** | Built on `main`, but only in the old local (all-in-one) mode. Needs porting to thin mode. |
| **State: Server only** | The server command exists; no screen calls it. |
| **State: Not on main** | Nothing on `main` does this. |
| **State: Ruling** | Needs Christopher's decision before anyone builds it. |
| **Domain (D01–D78)** | The repo's own data-domain numbers from `docs/ui/data-catalog/DATA_FLOW_MAP.md`. Use them to find the owner package. |
| **Thin mode / local mode** | `docs/architecture/desktop-modes.md`: thin = the product (data from the league server); local = legacy all-in-one, frozen. |

## The numbers
| Kind \ State | Hosted | Partly | Local only | Server only | Not on main | Ruling | Mock only | Total |
|---|---|---|---|---|---|---|---|---|
| UI | 14 | 2 | 18 | · | 28 | · | 1 | **63** |
| UI·S | · | 2 | 7 | · | 13 | · | · | **22** |
| IN | 26 | 25 | 14 | 2 | 11 | 1 | · | **79** |
| OUT | 18 | 5 | 20 | 2 | 15 | 6 | · | **66** |
| IN+OUT | 16 | 10 | 13 | 1 | 7 | · | · | **47** |
| MOCK | · | · | · | · | · | · | 10 | **10** |
| **Total** | **74** | **44** | **72** | **5** | **74** | **7** | **11** | **287** |

Plain reading:
- **95 of 287 stubs need no league data** (UI 63, saved-per-person 22, mock-only 10).
- **192 need data** (read, send or both).
- **74 are already built in the hosted app.** 72 are built but only in local mode (mostly the Control Room). 44 are partly built. 5 have a server command and no screen. **74 have nothing on `main`.** 7 need a ruling.

## 1. Works with no league data
These need no connection to make them real. The ones marked UI·S need one per-person layout store (domain D58) so they follow the person, not the device.

| Stub | What it does on screen | Kind | State | Real code on main |
|---|---|---|---|---|
| `shortcuts` | Pinned places, one press away | UI·S | Local only | Home tile sc + SHORTCUTS in shell/homeTiles.tsx (☆ pins) |
| `notes` | Free-form private scratch pad, autosaves | UI·S | Partly | personal/PersonalLists.tsx NoteEditor (per player/franchise, not free pad) |
| `control:situations` | Situations: 12 stories, each walks the tools that settle it | UI | Local only | control/room/SituationsView.tsx, control/room/guide.ts (B/R/G build marks) |
| `shell:home-boards` | App Home above all leagues: boards (tabs), free grid of 61 pieces, edit, stack, full, undo | UI·S | Not on main | shell/HomeBrief.tsx + shell/homeLayout.ts = per-league 3x2 grid of 13 tiles (local mode only) |
| `shell:rail` | Sidebar: app zone (Home + boards), my shortcuts, league zone with picker and screens | UI·S | Local only | shell/ShellRail.tsx, shell/leaguePicker.tsx |
| `shell:feeds` | Feed tags, peek strip, feed index (mock-only wiring view) | MOCK | Mock only | registry/EndpointIndex.tsx (developer endpoint index, local) |
| `shell:popout` | Pop a piece into its own window; keep live score on top | UI | Not on main | none (no window code in frontend) |
| `shell:tour` | First-login tour and labels toggle | UI | Not on main | none |
| `shell:mock-controls` | MOCK CONTROLS tab: moment, league count, sample switches | MOCK | Mock only | — |
| `closeLayer` | Close the top popover/drawer, focus returns | UI | Hosted | commands browser overlays.close |
| `edit` | Toggle Home edit mode | UI | Local only | shell/homeLayout.ts edit (local) |
| `undo` | Undo last layout change | UI | Not on main | — |
| `redo` | Redo layout change | UI | Not on main | — |
| `board` | Switch Home board | UI·S | Not on main | — |
| `boardAdd` | Open starter-board menu | UI | Not on main | — |
| `boardNew` | Add a starter board (or switch to it if open) | UI·S | Not on main | — |
| `boardMenu` | Board menu: rename, duplicate, publish, reset, close | UI | Not on main | — |
| `bm` | Run a board-menu choice; 'publish to league' sends the board to the league | UI·S | Not on main | — |
| `boardReset` | Reset board to its starter | UI·S | Local only | shell/homeLayout.ts reset (local, per league) |
| `boardClose` | Close a board with Undo | UI·S | Not on main | — |
| `boardMore` | Overflow menu for boards that do not fit | UI | Not on main | — |
| `pset` | Change a piece setting (filter, sort, size) | UI·S | Not on main | — |
| `pflip` | Flip a piece to its back side | UI | Not on main | — |
| `stk` | Switch tab inside a stacked piece | UI | Not on main | — |
| `premove` | Remove a piece | UI·S | Local only | shell/homeLayout.ts bump (local) |
| `pgear` | Open piece settings in the inspector | UI | Not on main | — |
| `pmenu` | Piece menu: bind league, size, full, pop out, keep on top | UI | Not on main | — |
| `pm` | Run a piece-menu choice (full, pop out, bind pin/follow/all) | UI·S | Not on main | — |
| `fullOpt` | Library choice: stack or place a piece | UI·S | Not on main | — |
| `pvApply` | Apply a previewed layout (from assistant or library) | UI·S | Not on main | — |
| `pvCancel` | Cancel layout preview | UI | Not on main | — |
| `bringBack` | Bring a popped-out piece back | UI | Not on main | — |
| `home` | Go Home | UI | Local only | nav.open home |
| `nav` | Go to a screen tab | UI | Hosted | nav.open (commands/browser.gen.ts) |
| `railBoard` | Open a board from the sidebar | UI | Not on main | — |
| `railCol` | Collapse/expand the sidebar | UI·S | Local only | shell/ShellRail.tsx |
| `pickLeague` | Select a league (Follow pieces switch) | UI | Local only | leagueState.choose (shell/leaguePicker.tsx) |
| `goLeague` | Open a league's screen | UI | Local only | shell/leaguePicker.tsx |
| `star` | Pin a tab to My shortcuts | UI·S | Local only | SHORTCUTS ☆ pins in shell/homeTiles.tsx |
| `inspClose` | Close the right panel | UI | Hosted | inspector.close |
| `pmSel` | Piece-menu choice from the inspector | UI | Not on main | — |
| `feeds` | Show/hide feed tags (mock wiring view) | MOCK | Mock only | — |
| `swPick` | Pick a league in the switcher (or pin a piece to it) | UI·S | Local only | shell/leaguePicker.tsx |
| `account` | Account menu | UI | Hosted | control/NativeAccount.tsx |
| `noop` | Nothing (rocker label) | UI | Mock only | — |
| `map` | Help map of the screen | UI | Not on main | — |
| `cmdRun` | Run a palette result | UI | Local only | commandbar candidates (commandbar.gen.ts) |
| `dkLane` | Switch Docket lane | UI | Not on main | — |
| `dkLg` | Docket: this league only / all | UI | Not on main | — |
| `dkAdd` | Add the Docket as a Home piece | UI·S | Not on main | — |
| `asstApply` | Apply what the assistant prepared (board, lineup, pop out) | UI·S | Not on main | — |
| `feedIndex` | Feed index drawer (mock wiring view) | MOCK | Mock only | registry/EndpointIndex.tsx (developer) |
| `peek` | Peek a feed tag (mock wiring view) | MOCK | Mock only | — |
| `peekClose` | Close peek | MOCK | Mock only | — |
| `demo` | Mock controls panel | MOCK | Mock only | — |
| `dset` | Mock control switch (moment, league count) | MOCK | Mock only | — |
| `notesBuild` | 'Notes for the real build' drawer | MOCK | Mock only | — |
| `tourStart` | Start first-login tour | UI | Not on main | — |
| `tourNext` | Next tour step | UI | Not on main | — |
| `tourSkip` | Skip tour | UI | Not on main | — |
| `etab` | Switch an inner tab on a screen | UI | Hosted | — |
| `pcol` | Collapse a screen panel | UI | Not on main | — |
| `sset` | Change a screen panel setting | UI | Not on main | — |
| `spmenu` | Screen panel menu (send to Home, pop out) | UI | Not on main | — |
| `spm` | Run panel menu choice | UI·S | Not on main | — |
| `pSel` | Pulse: select which matchup follows | UI | Hosted | PulseNow FOLLOWING (game.follow browser cmd) |
| `cmCancel` | Cancel proposal | UI | Local only | — |
| `trDir` | Trade desk: lens direction | UI | Hosted | shell/WarRoomTrade.tsx |
| `trFlt` | Trade desk: shelf filter | UI | Hosted | shell/WarRoomTrade.tsx |
| `trVis` | Trade desk: open or private offer | UI | Hosted | trade.offer visibility arg |
| `trVer` | Trade desk: switch/add version | UI·S | Partly | — |
| `trClear` | Clear the deal | UI | Hosted | shell/tradeBuilderState.ts |
| `trToggle` | Add/remove an asset to the deal | UI | Hosted | shell/warroomDrag.ts, shell/tradeBuilderState.ts |
| `stLive` | Spoiler delay menu | UI | Not on main | — |
| `stDelay` | Set spoiler delay (hold live scores back) | UI·S | Not on main | — |
| `stWeek` | Week clock → Season calendar | UI | Local only | nav |
| `eGo` | Card button: go to lineup/standings/game | UI | Hosted | nav.open |
| `eTrade` | Card button: open trade desk with player | UI | Hosted | nav.open war trade (Route.trade) |
| `offCtr` | Counter: put their offer on the desk | UI | Not on main | — |
| `wifDel` | What-if: remove a move | UI | Partly | — |
| `wifReset` | What-if: clear | UI | Partly | — |
| `dlGo` | Deadline row → open what is due | UI | Hosted | clock/deadlineRoute.ts |
| `copyLink` | Copy the invite link | UI | Not on main | — |
| `signin` | Mock sign in after sign-out | MOCK | Mock only | control/NativeAccount.tsx |
| `healthGo` | League health row → its tool | UI | Local only | — |
| `tOk` | Submit the open tool form | UI | Local only | commissioner/ActFields.tsx |
| `pkF` | Picks filter | UI | Local only | control/room/PicksView.tsx |
| `pkMenu` | Pick menu | UI | Local only | control/room/PicksView.tsx |
| `hJob` | Health row → open its job tool | UI | Local only | — |
| `cjFilter` | Jobs filter chips | UI | Local only | control/room/JobsView.tsx |
| `cjHome` | Cross-league jobs → that league's Jobs | UI | Local only | — |
| `cjReopen` | Reopen a settled job | UI | Not on main | — |
| `cjGo` | Job → open its tool | UI | Local only | control/room/openers.tsx |
| `askGo` | Assistant routes to a tool | UI | Local only | control/room/search.ts (runSearch) |
| `pfS` | People filter | UI | Local only | control/room/PeopleView.tsx |

## 2. Needs data — pieces (Home tiles that also appear inside screens)
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `franchise` | Record, seed, power and cap room for one league in a strip | IN | Hosted | `standings.read` · `odds.read` · `native.cap.check` | D02,D05,D20 | shell/HQHeader.tsx, shell/WarRoomHeader.tsx (same four facts) |
| `race` | Seed now, projected seed, playoff odds, next five opponents | IN | Hosted | `standings.read` · `schedule.read` · `odds.read` · `bracket.read` | D02,D05,D06 | shell/PulseRace.tsx; Home tile race (shell/homeTiles.tsx) |
| `docket` | Everything waiting on you or others, three lanes, across all leagues | IN+OUT | Partly | `moves.read` · `trades.read` · `bids.read` · `waivers.read` · `calendar.read` · `commish.jobs.read` · **NEW** one cross-league open-matters read · **NEW** private read marks | D56,D60,D21,D57 | Home alerts alertsOf() in shell/HomeBrief.tsx; commissioner/jobs.ts; shell/EnvelopeRail.tsx (local MFL) |
| `week` | Every matchup you play this week, all leagues, one plate | IN | Partly | `results.read` · `standings.read` · `week.read` · **NEW** cross-league summary (fan-out over leagues) | D02,D09 | Home tile mu (shell/homeTiles.tsx Matchup) is one league only |
| `today` | Your starters across every league grouped by kickoff | IN | Not on main | `lineup.read` · `nfl.games.read` · `projections.read` · **NEW** one player's exposure across my leagues | D09,D16,D17 | shell/PulseNow.tsx (one league); none cross-league |
| `cards` | One card per league: record, seed, next game, waiting count | IN | Partly | `standings.read` · `schedule.read` · `native.directory.read` · `commish.leagues.read` | D01,D02 | shell/leaguePicker.tsx (league list); control/NativeAccount.tsx (memberships) |
| `deadlines` | Next clocks that matter, soonest first; remind toggle | IN+OUT | Hosted | `calendar.read` · **NEW** reminder (private) | D40,D57 | clock/ClockStrip.tsx, clock/CalendarPanel.tsx, Home tile due |
| `cjobs` | Commissioner jobs across every league you run | IN | Local only | `commish.jobs.read` · `stats.queue.read` · `trades.read` | D60 | commissioner/jobs.ts, commissioner/JobsStrip.tsx, control/room/JobsView.tsx |
| `clock` | Countdown to the next deadline | IN | Hosted | `calendar.read` · `week.read` | D40 | clock/ClockStrip.tsx, clock/ticker.ts |
| `matchup` | Your live game: score, win chance, who is left to play | IN | Hosted | `results.read` · `lineup.read` · `projections.read` · `odds.read` · `nfl.games.read` | D09,D05 | shell/PulseNow.tsx; Home tile mu |
| `board` | Every matchup in the league this week | IN | Hosted | `results.read` · `schedule.read` · `week.read` | D09 | shell/PulseNow.tsx, shell/PulseSchedule.tsx |
| `nfl` | NFL scoreboard and the ball on the field | IN | Hosted | `nfl.games.read` · `nfl.game.read` | D07,D08 | now/LiveGame.tsx (field + plays), shell/NativeLeaguePulse.tsx |
| `pbp` | Play by play, newest first, with fantasy effect | IN | Partly | `nfl.game.read` · `results.read` | D08,D09 | now/LiveGame.tsx Scoring plays (scoring plays only; per-play fantasy delta missing) |
| `ingame` | Your starters in one NFL game | IN | Hosted | `lineup.read` · `nfl.game.read` | D09 | now/LiveGame.tsx follow + shell/PulseNow.tsx FOLLOWING |
| `statchg` | Live numbers vs final numbers with source | IN | Not on main | `results.read` · `stats.queue.read` | D51,D75 | no hosted screen; app/control CompareAlerts is local |
| `winp` | Win chance needle | IN | Partly | `odds.read` | D05 | Home tile WinChance (shell/homeTiles.tsx); live mid-game update not computed |
| `heat` | Live matchups ranked by closeness and time left | IN | Not on main | `results.read` · `odds.read` · **NEW** heat order (can be computed client-side from results + odds) | D09,D05 | none |
| `standings` | Seeds and records | IN | Hosted | `standings.read` | D02 | shell/PulseRace.tsx |
| `power` | Power rankings with components | IN | Hosted | `odds.read` · `standings.read` | D05 | shell/PulseRace.tsx, Home tile pwr |
| `bracket` | Seed now/projected and bracket | IN | Hosted | `bracket.read` · `odds.read` · `standings.read` | D04,D06 | shell/PulseRace.tsx, shell/PulseRaceDrawers.tsx |
| `road` | Your remaining schedule | IN | Server only | `schedule.franchise.read` | D02 | shell/PulseSchedule.tsx (league grid); schedule.franchise.read has no screen |
| `form` | Last five weeks vs projection; click a week | IN | Partly | `results.read` · `projections.read` | D12 | Home tile form (local research read; hosted absent per HQ map) |
| `trophy` | Titles, awards, custom trophies; award one | IN+OUT | Not on main | **NEW** trophies/awards read · **NEW** award trophy command | D49 | none (title.set only sets the title label) |
| `archive` | Seasons, eras, champions | IN | Not on main | **NEW** league history by season (app/archive facts not exposed hosted) | D47,D48 | none hosted; app/archive local |
| `wire` | Every transaction as it lands | IN | Hosted | `commish.ledger.filtered.read` · `franchise.moves.history.read` | D22,D23 | Home tile wire; shell/HQMoveHistory.tsx |
| `lineup` | Starters; drag to change; save | IN+OUT | Hosted | `lineup.read` · `lineup.check` · `projections.read` · `native.reserve.check` · `commish.lineup.set` | D17,D11 | shell/LineupEditor.tsx, shell/HQBoard.tsx, shell/hqNativeLineup.tsx, shell/hqPlacement.ts |
| `roster` | Everyone you hold with taxi and IR | IN+OUT | Hosted | `native.assets.read` · `lineup.read` · `native.reserve.check` · `roster.reserve` · `roster.cut` | D17 | shell/HQPanels.tsx, shell/HQRelease.tsx, commissioner/HostedLeague.tsx |
| `caprun` | Cap room this year and next four | IN | Hosted | `native.cap.check` | D20 | Home tile cap; shell/HQPanels.tsx (runway) |
| `contracts` | Every deal year by year | IN+OUT | Partly | `native.assets.read` · `native.cap.check` · `value.read` · `contract.action.set` | D18,D24 | contracts/Contracts.tsx (local), shell/HQPanels.tsx (hosted) |
| `expiring` | Deals that end and the calls you owe (extend/tag/let walk) | IN+OUT | Server only | `native.assets.read` · `calendar.read` · `contract.action.set` | D18,D40 | HQ 'Decisions due' (shell/HQPanels.tsx) placeholder; contract.action.set has no screen |
| `deadcap` | Money still owed for players gone | IN | Partly | `native.cap.check` | D19 | shell/HQPanels.tsx (dead in runway) |
| `whatif` | Stack cuts/restructures, watch the cap answer | IN | Partly | `contract.quote` · `native.cap.check` · **NEW** stacked what-if read | D19,D20 | shell/HQNativeWhatIf.tsx (one cut only), contracts/WhatIfPanel.tsx (local) |
| `picks` | Every pick you own; click for detail | IN | Hosted | `draft.read` · `native.assets.read` | D41,D24 | Home tile picks; draftroom/DraftBoard.tsx |
| `pcard` | One player, front and back, with a picker | IN | Partly | `value.read` · `projections.read` · `player.usage.read` · `player.agearc.read` · `contract.quote` · `player.note.read` | D24,D26,D13,D45 | shell/HQInspector.tsx, shell/WarRoomInspector.tsx, cards/PlayerCard.tsx |
| `offers` | Offers in and out; accept, decline, withdraw, counter | IN+OUT | Hosted | `trades.read` · `trade.accept` · `trade.decline` · `trade.withdraw` | D30 | commissioner/HostedWorkflows.tsx (trades task), shell/WarRoomDrawers.tsx |
| `desk` | Trade desk: send/get, two lenses, verdict, tune it | IN+OUT | Hosted | `tradelens.read` · `value.read` · `lineupdelta.read` · `native.cap.check` · `native.assets.read` · `trade.offer` | D29,D32,D33 | shell/WarRoomTrade.tsx, shell/warroomTradeReads.ts, shell/warroomLens.ts |
| `review` | Trades waiting on your ballot (DOT seat) | IN+OUT | Hosted | `trades.read` · `trade.review` · `trade.recuse` · `trade.appeal` | D31 | commissioner/HostedWorkflows.tsx; control/room/JobsView.tsx (local) |
| `pool` | Free agents and what they are worth; bid/claim | IN+OUT | Hosted | `native.assets.read` · `value.read` · `waivers.read` · `bids.read` · `bid.place` · `waivers.claim` · `player.watch.set` | D34,D24,D38 | shell/WarRoomMarket.tsx, shell/warroomMarketReads.ts |
| `bids` | Your open bids; raise, withdraw | IN+OUT | Hosted | `bids.read` · `bid.raise` · `bid.withdraw` | D34,D36 | Home tile bids; shell/WarRoomMarket.tsx |
| `waivers` | Your claims and the order; reorder, cancel | IN+OUT | Partly | `waivers.read` · `waivers.claim` · **NEW** claim reorder · **NEW** claim cancel | D34 | commissioner/HostedWorkflows.tsx (waivers task) |
| `draft` | Draft clock and board; make the pick | IN+OUT | Hosted | `draft.read` · `draft.pick` · `draft.queue.read` · `draft.queue.set` | D41,D43 | draftroom/DraftRoom.tsx, shell/WarRoomDraft.tsx |
| `plans` | Private plans on the calendar | IN+OUT | Not on main | `calendar.read` · **NEW** plans store (D39) | D39,D40 | shell/WarRoomPlans.tsx (empty state) |
| `looking` | Past offers and partners | IN | Hosted | `trades.read` · `moves.read` | D22,D30 | shell/WarRoomPlans.tsx Past offers/Counterparties |
| `nflnews` | News on your players, filters | IN | Not on main | **NEW** hosted status facts feed (injury, practice, depth) — no prose per phone ruling | D52,D14 | Home tile news (local research only) |
| `headlines` | What happened in your league this week | IN | Not on main | **NEW** week story (D50) | D50 | Home tile head = missing |
| `depth` | NFL depth charts coloured by owner | IN | Not on main | **NEW** depth chart feed · `native.assets.read` | D53 | HomeBrief Depth() (local research only) |
| `nflstats` | Stats scored with your league rules, filter and sort | IN | Not on main | **NEW** league-scored stats table read | D54 | HomeBrief NFL stats tab = explicit 'no read' |
| `describe` | Say what you want; see what would change | IN+OUT | Partly | `commish.rules.read` · `commish.controls.read` · `commish.rules.set` · **NEW** AI assistant (scoped token) | D61,D74 | commissioner/RuleChangeForm.tsx, commissioner/ruleDraft.ts (form path); no assistant |
| `house` | House rules in plain words with firing log | IN+OUT | Partly | `native.rules.read` · `commish.rules.set` · **NEW** house-rule firing log | D71 | control/room/RulesView.tsx; native.rules.read (TargetNativeHouseRules) |
| `layout` | Every screen/piece the league gets, and why; set on/off/auto | IN+OUT | Not on main | **NEW** league surface settings (fold 1) | D58 | none |
| `statq` | Stat corrections waiting for yes/no | IN+OUT | Local only | `stats.queue.read` · `commish.stat.queue.approve` · `commish.stat.queue.reject` | D51 | control/room/openers.tsx, control/room/FixView.tsx (local) |
| `rules` | Rulebook, editable, with preset compare | IN+OUT | Hosted | `commish.rules.read` · `commish.rules.preset.read` · `commish.rules.set` · `rule.exception` | D61,D62 | commissioner/HostedRules.tsx (hosted), control/room/RulesView.tsx, control/room/PresetCompare.tsx |
| `season` | Windows of the season; move them | IN+OUT | Local only | `calendar.read` · `commish.schedule.read` · `calendar.window.move` · `calendar.window.set` · `calendar.window.remove` | D40 | control/room/SeasonView.tsx (local; window.set/remove have no screen) |
| `fix` | Recent fixes and exceptions in force | IN+OUT | Local only | `commish.corrections.read` · `commish.ledger.read` · `commish.correction.revert` | D66,D23 | control/room/FixView.tsx, commissioner/UndoForm.tsx |
| `people` | Who sits where and when last seen; seat tools | IN+OUT | Local only | `commish.people.read` · `commish.seat.invite` · `commish.seat.replace` · `commish.seat.remove` · `commish.seat.caretaker` · `commish.role.grant` · `commish.role.revoke` | D64,D69 | control/room/PeopleView.tsx, commissioner/LeaguePeople.tsx |
| `ledger` | The record of everything, filterable | IN | Local only | `commish.ledger.read` · `commish.ledger.filtered.read` | D23 | commissioner/LeagueLedger.tsx, commissioner/LedgerFilters.tsx |
| `notices` | What you told the league; publish | IN+OUT | Local only | `commish.notices.read` · `commish.notice.publish` | D70 | commissioner/NoteForm.tsx, commissioner/notes.ts |
| `health` | Stalled matters, quiet owners, orphan teams | IN | Local only | `commish.people.read` · `commish.jobs.read` | D60,D69 | commissioner/jobs.ts (jobs) + commissioner/people.ts (last seen) |
| `server` | Backups, feeds, upgrades | IN+OUT | Local only | **NEW** server status read for commissioner (serverstatus is operator console only) | D72 | control/room/ServerView.tsx (local), commissioner/BackupNow.tsx (local backup.now) |

## 3. Needs data — screen tabs
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `hq:lineup` | Lineup and roster: starters/bench, drag to set, live game, win chance, form, taxi/IR | IN+OUT | Hosted | `lineup.read` · `lineup.check` · `projections.read` · `native.reserve.check` · `commish.lineup.set` · `roster.reserve` | D17,D09 | shell/NativeFranchiseHQ.tsx > HQShell > HQBoard/LineupEditor (hosted) |
| `hq:contracts` | Contracts and cap: deals by year, runway, decisions, release/restructure/extend forms | IN+OUT | Partly | `native.cap.check` · `native.assets.read` · `contract.quote` · `value.read` · `roster.cut` · `contract.action.set` | D18,D19,D20 | shell/HQPanels.tsx, shell/HQRelease.tsx, shell/HQNativeWhatIf.tsx |
| `hq:moves` | My moves: what you did and what is pending | IN | Hosted | `franchise.moves.history.read` · `moves.read` | D21,D22 | shell/HQMoveHistory.tsx, shell/HQMoves.tsx |
| `war:pool` | Valuations and pool: free agents, targets, bids, budget split, recent winning bids | IN+OUT | Hosted | `native.assets.read` · `value.read` · `bids.read` · `waivers.read` · `player.watch.read` · `player.watch.order` · `bid.place` · `bid.raise` · `bid.withdraw` · **NEW** recent winning bids (D35) | D34,D35,D38 | shell/WarRoomMarket.tsx |
| `war:trade` | Trade desk: shelves, ledger, two lenses, verdict, tune, versions, pocket, negotiations | IN+OUT | Hosted | `tradelens.read` · `value.read` · `lineupdelta.read` · `native.cap.check` · `trades.read` · `trade.offer` · **NEW** saved trade drafts (PR #338) | D29,D30,D32,D33 | shell/WarRoomTrade.tsx, shell/WarRoomDrawers.tsx |
| `war:draft` | Draft room: queue, scouting card, make the pick | IN+OUT | Hosted | `draft.read` · `draft.queue.read` · `draft.queue.set` · `draft.pick` · **NEW** ADP/tiers feed · **NEW** scouting store | D41,D42,D43,D44 | draftroom/DraftRoom.tsx embedded via shell/WarRoomDraft.tsx |
| `war:plans` | Plans: private plans, steps on the calendar, past offers | IN+OUT | Partly | `calendar.read` · `trades.read` · **NEW** plans store | D39,D40 | shell/WarRoomPlans.tsx |
| `pulse:now` | Now: league scoreboard, your matchup, NFL field, plays, stat pulls | IN | Hosted | `results.read` · `lineup.read` · `odds.read` · `nfl.games.read` · `nfl.game.read` · `week.read` | D07,D08,D09 | shell/NativeLeaguePulse.tsx > PulseNow, now/LiveGame.tsx |
| `pulse:race` | Race: standings, power + why it moved, tiebreaks, bracket | IN | Hosted | `standings.read` · `odds.read` · `bracket.read` · `schedule.read` | D02,D03,D05,D06 | shell/PulseRace.tsx, shell/PulseRaceDrawers.tsx |
| `pulse:archive` | Archive: seasons, champions, trophy case | IN+OUT | Not on main | **NEW** archive read · **NEW** trophies | D47,D48,D49 | none (not in real shell/nodes.ts) |
| `control:jobs` | Jobs: morphs by clock; lanes NOW/HOLDING/TODAY/WEEK/LATER; open the tool that settles it | IN+OUT | Local only | `commish.jobs.read` · `stats.queue.read` · `trades.read` · `calendar.read` | D60 | control/room/JobsView.tsx, commissioner/jobs.ts (local only) |
| `control:rules` | Rules: settings board with source tags, preset compare, house rules | IN+OUT | Partly | `commish.rules.read` · `commish.rules.preset.read` · `native.rules.read` · `commish.rules.set` · `rule.exception` | D61,D62,D71 | control/room/RulesView.tsx, commissioner/HostedRules.tsx |
| `control:season` | Season: windows, schedule, bracket, title, rollover | IN+OUT | Local only | `calendar.read` · `commish.schedule.read` · `schedule.edit` · `schedule.generate` · `schedule.publish` · `bracket.build` · `bracket.edit` · `title.set` · `season.rollover` · `calendar.window.move` | D40,D04 | control/room/SeasonView.tsx; season.rollover also in HostedWorkflows |
| `control:fix` | Fix: correction forms with preview, reason, safeguard; undo | IN+OUT | Local only | `commish.corrections.read` · `correct.roster` · `correct.transaction` · `correct.pick` · `commish.stat.correct` · `commish.correction.revert` | D66,D51 | control/room/FixView.tsx, commissioner/actTable.ts |
| `control:picks` | Picks: all picks at 32 teams, filters; grant, swap, forfeit, condition, void, create | IN+OUT | Local only | `draft.read` · `native.assets.read` · `correct.pick` · `draft.order.set` · **NEW** pick ops grant/swap/forfeit/condition/void/create (D67 ruled, not built) | D41,D67 | control/room/PicksView.tsx |
| `control:people` | People: seats, deputies, permissions, succession | IN+OUT | Local only | `commish.people.read` · `commish.role.grant` · `commish.role.revoke` · `commish.abilities.set` · `commish.deputy.order` · `commish.succession.*` · `commish.seat.*` | D64,D65,D69 | control/room/PeopleView.tsx |
| `control:league` | League: identity, divisions, layout, colour, ledger | IN+OUT | Local only | `commish.rules.read` · `commish.franchise.edit` · `commish.ledger.filtered.read` · **NEW** league colour · **NEW** league layout surfaces | D01,D23,D58 | control/room/LeagueView.tsx |
| `control:server` | Server: backups, feeds, upgrades, metrics | IN+OUT | Local only | **NEW** commissioner server status read · **NEW** feed pause/switch · **NEW** restore from UI | D72 | control/room/ServerView.tsx (local), serverstatus (operator socket only) |

## 4. Needs data — shell
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `shell:league-chip` | League switcher grouped NEEDS YOU / LIVE NOW / EVERYTHING ELSE | IN | Partly | **NEW** per-league waiting + live flags (fan-out of jobs/alerts/results) | D01,D56 | shell/leaguePicker.tsx |
| `shell:search` | Search or type a command (Ctrl K) | IN | Local only | **NEW** search over players/teams/leagues (D76) | D63,D76 | commands/CommandBar.tsx, commands/search.ts, commandbar.gen.ts |
| `shell:needs-you` | Needs you count and Docket drawer | IN | Partly | see docket | D56,D60 | alertsOf() in shell/HomeBrief.tsx |
| `shell:alert-band` | NOW band: top alert with its one action, cycle | IN | Partly | **NEW** alert feed per person across leagues (D56 producers) | D56 | shell/HomeBrief.tsx alerts (one league) |
| `shell:assistant` | Assistant drawer: scripted requests that build boards or settle jobs | IN+OUT | Not on main | **NEW** in-app assistant over scoped AI tokens (seam: scopes) | D74 | none in app; AI helpers = tokens in control/NativeAccount.tsx, cmd/twr MCP |
| `shell:account` | YO menu: labels, AI key, devices, sign out | IN+OUT | Hosted | account endpoints /v1/auth/* (not catalog commands) | D68,D74 | control/NativeAccount.tsx (devices revoke, sign-in, recovery) |
| `shell:status-bar` | CONNECTED · LIVE n s behind · week clock; each opens its panel | IN | Partly | **NEW** per-league server health read · UI:spoiler delay pref | D72,D75 | shell/StatusStrip.tsx, data/freshness.ts |
| `shell:inspector` | Right panel: full player/asset card | IN | Hosted | `value.read` · `projections.read` · `contract.quote` · `player.usage.read` · `player.agearc.read` · `player.note.read` | D24,D26,D13,D45 | shell/HQInspector.tsx, shell/WarRoomInspector.tsx, shell/PlayerInspector.tsx |
| `shell:cards` | Click any player/team/club/transaction name: popover card | IN | Partly | see ent action | D24,D02,D07,D23 | shell/EntityInspector.tsx (franchise/offer/pick only) |

## 5. Needs data — actions (buttons and handlers), by area

### Home layout and boards
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `library` | Open the piece library (greys pieces a league cannot use, with reason) | IN | Local only | commish.rules.read (league shape) | D58,D61 | shell/homeLayout.ts tile chooser (local, 13 tiles) |

### Shell (top bar, rail, status, cards, assistant)
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `inspect` | Open the player in the right panel | IN | Hosted | `value.read` · `projections.read` · `contract.quote` · `player.usage.read` · `player.agearc.read` · `player.note.read` | D24,D26,D13,D45 | shell/HQInspector.tsx, shell/WarRoomInspector.tsx |
| `switcher` | League switcher with groups | IN | Partly | `commish.leagues.read` · **NEW** per-league waiting/live flags | D01,D56 | shell/leaguePicker.tsx |
| `acct` | Account menu choice: labels, AI provider key, devices, sign out | IN+OUT | Partly | account /v1/auth/* (sign-out, device revoke) · **NEW** AI provider key store | D68,D74 | control/NativeAccount.tsx (Revoke this device / all devices, Forget saved sign-in) |
| `cmdk` | Command palette: screens, commands, players | IN | Local only | **NEW** search index (D76) | D76 | commands/CommandBar.tsx, commands/search.ts |
| `docket` | Open the Docket drawer, three lanes | IN | Partly | see docket piece | D56,D60 | alertsOf() in shell/HomeBrief.tsx |
| `dgo` | Open a Docket item's place and mark it read | IN+OUT | Not on main | **NEW** private read marks | D57 | — |
| `assistant` | Open the assistant drawer | IN+OUT | Not on main | **NEW** in-app assistant | D74 | — |
| `askSend` | Send typed request to the assistant | OUT | Not on main | **NEW** in-app assistant | D74 | — |
| `ask` | Send a suggested request | OUT | Not on main | **NEW** in-app assistant | D74 | — |
| `cmAsk` | Describe-your-league: ask in words | OUT | Not on main | **NEW** in-app assistant | D74 | — |
| `asstCm` | Apply a commissioner request from the assistant | OUT | Not on main | `commish.rules.set` | D61,D74 | — |
| `trInspect` | Inspect an asset on the desk | IN | Hosted | `value.read` · `contract.quote` | D24 | shell/WarRoomInspector.tsx |
| `stConn` | League servers drawer: one row per server, ping | IN | Not on main | **NEW** per-server health read (serverversion preflight is closest) | D72 | serverversion, data/freshness.ts |
| `ent` | Popover card for a player, fantasy team, NFL club or transaction | IN | Partly | `value.read` · `projections.read` · `standings.read` · `nfl.game.read` · `commish.ledger.filtered.read` · **NEW** status facts per player | D24,D02,D07,D23,D14 | shell/EntityInspector.tsx (team/offer/pick), cards/PlayerCard.tsx |
| `devOut` | Sign out a device with Undo | OUT | Hosted | account device revoke (/v1/auth) | D68 | control/NativeAccount.tsx Revoke this device |
| `askC` | Ask bar on a Control Room tab | OUT | Not on main | **NEW** in-app assistant | D74 | — |
| `askKeep` | Keep a request in writing (becomes a job) | OUT | Not on main | **NEW** written request store | D60 | — |
| `askHouse` | Turn a request into a house rule | OUT | Not on main | `commish.rules.set` | D71 | — |

### Franchise HQ
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `hqAct` | Lineup: reset / review / save | OUT | Hosted | `lineup.check` · `commish.lineup.set` | D17 | shell/hqNativeLineup.tsx, shell/LineupEditor.tsx |
| `decide` | Expiring deal choices: extend, tag, let walk | IN | Server only | `native.assets.read` · `contract.quote` | D18 | — |
| `decSet` | Record a contract decision | OUT | Server only | `contract.action.set` | D18 | catalog only (no screen) |
| `decUndo` | Undo a contract decision | OUT | Server only | `contract.action.set` | D18 | catalog only |
| `wifAdd` | What-if: add a cut/restructure to the stack | IN | Partly | `contract.quote` · `native.cap.check` · **NEW** stacked what-if | D19,D20 | shell/HQNativeWhatIf.tsx (single cut) |
| `pick` | Pick detail popover (projected slot, from whom) | IN | Hosted | `draft.read` | D41 | EntityInspector pick (shell/EntityInspector.tsx) |
| `remind` | Set a reminder 15 min before a deadline | OUT | Not on main | **NEW** private reminder | D57 | — |
| `formWk` | Form: open one week's result | IN | Partly | `results.read` | D12 | — |
| `cmGo` | Release / restructure / extend form with cap quote | IN+OUT | Partly | `contract.quote` · `native.cap.check` · `roster.cut` · `contract.action.set` | D18,D19 | shell/HQRelease.tsx (release); restructure/extend not on a screen |
| `cmUndo` | Undo a release/restructure | OUT | Ruling | **NEW** owner undo (only commish.correction.revert exists) | D66 | — |

### War Room
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `trTune` | Apply a tune-it suggestion | IN | Hosted | tradelens.read (Tune) | D32 | internal/tradelens Tune; shell/warroomLens.ts |
| `trSave` | Save deal to pocket | OUT | Partly | **NEW** saved trade drafts (PR #338) | D29 | — |
| `trSend` | Send the offer | OUT | Hosted | `trade.offer` | D30 | shell/warroomTradeReads.ts (#324) |
| `trLoad` | Load a saved deal or a live thread | IN | Partly | `trades.read` · **NEW** saved trade drafts | D29,D30 | shell/WarRoomDrawers.tsx |
| `trPocket` | Pocket sheet: saved deals re-read | IN | Partly | **NEW** saved trade drafts | D29 | — |
| `trNeg` | Negotiations sheet: offers in motion | IN | Hosted | `trades.read` | D30 | shell/WarRoomDrawers.tsx |
| `eWatch` | Watch/unwatch a player | OUT | Hosted | `player.watch.set` | D38 | shell/playerPersonal.ts, personal/WatchButton.tsx |
| `eDepth` | Depth chart for a club | IN | Not on main | **NEW** depth chart feed | D53 | — |
| `eBid` | Bid/claim form from a card | IN | Hosted | `bids.read` · `native.cap.check` · `waivers.read` | D34,D36 | shell/WarRoomMarket.tsx 'Review bid' |
| `eBidGo` | Place the bid | OUT | Hosted | `bid.place` · `bid.raise` | D34 | shell/warroomMarketReads.ts / HostedWorkflows (#329) |
| `bidStep` | Raise or lower an open bid | OUT | Hosted | `bid.raise` | D36 | HostedWorkflows bid.raise |
| `bidDrop` | Withdraw a bid with Undo | OUT | Hosted | `bid.withdraw` | D36 | HostedWorkflows bid.withdraw |
| `clMove` | Reorder waiver claims | OUT | Not on main | **NEW** claim priority reorder | D34 | — |
| `clDrop` | Cancel a waiver claim with Undo | OUT | Not on main | **NEW** claim cancel | D34 | — |
| `offAcc` | Accept a trade offer (confirm) | OUT | Hosted | `trade.accept` | D30 | HostedWorkflows trade.accept |
| `offDec` | Decline with optional note | OUT | Hosted | `trade.decline` · **NEW** decline note | D30 | HostedWorkflows trade.decline |
| `offWd` | Withdraw your offer | OUT | Hosted | `trade.withdraw` | D30 | HostedWorkflows trade.withdraw |
| `offSet` | Confirm accept/decline | OUT | Hosted | `trade.accept` · `trade.decline` | D30 | HostedWorkflows |
| `pBid` | Pool: bid form with budget left | IN+OUT | Hosted | `bids.read` · `native.cap.check` · `bid.place` · `bid.raise` | D34,D36 | shell/WarRoomMarket.tsx |
| `dqAdd` | Queue a prospect | OUT | Hosted | `draft.queue.set` | D43 | shell/warroomDraftReads.ts (#333) |
| `dqDel` | Remove from queue | OUT | Hosted | `draft.queue.set` | D43 | shell/warroomDraftReads.ts |
| `dqMove` | Reorder queue | OUT | Hosted | `draft.queue.set` | D43 | shell/warroomDraftReads.ts |
| `scout` | Scouting card for a prospect | IN | Partly | `draft.read` · **NEW** scouting store (D44) · **NEW** ADP/tiers | D42,D44 | shell/WarRoomScouting.tsx (mostly unavailable) |
| `dPick` | Make the pick (final) | OUT | Hosted | `draft.pick` | D41 | draftroom/YourPick.tsx |
| `trAward` | Award a trophy | OUT | Not on main | **NEW** award trophy | D49 | — |

### League Pulse
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `pull` | Pulse drawer for a matchup's stat pulls | IN | Partly | `results.read` · `nfl.game.read` | D09,D51 | shell/PulseRaceDrawers.tsx / PulseNow drawers |
| `whyMoved` | Power: why each team moved | IN | Partly | `odds.read` · **NEW** why-moved notes (D05) | D05 | shell/PulseRaceDrawers.tsx |
| `tieBreak` | Tiebreak order and which step decided | IN | Partly | `standings.read` · **NEW** tiebreak step-used narrative (D03) | D03 | shell/PulseRaceDrawers.tsx |
| `brkFull` | Full bracket drawer | IN | Hosted | `bracket.read` · `odds.read` | D04,D06 | shell/PulseRace.tsx |

### Control Room
| Stub | What it does on screen | Kind | State | Data it needs | Domain | Real code on main |
|---|---|---|---|---|---|---|
| `say` | Commissioner/account action box by type (approve, reject, vote, fix, rule, seat, revert, retry, devices, sign out…) | OUT | Local only | `commish.stat.queue.approve` · `commish.stat.queue.reject` · `trade.review` · `commish.rules.set` · `commish.correction.revert` · `correct.pick` | D51,D31,D61 | commissioner/CommissionerActTray.tsx, commissioner/ActFields.tsx (generic act forms) |
| `jobOpen` | Open a job drawer (old v2 jobs) | IN | Local only | `commish.jobs.read` | D60 | control/room/JobsView.tsx |
| `jobDone` | Mark a job settled | OUT | Local only | the job's own command | D60 | control/room/JobsView.tsx |
| `cmTry` | Describe-your-league: preview a proposal | IN | Local only | `commish.rules.read` | D61 | commissioner/useChangePreview.ts |
| `cmApply` | Apply the proposal | OUT | Hosted | `commish.rules.set` | D61 | commissioner/RuleChangeForm.tsx (preview → confirm) |
| `surf` | League layout: set a surface on/off/auto | OUT | Not on main | **NEW** league surface setting | D58 | — |
| `lcolor` | Change league identity colour | OUT | Not on main | **NEW** league colour setting | D01 | — |
| `cmOk` | Confirm a commissioner action box (seat change needs a reason) | OUT | Local only | `commish.seat.replace` · `commish.seat.remove` · `commish.seat.caretaker` | D64 | control/room/openers.tsx |
| `seatGo` | Seat tool: invite link, caretaker, replace, remove | OUT | Local only | `commish.seat.invite` · `commish.seat.caretaker` · `commish.seat.replace` · `commish.seat.remove` | D64 | control/room/openers.tsx; join seam internal/accounts/join.go |
| `cmDone` | Finish invite | OUT | Local only | `commish.seat.invite` | D64 | control/room/openers.tsx |
| `preCmp` | Compare your league to presets | IN | Local only | `commish.rules.preset.read` | D62 | control/room/PresetCompare.tsx |
| `preApply` | Apply a preset column | OUT | Local only | `commish.rules.set` | D62 | control/room/PresetCompare.tsx |
| `houseNew` | Write a new house rule | OUT | Partly | commish.rules.set (house text) · **NEW** house-rule firing log | D71 | control/room/RulesView.tsx |
| `schedTool` | Schedule drawer weeks 12–14 | IN | Local only | `commish.schedule.read` | D04 | control/room/SeasonView.tsx |
| `schedSwap` | Swap a matchup | OUT | Local only | `schedule.edit` | D04 | control/room/SeasonView.tsx |
| `brkTool` | Bracket settings: teams, reseed | OUT | Local only | `bracket.build` · `bracket.edit` · `bracket.preset.save` | D04 | control/room/SeasonView.tsx |
| `titleTool` | Crown a champion by ruling | OUT | Local only | `bracket.advance` · `commish.standings.override` | D04 | control/room/SeasonView.tsx (title.set is only the title label) |
| `rollTool` | Rollover preview | IN | Hosted | season.rollover (preview mode) | D40 | commissioner/CommissionerActTray.tsx |
| `rollRun` | Run rollover (hold to fire) | OUT | Hosted | `season.rollover` | D40 | HostedWorkflows rollover, commissioner/HoldToFire.tsx |
| `pkDo` | Grant/swap/forfeit/condition/void/restore a pick | OUT | Partly | `correct.pick` · **NEW** pick ops (D67) | D67 | control/room/PicksView.tsx (move only) |
| `pkNew` | Create a pick | OUT | Not on main | **NEW** create pick (D67) | D67 | — |
| `permT` | Toggle a deputy permission | OUT | Local only | `commish.abilities.set` | D65 | control/room/PeopleView.tsx |
| `depAdd` | Add a deputy with permissions | OUT | Local only | `commish.role.grant` · `commish.abilities.set` | D64,D65 | control/room/PeopleView.tsx |
| `depDel` | Remove a deputy | OUT | Local only | `commish.role.revoke` | D64 | control/room/PeopleView.tsx |
| `sucMove` | Reorder the succession line | OUT | Local only | `commish.deputy.order` | D64 | control/room/PeopleView.tsx |
| `bkNow` | Take a backup now | OUT | Local only | backup.now (local desktop only) · **NEW** hosted commissioner backup request | D72 | commissioner/BackupNow.tsx (local) |
| `bkRestore` | Restore preview | IN | Ruling | **NEW** backup list for commissioner | D72 | internal/offsite (operator) |
| `bkRun` | Restore a backup | OUT | Ruling | **NEW** restore from the app (operator-only today: deploy/server/RESTORE.md) | D72 | — |
| `feedT` | Pause/resume a feed | OUT | Ruling | **NEW** feed control | D72 | — |
| `feedSw` | Switch live scoring to the backup feed | OUT | Ruling | **NEW** truth-ladder override | D72,D75 | — |
| `upChk` | Check for a server update | IN | Partly | **NEW** update status for commissioner (releasetrust) | D72 | internal/releasetrust, control/UpdateRequired.tsx (desktop client) |
| `upRun` | Install server upgrade | OUT | Ruling | **NEW** upgrade from the app (operator-only today) | D72 | — |
| `cjSnooze` | Move a job back a day | OUT | Not on main | **NEW** job snooze (D57) | D57 | — |
| `cjOpen` | Open a job drawer | IN | Local only | `commish.jobs.read` | D60 | control/room/JobsView.tsx |
| `cjMark` | Mark a job settled with a note | OUT | Local only | the job's settling command · **NEW** settle note | D60 | control/room/JobsView.tsx |
| `cjTrade` | Trade review drawer: both sides, ballots | IN | Local only | `trades.read` | D31 | control/room/JobsView.tsx |
| `cjTx` | Approve / veto / override / extend review | OUT | Partly | `trade.review` · `trade.override` · **NEW** extend review window | D31 | commissioner/HostedWorkflows.tsx (review, override) |
| `cjStat` | Stat corrections drawer | IN | Local only | `stats.queue.read` | D51 | control/room/openers.tsx |
| `cjSq` | Approve/reject one or all corrections | OUT | Local only | `commish.stat.queue.approve` · `commish.stat.queue.reject` | D51 | control/room/openers.tsx |
| `cjTie` | Settle a blind-bid tie | OUT | Local only | `bids.award.override` · `waivers.award.override` | D34 | control/room/SeasonView.tsx |
| `cjLine` | Lineups with a hole drawer | IN | Not on main | **NEW** league-wide lineup holes read (job producer D60) | D60,D17 | — |
| `cjLn` | Set a lineup for an owner or nudge in app | OUT | Partly | `commish.lineup.set` · `commish.notice.publish` | D17,D70 | commissioner/HostedLeague.tsx (commish.lineup.set) |
| `cjCap` | Over-the-cap ruling | OUT | Local only | `commish.cap.relief` · `rule.exception` · `commish.notice.publish` | D20,D66 | control/room/guide.ts steps |
| `cjNotice` | Publish a notice | OUT | Local only | `commish.notice.publish` | D70 | commissioner/NoteForm.tsx |
| `cjOrder` | Set next year's draft order | OUT | Local only | `draft.order.set` · `draft.order.auto` · `draft.order.pattern` | D41 | commissioner/actTable.ts |
| `cfgEdit` | Edit one setting with timing and reason | OUT | Hosted | `commish.rules.set` | D61 | commissioner/RuleChangeForm.tsx, commissioner/RuleInput.tsx |
| `divTool` | Divisions drawer | IN | Local only | `commish.rules.read` | D02 | control/room/LeagueView.tsx |
| `divMove` | Move a team to another division | OUT | Ruling | commish.rules.set (division membership) · **NEW** if not a rule key | D02 | — |

## 6. What nothing on `main` supplies yet (79 new feeds or commands)
| New feed or command | Stubs that need it |
|---|---|
| in-app assistant | `assistant`, `askSend`, `ask`, `cmAsk`, `askC` |
| saved trade drafts (PR #338) | `war:trade`, `trVer`, `trSave` |
| private read marks | `docket`, `dgo` |
| claim cancel | `waivers`, `clDrop` |
| depth chart feed | `depth`, `eDepth` |
| house-rule firing log | `house`, `houseNew` |
| saved trade drafts | `trLoad`, `trPocket` |
| one cross-league open-matters read | `docket` |
| cross-league summary (fan-out over leagues) | `week` |
| one player's exposure across my leagues | `today` |
| reminder (private) | `deadlines` |
| layout store (D58) | `shortcuts` |
| private scratch note (personal store kind) | `notes` |
| heat order (can be computed client-side from results + odds) | `heat` |
| trophies/awards read | `trophy` |
| award trophy command | `trophy` |
| league history by season (app/archive facts not exposed hosted) | `archive` |
| stacked what-if read | `whatif` |
| claim reorder | `waivers` |
| plans store (D39) | `plans` |
| hosted status facts feed (injury, practice, depth) — no prose per phone ruling | `nflnews` |
| week story (D50) | `headlines` |
| league-scored stats table read | `nflstats` |
| AI assistant (scoped token) | `describe` |
| league surface settings (fold 1) | `layout` |
| server status read for commissioner (serverstatus is operator console only) | `server` |
| recent winning bids (D35) | `war:pool` |
| ADP/tiers feed | `war:draft` |
| scouting store | `war:draft` |
| plans store | `war:plans` |
| archive read | `pulse:archive` |
| trophies | `pulse:archive` |
| pick ops grant/swap/forfeit/condition/void/create (D67 ruled, not built) | `control:picks` |
| league colour | `control:league` |
| league layout surfaces | `control:league` |
| commissioner server status read | `control:server` |
| feed pause/switch | `control:server` |
| restore from UI | `control:server` |
| layout store per person (D58; localStorage until it exists) | `shell:home-boards` |
| per-league waiting counts | `shell:rail` |
| per-league waiting + live flags (fan-out of jobs/alerts/results) | `shell:league-chip` |
| search over players/teams/leagues (D76) | `shell:search` |
| alert feed per person across leagues (D56 producers) | `shell:alert-band` |
| in-app assistant over scoped AI tokens (seam: scopes) | `shell:assistant` |
| per-league server health read | `shell:status-bar` |
| publish suggested Home (commissioner) | `bm` |
| shortcut pins in layout store | `star` |
| per-league waiting/live flags | `switcher` |
| AI provider key store | `acct` |
| search index (D76) | `cmdk` |
| league surface setting | `surf` |
| league colour setting | `lcolor` |
| per-server health read (serverversion preflight is closest) | `stConn` |
| status facts per player | `ent` |
| claim priority reorder | `clMove` |
| decline note | `offDec` |
| stacked what-if | `wifAdd` |
| private reminder | `remind` |
| owner undo (only commish.correction.revert exists) | `cmUndo` |
| scouting store (D44) | `scout` |
| ADP/tiers | `scout` |
| why-moved notes (D05) | `whyMoved` |
| tiebreak step-used narrative (D03) | `tieBreak` |
| award trophy | `trAward` |
| pick ops (D67) | `pkDo` |
| create pick (D67) | `pkNew` |
| hosted commissioner backup request | `bkNow` |
| backup list for commissioner | `bkRestore` |
| restore from the app (operator-only today: deploy/server/RESTORE.md) | `bkRun` |
| feed control | `feedT` |
| truth-ladder override | `feedSw` |
| update status for commissioner (releasetrust) | `upChk` |
| upgrade from the app (operator-only today) | `upRun` |
| job snooze (D57) | `cjSnooze` |
| settle note | `cjMark` |
| extend review window | `cjTx` |
| league-wide lineup holes read (job producer D60) | `cjLine` |
| written request store | `askKeep` |
| if not a rule key | `divMove` |

## 7. Names corrected while checking
| In the mock | On `main` |
|---|---|
| `power.read` (Franchise strip, Race) | Does not exist. Power comes from `odds.read` (Home tile `pwr` and Pulse Race read it there). |
| `cap.read` (Franchise strip) | Does not exist. Cap room comes from `native.cap.check`. |
| Lineup save "calls `editor.save`; catalog name NOT IDENTIFIED" (v2 note on HQ tray) | Owner lineup save is `commish.lineup.set` through the hosted workflow, checked with `lineup.check` (`HostedLeague.tsx`, `hqNativeLineup.tsx`). |
| `commish.jobs.read` tagged LOCAL ONLY (kit list 11-15) | It is a native catalog read on `main`. The Control Room screen that shows it is local-only, not the read. Same for the other `commish.*` reads. |
| Notes, watch, draft queue tagged "new feed" | Now on `main`: `player.note.read/set`, `player.watch.read/set/order`, `draft.queue.read/set` (#333, #334, #336). |
| Moves history "new" | `franchise.moves.history.read` (#319). |
| Age arc `@TargetAgeArc` (local binding) | `player.agearc.read` and `player.usage.read` exist; they refuse until F51 player values are on the server. |
| Trade Tune-it "new engine" | `tradelens.read` returns `Tune` suggestions (`internal/tradelens/tradelens.go`). |
| "Crown a champion" → `title.set` | `title.set` only sets the title label. A champion by ruling is `bracket.advance` / `commish.standings.override`. |

Server commands with no screen anywhere on `main`: `calendar.window.set`, `calendar.window.remove`, `commish.succession.accept`, `commish.succession.activity`, `commish.succession.decline`, `contract.action.set`, `history.link`, `native.roster.check`, `schedule.franchise.read`, `score.player.read`.

## 8. Rulings the mock currently breaks
| Mock behaviour | Ruling | Where it is written |
|---|---|---|
| My bids lets you lower a placed bid and withdraw it, with Undo | R3: a placed bid is final; it ends only when legally outbid | `docs/ui/UI_Direction_v2_2026-10-30.md` (Rulings 10-30 late; War Room: "a placed bid is final") |
| 8-second Undo toasts after sent commands (accept, decline, bids, contract decisions, seat changes, rollover, restore) | Commands are preview → confirm → execute; reversal is per the catalog's Undo field (`checked-reversal` = commissioner reversal as a new ledger row; `none` = final); G3 is a 600 ms hold | `docs/architecture/command-spine.md`, `commands/native.gen.ts` Undo field, UI Direction v2 Interaction |
| Commit buttons inside read rows (bid rows, claim rows, job cards) | Commit controls live only in action trays and drawers, never in read tables | UI Direction v2 Interaction |
| NFL news as written prose, "rostered in 23% of leagues", "available in 71%" | Phone mock data ruling 11-17: status facts only, no ROST %, no trending adds, no written news (carried to desktop: decided 11-17) | `~/archive/2026-11-twr-mock-v1/HANDOFF.md` §4 |
| Commissioner restores backups, pauses feeds, switches the live feed, upgrades the server | Server operations are operator-only today (OPS class); restore runbook is offline | `DATA_FLOW_MAP.md` §1 (OPS), `deploy/server/RESTORE.md` |
