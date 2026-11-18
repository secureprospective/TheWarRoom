# TheWarRoom desktop mock v3 — a plan for every stub

**Written:** 2026-11-17 by Tom (Claude on the Beelink). **Pairs with:** `V3-STUB-INVENTORY.md` (what each stub is and needs) and `UIUX-INVENTORY-2026-11-17.md` (what the real app has). **Checked against:** `origin/main` `64e0507fa`.

The point: v3 is a picture of the finished app. Most of what it shows already has an owner in the real code. A session that builds a v3 stub must extend that owner, not write a new one beside it. Every one of the 287 stubs has a plan line below.

## Rules for every session that builds from v3
1. **The mock is a picture, not code.** Nothing from `~/scratch/twr-v3-1117/js/*.js` or the HTML is pasted into `frontend/`. Match the look; write the code in the real owner.
2. **Find the owner first.** Look up the stub in this file. If it says Hosted or Partly, open the named file and extend it. If two files could own it, pick the thin-mode one (`desktop-modes.md`: local mode is frozen).
3. **One read owner.** Thin screens read through `data/hostedReads.ts` + `data/query.ts` (one subscriber, identity fence, late-reply fencing). No screen calls the server its own way (slop item S01).
4. **One command path.** Every change goes preview → confirm → execute → reread through `commissioner/hostedRoom.ts` / `HostedWorkflows.tsx` and the command catalog. G3 actions use `HoldToFire.tsx` (600 ms in `hold.ts`; the Control Room wiring map's 1.2 s is the mock's number). Undo is what the catalog says per command: 92 are `checked-reversal` (a commissioner reversal, `commish.correction.revert`, writes a new ledger row), 65 are `none` (draft picks, league create, notes). No client-side Undo toast after a sent command.
5. **One form engine for commissioner tools.** `commissioner/actTable.ts` (CM-xxx rows) + `ActFields.tsx` + `CommissionerActTray.tsx`. A new tool is a new row, not a new form.
6. **New server features use the seams** (`docs/architecture/future-seams.md`): commands, scopes, events, stream, storage, join. New private data = a new kind or table through the storage seam. No side doors.
7. **No engine math in React.** Odds, power, cap, values, lens, tune, projections and live fantasy points are server reads. Pure ordering or grouping (heat order, kickoff groups) is fine as one tested pure function.
8. **Never invent a system state** (UI Direction v2). If no read produces it, show the read's own "unavailable" words, not a made-up label.
9. **Privacy classes hold** (`DATA_FLOW_MAP.md` §1): PRIV never in a league cache; SEALED bids never feed a public model; DOT votes only from DOT members.
10. **Speed is law** (UI Direction §18): one live subscriber per topic, no poll per piece, virtualised lists, inspector < 50 ms, trade card action < 100 ms UI.
11. **Generated files are never hand-edited** (`*.gen.ts`, `wailsjs/`, `verbs.gen.go`); change the Go catalog and run `make bindings`.
12. **Ask before** anything marked Ruling that the Decisions table has not settled, any new table (migration), any owner-to-owner text, and anything outside the app (push, email).

## Build order
Christopher's order for the real app (tracker, 11-17): **Franchise HQ → War Room → Home → Control Room.** HQ and War Room are now mostly on server data. The v3 stubs slot in like this:

| Step | Area | What it takes | Size |
|---|---|---|---|
| 0 | Finish HQ + War Room gaps already in flight | Saved trade drafts #338, recent winning bids (lane `wins`), F51 player values (unlocks usage + age arc), contract decisions (`contract.action.set` on a screen), stacked what-if read | in flight / M |
| 1 | **Home (app level)** | D58 per-person layout store (approved 11-17; storage seam migration), boards + pieces as an extension of `homeTiles.tsx` TILES, cross-league fan-out reads (week, today, cards), Docket read (one fan-out over existing producers), alert band in the shell | L |
| 2 | Shell | App zone in the rail, entity cards (extend EntityInspector), search results for players/teams/leagues, status bar health | M |
| 3 | **Control Room in thin mode** | Port `control/room/*` onto `hostedReads` + `hostedRoom`; every tool is already an actTable row. Then missing job producers (D60). Read `~/fleet/runs/twr-v3-1117/CONTROL-ROOM-WIRING.md` first (job engine behaviour, 10 mistakes, 4 open decisions) | L |
| 4 | Pulse extras | My Road (`schedule.franchise.read`), stat changes (D51 hosted read), per-play fantasy delta (D08/D09 server), Archive tab (D47) | M |
| 5 | New domains | Plans (D39), trophies (D49), week story (D50), depth charts (D53), stats table (D54), assistant (D74), pick ops (D67), permission sets (D65), pop-out windows | L each; each waits on a ruling or a feed |

## Decisions (Christopher, 2026-11-17: "go with yours on all 6")
| # | Question | Decided |
|---|---|---|
| 1 | v3 lets an owner lower or withdraw a placed bid | **R3 stands: a placed bid is locked.** The server refuses `bid.withdraw`; the app offers neither lowering nor withdrawal for placed bids. |
| 2 | Phone data rules on desktop | **Yes.** Desktop shows status facts (Out · hamstring, practice, depth), no written news, no ROST %, no trending adds. Projections stay: `projections.read` is our own range engine (D11). |
| 3 | Server operations in the Control Room | **Commissioners see read-only server health. Restore, feed pause/switch and upgrade stay operator-only (Christopher).** |
| 4 | Per-person layout for the app-level Home | **Approved: a per-person layout table on the league server (D58, storage seam, new migration).** Layout follows the person to any device. |
| 5 | New owner commands v3 implies (claim reorder/cancel, decline note, review-window extend, pick ops D67, house-rule firing log) | **Approve each one when its screen is next.** Ask then, one at a time. |
| 6 | Where this plan lives | **In the repo** (`docs/ui/v3/`) with a pointer in `CLAUDE.md`. |

## Plans — pieces
| Stub | State | Plan |
|---|---|---|
| `franchise` | Hosted | Reuse the HQ/War Room header readers (readNativeHQ, readWarRoomHeader). Power comes from odds.read, cap from native.cap.check; there is no power.read or cap.read (v3 names are wrong). Render only. |
| `race` | Hosted | Reuse shell/pulseReads.ts + PulseRace. Run-in win % per game needs D05 per-game win probability; show odds.read words when absent. No second race model. |
| `docket` | Partly | Build ONE server read that fans out the existing producers (alertsOf, commish.jobs.read, moves.read needs_you, trades.read, bids.read, waivers.read, calendar.read) per league; client merges leagues. Read marks = new PRIV personal-store kind via the storage seam. Do not compute lanes in React from raw reads in each piece. |
| `week` | Partly | Client fan-out of results.read + standings.read + week.read per membership through hostedReads (one subscriber per league). No new server read until fan-out is measured against §18. |
| `today` | Not on main | Fan-out lineup.read per league + nfl.games.read once; group by kickoff client-side. Exposure ('in 3 leagues') is derived from those reads; no new store. |
| `cards` | Partly | League list from the account memberships (NativeAccount view) + commish.leagues.read; per card standings.read and the docket count. Reuse leaguePicker.tsx rows. |
| `deadlines` | Hosted | Reuse clock/ (ClockStrip, labels, urgency, deadlineRoute). Reminder = PRIV personal kind (D57) shown only in-app (no push, no email). |
| `cjobs` | Local only | Port JobsView/jobs.ts to thin mode via hostedReads; cross-league = fan-out commish.jobs.read over leagues where the role is commish. The v3 Jobs engine (85-control.js) is a design reference for lanes, not code to copy. |
| `shortcuts` | Local only | Reuse SHORTCUTS + ☆ pins from homeTiles.tsx; persistence moves to the D58 layout store with the boards. |
| `notes` | Partly | Reuse personal/PersonalLists NoteEditor and player.note.* for per-player notes. A free scratch pad is a new PRIV kind (storage seam); ask before adding it. |
| `clock` | Hosted | Reuse clock/ticker.ts + useLeagueClock; no new timer per piece (one ticker). |
| `matchup` | Hosted | Reuse PulseNow matchup + Home tile Matchup; win chance from odds.read (complete-pair rule in hq-wire.md). |
| `board` | Hosted | Reuse PulseNow scoreboard + PulseSchedule. Nothing new. |
| `nfl` | Hosted | Reuse now/LiveGame.tsx (field + plays) on nfl.games.read/nfl.game.read. One live subscriber (hostedReads topic), never a poll per piece. |
| `pbp` | Partly | Extend LiveGame scoring plays to every play only when D08 full play-by-play lands on the server; per-play fantasy delta is D09 (server), never computed in React. |
| `ingame` | Hosted | Reuse LiveGame FOLLOWING + lineup.read. Done when Pulse is. |
| `statchg` | Not on main | Needs a hosted read over the existing app/control CompareAlerts + stats.queue.read (D51). Build in the Pulse lane; label provisional vs final (D75). |
| `winp` | Partly | Reuse Home WinChance. Mid-game update needs the server to recompute odds on the scores topic; until then show pre-game odds with its words. |
| `heat` | Not on main | Pure client ordering of results.read + odds.read (closeness × time left). No server work. Write it as one pure function with a test. |
| `standings` | Hosted | Reuse PulseRace standings. |
| `power` | Hosted | Reuse PulseRace power + Home tile pwr; components and horizons are D05 (powerrankings), never re-derived in UI. |
| `bracket` | Hosted | Reuse PulseRace bracket + PulseRaceDrawers. |
| `road` | Server only | Wire schedule.franchise.read (exists, no screen) into PulseRace My Road. Small job. |
| `form` | Partly | Hosted form from results.read (HQ map row: derive from results, no desktop research). One helper shared by HQ form bars, Home tile and this piece. |
| `trophy` | Not on main | Not on main. D49 in app/archive; award command through the commands seam with a ledger event. Wait for the Archive lane; ask Christopher before building (social builds are seams, not features yet). |
| `archive` | Not on main | Expose app/archive facts as a hosted read (D47); eras/stories D48 later. Archive is a Pulse tab in v3 but not in nodes.ts: add the tab only with the read. |
| `wire` | Hosted | Reuse Home tile wire (commish.ledger.filtered.read) and HQMoveHistory (franchise.moves.history.read). Kind chips = D23 filters already in LedgerFilters.tsx. |
| `lineup` | Hosted | Done in thin HQ (LineupEditor, hqPlacement, hqNativeLineup). Owner save is commish.lineup.set through the hosted workflow (the v3 'editor.save, name not identified' note is resolved). |
| `roster` | Hosted | Done in thin HQ (HQPanels, HQRelease G3 hold). Reuse; do not rebuild the taxi/IR drawers. |
| `caprun` | Hosted | Done: native.cap.check runway (native_runway.go). Reuse HQ runway renderer. |
| `contracts` | Partly | Hosted HQPanels has the cap/contract grid; contracts/Contracts.tsx is the local version. Extension/tag/let-walk need contract.action.set on a screen (exists server side). Port, don't fork. |
| `expiring` | Server only | Wire contract.action.set into HQ Decisions due (HQ map: project calendar.read events into the decisions list). Same form as cmGo, G3 hold. |
| `deadcap` | Partly | Dead rows come from native.cap.check; verify the hosted payload carries row-level dead money (HQ map row). Render in HQ runway, not a new read. |
| `whatif` | Partly | Hosted What-if is single-cut (HQNativeWhatIf). Stacked moves need a server stacked-quote read (deadcap + runway engines); one what-if drawer only (F71 merged the duplicate). |
| `picks` | Hosted | Reuse draft.read + EntityInspector pick. Pick values from value.read (pickvalue). |
| `pcard` | Partly | One player card: HQInspector/WarRoomInspector share warroomInspectorReads + hqPlayerData. Usage/age arc reads exist but refuse until F51 player values land on the server. Status facts (Out · hamstring, practice) come from injuries ingest (D14); no prose news. |
| `offers` | Hosted | Reuse HostedWorkflows trades task (trade.accept/decline/withdraw) and WarRoomDrawers. Trade card action < 100 ms UI (§18). Decline note is a new optional argument; ask before adding. |
| `desk` | Hosted | Done in thin War Room (#324 send). Versions/pocket = saved trade drafts (PR #338, owner-only, 50 cap). Do not add a second proposal state. |
| `review` | Hosted | Reuse HostedWorkflows trade.review/recuse/appeal; DOT votes only from DOT members (P2). |
| `pool` | Hosted | Done in thin War Room Market (#329 bids, #336 targets). Recent winning bids = D35 from revealed history only (lane 'wins' running 11-17). Never count sealed bids. |
| `bids` | Hosted | Done (bids.read, bid.place/raise). R3: a placed bid is final; the server refuses `bid.withdraw`, and the app offers neither lowering nor withdrawal on placed bids. |
| `waivers` | Partly | Claims via HostedWorkflows waivers.claim. Claim reorder and claim cancel have no command: add through the commands seam (internal/commissioner) with tests; ask before adding. |
| `draft` | Hosted | Done: embedded DraftRoom + draft.queue.* (#333). ADP/tiers wait on a server ADP feed (Christopher's call on which feed). |
| `plans` | Not on main | D39 private plans need a store (migration → Christopher). Until then one honest empty state (#337 did this). Nothing executes from Plans. |
| `looking` | Hosted | Done: WarRoomPlans Past offers/Counterparties over trades.read. |
| `nflnews` | Not on main | Per phone data ruling: status facts, not prose. Needs a hosted injuries/status read (D14; 'hosted injury feed' is on the later list). No ROST %, no trending adds. |
| `headlines` | Not on main | D50 week story: not on main. Later; frozen per week, server-written, never AI-invented numbers. |
| `depth` | Not on main | D53 depth charts: no feed on main. Christopher's call on source (Sleeper/ESPN). Park. |
| `nflstats` | Not on main | D54 league-scored stats table: needs a hosted read over fantasyscore weekly stats. Saved views = PRIV kind. Later. |
| `describe` | Partly | Form path exists (RuleChangeForm, ruleDraft, useChangePreview: preview → confirm). The words-to-proposal assistant is D74 over scoped AI tokens (scopes seam); never a second rule writer. |
| `house` | Partly | House rules are rules (native.rules.read/TargetNativeHouseRules + commish.rules.set). Firing log = ledger filter of rule events, not a new log (events seam). |
| `layout` | Not on main | League surface on/off/auto (fold 1) is a new league setting: add as rule keys in leaguerules so the commissioner preset machinery covers it. Ask before building. |
| `statq` | Local only | Port control/room openers stat queue to thin; commands exist. |
| `rules` | Hosted | Thin HostedRules exists; local RulesView + PresetCompare are the fuller board. Port RulesView onto hostedRoom; one rule input (RuleInput.tsx). |
| `season` | Local only | Port SeasonView; wire calendar.window.set/remove (catalog only today) into the same window editor as move. |
| `fix` | Local only | Port FixView + UndoForm + actTable correction rows. Corrections are new ledger rows (never edits). |
| `people` | Local only | Port PeopleView/LeaguePeople. Never-delegable: role grant/revoke, deputy order, succession settings. |
| `ledger` | Local only | Port LeagueLedger + LedgerFilters onto hostedReads. |
| `notices` | Local only | Port NoteForm/notes.ts. In-app only (no pestering outside the app). |
| `health` | Local only | Same producer as Jobs (commish.jobs.read + people last seen). No second engine; v3 R.health already reads Jobs. |
| `server` | Local only | Commissioner view of server health is OPS data: serverstatus is an operator socket. Show read-only status only; restore/upgrade/feed switches stay operator-only (decided 11-17). |

## Plans — screen tabs
| Stub | State | Plan |
|---|---|---|
| `hq:lineup` | Hosted | Done in thin HQ. Keep; match v3 look only where the HQ v10 mock is silent. |
| `hq:contracts` | Partly | Mostly done; add contract.action.set decisions and stacked what-if (see pieces). |
| `hq:moves` | Hosted | Done (#319 franchise.moves.history.read + moves.read). |
| `war:pool` | Hosted | Done (#329, #336); recent wins lane running; min-bid/value columns per wrplan job 3. |
| `war:trade` | Hosted | Done (#324); saved drafts #338 merging. |
| `war:draft` | Hosted | Done (#333); ADP waits on a feed ruling. |
| `war:plans` | Partly | Empty state until D39 store ruling. |
| `pulse:now` | Hosted | Done in thin Pulse. Per-play fantasy delta + stat changes are server work (D08/D09/D51). |
| `pulse:race` | Hosted | Done in thin Pulse; why-moved notes (D05) and tiebreak step (D03) are partial engine work. |
| `pulse:archive` | Not on main | New tab + hosted archive read (D47). After Home. |
| `control:jobs` | Local only | Port JobsView to thin mode (Christopher's order: after Home). Jobs come from commish.jobs.read; 7 producers still missing (D60). |
| `control:situations` | Local only | Port SituationsView + guide.ts as is; guide marks B/R/G tell which steps are built. |
| `control:rules` | Partly | Port RulesView/PresetCompare onto hostedRoom. |
| `control:season` | Local only | Port SeasonView; add calendar.window.set/remove. |
| `control:fix` | Local only | Port FixView. |
| `control:picks` | Local only | Port PicksView; D67 pick ops (grant/swap/forfeit/condition/void/create) are ruled and not built: commands seam. |
| `control:people` | Local only | Port PeopleView; permission sets D65 not built. |
| `control:league` | Local only | Port LeagueView; league colour + layout surfaces are new rule keys (ask). |
| `control:server` | Local only | Read-only status in thin; actions operator-only (ruling). |

## Plans — shell
| Stub | State | Plan |
|---|---|---|
| `shell:home-boards` | Not on main | Biggest new build. Replace per-league homeLayout with an app Home whose layout lives in a D58 PRIV store (storage seam: new table; approved 11-17). Pieces = registry entries that declare their reads (homeTiles TILES already do this: extend TILES, do not start a second registry). |
| `shell:rail` | Local only | Reuse ShellRail + leaguePicker. Add the app zone (Home + boards) above the league zone (ruled 11-16). |
| `shell:league-chip` | Partly | Reuse leaguePicker; waiting/live counts come from the docket read. |
| `shell:search` | Local only | Reuse CommandBar + commands/search.ts + commandbar.gen.ts; add players/teams/leagues as result kinds (D76). |
| `shell:needs-you` | Partly | Count = docket read; drawer = Docket. |
| `shell:alert-band` | Partly | Reuse HomeBrief alertsOf()/parseAlerts; lift it into the shell for every screen (UI Direction: one-row alert band outside the layout, all screens). |
| `shell:assistant` | Not on main | D74: AI helpers already act through scoped tokens + MCP (cmd/twr). An in-app drawer is a client for that same door; it prepares, the owner confirms (ruled). Never a second command path. |
| `shell:feeds` | Mock only | Mock-only. The real equivalent is registry/EndpointIndex (developer view). Do not ship. |
| `shell:account` | Hosted | Reuse NativeAccount (devices, recovery, sign-out). |
| `shell:status-bar` | Partly | Reuse StatusStrip + data/freshness.ts. Spoiler delay is a client preference (UI-SAVE). |
| `shell:inspector` | Hosted | Reuse HQInspector/WarRoomInspector; one inspector read module (warroomInspectorReads). |
| `shell:popout` | Not on main | No window code exists. Wails supports extra windows; keep-on-top only for live scoring (ruled). New, after Home. |
| `shell:cards` | Partly | Extend EntityInspector (team, offer, pick) with player/club/transaction kinds; center card exception is ruled. One card component per kind. |
| `shell:tour` | Not on main | New, small, client-only. Last. |
| `shell:mock-controls` | Mock only | Mock-only. Do not ship. |

## Plans — actions, by area
Each action points at its parent piece or screen above; the line says only what is specific to it.

### Home layout and boards
| Stub | Kind | State | Plan |
|---|---|---|---|
| `edit` | UI | Local only | Part of shell:home-boards: Biggest new build. |
| `undo` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `redo` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `board` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `boardAdd` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `boardNew` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `boardMenu` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `bm` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `boardReset` | UI·S | Local only | Part of shell:home-boards: Biggest new build. |
| `boardClose` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `boardMore` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `pset` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `pflip` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `stk` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `premove` | UI·S | Local only | Part of shell:home-boards: Biggest new build. |
| `pgear` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `pmenu` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `pm` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `fullOpt` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `pvApply` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `pvCancel` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `bringBack` | UI | Not on main | Part of shell:popout: No window code exists. |
| `railBoard` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `pmSel` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `swPick` | UI·S | Local only | Part of shell:home-boards: Biggest new build. |
| `dkAdd` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `library` | IN | Local only | Part of shell:home-boards: Biggest new build. |
| `asstApply` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |
| `spmenu` | UI | Not on main | Part of shell:home-boards: Biggest new build. |
| `spm` | UI·S | Not on main | Part of shell:home-boards: Biggest new build. |

### Shell (top bar, rail, status, cards, assistant)
| Stub | Kind | State | Plan |
|---|---|---|---|
| `inspect` | IN | Hosted | Part of shell:inspector: Reuse HQInspector/WarRoomInspector; one inspector read module (warroomInspectorReads).. |
| `inspClose` | UI | Hosted | Part of shell:inspector: Reuse HQInspector/WarRoomInspector; one inspector read module (warroomInspectorReads).. |
| `switcher` | IN | Partly | Reuse leaguePicker; groups need the docket read for waiting/live. |
| `account` | UI | Hosted | Part of shell:account: Reuse NativeAccount (devices, recovery, sign-out).. |
| `acct` | IN+OUT | Partly | Labels toggle = client pref; devices/sign-out = NativeAccount (/v1/auth). AI provider key: none on main; AI access is scoped tokens (D74). Ask before storing provider keys. |
| `map` | UI | Not on main | Part of shell:tour: New, small, client-only. |
| `cmdk` | IN | Local only | Reuse CommandBar; command candidates are generated (commandbar.gen.ts). Add entity results (D76). |
| `docket` | IN | Partly | Part of p:docket: Build ONE server read that fans out the existing producers (alertsOf, commish.jobs.read, moves.read needs_you, trades.read, bids.read, waivers.read, calendar.read) per league; client merges leagues. |
| `dkLane` | UI | Not on main | Part of p:docket: Build ONE server read that fans out the existing producers (alertsOf, commish.jobs.read, moves.read needs_you, trades.read, bids.read, waivers.read, calendar.read) per league; client merges leagues. |
| `dkLg` | UI | Not on main | Part of p:docket: Build ONE server read that fans out the existing producers (alertsOf, commish.jobs.read, moves.read needs_you, trades.read, bids.read, waivers.read, calendar.read) per league; client merges leagues. |
| `dgo` | IN+OUT | Not on main | Read marks = PRIV personal kind (storage seam). Navigation part is nav.open. |
| `assistant` | IN+OUT | Not on main | Part of shell:assistant: D74: AI helpers already act through scoped tokens + MCP (cmd/twr). |
| `askSend` | OUT | Not on main | Part of shell:assistant: D74: AI helpers already act through scoped tokens + MCP (cmd/twr). |
| `ask` | OUT | Not on main | Part of shell:assistant: D74: AI helpers already act through scoped tokens + MCP (cmd/twr). |
| `tourStart` | UI | Not on main | Part of shell:tour: New, small, client-only. |
| `tourNext` | UI | Not on main | Part of shell:tour: New, small, client-only. |
| `tourSkip` | UI | Not on main | Part of shell:tour: New, small, client-only. |
| `cmAsk` | OUT | Not on main | Words → proposal is the assistant (D74). It prepares a commish.rules.set preview; the commissioner confirms. |
| `asstCm` | OUT | Not on main | Part of shell:assistant: D74: AI helpers already act through scoped tokens + MCP (cmd/twr). |
| `trInspect` | IN | Hosted | Part of shell:inspector: Reuse HQInspector/WarRoomInspector; one inspector read module (warroomInspectorReads).. |
| `stConn` | IN | Not on main | Per-server health: one row per league server from the existing serverversion preflight/status; no ping loop in the client (§18). |
| `stLive` | UI | Not on main | Part of shell:status-bar: Reuse StatusStrip + data/freshness.ts. |
| `stDelay` | UI·S | Not on main | Client preference (spoiler delay). Saved with the person, not the league. |
| `ent` | IN | Partly | One popover per kind, extending EntityInspector. Player card shows status facts (no prose), his leagues, actions that route to existing screens. |
| `devOut` | OUT | Hosted | Done: NativeAccount Revoke this device. Confirm first; no undo (a revoked token cannot come back). |
| `askC` | OUT | Not on main | Part of shell:assistant: D74: AI helpers already act through scoped tokens + MCP (cmd/twr). |
| `askKeep` | OUT | Not on main | Written request that becomes a job: new CMSH store row via storage seam + a job producer. Ask first. |
| `askHouse` | OUT | Not on main | Turns words into a house rule draft; the write is commish.rules.set through the rule form. Assistant prepares, commissioner confirms. |

### Franchise HQ
| Stub | Kind | State | Plan |
|---|---|---|---|
| `hqAct` | OUT | Hosted | Done: lineup save via commish.lineup.set with lineup.check (hqNativeLineup). |
| `decide` | IN | Server only | Part of p:expiring: Wire contract.action.set into HQ Decisions due (HQ map: project calendar.read events into the decisions list). |
| `decSet` | OUT | Server only | contract.action.set exists on the server with no screen. Wire it into HQ Decisions due with preview → confirm; v3 Undo toast becomes "change before the deadline". |
| `decUndo` | OUT | Server only | Changing a decision before the deadline is another contract.action.set, not an undo. |
| `wifAdd` | IN | Partly | Part of p:whatif: Hosted What-if is single-cut (HQNativeWhatIf). |
| `wifDel` | UI | Partly | Part of p:whatif: Hosted What-if is single-cut (HQNativeWhatIf). |
| `wifReset` | UI | Partly | Part of p:whatif: Hosted What-if is single-cut (HQNativeWhatIf). |
| `pick` | IN | Hosted | Done: EntityInspector pick + draft.read. |
| `remind` | OUT | Not on main | Reminder = PRIV personal kind (D57) surfaced in Needs you only. No push, no email (ruled). |
| `formWk` | IN | Partly | One week from results.read; same helper as the form piece. |
| `cmGo` | IN+OUT | Partly | Part of p:contracts: Hosted HQPanels has the cap/contract grid; contracts/Contracts.tsx is the local version. |
| `cmUndo` | OUT | Ruling | The catalog marks roster.cut as checked-reversal: it is undone by a commissioner reversal (commish.correction.revert, a new ledger row), not by an owner Undo toast. Real flow: preview → confirm (G3 hold). |

### War Room
| Stub | Kind | State | Plan |
|---|---|---|---|
| `trDir` | UI | Hosted | Part of war:trade: Done (#324); saved drafts #338 merging.. |
| `trFlt` | UI | Hosted | Part of war:trade: Done (#324); saved drafts #338 merging.. |
| `trVis` | UI | Hosted | Part of war:trade: Done (#324); saved drafts #338 merging.. |
| `trVer` | UI·S | Partly | A version is a saved trade draft (PR #338). Same store. |
| `trClear` | UI | Hosted | Part of war:trade: Done (#324); saved drafts #338 merging.. |
| `trTune` | IN | Hosted | Done: tradelens.read returns Tune suggestions (internal/tradelens). Keep. |
| `trToggle` | UI | Hosted | Part of war:trade: Done (#324); saved drafts #338 merging.. |
| `trSave` | OUT | Partly | Saved trade drafts (PR #338: owner-only, 50 cap, 16 KB). Use that command; do not keep a client pocket. |
| `trSend` | OUT | Hosted | Done: trade.offer via hosted workflow (#324). Keep. |
| `trLoad` | IN | Partly | Loads a saved draft or a live offer (trades.read) onto the desk; no new state. |
| `trPocket` | IN | Partly | Reads saved trade drafts (PR #338); re-check each with the existing trade.offer preview for stale flags. |
| `trNeg` | IN | Hosted | Part of war:trade: Done (#324); saved drafts #338 merging.. |
| `eWatch` | OUT | Hosted | Done: player.watch.set (#334). Keep. |
| `eDepth` | IN | Not on main | Part of p:depth: D53 depth charts: no feed on main. |
| `eBid` | IN | Hosted | Opens the War Room bid review (WarRoomMarket "Review bid"); same component, not a card copy. |
| `eBidGo` | OUT | Hosted | bid.place / bid.raise via hosted workflow (#329). Budget left = native.cap.check + open bids (D36: open bids count against cap). |
| `bidStep` | OUT | Hosted | Raise only (bid.raise). Lowering a placed bid breaks R3 (a placed bid is final; only being outbid ends it). Remove the minus step. |
| `bidDrop` | OUT | Hosted | R3: the server refuses `bid.withdraw`; the app no longer offers withdrawal on placed bids. |
| `clMove` | OUT | Not on main | No claim-reorder command. Add through the commands seam with a test, after Christopher says claims are reorderable. |
| `clDrop` | OUT | Not on main | No claim-cancel command. Same as clMove. |
| `offAcc` | OUT | Hosted | Reuse HostedWorkflows trade.accept with preview → confirm. v3 shows an 8 s Undo toast; the real flow confirms first, and reversal is a checked reversal by the commissioner (catalog Undo metadata). |
| `offDec` | OUT | Hosted | trade.decline exists. The optional note to the other owner is a new argument; ask before adding (it is owner-to-owner text). |
| `offWd` | OUT | Hosted | trade.withdraw via HostedWorkflows; confirm first, no client undo toast. |
| `offSet` | OUT | Hosted | Part of p:offers: Reuse HostedWorkflows trades task (trade.accept/decline/withdraw) and WarRoomDrawers. |
| `offCtr` | UI | Not on main | Part of p:offers: Reuse HostedWorkflows trades task (trade.accept/decline/withdraw) and WarRoomDrawers. |
| `pBid` | IN+OUT | Hosted | Same as eBidGo: one bid form. |
| `dqAdd` | OUT | Hosted | Done: draft.queue.set (#333). |
| `dqDel` | OUT | Hosted | Done: draft.queue.set (#333). |
| `dqMove` | OUT | Hosted | Done: draft.queue.set (#333). |
| `scout` | IN | Partly | Scouting card is a ruled center-card exception; content needs D44 PRIV store + ADP feed. Reuse WarRoomScouting.tsx. |
| `dPick` | OUT | Hosted | Done: draft.pick via DraftRoom YourPick. Keep. |
| `trAward` | OUT | Not on main | Part of p:trophy: Not on main. |

### League Pulse
| Stub | Kind | State | Plan |
|---|---|---|---|
| `pSel` | UI | Hosted | Part of pulse:now: Done in thin Pulse. |
| `pull` | IN | Partly | Bottom pulls follow the selected matchup (Pulse v14). Stat changes = D51 hosted read. |
| `whyMoved` | IN | Partly | D05 why-moved notes are engine work in powerrankings; UI reuses PulseRaceDrawers. |
| `tieBreak` | IN | Partly | D03 step-used narrative belongs in engine/race Order; UI shows its words. |
| `brkFull` | IN | Hosted | Part of pulse:race: Done in thin Pulse; why-moved notes (D05) and tiebreak step (D03) are partial engine work.. |

### Control Room
| Stub | Kind | State | Plan |
|---|---|---|---|
| `say` | OUT | Local only | Generic commissioner act box. Real: CommissionerActTray + ActFields over actTable rows (CM-xxx). Every v3 "say" type maps to an actTable row. |
| `jobOpen` | IN | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `jobDone` | OUT | Local only | Same as cjMark. |
| `cmTry` | IN | Local only | Part of control:rules: Port RulesView/PresetCompare onto hostedRoom.. |
| `cmCancel` | UI | Local only | Part of control:rules: Port RulesView/PresetCompare onto hostedRoom.. |
| `cmApply` | OUT | Hosted | Rule change via RuleChangeForm preview → confirm (exists). |
| `surf` | OUT | Not on main | League surface setting = new rule keys (fold 1). Ask before building. |
| `lcolor` | OUT | Not on main | League colour + badge = commissioner setting (ruled 11-16 yes). New rule key; avoid green/blue/amber/red. |
| `cmOk` | OUT | Local only | Part of control:people: Port PeopleView; permission sets D65 not built.. |
| `seatGo` | OUT | Local only | Seat tools exist (commish.seat.*); invite links go through the join seam (internal/accounts/join.go: one revocable per-league secret). |
| `copyLink` | UI | Not on main | Client-only copy of the join link. |
| `cmDone` | OUT | Local only | Part of control:people: Port PeopleView; permission sets D65 not built.. |
| `tOk` | UI | Local only | Reuse the commissioner form engine (actTable.ts + ActFields.tsx + hostedRoom preview → confirm → execute). One form component, never a per-tool form. |
| `preCmp` | IN | Local only | Part of control:rules: Port RulesView/PresetCompare onto hostedRoom.. |
| `preApply` | OUT | Local only | PresetCompare.tsx exists (local). Port onto hostedRoom; presets are setting bundles (ruled). |
| `houseNew` | OUT | Partly | House rule = rules text + commish.rules.set. Firing log is a ledger filter (events seam). |
| `schedTool` | IN | Local only | Part of control:season: Port SeasonView; add calendar.window.set/remove.. |
| `schedSwap` | OUT | Local only | Part of control:season: Port SeasonView; add calendar.window.set/remove.. |
| `brkTool` | OUT | Local only | Part of control:season: Port SeasonView; add calendar.window.set/remove.. |
| `titleTool` | OUT | Local only | title.set only sets the title label. Crowning by ruling = bracket.advance / commish.standings.override through the same act form. |
| `rollTool` | IN | Hosted | Done: season.rollover preview. |
| `rollRun` | OUT | Hosted | Done: season.rollover via HostedWorkflows + HoldToFire (G3). Keep. |
| `pkF` | UI | Local only | Part of control:picks: Port PicksView; D67 pick ops (grant/swap/forfeit/condition/void/create) are ruled and not built: commands seam.. |
| `pkMenu` | UI | Local only | Part of control:picks: Port PicksView; D67 pick ops (grant/swap/forfeit/condition/void/create) are ruled and not built: commands seam.. |
| `pkDo` | OUT | Partly | Move = correct.pick (exists). Grant/swap/forfeit/condition/void/restore are D67: ruled, not built; add through the commands seam in internal/commissioner. |
| `pkNew` | OUT | Not on main | Create pick = D67, ruled not built. Commands seam. |
| `permT` | OUT | Local only | commish.abilities.set exists. Permission sets (D65) are not built; never-delegable list applies. |
| `depAdd` | OUT | Local only | commish.role.grant (+ abilities). Never-delegable: role grant/revoke, deputy order, succession settings. |
| `depDel` | OUT | Local only | Part of control:people: Port PeopleView; permission sets D65 not built.. |
| `sucMove` | OUT | Local only | Part of control:people: Port PeopleView; permission sets D65 not built.. |
| `bkNow` | OUT | Local only | Local mode has backup.now; hosted backups are automatic (R2 offsite). A commissioner "backup now" on hosted is new and operator-side: ask. |
| `bkRestore` | IN | Ruling | Restore is operator-only today (deploy/server/RESTORE.md). Ruling needed before any commissioner restore. |
| `bkRun` | OUT | Ruling | Same as bkRestore. |
| `feedT` | OUT | Ruling | Feed control is operator-side (serverstatus). Ruling needed. |
| `feedSw` | OUT | Ruling | Truth-ladder override (ESPN → Sleeper → nflverse) is server policy. Ruling needed. |
| `upChk` | IN | Partly | Read-only update status can show; desktop client already has UpdateRequired.tsx. |
| `upRun` | OUT | Ruling | Server upgrade is operator-only (signed releases, releasetrust). Ruling needed. |
| `cjFilter` | UI | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjReopen` | UI | Not on main | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjSnooze` | OUT | Not on main | Snooze = PRIV per-actor state (D57). A job is computed; snoozing never hides it from co-commissioners. |
| `cjOpen` | IN | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjMark` | OUT | Local only | A job settles when its command runs; the producer drops it on the next read. A manual "settled" mark without a command is new: ask. |
| `cjTrade` | IN | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjTx` | OUT | Partly | Approve/veto = trade.review (DOT only); override = trade.override. "Extend the review window" has no command: ask Christopher, then add via the commands seam. |
| `cjStat` | IN | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjSq` | OUT | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjTie` | OUT | Local only | bids.award.override / waivers.award.override exist (SeasonView). Port. |
| `cjLine` | IN | Not on main | Lineup holes across the league = a new job producer in app/control alerts (one of the 7 missing producers, D60). |
| `cjLn` | OUT | Partly | Commissioner sets a lineup = commish.lineup.set (exists). "Nudge" = in-app notice (commish.notice.publish), never outside the app. |
| `cjCap` | OUT | Local only | Ruling tools exist: commish.cap.relief, rule.exception, commish.notice.publish (guide.ts lists them). |
| `cjNotice` | OUT | Local only | Part of control:jobs: Port JobsView to thin mode (Christopher's order: after Home). |
| `cjOrder` | OUT | Local only | draft.order.set/auto/pattern exist in actTable. Port. |
| `cfgEdit` | OUT | Hosted | Done in thin: HostedRules + RuleInput (typed value, timing, reason, preview → confirm). |
| `divTool` | IN | Local only | Part of control:rules: Port RulesView/PresetCompare onto hostedRoom.. |
| `divMove` | OUT | Ruling | Check whether division membership is a rules key (leaguerules competition) before adding anything; if not, it is a new commish command via the seam (ask). |
| `pfS` | UI | Local only | Part of control:people: Port PeopleView; permission sets D65 not built.. |

### Navigation only
| Stub | Kind | State | Plan |
|---|---|---|---|
| `closeLayer` | UI | Hosted | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `home` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `nav` | UI | Hosted | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `railCol` | UI·S | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `pickLeague` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `goLeague` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `star` | UI·S | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `cmdRun` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `etab` | UI | Hosted | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `pcol` | UI | Not on main | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `sset` | UI | Not on main | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `stWeek` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `eGo` | UI | Hosted | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `eTrade` | UI | Hosted | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `dlGo` | UI | Hosted | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `healthGo` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `hJob` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `cjHome` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `cjGo` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |
| `askGo` | UI | Local only | Navigation only: reuse nav.open / leagueState.choose / route tables. No data, no new state. |

### Mock only
| Stub | Kind | State | Plan |
|---|---|---|---|
| `feeds` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `noop` | UI | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `feedIndex` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `peek` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `peekClose` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `demo` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `dset` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `notesBuild` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
| `signin` | MOCK | Mock only | Mock-only. Do not ship. The real developer view is registry/EndpointIndex. |
