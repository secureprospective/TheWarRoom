# Trade Floor trace — 2026-11-05 (incomplete)

This is a preliminary trace, not a deletion authorization. Reachability of every
component, command, IPC method, registry row, and test has not been established.
No registry rows have been cut or moved. Slices 1 and 2 wiring is recorded below.

## Slice 1 — local War Room draft mount

`WarRoomDraft` now lazy-mounts the existing `draftroom/DraftRoom` for the local
embedded app-league shell. It passes the selected league to the existing
`leagueState.choose` read. That room still owns the clock, board, commissioner
permission (`people.mayChangeRules`), owner turn, polling, tray and commands;
no second action implementation or IPC path was added. The old route remains
until the later cut. MFL mirror and hosted-server draft readings are unchanged.

The locked v6 mock has no commissioner-action area (its pause is a mock-draft
control). The local draft uses a War Room action card/header and existing button
style, with the original native room inside; it replaces, rather than duplicates,
the native mock-oriented board. The historical parity table below predates this
mount: all ten controls plus owner submission now share their old implementation
in this local War Room destination, under their original state/role conditions.

Evidence: TypeScript clean; 56 tests across six targeted files including the
locked-style test; plain `make build` passed. The added War Room mount tests
exercise selected-league commissioner pause/confirmation and owner submission
without commissioner launchers. No backend code changed.

Live copy: War Room > Draft room opens on DISPLAY :5. The supported identity
relink command restored Commissioner on the disposable profile. Its 2027 rookie
draft has no start time and needs 197 game results from 2026, so it has no clock
or live-action launchers: a live pause/resume and a screenshot of active controls
remain unverified. No draft write was attempted. MFL Draft room was captured
read-only, with no action pressed. Captures/logs are external evidence under the
operator's scratch soak directory, `soak-1105/sol1`; the copied profile is deleted
at the end. This is not deletion authorization or full live parity acceptance.

## Slice 2 — incoming navigation repointed

Flow: HQ checks its held snapshot roster, then `nav.open` carries scoped trade
input (league, franchise, player) into shell state. `WarRoom` adds the player to
its existing `WarRoomDeal.give`; `WarRoomTrade` renders its existing catalog
asset in YOU SEND. Native HQ now enables this local what-if handoff too. No old
builder call, proposal, submission, new IPC or parallel trade implementation.
War Room workspace state now selects the desk/draft tab.

Navigation inventory (`git grep` of slice-1 HEAD, including indirect callers):

| Caller | Old destination | New destination |
| --- | --- | --- |
| `frontend/src/app/shell/hqReads.ts` via `HQInspector.tsx` | trade/trade-desk + old builder give | war/trade-desk + War Room deal give |
| `frontend/src/app/clock/deadlineRoute.ts` trade window | trade/trade-desk | war/trade-desk |
| `frontend/src/app/shell/presets.ts` Trade Season | trade/trade-desk | war/trade-desk |
| Same file, Draft Mode | trade/draft-room | war/draft-room |
| `frontend/src/app/commands/registry.ts` DRAFT phase | trade/draft-room | war/draft-room |
| `frontend/src/app/commissioner/LeagueLanding.tsx` startup and live draft (two sites) | trade/draft-room | war/draft-room |
| `frontend/src/app/control/room/ControlRoom.tsx` draft controls | trade/draft-room | war/draft-room |
| `frontend/src/app/shell/EntityInspector.tsx` pick link | trade/draft-room | war/draft-room |
| `frontend/src/app/registry/index.ts` placement resolution for endpoint commands/reports | Trade Floor Trade desk / Draft room | war/trade-desk / war/draft-room |

The placement aliases cover all 27 historical registry rows listed below,
including command-bar endpoint navigation through `placementRoute`. CSV
placements/dispositions remain unchanged pending slice-3 parity/parking. A new
navigation destination does not prove every historical report is rendered
there. The CLI catalog has no Trade Floor workspace routes; its five trade IPC
methods remain unchanged.

Remaining old navigation is the explicit slice-3 rail/rendering cut:
`nodes.ts` still declares trade; `TargetApp.tsx` generates its rail from
`nodeKeys` and retains its old route renderer; `HQShell.tsx` has a hard-coded
Trade Floor rail entry. Existing trade hashes still parse to that node.
`bindings.test.ts`, `shell.test.ts` and `leaguePicker.interaction.test.tsx`
exercise the retained node. No compatibility guards added. Generator vocabulary
in `frontend/scripts/gen-endpoints.mjs`, generated types and fixture tests keeps
historical placements. Changed incoming-link assertions live in windows,
shellRows, actRouting, draftParams, draftreports and registry tests.

No production incoming links to offers-history or counterparties were found;
those slugs remain in invalid-hash tests / generator vocabulary, not current
node workspaces. War Room NEGOTIATIONS displays pending offers, not offers
history or counterparties reports: **park those historical homes for slice 3**.
`OfferCard` reject/withdraw still invokes `trade.builder` (action, not navigation).
War Room has no equivalent saved-plan/action UI. Keep that dependency until
slice 3 assigns its disposition. Saved versions / pocket persistence also lack
a home. No plan/action deletion is authorized by this routing change.

Evidence: TypeScript clean; 176 tests in 11 targeted files pass; plain
`make build` passes. New tests cover scoped HQ input, invalid held roster,
requested draft tab, tab navigation, league isolation and placement aliases.
On DISPLAY :5, the disposable Legacy NFL copy's HQ ADD TO TRADE opened War Room
Trade desk with Butler, Adam in YOU SEND. External scratch evidence:
`soak-1105/sol2/hq.png` and `soak-1105/sol2/hq-player-in-warroom.png`.
Draft-tab opening was captured too; no active draft write was tested. MFL
buttons were not pressed. Live MFL/hosted handoff and every individual registry
report/commissioner launcher remain unverified. Own GUI stopped and disposable
profile deleted; PID 289151 untouched. Pull remains blocked by the missing
configured bundle remote. No merge or push.

## Rail discrepancy

`frontend/src/app/shell/TargetApp.tsx` renders its rail from `nodeKeys`, which
includes `trade` in `nodes.ts`. Home uses that shell. For a selected MFL league,
when War Room is selected and the snapshot is available, `TargetAppContent`
returns `WarRoom` early instead. `WarRoom.tsx` renders its own hard-coded rail:
`home`, `war`, `hq`, `pulse`, `control`. It does not include `trade`. App
leagues embed War Room in the `TargetApp` shell, so their rail is not the same
as the standalone MFL War Room rail.

## Confirmed live dependencies and duplicate surfaces

- KEEP `shell/OfferCard.tsx`: extracted unchanged from `TradeDesk.tsx` for
  `shell/EntityInspector.tsx`; the old desk also imports it until cut. Its
  rejection/withdrawal action still invokes `trade.builder`, so that action's
  destination must be resolved before removing the old builder. The default
  TradeDesk workspace is reached from the MFL `trade/trade-desk` route in
  `TargetApp.tsx`.
- KEEP `shell/tradeBuilderState.ts`'s parser: `shell/warroomTradeReads.ts`
  uses `parseBuilder` for War Room catalog and preview. Separately,
  `shell/hqReads.ts` uses builder state for `addHQTrade`, invoked by
  `shell/HQInspector.tsx`'s ADD TO TRADE button, and navigates to the old desk.
  This is a live cross-screen action that must be rewired, including carrying
  the selected player, not just redirected.
- KEEP `trades/api.ts`'s `targetTrades`, builder read and preview:
  `shell/WarRoomDrawers.tsx` reads MFL offers and
  `shell/warroomTradeReads.ts` reads and previews the War Room catalog.
  KEEP `data/parseTrades.ts`: the War Room offers drawer parses that reading.
- KEEP Go IPC `TargetTrades`, `TargetDraftTradeAccept`, `TargetTradeBuilder`,
  `TargetTradeBuilderRead`, and `TargetTradeBuilderPreview`: all are bound in
  `internal/composition/command_bindings.go` and listed in
  `internal/commandcatalog/desktop.go`; the brief explicitly preserves CLI
  registry methods. `internal/composition/control_alerts.go` also calls
  `TargetTrades`.
- `shell/WarRoom.tsx` uses `WarRoomTrade`, `WarRoomMarket`, `WarRoomDraft`
  and a negotiations drawer; this is the newer mock-oriented surface. The old
  `TradeDesk.tsx` and its builder are a separate trade surface. Their feature
  parity has not been established: the old desk has offer-plan actions and
  saved plans, while the new drawer displays offers.
- Old `market/Market.tsx` dispatches `draft-room` to
  `draftroom/DraftRoom.tsx`. The newer `WarRoomDraft.tsx` is a separate draft
  surface. The old room includes commissioner draft controls in
  `draftroom/DraftBoard.tsx`, `DraftClock.tsx` and `DraftTray.tsx`; do not
  assume the newer room implements the same controls. Registry CM-043,
  CM-044 and CM-045 require particular attention.
- `shell/EntityInspector.tsx` links pick inspection to the old draft route;
  `shell/presets.ts` routes Draft Mode and Trade Season to the old node;
  `commands/registry.ts` routes DRAFT phase there. Their tests and navigation
  behavior must be updated with the cut.
- The brief names Offers history and Counterparties as rail workspaces, but at
  this checkout `nodes.ts` contains only Trade desk and Draft room. The old
  workspace hashes appear in `shell/bindings.test.ts` and the old placement
  names remain in `scripts/gen-endpoints.mjs` and generated registry types.

## Registry inventory (not dispositions)

Exactly 27 rows have a placement beginning `Trade Floor ›` in
`docs/ui/endpoint-registry.csv`:

- Trade desk: M-033, M-034, C-06.2, X-04, M-199.
- Draft room: M-063, M-064, M-065, M-066, M-068, M-087, M-102, M-103,
  M-176, M-179, M-183, L-15–L-22, CM-043, CM-044, CM-045.

All are currently marked kept. These placements are not proof that the new
War Room actually serves each report or commissioner action. Compare actual
implementations and the locked mocks before choosing moved versus parked;
regenerate the registry after a decision. Other rows mention Trade Floor in
notes without using it as their placement.

## Commissioner draft controls: needs a War Room home

Static comparison of `draftroom/DraftRoom.tsx`, `DraftClock.tsx`,
`DraftBoard.tsx`, `DraftTray.tsx`, `trayAction.ts`, and `YourPick.tsx` against
`shell/WarRoomDraft.tsx`: the new room has none of the following action
controls. Each is **needs a War Room home**, not deletion-authorized:

| Old control | Old action / command | War Room Draft equivalent |
| --- | --- | --- |
| Open startup draft | `open` / `draft.startup.open` | None |
| Skip current pick | `skip` / `draft.skip` | None |
| Pause clock | `pause` / `draft.timer` | None |
| Resume clock | `resume` / `draft.timer` | None |
| Set clock minutes | `set` / `draft.timer` | None |
| Assign player to current or skipped pick | `assign` / `draft.assign` | None |
| Change an existing pick | `change` / `draft.change` | None |
| Insert extra pick | `insert` / `draft.insert` | None |
| Restore checkpoint | `restore` / `draft.revert` | None |
| Close draft | `close` / `draft.archive` | None |

The old room also has the owner's **Make this pick** action through
`draftRoom.pick` / `TargetNativeDraftPick`; the new room lists available
players and opens their inspector, but has no pick submission control. This
is not a commissioner action, but also needs a home before removal. The
new room's Run Mock Draft button is disabled, not a substitute.

KEEP the old room and its action/tray code pending a working replacement and
routing verification. The code implements preview and confirmation via the
shared commissioner change flow; this audit did not execute those actions or
prove they currently work end-to-end. In fact, `TargetApp.tsx`'s app-league
branch shows Not wired for `trade`, before the `Market` route can render the
old room. `LeagueLanding.tsx` routes startup and live draft controls to that
node, and `control/room/ControlRoom.tsx` routes draft controls there too.
These routes are not evidence that the app-league UI presently reaches the old
room. Do not silently claim that simply retaining the files preserves access.
The MFL `draft-room` path does enter `Market`, but the old room reads native
league state, so its live behavior also needs verification.

`draftroom/draft.ts` and parser utilities are independently KEEP: the new
War Room, Home, and commissioner hosted room import them. `draftroom/state.ts`
is also imported by LeagueLanding and ControlRoom to select the draft, and by
commissioner routing tests; replacing its navigation callers does not by
itself prove the state implementation dead.

## Still required

Complete transitive callers, all tests, CLI bindings and IPC/Go paths, and
row-by-row mock coverage. Assign defensible KEEP/DEAD outcomes and counts;
rewire live cross-screen actions; cut only proven dead code; update registry,
soak script, and tests. Run the requested gates and the copied-profile live
checks. This preliminary trace alone is insufficient to make those changes.
