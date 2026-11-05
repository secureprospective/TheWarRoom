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

- KEEP `shell/TradeDesk.tsx`'s `OfferCard`: `shell/EntityInspector.tsx`
  imports it for offer inspection. The default TradeDesk workspace itself is
  reached from the MFL `trade/trade-desk` route in `TargetApp.tsx`; a component
  extraction may be needed before cutting the workspace.
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

## Still required

Complete transitive callers, all tests, CLI bindings and IPC/Go paths, and
row-by-row mock coverage. Assign defensible KEEP/DEAD outcomes and counts;
rewire live cross-screen actions; cut only proven dead code; update registry,
soak script, and tests. Run the requested gates and the copied-profile live
checks. This preliminary trace alone is insufficient to make those changes.
