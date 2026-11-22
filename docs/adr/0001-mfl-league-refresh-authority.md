# MFL league refresh authority split by data kind

Status: Accepted
Date: 2026-11-20

**In plain words:** While a league still plays on MFL, a refresh copies the field (rosters, results, contracts) from MFL but never overwrites what the league decided in TheWarRoom (seats, roles, rules, commissioner edits).

While a league still plays live weeks on MFL after its initial `twr-server league-load` (`cmd/twr-server/league_load.go`), an offline `twr-server league-refresh` reconciles the native league on the hosted server from a fresh read-only MFL mirror (`history.db`, zero network calls) by splitting authority per data kind: the hosted server wins for governance and rules (claimed seats, commissioner seat `franchise.25`, roles, co-commissioner order, deputies, invites, scoring/cap/calendar rules, and prior append-only ledger history/notices), while MFL wins for on-field and market assets (rosters, Active/IR/Taxi status, contracts and salaries, MFL-keyed salary adjustments, draft picks/selections, schedule and completed weekly results advancing `CutoverWeek` monotonically, and played-week lineups `< CutoverWeek`). Franchise names keep in-app `commish.franchise.edit` changes and otherwise take MFL's name; current-week lineups keep an owner's in-app `commish.lineup.set` unless explicitly overwritten; and refresh refuses to run while unsettled in-app trade offers (`PendingTradeCapCheck`, `internal/commissioner/native_override.go`) are open, supports `-dry-run`, and is strictly idempotent (a second run against the same mirror appends zero ledger records).

## Considered Options

- **Wipe and re-run `league-load` (`cmd/twr-server/league_load.go`)**: Rejected because replacing `leagues/<league>/league.db` destroys claimed owner seats, the commissioner's seat (`franchise.25`), role grants, invitations, in-app rule rulings, and ledger notices recorded since the initial load.
- **Rely on existing incremental sync (`SyncSchedule`, `SyncTransactions`, `SyncLineups` in `internal/leaguemigration/service.go`)**: Rejected because `SyncTransactions` adds players without full contract terms and never syncs contract edits, `TYPE=salaryAdjustments`, or draft selections, while `retainBoundary` freezes `CutoverWeek` at the initial load week.
- **Diff desired MFL state against projected native league state and commit one atomic `Ledger.Batch` (`internal/store/history/migration_import.go`)**: Chosen because orthogonal ledger projections (`governanceFold`, `seasonChanges`, `foldNativeAssets`) allow asset, schedule, and lineup corrections (`correct.roster`, `correct.pick`, `schedule.publish`, `commish.lineup.correct`) to reconcile rosters and caps to zero diffs without touching seats or rules.

## Consequences

- Operators can safely refresh a live MFL-backed league repeatedly without losing owner admissions, role grants, or in-app rule rulings.
- Refresh must be blocked whenever pending in-app trade offers exist so MFL roster diffs cannot invalidate in-flight cap checks.
