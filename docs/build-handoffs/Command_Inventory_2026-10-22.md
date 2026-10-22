# Command inventory, 2026-10-22

All 133 bound methods (15 APIs) were traced by Sol 6.1 in six read-only runs against main `d3dabe7a`.
Claude re-checked the leads marked **confirmed** in source. Per-method tables (what it does in plain
words, writes, network, files, guard, proposed who and gravity) are on the Beelink:
`~/fleet/runs/sol61-sol61-warroom-ipc-inventory-2026-10-22/stdout.log` and
`~/fleet/runs/sol61-sol61-warroom-ipc-inv-{c1,c2,k1,k2,k3}-2026-10-22/stdout.log`.

The catalog rows in `internal/app/*/api.go` `Commands()` stay the single list. They are filled in
from these tables once Christopher rules on who may run what.

## Catalog state today
| Field | Filled | `unset` |
|---|---|---|
| ID | 13 | 120 |
| Role | 15 | 118 |
| Gravity | 19 | 114 |
| Version | 0 | 133 |

## What the trace found
| Theme | Where | Why it matters |
|---|---|---|
| Nothing binds the caller to a franchise; commands trust a franchise ID or a plan ID they are given | bids, private lists, lineups, IR/taxi, trades, what-if plans | Harmless with one local user. The first blocker before any CLI, chat or server use. |
| Reads that write | snapshot, pool, bid preview, trade builder preview, research player, league settings, site news, move check | They fetch, archive or create what-if records. A command can't be listed as read-only. |
| "Force" refresh skips the freshness check | comms, race, draft, pool, research, now, cache clear | A loop could hammer MFL. Needs a rate limit, not a role. |
| League reads with no membership check | ledger, people, rules, directory, results, standings | Fine locally; needs scoping before sharing. |
| Engine settings and photo switch have no admin check; `SetPhotos` skips the startup check | `internal/composition/control_host.go:146,159`; `internal/app/sources/api.go:167` | **confirmed** |

## Bugs (confirmed in source)
1. **Open MFL accounting always fails once the host is known.** `LeagueHost()` returns
   `www47.myfantasyleague.com` (`internal/mfl/client.go:345`); `AccountingURL` accepts only `www47`
   (`internal/app/control/reading.go:214`). Its test passes `"www47"`, which hides it.
2. **A private lineup is readable for any franchise.** Setting one records `Visibility: "private"`
   (`internal/composition/native_lineup_app.go:44`); `TargetNativeLineup` reads it with no principal
   (`internal/app/commissioner/native_lineup_api.go:35`).

## Duplicates (same business rule)
- `sources.TargetCacheClear` for comms, race, draft, pool and research does what each feature's own
  forced refresh does (`internal/composition/cache_app.go:25–39`).
- `sources.LoadCrosswalk` and `sources.TargetDataRefresh("crosswalk")` (`internal/app/sources/api.go:95,205`).
- `commissioner.TargetCommissionerLedger` and `TargetCommissionerLedgerFiltered` with an empty filter
  (`internal/app/commissioner/commissioner_api.go:55–64`).

Not duplicates (checked): CommissionerSchedule vs LeagueSchedule; Digests vs Ledger;
contracts.TargetDraftLineup vs commissioner.TargetNativeLineupSet (both say `lineup.set`, but one saves a
move plan and the other sets a native lineup; the shared ID is wrong).

## Methods no screen calls (kept for the CLI)
23, confirmed: 20 commissioner `Target*` (Corrections, LeagueCalendar, LeagueSchedule and 17
`TargetNative*`), `control.TargetSeason`, `market.TargetMarket`, `platform.TargetCheckMoves`.
