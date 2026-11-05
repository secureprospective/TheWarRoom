# Trade Floor trace — 2026-11-05 (incomplete)

This is a preliminary trace, not a deletion authorization. Reachability of every
component, command, IPC method, registry row, and test has not been established.
No code or registry rows have been cut or moved.

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
