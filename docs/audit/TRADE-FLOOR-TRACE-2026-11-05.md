# Trade Floor cut — 2026-11-05

## Preserved destinations (slices 1–2)

- WarRoomDraft lazy-mounts the existing native DraftRoom. It owns permissions,
  clock, board, owner submission and commissioner tray; no parallel controls.
  The locked mock has no commissioner spot, so the mount uses a War Room action header.
- Commissioner controls retained: open, skip, pause/resume, set clock, assign,
  change, insert, restore and close, under the original role/state conditions.
- HQ roster-checked player input now enters the existing War Room deal, scoped
  by league/franchise. Deadline, preset, phase, commissioner and pick-inspector
  navigation all target War Room. Old hashes now normalize to Home.
- Keep trades/api, parseTrades, OfferCard, parser/receipt handlers, native draft
  code and all five trade IPC methods (including CLI). No Go code changed.

## Slice 3 — cut and disposition

Finding: the obsolete trade rail owns a duplicate builder UI and two routes. Cut `nodes.trade`,
both rail entries, the `TargetApp` desk branch and the old Market-to-native-draft branch.
Deleted `shell/TradeDesk.tsx` and `shell/TradeBuilder.tsx` after tracing their imports (including
indirect component imports in tests). Removed their UI-only tests/assertions, not the inspector,
receipt or parser coverage. Native DraftRoom now has one production mount: WarRoomDraft.

Preserved flow: WarRoomDrawers (MFL Negotiations) → selected offer ID → the existing
EntityInspector in the War Room inspector → OfferCard → trade.builder reject/revoke →
TargetTradeBuilder → receipt reads. The old desk’s move-read subscription moved into the
OfferInspector, so receipt actions update there without requiring the removed desk to mount.
TradePlan/EnvelopeRail still share handoff/removal. No MFL action was pressed during live checks.
Removed builder-only deal store, read/preview/save handlers; parser and retained owner/receipt
handlers remain in tradeBuilderState. War Room uses the existing API read/preview directly.
The five trade IPC methods and all Go code are unchanged.

Registry: 12 homes moved (Inspector 3, War Room Trade desk 1, Draft room 8), 15 newly parked;
11 previously retired offers-history/counterparties rows explicitly marked PARKED 11-05 trade.
IDs and names are in PARKED-REPORTS-2026-11-05.md. PLANS is a placeholder, not saved-plan parity.
HQ My moves retains existing receipt plans, but saving new proposals/versions has no UI home.
Historical draft analytics are parked rather than passed off as the new room’s functionality.

Evidence: TypeScript clean; full Vitest leaves only the four permitted inherited failures;
plain make build passes. Disposable Legacy NFL app copy and MFL mirror: Home and War Room
rails have no Trade Floor; Trade desk, Market, Draft room and Plans open. MFL checks read-only.
Native draft displays its existing refusal: 197 season-2026 games lack results. Active draft
actions, live owner offer exit/handoff, historical report parity and a long soak are unverified.
Screenshots/logs are external scratch evidence under soak-1105/sol3. Soak script backed up
and three Trade Floor steps removed; existing four War Room tab steps retained; following
rail coordinates shifted up. No new package, second implementation, IPC or backend mutation.
