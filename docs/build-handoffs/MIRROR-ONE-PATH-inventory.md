# MIRROR-ONE-PATH — inventory (step 1)

Scouted from code by Claude 10-31 21:15 CT on build/WR-wire bb40b41d. **Not yet confirmed on the live app**:
the builder confirms each row on Claude-OS before fixing it, and corrects any row that is wrong.

## Root cause (one finding, not many)

The screens are forked, not one path:

- **HQ:** `HostedLeague.tsx` mounts `NativeFranchiseHQ` (native leagues). `FranchiseHQMount.tsx` mounts
  `FranchiseHQ` (mirror). Two separate screens with separate data code.
- **War Room:** one component, but `native?` branches about 40 times across shell files (WarRoom.tsx 11,
  HQInspector 12, HQRelease 5, HQMoves 5, warroomTradeReads 4, …). Every `native ? X : Y` is a fork.
- **Native branch:** reads come through `native.*` / ledger commands (catalog source `ledger`). These are
  built from ledger records (`commissioner.NativeAssetsReading(records)`). A mirror league has no ledger,
  so they cannot answer it.
- **Mirror branch:** reads report tables (M-008 power/roster value, M-018, M-046 projections, M-063).
  These are joined BY PLAYER NAME and gated on the report title naming the current week
  (`hqPlayerData.projections`). On the real profile they come back empty, so the values show
  "Not available yet".
- **The native HQ never shows values at all:** `NativeFranchiseHQ` passes `values={new Map()}` to HQBoard.

**Fix direction (per the ruling):** make the `native.*` reads (assets, cap, reserve, moves, standings,
week, lineup, projections, calendar, trades, draft) answer a mirror league from the mirror's held
rosters/contracts/rules in the same output shape. Values come from the existing model runs. Then delete
the mirror branches and mount one HQ. Never add a third path.

## Region table

| Screen / region | Mirror read today | Why empty (scouted) | Target read |
|---|---|---|---|
| HQ projections, pts/game, This week | `projections(snapshot, M-046, week)` | name join + current-week title gate; report empty/stale on profile | `projections.read` serving mirror |
| HQ dynasty value, age arc | asset board / M-008 | report rows by name; native HQ passes empty Map | model run values by player ID |
| HQ snap share, contract details | contractsProvider / roster facts | to confirm live | `native.cap.check` contracts, mirror-backed |
| HQ roster, cap, 38/80 | snapshot | REAL (works) | keep, move to `native.assets.read` shape |
| WR league name (rail) | `controlAPI.settings` "League name" row | to confirm live | `commish.rules.read` name |
| WR My shelf, partner select | `readWarRoomCatalog` mirror branch | identities only on native assets | `native.assets.read` + Identities (d8c3f034) |
| WR Best 21, value by age, needs | race M-008 / reports | report empty | model run + teamneeds (D33, merged bb40b41d) |
| WR cap runway 2027/2028 | `capFromReads` | to confirm live | `native.cap.check` |
| WR lenses, preview | trade.offer preview (native only) | mirror branch has none | same command, mirror-backed |
| WR calendar strip | `readWarRoomCalendar` | shows raw codes (LINEUP_LOCK…) | plain labels as HQ/Home |

Debt seen: `OL` maps to no position in identities (from WR-IDENT).
