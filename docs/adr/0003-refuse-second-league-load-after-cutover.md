# Refuse one-time league-load for a league already moved over

Status: Accepted
Date: 2026-11-20

`twr-server league-load` (`cmd/twr-server/league_load.go`, `internal/leagueload/load.go:64-123`) is strictly a one-time creation and cutover bootstrap: once a league has been loaded and cut over onto the hosted server, a second load or cutover is refused at three independent gates—`leagueFiles.create` (`cmd/twr-server/league_files.go:144-169`, `"already on this server"`), `CheckNativeAppend` (`internal/commissioner/native_seed.go:330-334`, `"this league already has its starting snapshot"`), and `Service.Cutover` / `CheckCutoverAppend` (`internal/leaguemigration/service.go:261-263`, `internal/commissioner/migration.go:58-60`, `"this league has already been moved over from MFL"`). Ongoing synchronization from an MFL mirror is separated into an explicit `twr-server league-refresh` command so an operator can never accidentally wipe an active native league's append-only ledger, claimed seats, or audit chain by re-running the initial load recipe.

## Considered Options

- **Make `league-load` overwrite or reseed an existing league database**: Rejected because overwriting `leagues/<league>/league.db` or allowing a second `commish.league.seed_held` would destroy post-cutover governance events, owner seat claims, and the cryptographic audit chain (`internal/store/history`).
- **Have `league-load` automatically fall back to incremental refresh when the league exists**: Rejected because initial bootstrap (creating `league.db`, sealing/activating transfer, recording `commish.league.cutover`) and incremental refresh (diffing a mirror against live seats and rules with `-dry-run` and conflict rules) have fundamentally different preconditions and blast radii.

## Consequences

- Re-running `league-load` against an existing native league fails closed with a clear operator error at whichever layer is reached first.
- Operators must invoke `league-refresh` explicitly for ongoing MFL mirror reconciliation.
