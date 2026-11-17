# TheWarRoom — the real app's UI/UX, as of `origin/main` 64e0507fa

**Written:** 2026-11-17 by Tom (Claude on the Beelink). Read from source (`git archive origin/main`), not from memory or older plans. Where this disagrees with the code, the code wins (`docs/architecture/desktop-modes.md` rule 2).
**Pairs with:** `V3-STUB-INVENTORY.md` and `V3-STUB-PLAN.md`.

## 1. In one paragraph
The real desktop app is one binary with two modes. **Thin (hosted) mode is the product**: the window holds no league database and reads everything from the league server through the command catalog. It opens on **Franchise HQ**, and also has **League Pulse** (Now, Race), **War Room** (Trade desk, Market, Draft room, Plans) and a hosted work room (lineup, roster, draft, rules, bids, waivers, trades, results, next season). **Local mode is legacy and frozen**: it has the full shell (Home, rail, command bar, inspector, status strip) and the **Control Room** — none of which run in thin mode yet. So for v3: HQ, War Room and Pulse are largely real; Home and the Control Room exist only as local-mode code to port; the app-level Home with boards, the Docket, the assistant and pop-out windows do not exist anywhere.

## 2. The two modes (what mounts where)
| | Thin (hosted) — the product | Local (all-in-one) — frozen |
|---|---|---|
| Entry | `control/NativeStartup.tsx` → `NativeAccount.tsx` (sign-in, authenticator, recovery, devices) → `commissioner/HostedLeague.tsx` | `TargetMount.tsx` → `shell/TargetApp.tsx` |
| Owner lands on | Franchise HQ (`shell/NativeFranchiseHQ.tsx` in `HQShell`) | Home (`shell/HomeBrief.tsx`) |
| Screens | HQ · League Pulse (`shell/NativeLeaguePulse.tsx`) · War Room (`shell/WarRoomNative.tsx` → `WarRoom.tsx`) · hosted work room tabs (`HostedLeague.tsx` + `HostedWorkflows.tsx`) | Home · War Room · Franchise HQ · League Pulse · Control Room (`control/room/ControlRoom.tsx`) · command bar · inspector · status strip |
| Commissioner with no team | `HostedCommissionerLeague.tsx`: rules + league-wide Pulse | Control Room |
| Data | `data/hostedReads.ts` (one subscriber, identity fence, change topics over `/desktop/changes`) + `data/query.ts` | Wails bindings (`data/provider.ts` picks Wails when `window.go` exists) |
| Writes | `commissioner/hostedRoom.ts`: preview → confirm → execute → reread, request key + expected revision | `commands/registry.ts` dispatch + feature `api.ts` |
| Rule | Grows until it carries everything | Do not grow it (`desktop-modes.md`) |

## 3. Screen map
| Node | Tabs on `main` | Thin component | Local component | Locked mock (look authority) | Palette |
|---|---|---|---|---|---|
| Home | Brief (3×2 tile grid, 13 tiles), Depth charts, NFL stats | — (none in thin) | `HomeBrief.tsx`, `homeTiles.tsx`, `homeLayout.ts`, `homeReads.ts` | `docs/ui/mocks/home-mock-v5.html` | slate |
| Franchise HQ | Lineup and roster · Contracts and cap · My moves (a queue control, not a tab) | `NativeFranchiseHQ.tsx`, `HQShell.tsx`, `HQBoard.tsx`, `HQPanels.tsx`, `HQInspector.tsx`, `HQRelease.tsx`, `HQNativeWhatIf.tsx`, `HQMoveHistory.tsx`, `hqNativeLineup.tsx` | `FranchiseHQ.tsx`, `contracts/Contracts.tsx` | `franchise-hq-mock-v10.html` (`docs/architecture/hq-wire.md`) | cool grey |
| War Room | Trade desk · Market · Draft room · Plans | `WarRoomNative.tsx` → `WarRoom.tsx`, `WarRoomTrade.tsx`, `WarRoomMarket.tsx`, `WarRoomDraft.tsx` (embeds `draftroom/DraftRoom.tsx`), `WarRoomPlans.tsx`, `WarRoomDrawers.tsx`, `WarRoomInspector.tsx`, `WarRoomScouting.tsx` | same files, local branch | `warroom-mock-v6.html` (`warroom-wire.md`; CSS scoped `wr-v6`) | smoky |
| League Pulse | Now · Race (Archive is ruled, not in `nodes.ts`) | `NativeLeaguePulse.tsx`, `PulseNow.tsx`, `PulseRace.tsx`, `PulseRaceDrawers.tsx`, `PulseSchedule.tsx`, `now/LiveGame.tsx` | `now/Now.tsx`, `race/Race.tsx` | `league-pulse-mock-v14.html` | jet black |
| Control Room | Jobs · Situations · Rules · Season desk · Fix · Picks · People · League · Server (same nine as v3) | — (thin commissioner gets `HostedRules.tsx`, `HostedWorkflows.tsx` only) | `control/room/*View.tsx`, `openers.tsx`, `guide.ts`, `roomReads.ts`, `EditorTray.tsx`, `search.ts` | `control-room-mock-v1.html` | functional first |
| Hosted work room | Set lineup · Roster and reserves · Draft room · League rules · Free-agent bids · Waivers · Trades · Results and standings · Next season | `HostedLeague.tsx` (HostedWorkRoom), `HostedWorkflows.tsx` (generic owner/commish task forms) | — | none (plain forms) | — |

Rail: `nodes.ts` has six fixed nodes (Home, War Room, HQ, Pulse, Control Room) with fixed leaves; `ShellRail.tsx` + `rail.css` (rail-mock-v1 option 3). Routes are `#/node/workspace` (`routes.ts`).

## 4. Building blocks that already exist — reuse these, do not rewrite them
| Need | Use | File(s) | Rule it carries |
|---|---|---|---|
| Any clickable action | `Act` (variants rail/tab/icon/text/button/card/command/surface) over the command registry | `commands/Act.tsx`, `commands/registry.ts`, `browser.gen.ts` (65 UI-local ids: nav.open, inspector.*, lineup.*, watch.add, game.follow, density.set…) | Every action has one named definition; UI-local actions never gain server authority (`command-spine.md`) |
| A server read (thin) | `createReadQuery` / `joinRead` | `data/query.ts`, `data/hostedReads.ts`, `commands/reads.gen.ts` | One reader per scope, no stacked reads, late replies fenced, identity change clears private state (`frontend-reads.md`) |
| A server change (thin) | hosted room / workflows | `commissioner/hostedRoom.ts`, `HostedWorkflows.tsx`, `HostedReview.tsx`, `forms.ts` (`newKey`) | preview → explicit confirm → execute → reread; expected revision + request key |
| Heavy (G3) confirm | hold to fire | `commissioner/HoldToFire.tsx`, `hold.ts` | ~600 ms hold; used today by HQ release, hosted lineup, Market bids, HostedReview, PresetCompare, SeedTray, ChangeBar |
| Commissioner tools | act table + fields + tray | `commissioner/actTable.ts` (every `CM-xxx` command with typed fields), `ActFields.tsx`, `CommissionerActTray.tsx`, `actBody.ts`, `actReadings.ts` | A new tool is a new row; paths mirror the Go Raw* types |
| Commissioner help | situations guide | `control/room/guide.ts` | Each step names its command and marks it B (built), R (ruled), G (gap) |
| Rule editing | typed rule input + preview | `RuleInput.tsx`, `RuleChangeForm.tsx`, `ruleDraft.ts`, `useChangePreview.ts`, `HostedRules.tsx`, `PresetCompare.tsx` | Only commissioner/co-commissioner see Change; server authorises and records |
| Player panel | shared inspector reads | `shell/warroomInspectorReads.ts`, `hqPlayerData.ts`, `HQInspector.tsx`, `WarRoomInspector.tsx`, `cards/PlayerCard.tsx`, `playerModel.ts`, `PlayerPhoto.tsx` | One read path for HQ + War Room (#315); missing facts stay absent |
| Entity popovers | entity inspector | `shell/EntityInspector.tsx` (franchise, offer, pick) | Center cards are a ruled exception to inspector-only |
| Private lists | personal store | `personal/PersonalLists.tsx` (NoteEditor), `WatchButton.tsx`, `shell/playerPersonal.ts`; server `player.note.*`, `player.watch.*`, `draft.queue.*` | PRIV: owner only, no commissioner override, never cached league-wide |
| Lineup drag and autofill | placement | `shell/hqPlacement.ts`, `LineupEditor.tsx`, `LineupRoster.tsx`, `lineups.ts`, `eligibilityLine.ts` | Position tags are drag handles; bumped starters cascade; app never second-guesses a deliberate move |
| Trade building | builder state + drag + lens | `shell/tradeBuilderState.ts`, `warroomDrag.ts`, `warroomLens.ts`, `warroomTradeReads.ts` | Proposal is DERIV-PRIV; send = `trade.offer` preview/execute |
| Clocks and deadlines | clock kit | `clock/ClockStrip.tsx`, `CalendarPanel.tsx`, `ticker.ts`, `labels.ts`, `urgency.ts`, `deadlineRoute.ts`, `useLeagueClock.ts`, `localTime.ts` | Deadlines are read, never rebuilt from a weekday; one ticker |
| Alerts | alert builders | `alertsOf`, `tradeAlerts`, `seasonAlerts` in `shell/HomeBrief.tsx`; `data/parseAlerts.ts` | Alert colour is an accent, not a wash |
| Live football | live game | `now/LiveGame.tsx`, `now/liveContract.ts`, `now/liveState.ts`, `shell/hqLive.ts` | One visibility-aware poll / topic; failed feeds never paint a new verdict |
| Freshness and failure words | | `data/freshness.ts`, `data/plainFailure.ts`, `internal/offline` | "Not available yet" with the reason in the title; never fabricate |
| Look primitives | four channels | `look/channels.ts` (gravity G0–G3, urgency U0–U3, signals green/blue/amber/red/grey), `look/Glyph.tsx`, `look/Slots.tsx`, `governed/GovernedGlyph.tsx`, `look/look.css` | colour = meaning, frame = consequence, countdown = urgency, glyph = classification |
| Density | | `cards/Card.tsx` (DENSITIES), `shell/density.ts`, `commissioner/DensityScreen.tsx` | Density/theme switch < 16 ms, no React re-render (§18) |
| Search / command bar | | `commands/CommandBar.tsx`, `commands/search.ts`, `commandbar.gen.ts`, `control/room/search.ts` | Generated candidates; same dispatcher and confirmations |
| Developer wiring view | endpoint index | `registry/EndpointIndex.tsx`, `endpoints.gen.ts`, `docs/ui/endpoint-registry.csv` | Not for owners |

## 5. UX laws the code is held to
- **No page scroll.** Tables scroll inside their own frame. 1920×1080 first; checked at 2550×1352.
- **Drawers for detail** (side drawer for a task, bottom drawer for right-now data, nested drawer only for confirms, max stack 2). Player and team detail go to the inspector; center cards (scouting, player career, news, trade) are the ruled exception.
- **Commit controls only in trays and drawers**, never in read tables. G3 = hold.
- **Never invent a system state** for an owner judgment.
- **Look:** textured rubber grip, hard borders, hard offset shadows, hazard stripes. Never glow, bubbles, soft gradients, blur. Hover 1–2 px, < 80 ms.
- **Palettes per screen:** Home slate, HQ cool grey, War Room smoky, Pulse jet black. Amber = urgency, red = danger, green = done, blue = live or mine.
- **Speed (UI Direction §18, acceptance criteria):** inspector < 50 ms; trade card action < 100 ms UI; panel collapse 120 ms with no JS blocking; drag 60 fps with no refetch; lists virtualised; density/theme switch < 16 ms. Christopher: "snappy and fast is law".
- **R3 bids:** a placed bid is final; open bids count against cap.
- **Confidence scores never show in the UI. MFL IDs are strings.**

## 6. The command catalog on `main`
158 command names (45 reads). Hosted workflow exposure: 42 (13 reads + 29 changes). UI-local browser ids: 65. Families:
- **Owner reads:** `lineup.read`, `lineup.check`, `lineupdelta.read`, `native.assets.read`, `native.cap.check`, `native.reserve.check`, `native.roster.check`, `contract.quote`, `value.read`, `tradelens.read`, `projections.read`, `score.player.read`, `player.usage.read`, `player.agearc.read`, `player.note.read`, `player.watch.read`, `draft.read`, `draft.queue.read`, `bids.read`, `waivers.read`, `trades.read`, `moves.read`, `franchise.moves.history.read`.
- **League reads:** `standings.read`, `results.read`, `odds.read`, `bracket.read`, `schedule.read`, `schedule.franchise.read`, `week.read`, `calendar.read`, `native.directory.read`, `native.rules.read`, `nfl.games.read`, `nfl.game.read`.
- **Commissioner reads:** `commish.actor/controls/digests/jobs/leagues/ledger/ledger.filtered/market/notices/people/rules/rules.preset/schedule/corrections.read`, `stats.queue.read`.
- **Owner changes:** `commish.lineup.set` (owner lineup via workflow), `roster.reserve`, `roster.cut`, `contract.action.set`, `bid.place/raise/withdraw`, `waivers.claim`, `trade.offer/accept/decline/withdraw/appeal`, `draft.pick`, `draft.queue.set`, `player.note.set`, `player.watch.set/order`, `commish.franchise.direction.set`.
- **Commissioner changes:** rules, calendar, schedule, bracket, title, draft order and room, waivers/bids processing and awards, trades review/override/recuse/appeal ruling, corrections (`correct.*`, `commish.stat.*`, `commish.score.*`, `commish.correction.revert`), standings overrides, cap adjust/relief, contract edit, player position, reserve exception, people and seats (`commish.seat.*`, `commish.role.*`, `commish.abilities.set`, `commish.deputy.order`, `commish.visibility.set`), succession and votes, notices and what's new, league create/seed, `season.rollover`, `oline.auction.close`, `history.link`.
- **No screen anywhere:** `calendar.window.set`, `calendar.window.remove`, `commish.succession.accept/activity/decline`, `contract.action.set`, `history.link`, `native.roster.check`, `schedule.franchise.read`, `score.player.read`.
Full table with the files that call each: `~/fleet/runs/twr-v3-stubs-1117/data/cmd-usage.tsv`.

## 7. Domain status — what changed since the 10-30 data map
`docs/ui/data-catalog/DATA_FLOW_MAP.md` lists D01–D78 as of 10-30. Moved since (source-checked 11-17):
| Domain | 10-30 | Now |
|---|---|---|
| D20 cap runway | PARTIAL | EXISTS — `internal/commissioner/native_runway.go`, via `native.cap.check` |
| D32 tune-it | NEW | EXISTS — `internal/tradelens` returns `Tune` (max 6) |
| D13 usage, D26 age arc | NEW | Reads exist (`player.usage.read`, `player.agearc.read`); refuse until F51 server player values land |
| D38 watch/targets | PARTIAL | Hosted (`player.watch.*`, #334, #336) |
| D43 draft queue | PARTIAL | Hosted (`draft.queue.*`, #333) |
| D45 player notes | PARTIAL | Hosted (`player.note.*`, #334, private-entries table #325) |
| D22 franchise history | EXISTS (local) | Hosted (`franchise.moves.history.read`, #319) |
| D29 builder versions/pocket | NEW | Saved trade drafts PR #338 (merging 11-17 5:00 PM) |
| D06 playoff odds | NEW | `odds.read` on the server (read once a season + on league topic) |
| D11 projections | PARTIAL | Native range engine (`internal/projection`: floor/median/ceiling from the player's own weeks) |

## 8. Work in flight on 11-17 (tracker, 4:54 PM CT)
Claudebox (head brain) with Luna lanes: #338 saved trade drafts merging; `wins` lane (recent winning bids, job 9); `wrslop` (War Room front-end slop); F51 player values on the server (#304/#307, draft); F65 contracts not recomputed per lineup save (#312); F69 HQ My moves (#319); F73 server memory. Later list includes: hosted injury feed, ADP feed (Christopher's call), playoff odds/values on the server, commissioner sees all open claims, phone sign-in. **Do not start a v3 stub that overlaps a running lane; check the tracker (`~/Downloads/twr-usable-week-plan.html`) first.**

## 9. Known slop on `main` (do not copy its pattern)
| Id | What | State |
|---|---|---|
| S01 | Each screen has its own read file asking for the same things its own way | HQ + War Room player panel fixed (#315, #318); other screens later |
| S02 | Pulse NFL live bypasses the data port and calls the old local engine | found |
| S04 | HQ/War Room panels called the local database directly | fixed via shared read owner |
| S06 | Duplicated what-if drawer (desktop vs hosted), overlapping HQ subscriptions | F71 |
| wrplan | `WarRoomMarket.tsx` hosted path called the desktop pool provider | fixed by #329 |
| wrplan | SAVE TO POCKET / SEND permanently disabled | SEND fixed #324; pocket = #338 |
| wrplan | Plans tab repeated empty boxes | one honest empty state (#337) |
| wrplan | `warroomTradeReads.ts` re-parses standings after the parser | open |

## 10. v3 next to the real app, screen by screen
| v3 | Real app on `main` | Gap in one line |
|---|---|---|
| App Home above all leagues: boards, 61 pieces, free grid, edit/stack/full/undo, pop out | Per-league 3×2 grid of 13 tiles, local mode only | New app-level Home + per-person layout store; extend the TILES registry |
| Rail: app zone + league zone, shortcuts | Six-node rail (local), HQ shell rail (thin) | Add the app zone; keep leaves fixed |
| Docket / Needs you across leagues | One-league alert band (local) | One fan-out read over existing producers |
| Franchise HQ (3 tabs) | Thin HQ, 3 tabs | Contract decisions, stacked what-if, usage/arc data |
| War Room (pool, trade, draft, plans) | Thin War Room (market, desk, draft, plans) | Saved drafts (#338), recent wins, ADP feed, Plans store |
| League Pulse (now, race, archive) | Thin Pulse (now, race) | My Road, stat changes, per-play delta, Archive |
| Control Room (9 tabs, Jobs engine, ask bar) | Local Control Room with the same 9 tabs | Port to thin; 7 job producers; assistant |
| Player/team/club/transaction cards on every name | Entity inspector for team/offer/pick | Extend EntityInspector |
| Assistant drawer | AI helpers act via scoped tokens + MCP CLI only | In-app client for the same door |
| Status bar links, spoiler delay | Status strip + freshness | Health read; client pref |
| First-login tour, labels | — | Small, client-only |
