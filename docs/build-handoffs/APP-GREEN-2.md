# APP-GREEN-2 — full-suite failures on 786edadf

Branch `build/GREEN-2` from `build/INTEGRATE-1102` (786edadf).

## A. Release contract for history schema 13
- Cause: SCORE-3 (a5d60cfe) added migration 13 (`addLiveEvidence`) with no release row.
- Fix: appended `{Release: 6, Schema: 13, ...}` to `internal/serverversion/version.go`. The previous
  bump (eab87ded, schema 12) did the same thing and nothing else. deploy/server/UPGRADE.md has no
  per-release table, so it was not changed.
- `TestUpgradeBacksUpMigratesAndVerifiesMatchedCheckpoint` ("table live_evidence already exists"):
  **not a production bug.** The test fixture `oldUpgradeFile` (cmd/twr-server/upgrade_test.go:55-61)
  builds a current-schema file, then strips each later migration's objects by hand and sets
  user_version=7. Each schema bump has added its own objects to that strip list (ff57b555 added
  identity, e3bc9d41 added player_record). SCORE-3 did not add the live tables, so the fake v7 file
  still has them and migration 13 fails correctly. Migrations here are deliberately non-idempotent
  (plain CREATE TABLE, with each step's DDL and version marker committed in one transaction), so the fix
  belongs in the fixture, not in a CREATE IF NOT EXISTS. Nothing creates live_evidence outside migration 13.
  **OPEN:** the one-line fixture edit was blocked by the agent's destructive-SQL hook, which matched
  the SQL text inside a test file. On line 59, after the player_record drop, also drop
  `live_board`, `live_chronology` and `live_evidence`, in that order (live_board references live_evidence).
- Migration proof: a reflink copy of the real `history.db` and its WAL started at user_version 12 with
  no live_* tables. Opening it with `db.Open` + `history.Initialize` + `RestoreLiveStats` migrated it
  to 13 (all 4 tables/index + 4 append-only triggers present, integrity_check ok). A second open of the v13
  file succeeded. The copy was deleted.

## B. Goldens regenerated
commands/controls (TWR_WRITE_WIRE=1), hosted/rollover (TWR_UPDATE_HOSTED_WIRE=1). The only
substantive additions are the `league.startingAssets` and `lineup.lock` fields (labels, choices,
safeguards, citations, rollover StartingAssets/Lock). In commands, the safeguard sample field moved from
league.teams to league.startingAssets, the new first field. Everything else is run-to-run values
(timestamps, account ids, revisions, digests) that `normaliseHostedWire` already masks.
No preset-apply lines appeared.

## C. Left alone
internal/commissioner TestOnlyTheCatalogNamesAPreset (stale `.worktrees/fix` scan).

## Verify (-race -p 2, count=1)
serverversion 9/9; app/commissioner 1975/1975; serverhttp 313/313; store/history 525 pass 4 skip;
tools/twr-practice 14/14 (after make build, because it needs frontend/dist); cmd/twr-server 23 pass 1 fail
(the fixture item above). make build: OK.
